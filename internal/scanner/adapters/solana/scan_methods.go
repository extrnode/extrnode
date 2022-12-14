package solana

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage"
)

func (a *SolanaAdapter) HostAsPeer(host string) (hostPeer storage.PeerWithIp, err error) {
	hostURL, err := url.Parse(host)
	if err != nil {
		return hostPeer, err
	}

	host, port, err := net.SplitHostPort(hostURL.Host)
	if err != nil {
		return hostPeer, err
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return hostPeer, err
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return hostPeer, err
	}

	hostPeers, err := a.storage.GetPeerByPortAndIP(portInt, ip)
	if err != nil {
		return hostPeer, err
	}

	return hostPeers, nil
}

func (a *SolanaAdapter) ScanMethods(host storage.PeerWithIp) error {
	log.Logger.Scanner.Debugf("start ScanMethods")
	defer log.Logger.Scanner.Debugf("fin ScanMethods")

	now := time.Now()
	methods, err := a.storage.GetRpcMethodsMapByBlockchainID(a.blockchainID)
	if err != nil {
		return fmt.Errorf("GetRpcMethodsMapByBlockchainID: %s", err)
	}

	rpcClient, isSSL, err := a.validRpc(host)
	if err != nil {
		log.Logger.Scanner.Errorf("ScanMethods finished with error: %s", err)
		err = a.storage.CreateScannerPeer(host.ID, now, 0, false)
		if err != nil {
			return fmt.Errorf("CreateScannerPeer: %s", err)
		}

		err = a.storage.UpdatePeerByID(host.ID, false, false, isSSL)
		if err != nil {
			return fmt.Errorf("UpdatePeerByID: %s", err)
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
			err = a.storage.CreateRpcPeerMethod(host.ID, methods[m])
			if err != nil {
				return fmt.Errorf("CreateRpcPeerMethod: %s", err)
			}
		} else {
			isRpc = false
			err = a.storage.DeleteRpcPeerMethod(host.ID, methods[m])
			if err != nil {
				return fmt.Errorf("DeleteRpcPeerMethod: %s", err)
			}
		}
		err = a.storage.CreateScannerMethod(host.ID, methods[m], now, 0, responseTime, statusCode, responseValid)
		if err != nil {
			return fmt.Errorf("CreateScannerMethod: %s", err)
		}
	}

	err = a.storage.CreateScannerPeer(host.ID, now, 0, isAlive)
	if err != nil {
		return fmt.Errorf("CreateScannerPeer: %s", err)
	}

	err = a.storage.UpdatePeerByID(host.ID, isRpc, isAlive, isSSL)
	if err != nil {
		return fmt.Errorf("UpdatePeerByID: %s", err)
	}

	return nil
}
