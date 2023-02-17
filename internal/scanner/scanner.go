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
	"extrnode-be/internal/scanner/adapters"
	"extrnode-be/internal/scanner/adapters/solana"
	"extrnode-be/internal/scanner/scaners/nmap"
)

type scanner struct {
	cfg       config.Config
	pgStorage postgres.Storage

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
	chStorage, err := clickhouse.New(cfg.CH.DSN, cfg.Scanner.Hostname)
	if err != nil {
		return nil, fmt.Errorf("CH storage init: %s", err)
	}

	scannerMethodsCollector := delayed_insertion.New[clickhouse.ScannerMethod](ctx, cfg, chStorage, collectorInterval)
	scannerPeersCollector := delayed_insertion.New[clickhouse.ScannerPeer](ctx, cfg, chStorage, collectorInterval)
	solanaAdapter, err := solana.NewSolanaAdapter(ctx, pgStorage, scannerMethodsCollector, scannerPeersCollector)
	if err != nil {
		return nil, fmt.Errorf("NewSolanaAdapter: %s", err)
	}

	return &scanner{
		cfg:           cfg,
		pgStorage:     pgStorage,
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
			err := nmap.ScanAndInsertPorts(s.ctx, s.pgStorage, task.peer)
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
