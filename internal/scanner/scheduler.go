package scanner

import (
	"context"
	"fmt"
	"time"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage/sqlite"
)

type chainType string

const (
	chainTypeSolana            chainType = "solana"
	scannerInterval                      = time.Hour
	nmapInterval                         = 3 * time.Hour
	checkOutdatedNodesInterval           = time.Minute
)

type scannerTask struct {
	peer  sqlite.PeerWithIpAndBlockchain
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

func (s *scanner) scheduleNmap(ctx context.Context) {
	for {
		peers, err := s.slStorage.GetPeers(true, nil, nil, nil, nil)
		if err != nil {
			log.Logger.Scanner.Fatalf("scheduleNmap: GetPeers: %s", err)
		}

		log.Logger.Scanner.Debugf("scheduleNmap: get %d uniq IP for nmap. Creating scanner tasks", len(peers))

		for _, p := range peers {
			select {
			case <-ctx.Done():
				log.Logger.Scanner.Info("stopping nmap scheduler")
				return

			case s.nmapTaskQueue <- scannerTask{peer: p, chain: chainType(p.BlockchainName)}:
			}
		}

		select {
		case <-ctx.Done():
			log.Logger.Scanner.Info("stopping nmap scheduler")
			return

		case <-time.After(nmapInterval):
			continue
		}
	}
}

func (s *scanner) scheduleScans(ctx context.Context) {
	for {
		peers, err := s.slStorage.GetPeers(false, nil, nil, nil, nil)
		if err != nil {
			log.Logger.Scanner.Fatalf("scheduleScans: GetPeers: %s", err)
		}

		log.Logger.Scanner.Debugf("scheduleScans: get %d peers. Creating scanner tasks", len(peers))

		for _, p := range peers {
			select {
			case <-ctx.Done():
				log.Logger.Scanner.Info("stopping scanner scheduler")
				return

			default:
				log.Logger.Scanner.Debugf("Scheduling scan for peer %s", p.Address)

				s.taskQueue <- scannerTask{peer: p, chain: chainType(p.BlockchainName)}
			}
		}

		select {
		case <-ctx.Done():
			log.Logger.Scanner.Info("stopping scanner scheduler")
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
