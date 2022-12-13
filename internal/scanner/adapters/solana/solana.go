package solana

import (
	"context"
	"fmt"

	"extrnode-be/internal/pkg/storage"
	"extrnode-be/internal/scanner/scaners/asn"
)

const (
	solanaBlockchain = "solana"
	httpPort         = 80
)

type SolanaAdapter struct {
	storage      storage.PgStorage
	blockchainID int
	ctx          context.Context
}

func NewSolanaAdapter(ctx context.Context, storage storage.PgStorage) (*SolanaAdapter, error) {
	blockchain, err := storage.GetBlockchainByName(solanaBlockchain)
	if err != nil {
		return nil, fmt.Errorf("GetBlockchainByName: %s", err)
	}
	if blockchain.ID == 0 {
		return nil, fmt.Errorf("empty blockchain.ID")
	}

	return &SolanaAdapter{
		storage:      storage,
		blockchainID: blockchain.ID,
		ctx:          ctx,
	}, nil
}

func (a *SolanaAdapter) Scan(host string) error {
	hostPeer, err := a.HostAsPeer(host)
	if err != nil {
		return err
	}

	err = a.ScanMethods(hostPeer)
	if err != nil {
		return err
	}

	return nil
}

func (s *SolanaAdapter) GetNewNodes(host string, isAlive bool) error {
	if !isAlive {
		return nil
	}
	nodes, err := s.getNodes(host)
	if err != nil {
		return err
	}

	nodes, err = s.filterNodes(nodes)
	if err != nil {
		return fmt.Errorf("filterNodes: %s", err)
	}
	records, err := asn.GetWhoisRecords(nodes)
	if err != nil {
		return err
	}

	err = s.insertData(records)
	if err != nil {
		return err
	}

	return nil
}
