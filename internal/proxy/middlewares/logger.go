package middlewares

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"extrnode-be/internal/pkg/log"
	echo2 "extrnode-be/internal/pkg/util/echo"
)

func NewLoggerMiddleware(saveLog func(ip, requestId string, statusCode int, latency int64, endpoint string, attempts int, responseTime int64, rpcMethods []string, rpcErrorCodes []int, userAgent, reqBody string)) echo.MiddlewareFunc {
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
			cc := c.(*echo2.CustomContext)
			saveLog(v.RemoteIP, v.RequestID, v.Status, v.Latency.Milliseconds(), cc.GetProxyEndpoint(), cc.GetProxyAttempts(), cc.GetProxyResponseTime(), cc.GetReqMethods(), cc.GetRpcErrors(), v.UserAgent, string(cc.GetReqBody()))

			// truncate before log
			reqBody := cc.GetReqBody()
			if len(cc.GetReqBody()) > bodyLimit {
				reqBody = reqBody[:bodyLimit]
			}

			if v.Error != nil || len(cc.GetRpcErrors()) != 0 || v.Status >= http.StatusBadRequest {
				log.Logger.Proxy.Errorf("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %v, attempts: %d, node_response_time: %dms, "+
					"rpc_error_code: %v, error: %s, request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s, path: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), cc.GetProxyEndpoint(), cc.GetReqMethods(), cc.GetProxyAttempts(), cc.GetProxyResponseTime(),
					cc.GetRpcErrors(), errMsg(v.Error), reqBody, cc.GetResBody(), v.RemoteIP, v.UserAgent, v.URI)
			} else {
				log.Logger.Proxy.Infof("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %v, attempts: %d, node_response_time: %dms, "+
					"request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s, path: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), cc.GetProxyEndpoint(), cc.GetReqMethods(), cc.GetProxyAttempts(), cc.GetProxyResponseTime(),
					reqBody, cc.GetResBody(), v.RemoteIP, v.UserAgent, v.URI)
			}

			return nil
		},
	})
}
