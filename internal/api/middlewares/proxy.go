package middlewares

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"

	"extrnode-be/internal/pkg/metrics"
)

// TODO: Handle TLS proxy

const (
	NodeReqAttempts  = "X-NODE-REQ-ATTEMPTS"
	NodeResponseTime = "X-NODE-RESPONSE-TIME"
)

type (
	// ProxyConfig defines the config for Proxy middleware.
	ProxyConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper Skipper

		// Balancer defines a load balancing technique.
		// Required.
		Balancer ProxyBalancer

		// Rewrite defines URL path rewrite rules. The values captured in asterisk can be
		// retrieved by index e.g. $1, $2 and so on.
		// Examples:
		// "/old":              "/new",
		// "/api/*":            "/$1",
		// "/js/*":             "/public/javascripts/$1",
		// "/users/*/orders/*": "/user/$1/order/$2",
		Rewrite map[string]string

		// RegexRewrite defines rewrite rules using regexp.Rexexp with captures
		// Every capture group in the values can be retrieved by index e.g. $1, $2 and so on.
		// Example:
		// "^/old/[0.9]+/":     "/new",
		// "^/api/.+?/(.*)":    "/v2/$1",
		RegexRewrite map[*regexp.Regexp]string

		// Context key to store selected ProxyTarget into context.
		// Optional. Default value "target".
		ContextKey string

		// To customize the transport to remote.
		// Examples: If custom TLS certificates are required.
		Transport http.RoundTripper

		// ModifyResponse defines function to modify response from ProxyTarget.
		ModifyResponse func(*http.Response) error
	}

	// ProxyTarget defines the upstream target.
	ProxyTarget struct {
		Name string
		URL  *url.URL
		Meta echo.Map
		Rate int
	}

	// ProxyBalancer defines an interface to implement a load balancing technique.
	ProxyBalancer interface {
		AddTarget(*ProxyTarget) bool
		RemoveTarget(string) bool
		Next(echo.Context) *ProxyTarget
		GetTargetsLen() int
	}

	commonBalancer struct {
		targets []*ProxyTarget
		mutex   sync.RWMutex
	}

	// RoundRobinBalancer implements a round-robin load balancing technique.
	roundRobinBalancer struct {
		*commonBalancer
		i uint32
	}
)

var (
	// DefaultProxyConfig is the default Proxy middleware config.
	DefaultProxyConfig = ProxyConfig{
		Skipper:    DefaultSkipper,
		ContextKey: "target",
	}
)

const contextErrorField = "_error"

func proxyRaw(t *ProxyTarget, c echo.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		in, _, err := c.Response().Hijack()
		if err != nil {
			c.Set(contextErrorField, fmt.Sprintf("proxy raw, hijack error=%v, url=%s", t.URL, err))
			return
		}
		defer in.Close()

		out, err := net.Dial("tcp", t.URL.Host)
		if err != nil {
			c.Set(contextErrorField, echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("proxy raw, dial error=%v, url=%s", t.URL, err)))
			return
		}
		defer out.Close()

		// Write header
		err = r.Write(out)
		if err != nil {
			c.Set(contextErrorField, echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("proxy raw, request header copy error=%v, url=%s", t.URL, err)))
			return
		}

		errCh := make(chan error, 2)
		cp := func(dst io.Writer, src io.Reader) {
			_, err = io.Copy(dst, src)
			errCh <- err
		}

		go cp(out, in)
		go cp(in, out)
		err = <-errCh
		if err != nil && err != io.EOF {
			c.Set(contextErrorField, fmt.Errorf("proxy raw, copy body error=%v, url=%s", t.URL, err))
		}
	})
}

// NewRoundRobinBalancer returns a round-robin proxy balancer.
func NewRoundRobinBalancer(targets []*ProxyTarget) ProxyBalancer {
	b := &roundRobinBalancer{commonBalancer: new(commonBalancer)}
	b.targets = targets
	return b
}

// AddTarget adds an upstream target to the list.
func (b *commonBalancer) AddTarget(target *ProxyTarget) bool {
	for _, t := range b.targets {
		if t.Name == target.Name {
			return false
		}
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.targets = append(b.targets, target)
	return true
}

// RemoveTarget removes an upstream target from the list.
func (b *commonBalancer) RemoveTarget(name string) bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for i, t := range b.targets {
		if t.Name == name {
			b.targets = append(b.targets[:i], b.targets[i+1:]...)
			return true
		}
	}
	return false
}

// Next returns an upstream target using round-robin technique.
func (b *roundRobinBalancer) Next(c echo.Context) *ProxyTarget {
	b.i = b.i % uint32(len(b.targets))
	t := b.targets[b.i]
	atomic.AddUint32(&b.i, 1)
	return t
}

func (b *roundRobinBalancer) GetTargetsLen() int {
	return len(b.targets)
}

// Proxy returns a Proxy middleware.
//
// Proxy middleware forwards the request to upstream server using a configured load balancing technique.
func Proxy(balancer ProxyBalancer) echo.MiddlewareFunc {
	c := DefaultProxyConfig
	c.Balancer = balancer
	return ProxyWithConfig(c)
}

// ProxyWithConfig returns a Proxy middleware with config.
// See: `Proxy()`
func ProxyWithConfig(config ProxyConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultProxyConfig.Skipper
	}
	if config.Balancer == nil {
		panic("echo: proxy middleware requires balancer")
	}

	if config.Rewrite != nil {
		if config.RegexRewrite == nil {
			config.RegexRewrite = make(map[*regexp.Regexp]string)
		}
		for k, v := range rewriteRulesRegex(config.Rewrite) {
			config.RegexRewrite[k] = v
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			if config.Balancer.GetTargetsLen() == 0 {
				return errors.New("no nodes available")
			}

			req := c.Request()
			res := c.Response()

			if err := rewriteURL(config.RegexRewrite, req); err != nil {
				return err
			}

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

			clonedBody, err := io.ReadAll(req.Body)
			if err != nil {
				return fmt.Errorf("ReadAll: %s", err)
			}
			clonedContentLength := req.ContentLength

			var (
				i        int
				now      time.Time
				tgt      *ProxyTarget
				proxyReq *http.Request
				proxyRes *echo.Response
				writer   *fakeWriter
			)

			for ; i < config.Balancer.GetTargetsLen(); i++ {
				tgt = config.Balancer.Next(c)
				proxyReq, proxyRes, writer = prepareProxyReqRes(req, res, clonedBody, clonedContentLength)

				now = time.Now()
				httpError := newReverseProxy(tgt, config, proxyReq, proxyRes, writer)
				if httpError != nil {
					err = errors.New(httpError.Error())
					log.Errorf("solana proxy (%s): %s", tgt.URL.String(), err)
					tgt.DecreaseRate()
					continue
				}

				err = nil // unset err in success case
				tgt.IncreaseRate()
				break // find a working one
			}

			if writer == nil {
				return errors.New("writer is nil")
			}

			// Write response
			res.Writer.WriteHeader(writer.statusCode)
			res.Writer.Write(writer.buf.Bytes())
			// Flush headers
			writer.FlushHeaders(res.Writer)

			c.Set(config.ContextKey, tgt)

			nodeResponseTime := time.Since(now)

			metrics.ObserveNodeAttemptsPerRequest(tgt.URL.String(), i+1)
			metrics.ObserveNodeResponseTime(tgt.URL.String(), nodeResponseTime)

			res.Header().Set(NodeReqAttempts, fmt.Sprintf("%d", i+1))
			res.Header().Set(NodeResponseTime, fmt.Sprintf("%dms", nodeResponseTime.Milliseconds()))

			if err != nil {
				c.Set(contextErrorField, err.Error())
			}

			return
		}
	}
}

// StatusCodeContextCanceled is a custom HTTP status code for situations
// where a client unexpectedly closed the connection to the server.
// As there is no standard error code for "client closed connection", but
// various well-known HTTP clients and server implement this HTTP code we use
// 499 too instead of the more problematic 5xx, which does not allow to detect this situation
const StatusCodeContextCanceled = 499

func newReverseProxy(tgt *ProxyTarget, config ProxyConfig, req *http.Request, res *echo.Response, writer http.ResponseWriter) *echo.HTTPError {
	var httpError *echo.HTTPError

	proxy := httputil.NewSingleHostReverseProxy(tgt.URL)
	proxy.ErrorHandler = func(_ http.ResponseWriter, _ *http.Request, err error) {
		desc := tgt.URL.String()
		if tgt.Name != "" {
			desc = fmt.Sprintf("%s(%s)", tgt.Name, tgt.URL.String())
		}
		// If the client canceled the request (usually by closing the connection), we can report a
		// client error (4xx) instead of a server error (5xx) to correctly identify the situation.
		// The Go standard library (at of late 2020) wraps the exported, standard
		// context.Canceled error with unexported garbage value requiring a substring check, see
		// https://github.com/golang/go/blob/6965b01ea248cabb70c3749fd218b36089a21efb/src/net/net.go#L416-L430
		if err == context.Canceled || strings.Contains(err.Error(), "operation was canceled") {
			httpError = echo.NewHTTPError(StatusCodeContextCanceled, fmt.Sprintf("client closed connection: %v", err))
			httpError.Internal = err
		} else {
			httpError = echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("remote %s unreachable, could not forward: %v", desc, err))
			httpError.Internal = err
		}
	}
	proxy.Transport = config.Transport
	proxy.ModifyResponse = config.ModifyResponse

	proxy.ServeHTTP(res, req)
	return httpError
}

func prepareProxyReqRes(req *http.Request, res *echo.Response, clonedBody []byte, clonedContentLength int64) (*http.Request, *echo.Response, *fakeWriter) {
	// create temp response writer
	writer := NewFakeWriter(res.Writer.Header())

	// create a copy for req and response, don't use original
	proxyRes := new(echo.Response)
	proxyRes.Writer = writer

	// req
	proxyReq := req.Clone(req.Context())
	proxyReq.Body = io.NopCloser(bytes.NewBuffer(clonedBody)) // refill body
	proxyReq.ContentLength = clonedContentLength

	return proxyReq, proxyRes, writer
}

func (p *ProxyTarget) IncreaseRate() {
	// TODO: use formula
	p.Rate++
}
func (p *ProxyTarget) DecreaseRate() {
	// TODO: use formula
	p.Rate--
}

type fakeWriter struct {
	statusCode int
	header     http.Header
	buf        *bytes.Buffer
}

func NewFakeWriter(headers http.Header) *fakeWriter {
	h := make(http.Header)
	for k, v := range headers {
		h[k] = v
	}

	return &fakeWriter{
		header: h,
		buf:    &bytes.Buffer{},
	}
}

func (fw *fakeWriter) Header() http.Header {
	return fw.header
}

func (fw *fakeWriter) Write(data []byte) (int, error) {
	return fw.buf.Write(data)
}

func (fw *fakeWriter) WriteHeader(statusCode int) {
	fw.statusCode = statusCode
}

func (fw *fakeWriter) FlushHeaders(w http.ResponseWriter) {
	for k, v := range fw.header {
		if len(v) == 0 {
			continue
		}

		w.Header().Set(k, v[0])
	}

}
