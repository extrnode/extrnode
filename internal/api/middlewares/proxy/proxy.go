package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"

	"extrnode-be/internal/api/middlewares"
)

func NewProxyMiddleware(transport *ProxyTransport) echo.MiddlewareFunc {
	// set some basic url so validation does not fail, later we get proxy url from transport in roundtripper
	baseUrl, _ := url.Parse("https://localhost:8080")
	responseModifier := responseModifier{}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			req := c.Request()
			res := c.Response()

			// Fix header
			// Basically it's not good practice to unconditionally pass incoming x-real-ip header to upstream.
			// However, for backward compatibility, legacy behavior is preserved unless you configure Echo#IPExtractor.
			if req.Header.Get(echo.HeaderXRealIP) == "" || c.Echo().IPExtractor != nil {
				req.Header.Set(echo.HeaderXRealIP, c.RealIP())
			}
			if req.Header.Get(echo.HeaderXForwardedProto) == "" {
				req.Header.Set(echo.HeaderXForwardedProto, c.Scheme())
			}
			if c.IsWebSocket() && req.Header.Get(echo.HeaderXForwardedFor) == "" { // For HTTP, it is automatically set by Go HTTP reverse proxy.
				req.Header.Set(echo.HeaderXForwardedFor, c.RealIP())
			}

			proxy := httputil.NewSingleHostReverseProxy(baseUrl)
			proxy.Transport = transport.WithContext(c)
			proxy.ModifyResponse = responseModifier.WithContext(c)

			eh := errorHandler{}
			proxy.ErrorHandler = eh.WithContext(c)

			proxy.ServeHTTP(res, req)

			return eh.err
		}
	}
}

const (
	// headerProcessingTime     = "X-RESPONSE-PROCESSING-TIME"
	headerNodeReqAttempts  = "X-NODE-REQ-ATTEMPTS"
	headerNodeResponseTime = "X-NODE-RESPONSE-TIME"
	headerNodeEndpoint     = "X-NODE-ENDPOINT"
)

type responseModifier struct{}

func (rm *responseModifier) WithContext(c echo.Context) func(*http.Response) error {
	return func(res *http.Response) error {
		cc := c.(*middlewares.CustomContext)

		res.Header.Set(headerNodeEndpoint, cc.GetProxyEndpoint())
		res.Header.Set(headerNodeReqAttempts, fmt.Sprintf("%d", cc.GetProxyAttempts()))
		res.Header.Set(headerNodeResponseTime, fmt.Sprintf("%dms", cc.GetProxyResponseTime()))

		return nil
	}
}

// StatusCodeContextCanceled is a custom HTTP status code for situations
// where a client unexpectedly closed the connection to the server.
// As there is no standard error code for "client closed connection", but
// various well-known HTTP clients and server implement this HTTP code we use
// 499 too instead of the more problematic 5xx, which does not allow to detect this situation
const StatusCodeContextCanceled = 499

type errorHandler struct {
	err error
}

func (eh *errorHandler) WithContext(c echo.Context) func(http.ResponseWriter, *http.Request, error) {
	return func(_ http.ResponseWriter, _ *http.Request, err error) {
		// If the client canceled the request (usually by closing the connection), we can report a
		// client error (4xx) instead of a server error (5xx) to correctly identify the situation.
		// The Go standard library (at of late 2020) wraps the exported, standard
		// context.Canceled error with unexported garbage value requiring a substring check, see
		// https://github.com/golang/go/blob/6965b01ea248cabb70c3749fd218b36089a21efb/src/net/net.go#L416-L430
		if err == context.Canceled || strings.Contains(err.Error(), "operation was canceled") {
			httpError := echo.NewHTTPError(StatusCodeContextCanceled, fmt.Sprintf("client closed connection: %v", err))
			httpError.Internal = err
			eh.err = httpError
		} else if httErr, ok := err.(*echo.HTTPError); ok {
			eh.err = httErr // return not changed err for user
		} else {
			httpError := echo.NewHTTPError(http.StatusBadGateway, err.Error())
			httpError.Internal = err
			eh.err = httpError
		}
	}
}
