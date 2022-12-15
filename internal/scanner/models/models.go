package models

import (
	"net"

	"github.com/gagliardetto/solana-go"
)

type (
	NodeInfo struct {
		Version string
		Pubkey  solana.PublicKey
		IP      net.IP
		Port    int
		AsnInfo
	}

	AsnInfo struct {
		As      uint64
		Network *net.IPNet
		Alpha2  string
		Alpha3  string
		Name    string
		Isp     string
	}
)
