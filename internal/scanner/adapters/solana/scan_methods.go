package solana

import (
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage/postgres"
)

var solanaMainNetGenesisHash = solana.MustHashFromBase58("5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d")

func (a *SolanaAdapter) updatePeerInfo(peer postgres.PeerWithIpAndBlockchain, now time.Time, isALive, isRpc, isSSL, isMainNet, isValidator, deleteAllRpcMethods bool, version string) error {
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

	if peer.IsRpc != isRpc {
		log.Logger.Scanner.Debugf("peer updated %s:%d: isRpc %t", peer.Address, peer.Port, isRpc)
	}

	return nil
}

func (a *SolanaAdapter) ScanMethods(peer postgres.PeerWithIpAndBlockchain) error {
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

	isRpc := true
	for mName, mID := range methods {
		responseValid, responseTime, statusCode, err := a.checkRpcMethod(mName, rpcClient)
		if err != nil || !responseValid { // responseValid always == false when err != nil
			if err != nil {
				log.Logger.Scanner.Errorf("checkRpcMethod %s %s:%d: %s", mName, peer.Address, peer.Port, err)
			}
			isRpc = false
			err = a.storage.DeleteRpcPeerMethod(peer.ID, &mID)
			if err != nil {
				return fmt.Errorf("DeleteRpcPeerMethod: %s", err)
			}
		} else {
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
	err = a.updatePeerInfo(peer, now, true, isRpc, isSSL, true, isValidator, false, version)
	if err != nil {
		return fmt.Errorf("updatePeerInfo 3: %s", err)
	}

	return nil
}
