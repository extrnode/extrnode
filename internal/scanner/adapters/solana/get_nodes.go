package solana

import (
	"fmt"
	"net"
	"strconv"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/scanner/models"
)

func (s *SolanaAdapter) getNodeIpAndVersion(nodeGossip, nodeVersion string) (node models.NodeInfo, err error) {
	node.Version = nodeVersion
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

func (s *SolanaAdapter) getNodes(host string) (nodes []models.NodeInfo, err error) {
	rpcClient := createRpcWithTimeout(host)
	clusterNodes, err := rpcClient.GetClusterNodes(s.ctx)
	if err != nil {
		return nodes, fmt.Errorf("GetClusterNodes: %s", err)
	}

	log.Logger.Scanner.Debugf("getNodes: host %s got %d cluster nodes", host, len(clusterNodes))

	for _, node := range clusterNodes {
		if node.Gossip == nil || node.Version == nil {
			continue
		}

		nodeInfo, err := s.getNodeIpAndVersion(*node.Gossip, *node.Version)
		if err != nil {
			return nodes, fmt.Errorf("getNodeIpAndVersion gossip: %s", err)
		}
		nodes = append(nodes, nodeInfo)

		if node.RPC != nil && *node.RPC != *node.Gossip {
			nodeInfo, err = s.getNodeIpAndVersion(*node.RPC, *node.Version)
			if err != nil {
				return nodes, fmt.Errorf("getNodeIpAndVersion rpc: %s", err)
			}

			nodes = append(nodes, nodeInfo)
		}
	}

	return nodes, nil
}

func (s *SolanaAdapter) insertData(records []models.NodeInfo) error {
	for _, r := range records {
		countryID, err := s.storage.GetOrCreateGeoCountry(r.Alpha2, r.Alpha3, r.Name)
		if err != nil {
			return fmt.Errorf("GetOrCreateGeoCountry: %s; req %+v", err, r)
		}

		networkID, err := s.storage.GetOrCreateGeoNetwork(countryID, *r.AsnInfo.Network, int(r.AsnInfo.As), r.AsnInfo.Isp)
		if err != nil {
			return fmt.Errorf("GetOrCreateGeoNetwork: %s; req %+v cntId %d", err, r, countryID)
		}

		ipID, err := s.storage.GetOrCreateIP(networkID, r.IP)
		if err != nil {
			return fmt.Errorf("GetOrCreateIP: %s; req %+v; ipID %d", err, r, ipID)
		}

		_, err = s.storage.GetOrCreatePeer(s.blockchainID, ipID, r.Port, r.Version, false, false, false)
		if err != nil {
			return fmt.Errorf("GetOrCreatePeer: %s; req %+v blcId %d ipId %d", err, r, s.blockchainID, ipID)
		}
	}

	return nil
}

func (s *SolanaAdapter) filterNodes(nodes []models.NodeInfo) (res []models.NodeInfo, err error) {
	ips := make([]string, 0, len(nodes))
	for _, n := range nodes {
		ips = append(ips, n.IP.String())
	}

	existentPeers, err := s.storage.ReturnExistentPeers(s.blockchainID, ips)
	if err != nil {
		return res, fmt.Errorf("storage.ReturnExistentPeers: %s", err)
	}
	existentPeersMap := make(map[string]map[int]struct{}, len(existentPeers)) // ip -> port
	for _, p := range existentPeers {
		if _, ok := existentPeersMap[p.Address.String()]; !ok {
			existentPeersMap[p.Address.String()] = make(map[int]struct{})
		}

		existentPeersMap[p.Address.String()][p.Port] = struct{}{}
	}

	for _, n := range nodes {
		if _, ok := existentPeersMap[n.IP.String()]; ok {
			if _, ok = existentPeersMap[n.IP.String()][n.Port]; ok {
				continue
			}
		}

		res = append(res, n)
	}

	return res, nil
}
