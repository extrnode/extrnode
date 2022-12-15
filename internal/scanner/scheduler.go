package scanner

import (
	"context"
	"fmt"
	"time"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage"
)

const scannerInterval = time.Hour * 4

type chainType string

const chainTypeSolana chainType = "solana"

type scannerTask struct {
	peer  storage.PeerWithIpAndBlockchain
	chain chainType
}

func (s *scanner) updateAdapters() error {
	for chain, a := range s.adapters {
		err := a.BeforeRun()
		if err != nil {
			return fmt.Errorf("BeforeRun %s: %s", chain, err)
		}
	}

	return nil
}

func (s *scanner) scheduleScans(ctx context.Context) {
	for {
		peers, err := s.storage.GetPeers()
		if err != nil {
			log.Logger.Scanner.Fatalf("scheduleScans: GetPeers: %s", err)
		}

		log.Logger.Scanner.Debugf("scheduleScans: get %d peers. Creating scanner tasks", len(peers))

		for _, p := range peers {
			s.taskQueue <- scannerTask{peer: p, chain: chainType(p.BlockchainName)}
		}

		select {
		case <-ctx.Done():
			log.Logger.Scanner.Info("stopping scheduler")
			return

		case <-time.After(scannerInterval):
			err = s.updateAdapters()
			if err != nil {
				log.Logger.Scanner.Fatalf("scheduleScans: updateAdapters: %s", err)
			}

			continue
		}
	}
}
