package solana

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"extrnode-be/internal/pkg/storage"
	"extrnode-be/internal/scanner/scaners/asn"
)

const (
	solanaBlockchain = "solana"
	httpPort         = 80
	slotShift        = 2000
)

type SolanaAdapter struct {
	ctx                    context.Context
	storage                storage.PgStorage
	blockchainID           int
	voteAccountsNodePubkey map[string]struct{} // solana.PublicKey
	signatureForAddress    solana.Signature
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
		return fmt.Errorf("getNodes: %s", err)
	}

	nodes, err = a.filterAndUpdateNodes(nodes)
	if err != nil {
		return fmt.Errorf("filterAndUpdateNodes: %s", err)
	}

	records, err := asn.GetWhoisRecords(nodes)
	if err != nil {
		return fmt.Errorf("GetWhoisRecords: %s", err)
	}

	err = a.insertData(records)
	if err != nil {
		return fmt.Errorf("insertData: %s", err)
	}

	return nil
}

func (a *SolanaAdapter) BeforeRun() error {
	voteAccounts, err := a.baseRpcClient.GetVoteAccounts(a.ctx, &rpc.GetVoteAccountsOpts{Commitment: rpc.CommitmentFinalized})
	if err != nil {
		return fmt.Errorf("GetVoteAccounts: %s", err)
	}

	res, err := a.baseRpcClient.GetSignaturesForAddress(a.ctx, testKey3)
	if len(res) <= 0 || err != nil {
		return fmt.Errorf("GetSignaturesForAddress: %s", err)
	}
	slot, err := a.baseRpcClient.GetSlot(a.ctx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("GetSlot: %s", err)
	}

	var (
		i     uint64 = 0
		block *rpc.GetBlockResult
	)
	slot = slot - slotShift
	ops := rpc.GetBlockOpts{
		MaxSupportedTransactionVersion: &i,
		TransactionDetails:             rpc.TransactionDetailsSignatures,
	}
	if block, err = a.baseRpcClient.GetBlockWithOpts(a.ctx, slot, &ops); err != nil || block == nil || len(block.Signatures) <= 0 {
		return fmt.Errorf("GetBlockWithOpts: %s", err)
	}

	a.signatureForAddress = block.Signatures[0]
	a.voteAccountsNodePubkey = make(map[string]struct{}, len(voteAccounts.Current))
	for _, va := range voteAccounts.Current {
		a.voteAccountsNodePubkey[va.NodePubkey.String()] = struct{}{}
	}

	return nil
}
