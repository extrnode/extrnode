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
	"time"

	"extrnode-be/internal/pkg/log"

	"github.com/labstack/echo/v4"
)

type proxyTarget struct {
	URL *url.URL
}

type ProxyTransport struct {
	i           int
	maxAttempts int
	transport   *http.Transport
	targets     []proxyTarget
	sync.Mutex
}

type proxyTransportWithContext struct {
	transport *ProxyTransport
	c         echo.Context
	config    ProxyContextConfig
}

const transportDialerTimeout = 2 * time.Second

func NewProxyTransport() *ProxyTransport {
	return &ProxyTransport{
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
		maxAttempts: 10,
	}
}

func (pt *ProxyTransport) WithContext(c echo.Context, config ProxyContextConfig) *proxyTransportWithContext {
	return &proxyTransportWithContext{
		transport: pt,
		c:         c,
		config:    config,
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

	ptc.c.Set(ptc.config.ProxyEndpointContextKey, target.URL.String())
	ptc.c.Set(ptc.config.ProxyAttemptsContextKey, i+1)
	ptc.c.Set(ptc.config.ProxyResponseTimeContextKey, time.Since(startTime).Milliseconds())

	return resp, err
}

// Next returns an upstream target using round-robin technique.
func (pt *ProxyTransport) NextProxyTarget() *proxyTarget {
	pt.Lock()
	defer pt.Unlock()

	pt.i = pt.i % len(pt.targets)
	t := &pt.targets[pt.i]
	pt.i++
	return t
}

func (pt *ProxyTransport) GetTargetsLen() int {
	return len(pt.targets)
}

// AddTarget adds an upstream target to the list.
func (pt *ProxyTransport) AddTarget(url *url.URL) bool {
	for _, t := range pt.targets {
		if strings.EqualFold(t.URL.String(), url.String()) {
			return false
		}
	}

	pt.Lock()
	pt.targets = append(pt.targets, proxyTarget{
		URL: url,
	})
	pt.Unlock()

	log.Logger.Api.Debugf("Transport added target: %s", url.String())
	return true
}

// RemoveTarget removes an upstream target from the list.
func (pt *ProxyTransport) RemoveTarget(url *url.URL) bool {
	for i, t := range pt.targets {
		if strings.EqualFold(t.URL.String(), url.String()) {
			pt.Lock()
			pt.targets = append(pt.targets[:i], pt.targets[i+1:]...)
			pt.Unlock()

			log.Logger.Api.Debugf("Transport removed target: %s", url.String())
			return true
		}
	}

	return false
}

// AddTarget adds an upstream target to the list.
func (pt *ProxyTransport) UpdateTargets(urls []*url.URL) {
	// Remove targets
	for _, t := range pt.targets {
		var found bool
		for _, u := range urls {
			if strings.EqualFold(t.URL.String(), u.String()) {
				found = true
				break
			}
		}

		if !found {
			pt.RemoveTarget(t.URL)
		}
	}

	for _, u := range urls {
		pt.AddTarget(u)
	}
}
