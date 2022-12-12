package solana

import (
	"net"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/klauspost/compress/gzhttp"
)

var (
	defaultMaxIdleConnsPerHost = 1
	defaultTimeout             = 5 * time.Second
	defaultKeepAlive           = 100 * time.Second
)

func newHTTPTransport() *http.Transport {
	return &http.Transport{
		IdleConnTimeout:     defaultTimeout,
		MaxConnsPerHost:     defaultMaxIdleConnsPerHost,
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		Proxy:               http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   defaultTimeout,
			KeepAlive: defaultKeepAlive,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2: true,
		// MaxIdleConns:          100,
		TLSHandshakeTimeout: defaultTimeout,
		// ExpectContinueTimeout: 1 * time.Second,
	}
}

func createRpcWithTimeout(host string) *rpc.Client {
	jsonrpcClient := jsonrpc.NewClientWithOpts(host, &jsonrpc.RPCClientOpts{HTTPClient: &http.Client{
		Timeout:   defaultTimeout,
		Transport: gzhttp.Transport(newHTTPTransport()),
	}})

	return rpc.NewWithCustomRPCClient(jsonrpcClient)
}
