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

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"

	"github.com/labstack/echo/v4"
)

type ProxyTransport struct {
	maxAttempts int
	transport   *http.Transport
	withJail    bool

	targets []*proxyTarget
	i       int

	failoverTargets []*proxyTarget
	fi              int

	sync.Mutex
}

type proxyTransportWithContext struct {
	transport *ProxyTransport
	c         echo.Context
	config    ProxyContextConfig
}

const (
	transportDialerTimeout     = 2 * time.Second
	targetJailTime             = time.Second
	consecutiveSucessResponses = 10
	limitWindowSeconds         = 10
	secondsInHour              = 3600
)

func NewProxyTransport(withJail bool, failoverTargets config.FailoverTargets) (*ProxyTransport, error) {
	pt := &ProxyTransport{
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

	for _, ft := range failoverTargets {
		parsedUrl, err := url.Parse(ft.Url)
		if err != nil {
			return nil, fmt.Errorf("url.Parse: %s", err)
		}

		reqLimit := ft.ReqLimitHourly / (secondsInHour / limitWindowSeconds)
		pt.failoverTargets = append(pt.failoverTargets, newProxyTarget(parsedUrl, reqLimit))
	}

	return pt, nil
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
			return nil, echo.NewHTTPError(http.StatusServiceUnavailable, err.Error()) // need text from err
		}

		// modify req url
		req.URL = target.url
		req.Host = target.url.Host

		// refill body
		req.Body = io.NopCloser(bytes.NewBuffer(clonedBody))
		req.ContentLength = clonedContentLength

		startTime = time.Now()
		mustContinue, isAvailable := func() (bool, bool) {
			resp, err = ptc.transport.transport.RoundTrip(req)
			if err != nil {
				log.Logger.Proxy.Errorf("RoundTrip: %s", err)
				return true, false
			}

			if resp.StatusCode >= 300 {
				return true, false
			}

			analysisErr := ptc.getResponseError(resp)
			if analysisErr != nil {
				if analysisErr == ErrInvalidRequest {
					ptc.c.Set(ptc.config.ProxyUserErrorContextKey, true)
					return false, true
				}

				log.Logger.Proxy.Errorf("responseError: %s", analysisErr)

				return true, false
			}

			return false, true
		}()

		target.UpdateStats(isAvailable)

		if mustContinue {
			continue
		}

		break
	}

	ptc.c.Set(ptc.config.ProxyEndpointContextKey, target.url.String())
	ptc.c.Set(ptc.config.ProxyAttemptsContextKey, i+1)
	ptc.c.Set(ptc.config.ProxyResponseTimeContextKey, time.Since(startTime).Milliseconds())
	ptc.c.Set(ptc.config.ProxyHasErrorContextKey, err != nil)

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

// Next returns an upstream target using round-robin technique.
func (pt *ProxyTransport) getNextFailoverTarget() *proxyTarget {
	pt.Lock()
	defer pt.Unlock()

	pt.fi = pt.fi % len(pt.failoverTargets)
	t := pt.failoverTargets[pt.fi]
	pt.fi++
	return t
}

func (pt *ProxyTransport) NextAvailableTarget() (*proxyTarget, error) {
	for i := 0; i < len(pt.targets); i++ {
		target := pt.getNextTarget()
		if !target.isAvailable(pt.withJail) {
			continue
		}

		return target, nil
	}

	for i := 0; i < len(pt.failoverTargets); i++ {
		target := pt.getNextFailoverTarget()
		if !target.isAvailable(pt.withJail) {
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
	pt.targets = append(pt.targets, newProxyTarget(url, 0))
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
	url      *url.URL
	reqLimit uint64

	errCounter     uint64
	successCounter uint64
	reqCounter     uint64

	jailExpireTime int64
	reqWindow      int64

	sync.Mutex
}

func newProxyTarget(url *url.URL, reqLimit uint64) *proxyTarget {
	return &proxyTarget{
		url:      url,
		reqLimit: reqLimit,
	}
}

// RemoveTarget removes an upstream target from the list.
func (t *proxyTarget) UpdateStats(success bool) {
	t.Lock()
	defer t.Unlock()

	// truncate req counter by window
	currentWindow := getCurrentTimeWindow()
	if currentWindow > t.reqWindow {
		t.reqWindow = currentWindow
		t.reqCounter = 0
	}

	// increment req counter
	t.reqCounter++

	if !success {
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

func (t *proxyTarget) isAvailable(withJail bool) bool {
	// check jail time
	if withJail && t.jailExpireTime > time.Now().Unix() {
		return false
	}

	// check req limit
	currentWindow := getCurrentTimeWindow()
	if t.reqLimit > 0 && currentWindow == t.reqWindow && t.reqCounter >= t.reqLimit {
		return false
	}

	return true
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

func getCurrentTimeWindow() int64 {
	return time.Now().Truncate(time.Second * limitWindowSeconds).Unix()
}
