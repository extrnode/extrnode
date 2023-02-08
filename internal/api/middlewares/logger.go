package middlewares

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"extrnode-be/internal/pkg/log"
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

func NewLoggerMiddleware(config LoggerContextConfig, saveLog func(ip, requestId string, statusCode int, latency int64, endpoint string, attempts int, responseTime int64, rpcMethods []string, rpcErrorCodes []int, userAgent, reqBody string)) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:    true,
		LogMethod:    true,
		LogRequestID: true,
		LogLatency:   true,
		LogError:     true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogURI:       true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			rpcMethods, _ := c.Get(config.ReqMethodContextKey).([]string)
			rpcErrorCodes, _ := c.Get(config.RpcErrorContextKey).([]int)
			endpoint, _ := c.Get(config.ProxyEndpointContextKey).(string)
			attempts, _ := c.Get(config.ProxyAttemptsContextKey).(int)
			responseTime, _ := c.Get(config.ProxyResponseTimeContextKey).(int64)
			reqBody, _ := c.Get(config.ReqBodyContextKey).(string)

			saveLog(v.RemoteIP, v.RequestID, v.Status, v.Latency.Milliseconds(), endpoint, attempts, responseTime, rpcMethods, rpcErrorCodes, v.UserAgent, reqBody)

			// truncate before log
			if len(reqBody) > bodyLimit {
				reqBody = reqBody[:bodyLimit]
			}

			if v.Error != nil || len(rpcErrorCodes) != 0 || v.Status >= http.StatusBadRequest {
				log.Logger.Proxy.Errorf("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %v, attempts: %d, node_response_time: %dms, "+
					"rpc_error_code: %v, error: %s, request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s, path: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), endpoint, rpcMethods, attempts, responseTime,
					rpcErrorCodes, v.Error, reqBody, c.Get(config.ResBodyContextKey), v.RemoteIP, v.UserAgent, v.URI)
			} else {
				log.Logger.Proxy.Infof("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %v, attempts: %d, node_response_time: %dms, "+
					"request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s, path: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), endpoint, rpcMethods, attempts, responseTime,
					reqBody, c.Get(config.ResBodyContextKey), v.RemoteIP, v.UserAgent, v.URI)
			}

			return nil
		},
	})
}
