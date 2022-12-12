package adapters

type Adapter interface {
	Scan(host string) error
	GetNewNodes(host string) error
}
