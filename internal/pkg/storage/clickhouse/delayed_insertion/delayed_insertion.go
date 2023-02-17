package delayed_insertion

import (
	"context"
	"fmt"
	"sync"
	"time"

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage/clickhouse"
)

const (
	flushAmount = 1000
)

type (
	collectorPossibleTypes interface {
		clickhouse.Stat | clickhouse.ScannerMethod | clickhouse.ScannerPeer
	}
	Collector[T collectorPossibleTypes] struct {
		ctx           context.Context
		cfg           config.Config
		chStorage     *clickhouse.Storage
		mx            sync.Mutex
		flushInterval time.Duration
		cache         []T
	}
)

func New[T collectorPossibleTypes](ctx context.Context, cfg config.Config, chStorage *clickhouse.Storage, flushInterval time.Duration) (c *Collector[T]) {
	c = &Collector[T]{
		ctx:           ctx,
		cfg:           cfg,
		chStorage:     chStorage,
		flushInterval: flushInterval,
		cache:         make([]T, 0, flushAmount),
	}

	if chStorage == nil {
		return
	}

	go c.start()

	return
}

func (c *Collector[T]) start() {
	for {
		select {
		case <-c.ctx.Done():
			err := c.flushData()
			if err != nil {
				log.Logger.Collector.Errorf("flushData: %s", err)
			}

			return

		case <-time.After(c.flushInterval):
			err := c.flushData()
			if err != nil {
				log.Logger.Collector.Errorf("flushData: %s", err)
			}
		}
	}
}

func (c *Collector[T]) flushData() error {
	if c.chStorage == nil {
		return nil
	}

	entries := c.getCachedEntries()
	if len(entries) == 0 {
		return nil
	}

	var (
		err     error
		caller  string
		timeNow = time.Now()
	)
	switch e := any(entries).(type) {
	case []clickhouse.Stat:
		caller = "InsertStats"
		err = c.chStorage.BatchInsertStats(e)
	case []clickhouse.ScannerMethod:
		caller = "InsertScannerMethods"
		err = c.chStorage.BatchInsertScannerMethods(e)
	case []clickhouse.ScannerPeer:
		caller = "InsertScannerPeers"
		err = c.chStorage.BatchInsertScannerPeers(e)
	default:
		return fmt.Errorf("unknow type to handle: %T", e)
	}
	if err != nil {
		return fmt.Errorf("%s: %s", caller, err)
	}

	log.Logger.Collector.Debugf("fin flushData %s len %d. Elapsed %s", caller, len(entries), time.Since(timeNow))

	return nil
}
