package middlewares

import (
	"net/http"

	"extrnode-be/internal/pkg/log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type LoggerContextConfig struct {
	ReqMethodContextKey         string
	ReqBodyContextKey           string
	ResBodyContextKey           string
	RpcErrorContextKey          string
	ProxyEndpointContextKey     string
	ProxyAttemptsContextKey     string
	ProxyResponseTimeContextKey string
}

func NewLoggerMiddleware(config LoggerContextConfig) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:    true,
		LogMethod:    true,
		LogRequestID: true,
		LogLatency:   true,
		LogError:     true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			rpcMethod, _ := c.Get(config.ReqMethodContextKey).(string)
			rpcErrorCode, _ := c.Get(config.RpcErrorContextKey).(int)
			endpoint, _ := c.Get(config.ProxyEndpointContextKey).(string)
			attempts, _ := c.Get(config.ProxyAttemptsContextKey).(int)
			responseTime, _ := c.Get(config.ProxyResponseTimeContextKey).(int64)

			if v.Error != nil || rpcErrorCode != 0 || v.Status >= http.StatusBadRequest {
				log.Logger.Proxy.Errorf("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %s, attempts: %d, node_response_time: %dms, "+
					"rpc_error_code: %d, error: %s, request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), endpoint, rpcMethod, attempts, responseTime,
					rpcErrorCode, v.Error, c.Get(config.ReqBodyContextKey), c.Get(config.ResBodyContextKey), v.RemoteIP, v.UserAgent)
			} else {
				log.Logger.Proxy.Infof("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %s, attempts: %d, node_response_time: %dms, "+
					"request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), endpoint, rpcMethod, attempts, responseTime,
					c.Get(config.ReqBodyContextKey), c.Get(config.ResBodyContextKey), v.RemoteIP, v.UserAgent)
			}

			return nil
		},
	})
}
