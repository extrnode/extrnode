package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage/clickhouse"
	"extrnode-be/internal/pkg/storage/clickhouse/delayed_insertion"
	"extrnode-be/internal/pkg/storage/postgres"
	"extrnode-be/internal/pkg/storage/sqlite"
	"extrnode-be/internal/scanner/adapters"
	"extrnode-be/internal/scanner/adapters/solana"
	"extrnode-be/internal/scanner/scaners/nmap"
)

type scanner struct {
	cfg       config.Config
	pgStorage postgres.Storage
	slStorage sqlite.Storage

	taskQueue     chan scannerTask
	nmapTaskQueue chan scannerTask

	waitGroup *sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc
	adapters  map[chainType]adapters.Adapter
}

const (
	collectorInterval = 5 * time.Minute
)

func NewScanner(cfg config.Config) (*scanner, error) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	pgStorage, err := postgres.New(ctx, cfg.PG)
	if err != nil {
		return nil, fmt.Errorf("PG storage init: %s", err)
	}
	slStorage, err := sqlite.New(ctx, cfg.SL)
	if err != nil {
		return nil, fmt.Errorf("SL storage init: %s", err)
	}
	chStorage, err := clickhouse.New(cfg.CH.DSN, cfg.Scanner.Hostname)
	if err != nil {
		return nil, fmt.Errorf("CH storage init: %s", err)
	}

	scannerMethodsCollector := delayed_insertion.New[clickhouse.ScannerMethod](ctx, cfg, chStorage, collectorInterval)
	scannerPeersCollector := delayed_insertion.New[clickhouse.ScannerPeer](ctx, cfg, chStorage, collectorInterval)
	solanaAdapter, err := solana.NewSolanaAdapter(ctx, slStorage, scannerMethodsCollector, scannerPeersCollector)
	if err != nil {
		return nil, fmt.Errorf("NewSolanaAdapter: %s", err)
	}

	return &scanner{
		cfg:           cfg,
		pgStorage:     pgStorage,
		slStorage:     slStorage,
		taskQueue:     make(chan scannerTask),
		nmapTaskQueue: make(chan scannerTask),
		waitGroup:     &sync.WaitGroup{},
		ctx:           ctx,
		ctxCancel:     cancelFunc,
		adapters:      map[chainType]adapters.Adapter{chainTypeSolana: solanaAdapter},
	}, nil
}

func (s *scanner) Run() error {
	s.runWithWaitGroup(s.ctx, s.scheduleScans)

	for i := 0; i < s.cfg.Scanner.ThreadsNum; i++ {
		s.runWithWaitGroup(s.ctx, s.runScanner)
	}

	return nil
}

func (s *scanner) RunNmap() error {
	s.runWithWaitGroup(s.ctx, s.scheduleNmap)

	for i := 0; i < s.cfg.Scanner.ThreadsNum; i++ {
		s.runWithWaitGroup(s.ctx, s.runNmap)
	}

	return nil
}
func (s *scanner) CheckOutdatedNodes() error {
	for {
		for _, a := range s.adapters {
			err := a.CheckOutdatedNodes()
			if err != nil {
				return err
			}
		}

		select {
		case <-s.ctx.Done():
			log.Logger.Scanner.Info("stopping CheckOutdatedNodes")
			return nil

		case <-time.After(checkOutdatedNodesInterval):
			continue
		}
	}
}

func (s *scanner) getAdapter(task scannerTask) (adapter adapters.Adapter, ok bool) {
	adapter, ok = s.adapters[task.chain]
	if !ok {
		log.Logger.Scanner.Errorf("fail to get adapter for %s", task.chain)
	}

	return adapter, ok
}

func (s *scanner) runScanner(ctx context.Context) {
	var isIdle bool

	for {
		select {
		case <-ctx.Done():
			log.Logger.Scanner.Info("stopping scanner")
			return

		case task := <-s.taskQueue:
			log.Logger.Scanner.Debugf("Scanning peer %s", task.peer.Address)

			adapter, ok := s.getAdapter(task)
			if !ok {
				continue
			}

			err := adapter.GetNewNodes(task.peer)
			if err != nil {
				log.Logger.Scanner.Errorf("GetNewNodes (%s %s:%d): %s", task.chain, task.peer.Address, task.peer.Port, err)
				// continue not needed
			}

			err = adapter.Scan(task.peer)
			if err != nil {
				log.Logger.Scanner.Errorf("Scan (%s %s:%d): %s", task.chain, task.peer.Address, task.peer.Port, err)
			}

		case <-time.After(time.Minute):
			if !isIdle {
				log.Logger.Scanner.Debug("scanner: no more tasks")
				isIdle = true
			}
		}
	}
}

func (s *scanner) runNmap(ctx context.Context) {
	var isIdle bool

	for {
		select {
		case <-ctx.Done():
			log.Logger.Scanner.Info("stopping nmap")
			return

		case task := <-s.nmapTaskQueue:
			isIdle = false
			err := nmap.ScanAndInsertPorts(s.ctx, s.slStorage, task.peer)
			if err != nil {
				log.Logger.Scanner.Errorf("NmapCheck (%s %s:%d): %s", task.chain, task.peer.Address, task.peer.Port, err)
			}

		case <-time.After(time.Minute):
			if !isIdle {
				log.Logger.Scanner.Debug("nmap: no more tasks")
				isIdle = true
			}
		}
	}
}

func (s *scanner) runWithWaitGroup(ctx context.Context, fn func(context.Context)) {
	s.waitGroup.Add(1)
	go func() {
		fn(ctx)
		s.waitGroup.Done()
	}()
}

func (s *scanner) WaitGroup() *sync.WaitGroup {
	return s.waitGroup
}

func (s *scanner) Stop() (err error) {
	s.ctxCancel()
	return nil
}

func (s *scanner) LoadPostgresData() error {
	log.Logger.Scanner.Info("LoadPostgresData")
	solanaBlockchain, err := s.pgStorage.GetBlockchainByName("solana")
	if err != nil {
		return fmt.Errorf("GetBlockchainByName: %s", err)
	}
	endpoints, err := s.pgStorage.GetEndpoints()
	if err != nil {
		return fmt.Errorf("GetEndpoints: %s", err)
	}

	for _, e := range endpoints {
		countryID, err := s.slStorage.GetOrCreateGeoCountry(e.CountryAlpha2, e.CountryAlpha3, e.CountryName)
		if err != nil {
			return fmt.Errorf("GetOrCreateGeoCountry: %s; req %+v", err, e)
		}

		networkID, err := s.slStorage.GetOrCreateGeoNetwork(countryID, e.NetworkMask, e.NetworkAs, e.NetworkName)
		if err != nil {
			return fmt.Errorf("GetOrCreateGeoNetwork: %s; req %+v cntId %d", err, e, countryID)
		}

		ipID, err := s.slStorage.GetOrCreateIP(networkID, e.Address)
		if err != nil {
			return fmt.Errorf("GetOrCreateIP: %s; req %+v; networkID %d", err, e, networkID)
		}

		_, err = s.slStorage.GetOrCreatePeer(solanaBlockchain.ID, ipID, e.Port, e.Version, e.IsRpc, e.IsAlive, e.IsSSL, e.IsMainNet, e.IsValidator, e.NodePubkey)
		if err != nil {
			return fmt.Errorf("GetOrCreatePeer: %s; req %+v blcId %d ipId %d", err, e, solanaBlockchain.ID, ipID)
		}
	}

	log.Logger.Scanner.Infof("loaded %d rows to sqlite", len(endpoints))

	return nil
}
