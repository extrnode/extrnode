package solana

import (
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage"
)

var solanaMainNetGenesisHash = solana.MustHashFromBase58("5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d")

func (a *SolanaAdapter) updatePeerInfo(peer storage.PeerWithIpAndBlockchain, now time.Time, isALive, isRpc, isSSL, isMainNet, isValidator bool) error {
	err := a.storage.CreateScannerPeer(peer.ID, now, 0, isALive)
	if err != nil {
		return fmt.Errorf("CreateScannerPeer: %s", err)
	}

	err = a.storage.UpdatePeerByID(peer.ID, isRpc, isALive, isSSL, isMainNet, isValidator)
	if err != nil {
		return fmt.Errorf("UpdatePeerByID: %s", err)
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
	rpcClient, isSSL, err := a.getValidRpc(peer)
	if err != nil {
		log.Logger.Scanner.Errorf("ScanMethods finished with error prdId %d: %s", peer.ID, reformatSolanaRpcError(err))

		err = a.updatePeerInfo(peer, now, false, false, isSSL, peer.IsMainNet, isValidator)
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
		log.Logger.Scanner.Debugf("ScanMethods skip dennet peer prdId %d", peer.ID)

		err = a.updatePeerInfo(peer, now, false, false, isSSL, false, isValidator)
		if err != nil {
			return fmt.Errorf("updatePeerInfo 2: %s", err)
		}

		return nil
	}

	var isAlive bool
	isRpc := true
	for m := range methods {
		responseValid, responseTime, statusCode, err := checkRpcMethod(TopRpcMethod(m), rpcClient, a.ctx)
		if err != nil {
			log.Logger.Scanner.Warn("ScanMethods: checkRpcMethod: ", err)
			continue
		}

		if responseValid {
			isAlive = true
			err = a.storage.UpsertRpcPeerMethod(peer.ID, methods[m], responseTime)
			if err != nil {
				return fmt.Errorf("CreateRpcPeerMethod: %s", err)
			}
		} else {
			isRpc = false
			err = a.storage.DeleteRpcPeerMethod(peer.ID, methods[m])
			if err != nil {
				return fmt.Errorf("DeleteRpcPeerMethod: %s", err)
			}
		}
		err = a.storage.CreateScannerMethod(peer.ID, methods[m], now, 0, responseTime, statusCode, responseValid)
		if err != nil {
			return fmt.Errorf("CreateScannerMethod: %s", err)
		}
	}

	// isMainNet == true because devnet is skipped
	err = a.updatePeerInfo(peer, now, isAlive, isRpc, isSSL, true, isValidator)
	if err != nil {
		return fmt.Errorf("updatePeerInfo 3: %s", err)
	}

	return nil
}
