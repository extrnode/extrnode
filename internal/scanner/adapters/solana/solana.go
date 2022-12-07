package solana

import "fmt"

type SolanaAdapter struct{}

func (a *SolanaAdapter) Scan(host string) error {
	fmt.Println(host)
	// TODO:
	// err = s.getNodes()

	// Check if this node is validator or RPC
	// err = s.checkNodeType()

	// TODO: nmap scanner (https://github.com/Ullaakut/nmap)

	// TODO: asn scanner

	// TODO: methods scanner

	return nil
}

func (a *SolanaAdapter) getNodes() error {
	// Get list of nodes from host (getClusterNodes)
	// put these nodes to DB, so they can be scheduled by scheduler on next iteration

	return nil
}
