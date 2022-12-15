package adapters

import "extrnode-be/internal/pkg/storage"

type Adapter interface {
	Scan(peer storage.PeerWithIpAndBlockchain) error
	GetNewNodes(peer storage.PeerWithIpAndBlockchain) error
	BeforeRun() error
}
