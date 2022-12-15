package solana

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go/rpc"

	"extrnode-be/internal/pkg/storage"
	"extrnode-be/internal/scanner/scaners/asn"
)

const (
	solanaBlockchain = "solana"
	httpPort         = 80
)

type SolanaAdapter struct {
	ctx                    context.Context
	storage                storage.PgStorage
	blockchainID           int
	voteAccountsNodePubkey map[string]struct{} // solana.PublicKey
	baseRpcClient          *rpc.Client
}

func NewSolanaAdapter(ctx context.Context, storage storage.PgStorage) (*SolanaAdapter, error) {
	blockchain, err := storage.GetBlockchainByName(solanaBlockchain)
	if err != nil {
		return nil, fmt.Errorf("GetBlockchainByName: %s", err)
	}
	if blockchain.ID == 0 {
		return nil, fmt.Errorf("empty blockchain.ID")
	}

	sa := SolanaAdapter{
		storage:       storage,
		blockchainID:  blockchain.ID,
		ctx:           ctx,
		baseRpcClient: createRpcWithTimeout(rpc.MainNetBeta_RPC),
	}

	err = sa.BeforeRun()
	if err != nil {
		return nil, fmt.Errorf("BeforeRun: %s", err)
	}

	return &sa, nil
}

func (a *SolanaAdapter) Scan(peer storage.PeerWithIpAndBlockchain) error {
	err := a.ScanMethods(peer)
	if err != nil {
		return err
	}

	return nil
}

func (a *SolanaAdapter) GetNewNodes(peer storage.PeerWithIpAndBlockchain) error {
	if !peer.IsAlive || !peer.IsMainNet {
		return nil
	}

	nodes, err := a.getNodes(createNodeUrl(peer, peer.IsSSL))
	if err != nil {
		return err
	}

	nodes, err = a.filterAndUpdateNodes(nodes)
	if err != nil {
		return fmt.Errorf("filterAndUpdateNodes: %s", err)
	}

	records, err := asn.GetWhoisRecords(nodes)
	if err != nil {
		return err
	}

	err = a.insertData(records)
	if err != nil {
		return err
	}

	return nil
}

func (a *SolanaAdapter) BeforeRun() error {
	voteAccounts, err := a.baseRpcClient.GetVoteAccounts(a.ctx, &rpc.GetVoteAccountsOpts{Commitment: rpc.CommitmentFinalized})
	if err != nil {
		return fmt.Errorf("GetVoteAccounts: %s", err)
	}

	a.voteAccountsNodePubkey = make(map[string]struct{}, len(voteAccounts.Current))
	for _, va := range voteAccounts.Current {
		a.voteAccountsNodePubkey[va.NodePubkey.String()] = struct{}{}
	}

	return nil
}
