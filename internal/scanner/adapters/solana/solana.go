package solana

import "fmt"

type SolanaAdapter struct{}

func (a *SolanaAdapter) Scan(host string) error {
	fmt.Println(host)

	return nil
}
