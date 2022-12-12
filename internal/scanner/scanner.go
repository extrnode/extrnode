package scanner

import (
	"context"
	"fmt"
	"sync"

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage"
	"extrnode-be/internal/scanner/adapters"
	"extrnode-be/internal/scanner/adapters/solana"
)

type scanner struct {
	cfg     config.Config
	storage storage.PgStorage

	taskQueue chan scannerTask

	waitGroup *sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func NewScanner(cfg config.Config) (*scanner, error) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	s, err := storage.New(ctx, cfg.Postgres)
	if err != nil {
		cancelFunc()
		return nil, fmt.Errorf("storage init: %s", err)
	}

	return &scanner{
		cfg:       cfg,
		storage:   s,
		taskQueue: make(chan scannerTask),
		waitGroup: &sync.WaitGroup{},
		ctx:       ctx,
		ctxCancel: cancelFunc,
	}, nil
}

func (s *scanner) Run() error {
	s.runWithWaitGroup(s.ctx, s.scheduleScans)

	for i := 0; i < s.cfg.Scanner.ThreadsNum; i++ {
		s.runWithWaitGroup(s.ctx, s.runScanner)
	}

	return nil
}

func (s *scanner) runScanner(ctx context.Context) {
	var err error

	for {
		select {
		case <-ctx.Done():
			log.Logger.Scanner.Info("stopping scanner")
			return

		case task := <-s.taskQueue:
			var adapter adapters.Adapter
			switch task.chain {
			case chainTypeSolana:
				adapter, err = solana.NewSolanaAdapter(s.ctx, s.storage)
				if err != nil {
					log.Logger.Scanner.Errorf("NewSolanaAdapter (%s %s): %s", task.chain, task.host, err)
					continue
				}
			default:
				log.Logger.Scanner.Errorf("adapter not found for chain: %s", task.chain)
				continue
			}

			err = adapter.GetNewNodes(task.host)
			if err != nil {
				log.Logger.Scanner.Errorf("GetNewNodes (%s %s): %s", task.chain, task.host, err)
				// continue not needed
			}

			err = adapter.Scan(task.host)
			if err != nil {
				log.Logger.Scanner.Errorf("Scan (%s %s): %s", task.chain, task.host, err)
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
