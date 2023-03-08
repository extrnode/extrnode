package adapters

import "extrnode-be/internal/pkg/storage/sqlite"

type Adapter interface {
	Scan(peer sqlite.PeerWithIpAndBlockchain) error
	GetNewNodes(peer sqlite.PeerWithIpAndBlockchain) error
	BeforeRun() error
	CheckOutdatedNodes() error
}
