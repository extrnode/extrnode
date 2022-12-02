package types

import "github.com/gagliardetto/solana-go"

type RawAccountData struct {
	Slot      uint64
	PublicKey solana.PublicKey
	Version   uint64
	Owner     solana.PublicKey
	Data      []byte
}
