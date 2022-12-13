package adapters

type Adapter interface {
	Scan(host string) error
	GetNewNodes(host string, isAlive bool) error
}
