package middlewares

import (
	"time"

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
	ProxyHasErrorContextKey     string
	ReqDurationContextKey       string
}

const multipleValuesRequested = "multiple_values"

func NewMetricsMiddleware(config MetricsContextConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			reqDuration, _ := c.Get(config.ReqDurationContextKey).(time.Time)
			rpcMethods, _ := c.Get(config.ReqMethodContextKey).([]string)
			// rpcErrorCodes, _ := c.Get(config.RpcErrorContextKey).([]int)
			// endpoint, _ := c.Get(config.ProxyEndpointContextKey).(string)
			attempts, _ := c.Get(config.ProxyAttemptsContextKey).(int)
			nodeResponseTime, _ := c.Get(config.ProxyResponseTimeContextKey).(int64)
			hasError, _ := c.Get(config.ProxyHasErrorContextKey).(bool)
			userError, _ := c.Get(config.ProxyUserErrorContextKey).(bool)
			// cl := c.Request().Header.Get(echo.HeaderContentLength)

			var rpcMethod string
			if len(rpcMethods) > 1 {
				rpcMethod = multipleValuesRequested
			} else if len(rpcMethods) == 1 {
				rpcMethod = rpcMethods[0]
			}

			success := !hasError || userError

			metrics.IncHttpResponsesTotalCnt(rpcMethod, success)
			metrics.ObserveNodeAttempts(rpcMethod, success, attempts)
			metrics.ObserveNodeResponseTime(rpcMethod, success, nodeResponseTime)
			metrics.ObserveExecutionTime(rpcMethod, success, time.Since(reqDuration))

			return err
		}
	}
}
