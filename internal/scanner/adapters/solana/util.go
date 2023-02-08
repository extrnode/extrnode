package solana

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/klauspost/compress/gzhttp"

	"extrnode-be/internal/pkg/storage/postgres"
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

func (a *SolanaAdapter) getValidRpc(peer postgres.PeerWithIpAndBlockchain) (rpcClient *rpc.Client, isSSl bool, version string, err error) {
	rpcClient = createRpcWithTimeout(createNodeUrl(peer, isSSl))
	versionRes, err := rpcClient.GetVersion(a.ctx)
	if err != nil {
		isSSl = true
		rpcClient = createRpcWithTimeout(createNodeUrl(peer, isSSl))
		versionRes, err = rpcClient.GetVersion(a.ctx)
		if err != nil {
			return rpcClient, false, version, err // don't change isSsl in err case
		}
	}

	if versionRes != nil {
		version = versionRes.SolanaCore
	}

	return rpcClient, isSSl, version, nil
}

func createNodeUrl(p postgres.PeerWithIpAndBlockchain, isSSL bool) string {
	schema := "http://"
	if isSSL {
		schema = "https://"
	}

	return fmt.Sprintf("%s%s:%d", schema, p.Address.String(), p.Port)
}

func reformatSolanaRpcError(err error) error {
	if err == nil {
		return nil
	}
	rpcErr, ok := err.(*jsonrpc.RPCError)
	if !ok {
		return err
	}

	return fmt.Errorf("rpcErr: code %d %s", rpcErr.Code, rpcErr.Message)
}
