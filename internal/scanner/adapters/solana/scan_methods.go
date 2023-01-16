package solana

import (
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"

	"extrnode-be/internal/pkg/storage"
)

var solanaMainNetGenesisHash = solana.MustHashFromBase58("5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d")

func (a *SolanaAdapter) updatePeerInfo(peer storage.PeerWithIpAndBlockchain, now time.Time, isALive, isRpc, isSSL, isMainNet, isValidator, deleteAllRpcMethods bool, version string) error {
	err := a.storage.CreateScannerPeer(peer.ID, now, 0, isALive)
	if err != nil {
		return fmt.Errorf("CreateScannerPeer: %s", err)
	}

	err = a.storage.UpdatePeerByID(peer.ID, isRpc, isALive, isSSL, isMainNet, isValidator, version)
	if err != nil {
		return fmt.Errorf("UpdatePeerByID: %s", err)
	}

	if deleteAllRpcMethods {
		err = a.storage.DeleteRpcPeerMethod(peer.ID, nil)
		if err != nil {
			return fmt.Errorf("DeleteRpcPeerMethod: %s", err)
		}
	}

	return nil
}

func (a *SolanaAdapter) ScanMethods(peer storage.PeerWithIpAndBlockchain) error {
	now := time.Now()
	methods, err := a.storage.GetRpcMethodsMapByBlockchainID(a.blockchainID)
	if err != nil {
		return fmt.Errorf("GetRpcMethodsMapByBlockchainID: %s", err)
	}

	_, isValidator := a.voteAccountsNodePubkey[peer.NodePubkey] // peer.NodePubkey can be empty on first iteration
	rpcClient, isSSL, version, err := a.getValidRpc(peer)
	if err != nil {
		err = a.updatePeerInfo(peer, now, false, false, isSSL, peer.IsMainNet, isValidator, true, version) // version may be empty
		if err != nil {
			return fmt.Errorf("updatePeerInfo 1: %s", err)
		}

		return nil
	}

	hash, err := rpcClient.GetGenesisHash(a.ctx)
	if err != nil {
		return fmt.Errorf("GetGenesisHash: %s", reformatSolanaRpcError(err))
	}
	// skip method checking for devnet
	if hash != solanaMainNetGenesisHash {
		err = a.updatePeerInfo(peer, now, false, false, isSSL, false, isValidator, true, version)
		if err != nil {
			return fmt.Errorf("updatePeerInfo 2: %s", err)
		}

		return nil
	}

	var isAlive bool
	isRpc := true
	for mName, mID := range methods {
		responseValid, responseTime, statusCode, err := checkRpcMethod(TopRpcMethod(mName), rpcClient, a.ctx)
		if err != nil || !responseValid { // responseValid always == false when err != nil
			isRpc = false
			err = a.storage.DeleteRpcPeerMethod(peer.ID, &mID)
			if err != nil {
				return fmt.Errorf("DeleteRpcPeerMethod: %s", err)
			}
		} else {
			isAlive = true
			err = a.storage.UpsertRpcPeerMethod(peer.ID, mID, responseTime)
			if err != nil {
				return fmt.Errorf("UpsertRpcPeerMethod: %s", err)
			}
		}

		err = a.storage.CreateScannerMethod(peer.ID, mID, now, 0, responseTime, statusCode, responseValid)
		if err != nil {
			return fmt.Errorf("CreateScannerMethod: %s", err)
		}
	}

	// isMainNet == true because devnet is skipped
	err = a.updatePeerInfo(peer, now, isAlive, isRpc, isSSL, true, isValidator, false, version)
	if err != nil {
		return fmt.Errorf("updatePeerInfo 3: %s", err)
	}

	return nil
}
