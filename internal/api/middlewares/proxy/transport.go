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

type ProxyTransport struct {
	i           int
	maxAttempts int
	transport   *http.Transport
	targets     []*proxyTarget
	withJail    bool
	sync.Mutex
}

type proxyTransportWithContext struct {
	transport *ProxyTransport
	c         echo.Context
	config    ProxyContextConfig
}

const transportDialerTimeout = 2 * time.Second
const targetJailTime = time.Second
const consecutiveSucessResponses = 10

func NewProxyTransport(withJail bool) *ProxyTransport {
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
		maxAttempts: 5,
		withJail:    withJail,
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

		target, err = ptc.transport.NextAvailableTarget()
		if err != nil {
			return nil, err
		}

		// modify req url
		req.URL.Scheme = target.url.Scheme
		req.URL.Host = target.url.Host

		// refill body
		req.Body = io.NopCloser(bytes.NewBuffer(clonedBody))
		req.ContentLength = clonedContentLength

		startTime = time.Now()
		resp, err = ptc.transport.transport.RoundTrip(req)
		if err != nil {
			target.UpdateAvailability(true)

			log.Logger.Proxy.Errorf("RoundTrip: %s", err)
			continue
		}

		if resp.StatusCode >= 300 {
			target.UpdateAvailability(true)

			return resp, echo.NewHTTPError(resp.StatusCode)
		}

		analysisErr := ptc.getResponseError(resp)
		if analysisErr != nil {
			if analysisErr == ErrInvalidRequest {
				target.UpdateAvailability(false)
				ptc.c.Set(ptc.config.ProxyUserErrorContextKey, true)
				break
			}

			log.Logger.Proxy.Errorf("responseError: %s", analysisErr)

			target.UpdateAvailability(true)
			continue
		}

		// success case
		target.UpdateAvailability(false)
		break
	}

	ptc.c.Set(ptc.config.ProxyEndpointContextKey, target.url.String())
	ptc.c.Set(ptc.config.ProxyAttemptsContextKey, i+1)
	ptc.c.Set(ptc.config.ProxyResponseTimeContextKey, time.Since(startTime).Milliseconds())

	return resp, err
}

// Next returns an upstream target using round-robin technique.
func (pt *ProxyTransport) getNextTarget() *proxyTarget {
	pt.Lock()
	defer pt.Unlock()

	pt.i = pt.i % len(pt.targets)
	t := pt.targets[pt.i]
	pt.i++
	return t
}

func (pt *ProxyTransport) NextAvailableTarget() (*proxyTarget, error) {
	if !pt.withJail {
		return pt.getNextTarget(), nil
	}

	for i := 0; i < len(pt.targets); i++ {
		target := pt.getNextTarget()
		if !target.isAvailable() {
			continue
		}

		return target, nil
	}

	return nil, fmt.Errorf("no available targets")
}

// AddTarget adds an upstream target to the list.
func (pt *ProxyTransport) AddTarget(url *url.URL) bool {
	for _, t := range pt.targets {
		if strings.EqualFold(t.url.String(), url.String()) {
			return false
		}
	}

	pt.Lock()
	pt.targets = append(pt.targets, &proxyTarget{
		url: url,
	})
	pt.Unlock()

	log.Logger.Proxy.Debugf("Transport added target: %s", url.String())
	return true
}

// RemoveTarget removes an upstream target from the list.
func (pt *ProxyTransport) RemoveTarget(url *url.URL) bool {
	for i, t := range pt.targets {
		if strings.EqualFold(t.url.String(), url.String()) {
			pt.Lock()
			pt.targets = append(pt.targets[:i], pt.targets[i+1:]...)
			pt.Unlock()

			log.Logger.Proxy.Debugf("Transport removed target: %s", url.String())
			return true
		}
	}

	return false
}

type proxyTarget struct {
	url            *url.URL
	jailExpireTime int64
	errCounter     int
	successCounter int
	sync.Mutex
}

// RemoveTarget removes an upstream target from the list.
func (t *proxyTarget) UpdateAvailability(isNodeErr bool) {
	t.Lock()
	defer t.Unlock()

	if !t.isAvailable() {
		return
	}

	if isNodeErr {
		t.successCounter = 0
		t.errCounter++
		t.jailExpireTime = time.Now().Add(targetJailTime * time.Duration(t.errCounter)).Unix()
	} else if t.successCounter < consecutiveSucessResponses {
		t.successCounter++
	} else {
		t.successCounter = 0
		t.errCounter = 0
	}
}

func (t *proxyTarget) isAvailable() bool {
	return time.Now().Unix() > t.jailExpireTime
}

// AddTarget adds an upstream target to the list.
func (pt *ProxyTransport) UpdateTargets(urls []*url.URL) {
	// Remove targets
	for _, t := range pt.targets {
		var found bool
		for _, u := range urls {
			if strings.EqualFold(t.url.String(), u.String()) {
				found = true
				break
			}
		}

		if !found {
			pt.RemoveTarget(t.url)
		}
	}

	for _, u := range urls {
		pt.AddTarget(u)
	}
}
