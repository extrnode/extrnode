package scanner

import (
	"context"
	"time"

	"extrnode-be/internal/pkg/log"
)

const scannerInterval = time.Second * 5

type chainType string

const chainTypeSolana = "solana"

type scannerTask struct {
	host  string
	chain chainType
}

func (s *scanner) scheduleScans(ctx context.Context) {
	for {
		// TODO: get from database all hosts that need to be parsed and stream to worker channel
		s.taskQueue <- scannerTask{host: "178.237.58.142", chain: chainTypeSolana}

		select {
		case <-ctx.Done():
			log.Logger.Scanner.Info("stopping scheduler")
			return

		case <-time.After(scannerInterval):
			continue
		}
	}
}
