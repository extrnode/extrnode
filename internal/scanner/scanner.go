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
	var wg sync.WaitGroup
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
		waitGroup: &wg,
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
	for {
		select {
		case <-ctx.Done():
			log.Logger.Scanner.Info("stopping scanner")
			return

		case task := <-s.taskQueue:
			var adapter adapters.Adapter
			switch task.chain {
			case chainTypeSolana:
				adapter = &solana.SolanaAdapter{}
			}

			if adapter == nil {
				log.Logger.Scanner.Errorf("adapter not found for chain: %s", task.chain)
				continue
			}

			err := adapter.Scan(task.host)
			if err != nil {
				log.Logger.Scanner.Errorf("scanner error. node %s. chain %s: %s", task.host, task.chain, err)
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
