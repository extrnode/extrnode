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

func NewMetricsMiddleware(config MetricsContextConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			rpcMethod, _ := c.Get(config.ReqMethodContextKey).(string)
			rpcErrorCode, _ := c.Get(config.RpcErrorContextKey).(int)
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

			metrics.AddBytesReadTotalCnt(httpStatusString, rpcMethod, endpoint, clFloat)
			metrics.IncHttpResponsesTotalCnt(httpStatusString, rpcMethod, endpoint)
			if rpcErrorCode != 0 {
				metrics.IncRpcErrorCnt(fmt.Sprintf("%d", rpcErrorCode), httpStatusString, rpcMethod, endpoint)
			}
			metrics.ObserveNodeAttemptsPerRequest(rpcMethod, endpoint, attempts)
			metrics.ObserveNodeResponseTime(rpcMethod, endpoint, nodeResponseTime)
			if userError == true {
				metrics.IncUserFailedRequestsCnt(fmt.Sprintf("%d", rpcErrorCode), httpStatusString, rpcMethod, endpoint)
			}

			return err
		}
	}
}
