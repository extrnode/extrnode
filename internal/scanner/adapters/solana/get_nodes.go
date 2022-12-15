package solana

import (
	"fmt"
	"net"
	"strconv"

	"github.com/gagliardetto/solana-go"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/scanner/models"
)

func (a *SolanaAdapter) getNodeIpAndVersion(nodeGossip, nodeVersion string, nodePubkey solana.PublicKey) (node models.NodeInfo, err error) {
	node.Version = nodeVersion
	node.Pubkey = nodePubkey
	host, port, err := net.SplitHostPort(nodeGossip)
	if err != nil {
		if node.IP = net.ParseIP(nodeGossip); node.IP == nil {
			return node, fmt.Errorf("invalid argument for ip %s", nodeGossip)
		}
		node.Port = httpPort
	} else {
		if node.IP = net.ParseIP(host); node.IP == nil {
			return node, fmt.Errorf("invalid argument for ip %s", host)
		}

		node.Port, err = strconv.Atoi(port)
		if err != nil {
			return node, fmt.Errorf("atoi: |%s| %s", port, err)
		}
	}

	return node, nil
}

func (a *SolanaAdapter) getNodes(host string) (nodes []models.NodeInfo, err error) {
	rpcClient := createRpcWithTimeout(host)
	clusterNodes, err := rpcClient.GetClusterNodes(a.ctx)
	if err != nil {
		return nodes, fmt.Errorf("GetClusterNodes: %s", reformatSolanaRpcError(err))
	}

	log.Logger.Scanner.Debugf("getNodes: host %s got %d cluster nodes", host, len(clusterNodes))

	for _, node := range clusterNodes {
		if node.Gossip == nil || node.Version == nil {
			continue
		}

		nodeInfo, err := a.getNodeIpAndVersion(*node.Gossip, *node.Version, node.Pubkey)
		if err != nil {
			return nodes, fmt.Errorf("getNodeIpAndVersion gossip: %s", err)
		}
		nodes = append(nodes, nodeInfo)

		if node.RPC != nil && *node.RPC != *node.Gossip {
			nodeInfo, err = a.getNodeIpAndVersion(*node.RPC, *node.Version, node.Pubkey)
			if err != nil {
				return nodes, fmt.Errorf("getNodeIpAndVersion rpc: %s", err)
			}

			nodes = append(nodes, nodeInfo)
		}
	}

	return nodes, nil
}

func (a *SolanaAdapter) insertData(records []models.NodeInfo) error {
	for _, r := range records {
		countryID, err := a.storage.GetOrCreateGeoCountry(r.Alpha2, r.Alpha3, r.Name)
		if err != nil {
			return fmt.Errorf("GetOrCreateGeoCountry: %s; req %+v", err, r)
		}

		networkID, err := a.storage.GetOrCreateGeoNetwork(countryID, *r.AsnInfo.Network, int(r.AsnInfo.As), r.AsnInfo.Isp)
		if err != nil {
			return fmt.Errorf("GetOrCreateGeoNetwork: %s; req %+v cntId %d", err, r, countryID)
		}

		ipID, err := a.storage.GetOrCreateIP(networkID, r.IP)
		if err != nil {
			return fmt.Errorf("GetOrCreateIP: %s; req %+v; networkID %d", err, r, networkID)
		}

		_, err = a.storage.GetOrCreatePeer(a.blockchainID, ipID, r.Port, r.Version, false, false, false, true, false, r.Pubkey.String())
		if err != nil {
			return fmt.Errorf("GetOrCreatePeer: %s; req %+v blcId %d ipId %d", err, r, a.blockchainID, ipID)
		}
	}

	return nil
}

func (a *SolanaAdapter) filterAndUpdateNodes(nodes []models.NodeInfo) (res []models.NodeInfo, err error) {
	ips := make([]net.IP, 0, len(nodes))
	for _, n := range nodes {
		ips = append(ips, n.IP)
	}

	existentPeersMap, err := a.storage.GetExistentPeers(a.blockchainID, ips)
	if err != nil {
		return res, fmt.Errorf("storage.GetExistentPeers: %s", err)
	}

	for _, n := range nodes {
		if _, ok := existentPeersMap[n.IP.String()]; ok {
			if peer, ok := existentPeersMap[n.IP.String()][n.Port]; ok {
				if peer.Version != n.Version || peer.NodePubkey != n.Pubkey.String() {
					err = a.storage.UpdatePeerVersionAndNodePubkey(peer.ID, n.Version, n.Pubkey.String())
					if err != nil {
						return res, fmt.Errorf("UpdatePeerVersionAndNodePubkey: %s", err)
					}
				}

				continue
			}
		}

		res = append(res, n)
	}

	return res, nil
}
