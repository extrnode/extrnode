package models

import (
	"net"
)

type (
	NodeInfo struct {
		Version string
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
