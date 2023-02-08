package middlewares

import (
	"time"

	"github.com/labstack/echo/v4"

	"extrnode-be/internal/pkg/metrics"
)

func NewMetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			cc := c.(*CustomContext)

			rpcMethod := cc.GetReqMethod()
			success := !cc.GetProxyHasError() || cc.GetProxyUserError()

			metrics.IncHttpResponsesTotalCnt(rpcMethod, success)
			metrics.ObserveNodeAttempts(rpcMethod, success, cc.GetProxyAttempts())
			metrics.ObserveNodeResponseTime(rpcMethod, success, cc.GetProxyResponseTime())
			metrics.ObserveExecutionTime(rpcMethod, success, time.Since(cc.GetReqDuration()))

			return err
		}
	}
}
