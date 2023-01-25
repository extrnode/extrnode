package middlewares

import (
	"fmt"
	"strconv"

	"github.com/labstack/echo/v4"

	"extrnode-be/internal/pkg/metrics"
)

type MetricsContextConfig struct {
	ReqMethodContextKey         string
	RpcErrorContextKey          string
	ProxyEndpointContextKey     string
	ProxyAttemptsContextKey     string
	ProxyResponseTimeContextKey string
	ProxyUserErrorContextKey    string
}

const multipleValuesRequested = "multiple_values"

func NewMetricsMiddleware(config MetricsContextConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			rpcMethods, _ := c.Get(config.ReqMethodContextKey).([]string)
			rpcErrorCodes, _ := c.Get(config.RpcErrorContextKey).([]int)
			endpoint, _ := c.Get(config.ProxyEndpointContextKey).(string)
			attempts, _ := c.Get(config.ProxyAttemptsContextKey).(int)
			nodeResponseTime, _ := c.Get(config.ProxyResponseTimeContextKey).(int64)
			userError := c.Get(config.ProxyUserErrorContextKey)
			cl := c.Request().Header.Get(echo.HeaderContentLength)
			if cl == "" {
				cl = "0"
			}
			clFloat, _ := strconv.ParseFloat(cl, 64)

			httpStatusString := fmt.Sprintf("%d", c.Response().Status)

			var rpcMethod string
			if len(rpcMethods) > 1 {
				rpcMethod = multipleValuesRequested
			} else if len(rpcMethods) == 1 {
				rpcMethod = rpcMethods[0]
			}
			var rpcErrorCodesString string
			if len(rpcErrorCodes) > 1 {
				rpcErrorCodesString = multipleValuesRequested
			} else if len(rpcErrorCodes) == 1 {
				rpcErrorCodesString = fmt.Sprintf("%d", rpcErrorCodes[0])
			}

			metrics.AddBytesReadTotalCnt(httpStatusString, rpcMethod, endpoint, clFloat)
			metrics.IncHttpResponsesTotalCnt(httpStatusString, rpcMethod, endpoint)
			if len(rpcErrorCodes) != 0 {
				metrics.IncRpcErrorCnt(rpcErrorCodesString, httpStatusString, rpcMethod, endpoint)
			}
			metrics.ObserveNodeAttemptsPerRequest(rpcMethod, endpoint, attempts)
			metrics.ObserveNodeResponseTime(rpcMethod, endpoint, nodeResponseTime)
			if userError == true {
				metrics.IncUserFailedRequestsCnt(rpcErrorCodesString, httpStatusString, rpcMethod, endpoint)
			}

			return err
		}
	}
}
