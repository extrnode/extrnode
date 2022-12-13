package scanner

import (
	"context"
	"fmt"
	"time"

	"extrnode-be/internal/pkg/log"
)

const scannerInterval = time.Minute * 10

type chainType string

const chainTypeSolana = "solana"

type scannerTask struct {
	host    string
	isAlive bool
	chain   chainType
}

func (s *scanner) scheduleScans(ctx context.Context) {
	for {
		res, err := s.storage.GetPeers()
		if err != nil {
			log.Logger.Scanner.Fatalf("GetPeers: %s", err)
		}

		log.Logger.Scanner.Debugf("scheduleScans: get %d peers. Creating scanner tasks", len(res))

		for _, r := range res {
			schema := "http://"
			if r.IsSSL {
				schema = "https://"
			}

			s.taskQueue <- scannerTask{host: fmt.Sprintf("%s%s:%d", schema, r.Address.String(), r.Port), isAlive: r.IsAlive, chain: chainType(r.BlockchainName)}
		}

		select {
		case <-ctx.Done():
			log.Logger.Scanner.Info("stopping scheduler")
			return

		case <-time.After(scannerInterval):
			continue
		}
	}
}
