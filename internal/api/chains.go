package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"extrnode-be/internal/api/middlewares"
	"extrnode-be/internal/pkg/log"
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
	endpoints, err := a.storage.GetEndpoints(blockchainID, maxLimit, &isRpc, nil, nil, nil, nil)
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

	// TODO: use when multichain will be implemented
	//chainsGroup.POST("/solana", nil,
	chainsGroup.POST("/", nil,
		middlewares.ProxyWithConfig(middlewares.ProxyConfig{
			Balancer: middlewares.NewRoundRobinBalancer(targets),
			//Rewrite: map[string]string{
			//	"/solana": "/", // empty string not working
			//},
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
				//now := time.Now()

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
				//res.Header.Set(responseProcessingTimeHeader, fmt.Sprintf("%dms", time.Since(now).Milliseconds()))

				return nil
			},
		}))

	return nil
}

func chainsMiddlewares() []echo.MiddlewareFunc {
	bodyDumpMiddleware := middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		var (
			parsedReq RPCRequest
			parsedRes RPCResponse
		)
		_ = json.Unmarshal(reqBody, &parsedReq) // ignore err
		_ = json.Unmarshal(resBody, &parsedRes) // ignore err

		if len(reqBody) > bodyLimit {
			reqBody = reqBody[:bodyLimit]
		}
		if len(resBody) > bodyLimit {
			resBody = resBody[:bodyLimit]
		}

		reqBody = []byte(strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, string(reqBody)))
		resBody = []byte(strings.TrimSpace(string(resBody)))

		c.Set(reqBodyContextKey, reqBody)
		c.Set(resBodyContextKey, resBody)
		c.Set(reqRpcMethodContextKey, parsedReq.Method)
		if parsedRes.Error.Code != 0 {
			c.Set(rpcErrorContextKey, parsedRes.Error.Code)
		}
	})

	loggerMiddleware := middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:    true,
		LogMethod:    true,
		LogRequestID: true,
		LogLatency:   true,
		LogError:     true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			rpcMethod, _ := c.Get(reqRpcMethodContextKey).(string)
			rpcErrorCode, _ := c.Get(rpcErrorContextKey).(int)

			endpoint := c.Response().Header().Get(nodeEndpointHeader)
			cl := c.Request().Header.Get(echo.HeaderContentLength)
			if cl == "" {
				cl = "0"
			}
			clFloat, _ := strconv.ParseFloat(cl, 64)
			// ignore err

			httpStatusString := fmt.Sprintf("%d", c.Response().Status)
			metrics.AddBytesReadTotalCnt(httpStatusString, rpcMethod, endpoint, clFloat)
			metrics.IncHttpResponsesTotalCnt(httpStatusString, rpcMethod, endpoint)
			if rpcErrorCode != 0 {
				metrics.IncRpcErrorCnt(fmt.Sprintf("%d", rpcErrorCode), httpStatusString, rpcMethod, endpoint)
			}
			attempts := c.Response().Header().Get(middlewares.NodeReqAttempts)
			nodeResponseTime := c.Response().Header().Get(middlewares.NodeResponseTime)

			if v.Error != nil || rpcErrorCode != 0 {
				log.Logger.Proxy.Errorf("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %s, attempts: %s, node_response_time: %s, "+
					"rpc_error_code: %d, error: %s, request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), endpoint, rpcMethod, attempts, nodeResponseTime,
					rpcErrorCode, v.Error, c.Get(reqBodyContextKey), c.Get(resBodyContextKey), v.RemoteIP, v.UserAgent)
			} else {
				log.Logger.Proxy.Infof("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %s, attempts: %s, node_response_time: %s, "+
					"request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), endpoint, rpcMethod, attempts, nodeResponseTime,
					c.Get(reqBodyContextKey), c.Get(resBodyContextKey), v.RemoteIP, v.UserAgent)
			}

			return nil
		},
	})

	return []echo.MiddlewareFunc{middlewares.RequestID(), loggerMiddleware, bodyDumpMiddleware}
}
