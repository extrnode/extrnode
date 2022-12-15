package solana

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/klauspost/compress/gzhttp"

	"extrnode-be/internal/pkg/storage"
)

var (
	defaultMaxIdleConnsPerHost = 1
	defaultTimeout             = 10 * time.Second
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

func (a *SolanaAdapter) getValidRpc(peer storage.PeerWithIpAndBlockchain) (*rpc.Client, bool, error) {
	rpcClient := createRpcWithTimeout(createNodeUrl(peer, false))
	_, err := rpcClient.GetVersion(a.ctx)
	if err != nil {
		rpcClient = createRpcWithTimeout(createNodeUrl(peer, true))
		_, err := rpcClient.GetVersion(a.ctx)
		return rpcClient, true, err
	}

	return rpcClient, false, nil
}

func createNodeUrl(p storage.PeerWithIpAndBlockchain, isSSL bool) string {
	schema := "http://"
	if isSSL {
		schema = "https://"
	}

	return fmt.Sprintf("%s%s:%d", schema, p.Address.String(), p.Port)
}
