package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"extrnode-be/internal/api/middlewares"
	"extrnode-be/internal/pkg/metrics"
)

const (
	nodeEndpointHeader           = "X-NODE-ENDPOINT"
	signatureHeader              = "X-SIGNATURE"
	responseProcessingTimeHeader = "X-RESPONSE-PROCESSING-TIME"
)

func (a *api) solanaProxyHandler(chainsGroup *echo.Group) error {
	blockchainID, ok := a.blockchainIDs[solanaBlockchain]
	if !ok {
		return fmt.Errorf("fail to get blockchainID")
	}
	isRpc := true

	// TODO: update endpoints
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

				// Temporary not needed
				//body, err := io.ReadAll(res.Body)
				//if err != nil {
				//	return fmt.Errorf("ReadAll: %s", err)
				//}
				//res.Body = io.NopCloser(bytes.NewBuffer(body)) // refill body

				//hash := blake2b.Sum256(body) // high performance hash func
				//signature, err := a.apiPrivateKey.Sign(hash[:])
				//if err != nil {
				//	return fmt.Errorf("Sign: %s", err)
				//}
				//res.Header.Set(signatureHeader, signature.String())
				res.Header.Set(nodeEndpointHeader, strings.TrimSuffix(res.Request.URL.String(), "/")) // temp hack with trailing slash

				timeConsumed := time.Since(now)
				//res.Header.Set(responseProcessingTimeHeader, timeConsumed.String())
				metrics.ObserveProcessingTime(solanaBlockchain, timeConsumed)

				return nil
			},
		}))

	return nil
}

func chainsMiddlewares() []echo.MiddlewareFunc {
	bodyDumpMiddleware := middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		reqBody, _ = json.Marshal(string(reqBody))
		// ignore err
		resBody, _ = json.Marshal(string(resBody))
		// ignore err

		if len(reqBody) > 1 {
			reqBody = []byte(strings.Trim(string(reqBody), `"`)) // remove extra trailing quotes
		}
		if len(resBody) > 1 {
			resBody = []byte(strings.Trim(string(resBody), `"`)) // remove extra trailing quotes
		}

		if len(reqBody) > bodyLimit {
			reqBody = []byte(strings.Trim(string(reqBody[:bodyLimit]), `\`))
		}
		if len(resBody) > bodyLimit {
			resBody = []byte(strings.Trim(string(resBody[:bodyLimit]), `\`))
		}

		c.Set(reqBodyContextKey, reqBody)
		c.Set(resBodyContextKey, resBody)
	})
	loggerMiddleware := middleware.LoggerWithConfig(
		middleware.LoggerConfig{
			Format: `{"time":"${time_rfc3339}","id":"${id}","remote_ip":"${remote_ip}",` +
				`"method":"${method}","user_agent":"${user_agent}","status":${status},` +
				`"error":"${error}","latency":${latency},${custom}}` + "\n",
			CustomTagFunc: func(c echo.Context, buf *bytes.Buffer) (int, error) {
				return buf.WriteString(fmt.Sprintf(`"endpoint":"%s","attempts":"%s","node_response_time":"%s","request_body":"%s","response_body":"%s"`,
					c.Response().Header().Get(nodeEndpointHeader), c.Response().Header().Get(middlewares.NodeReqAttempts), c.Response().Header().Get(middlewares.NodeResponseTime), c.Get(reqBodyContextKey), c.Get(resBodyContextKey)))
			}},
	)

	return []echo.MiddlewareFunc{middlewares.RequestID(), loggerMiddleware, bodyDumpMiddleware}
}
