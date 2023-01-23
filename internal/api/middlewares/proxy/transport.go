package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"extrnode-be/internal/pkg/log"

	"github.com/labstack/echo/v4"
)

type proxyTarget struct {
	URL *url.URL
}

type proxyTransport struct {
	i           uint32
	maxAttempts int
	transport   *http.Transport
	targets     []proxyTarget
	config      ProxyContextConfig
	sync.Mutex
}

type proxyTransportWithContext struct {
	transport *proxyTransport
	c         echo.Context
}

const transportDialerTimeout = 2 * time.Second

func newProxyTransport(targets []*url.URL, config ProxyContextConfig) *proxyTransport {
	pt := make([]proxyTarget, len(targets))
	for i := range targets {
		pt[i] = proxyTarget{
			URL: targets[i],
		}
	}

	return &proxyTransport{
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   transportDialerTimeout,
				KeepAlive: transportDialerTimeout,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          1,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   3 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		targets:     pt,
		config:      config,
		maxAttempts: 10,
	}
}

func (pt *proxyTransport) WithContext(c echo.Context) *proxyTransportWithContext {
	return &proxyTransportWithContext{
		transport: pt,
		c:         c,
	}
}

func (ptc *proxyTransportWithContext) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	clonedContentLength := req.ContentLength
	clonedBody, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("ReadAll: %s", err)
	}

	var target *proxyTarget
	var i int
	var startTime time.Time
	for ; i < ptc.transport.maxAttempts; i++ {
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		default:
		}

		target = ptc.transport.NextProxyTarget()

		// modify req url
		req.URL.Scheme = target.URL.Scheme
		req.URL.Host = target.URL.Host

		// refill body
		req.Body = io.NopCloser(bytes.NewBuffer(clonedBody))
		req.ContentLength = clonedContentLength

		startTime = time.Now()
		resp, err = ptc.transport.transport.RoundTrip(req)
		if err != nil {
			log.Logger.Api.Errorf("solana proxy (%s): %s", target.URL.String(), err)
			continue
		}

		break
	}

	ptc.c.Set(ptc.transport.config.ProxyEndpointContextKey, target.URL.String())
	ptc.c.Set(ptc.transport.config.ProxyAttemptsContextKey, i+1)
	ptc.c.Set(ptc.transport.config.ProxyResponseTimeContextKey, time.Since(startTime).Milliseconds())

	return resp, err
}

// Next returns an upstream target using round-robin technique.
func (pt *proxyTransport) NextProxyTarget() *proxyTarget {
	pt.i = pt.i % uint32(len(pt.targets))
	t := &pt.targets[pt.i]
	atomic.AddUint32(&pt.i, 1)
	return t
}

func (pt *proxyTransport) GetTargetsLen() int {
	return len(pt.targets)
}

// AddTarget adds an upstream target to the list.
func (pt *proxyTransport) AddTarget(url *url.URL) bool {
	for _, t := range pt.targets {
		if strings.EqualFold(t.URL.String(), url.String()) {
			return false
		}
	}
	pt.Lock()
	defer pt.Unlock()
	pt.targets = append(pt.targets, proxyTarget{
		URL: url,
	})
	return true
}

// RemoveTarget removes an upstream target from the list.
func (pt *proxyTransport) RemoveTarget(url *url.URL) bool {
	pt.Lock()
	defer pt.Unlock()
	for i, t := range pt.targets {
		if strings.EqualFold(t.URL.String(), url.String()) {
			pt.targets = append(pt.targets[:i], pt.targets[i+1:]...)
			return true
		}
	}
	return false
}
