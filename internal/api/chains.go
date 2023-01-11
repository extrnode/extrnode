package api

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/blake2b"

	"extrnode-be/internal/api/middlewares"
	"extrnode-be/internal/pkg/metrics"
)

func (a *api) solanaProxyHandler(chainsGroup *echo.Group) error {
	blockchainID, ok := a.blockchainIDs[solanaBlockchain]
	if !ok {
		return fmt.Errorf("fail to get blockchainID")
	}
	isRpc := true
	endpoints, err := a.storage.GetEndpoints(blockchainID, 1000, &isRpc, nil, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("GetEndpoints: %s", err)
	}

	// temp sort solution
	// TODO: sort by methods
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].SupportedMethods.AverageResponseTime() < endpoints[j].SupportedMethods.AverageResponseTime()
	})

	targets := make([]*middlewares.ProxyTarget, 0, len(endpoints))
	for _, e := range endpoints {
		schema := "http://"
		if e.IsSsl {
			schema = "https://"
		}
		parsedUrl, err := url.Parse(fmt.Sprintf("%s%s", schema, e.Endpoint))
		if err != nil {
			return fmt.Errorf("url.Parse: %s", err)
		}

		targets = append(targets, &middlewares.ProxyTarget{
			URL: parsedUrl,
		})
	}

	chainsGroup.POST("/solana", nil,
		middlewares.ProxyWithConfig(middlewares.ProxyConfig{
			ProxyName: solanaBlockchain,
			Balancer:  middlewares.NewRoundRobinBalancer(targets),
			Rewrite: map[string]string{
				"/solana": "/", // empty string not working
			},
			ContextKey: "target", // default from lib
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   customTransportDialerTimeout,
					KeepAlive: customTransportDialerTimeout,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          1,
				IdleConnTimeout:       30 * time.Second,
				TLSHandshakeTimeout:   3 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			ModifyResponse: func(res *http.Response) error {
				now := time.Now()
				body, err := io.ReadAll(res.Body)
				if err != nil {
					return fmt.Errorf("ReadAll: %s", err)
				}
				res.Body = io.NopCloser(bytes.NewBuffer(body)) // refill body

				hash := blake2b.Sum256(body) // high performance hash func
				signature, err := a.apiPrivateKey.Sign(hash[:])
				if err != nil {
					return fmt.Errorf("Sign: %s", err)
				}
				res.Header.Set(signatureHeader, signature.String())
				res.Header.Set(endpointHeader, strings.TrimSuffix(res.Request.URL.String(), "/")) // temp hack with trailing slash

				timeConsumed := time.Since(now)
				res.Header.Set(elapsedTimeHeader, timeConsumed.String())
				metrics.ObserveProcessingTime(solanaBlockchain, timeConsumed)

				return nil
			},
		}))

	return nil
}
