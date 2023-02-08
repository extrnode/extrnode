package adapters

import "extrnode-be/internal/pkg/storage/postgres"

type Adapter interface {
	Scan(peer postgres.PeerWithIpAndBlockchain) error
	GetNewNodes(peer postgres.PeerWithIpAndBlockchain) error
	BeforeRun() error
	CheckOutdatedNodes() error
}
