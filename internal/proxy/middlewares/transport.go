package middlewares

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
	echo2 "extrnode-be/internal/pkg/util/echo"

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

	scannedMethodList map[string]int

	endpointTargetsMutex   sync.Mutex
	failoverTargetsMutex   sync.Mutex
	scannedMethodListMutex sync.Mutex
}

type proxyTransportWithContext struct {
	transport *ProxyTransport
	c         *echo2.CustomContext
}

type UrlWithMethods struct {
	Url              *url.URL
	SupportedMethods map[string]struct{}
}

const (
	transportDialerTimeout     = 2 * time.Second
	targetJailTime             = time.Second
	consecutiveSucessResponses = 10
	limitWindowSeconds         = 10
	secondsInHour              = 3600
)

func NewProxyTransport(withJail bool, failoverTargets config.FailoverTargets, scannedMethodList map[string]int) (*ProxyTransport, error) {
	pt := &ProxyTransport{
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   transportDialerTimeout,
				KeepAlive: transportDialerTimeout,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          1,
			TLSHandshakeTimeout:   3 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		maxAttempts:       5,
		withJail:          withJail,
		scannedMethodList: scannedMethodList,
	}

	for _, ft := range failoverTargets {
		parsedUrl, err := url.Parse(ft.Url)
		if err != nil {
			return nil, fmt.Errorf("url.Parse: %s", err)
		}

		reqLimit := ft.ReqLimitHourly / (secondsInHour / limitWindowSeconds)
		pt.failoverTargets = append(pt.failoverTargets, newProxyTarget(UrlWithMethods{Url: parsedUrl}, reqLimit))
	}

	return pt, nil
}

func (pt *ProxyTransport) WithContext(c echo.Context) *proxyTransportWithContext {
	return &proxyTransportWithContext{
		transport: pt,
		c:         c.(*echo2.CustomContext),
	}
}

func (ptc *proxyTransportWithContext) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	reqMethods := ptc.c.GetReqMethods()
	clonedContentLength := req.ContentLength
	clonedBody, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("ReadAll: %s", err)
	}

	var (
		i         int
		target    *proxyTarget
		startTime time.Time
	)
outerLoop:
	for ; i < ptc.transport.maxAttempts; i++ {
		select {
		case <-req.Context().Done():
			err = req.Context().Err()
			break outerLoop
		default:
		}
		target, err = ptc.transport.NextAvailableTarget(reqMethods)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusServiceUnavailable, extraNodeNoAvailableTargetsErrorResponse)
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
					ptc.c.SetProxyUserError(true)
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

	if resp == nil && err == nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, extraNodeAttemptsExceededErrorResponse)
	}

	if target != nil {
		ptc.c.SetProxyEndpoint(target.url.String())
		ptc.c.SetProxyAttempts(i + 1)
	}

	if !startTime.IsZero() {
		ptc.c.SetProxyResponseTime(time.Since(startTime).Milliseconds())
	}

	ptc.c.SetProxyHasError(err != nil)

	return resp, err
}

// Next returns an upstream target using round-robin technique.
func (pt *ProxyTransport) getNextTarget(reqMethods []string) (t *proxyTarget) {
	var isContainUnscannedMethod, isFound bool
	pt.scannedMethodListMutex.Lock()
	for _, method := range reqMethods {
		if _, ok := pt.scannedMethodList[method]; !ok {
			isContainUnscannedMethod = true
			break
		}
	}
	pt.scannedMethodListMutex.Unlock()

	pt.endpointTargetsMutex.Lock()
out:
	for i := 0; i < len(pt.targets); i++ {
		pt.i = pt.i % len(pt.targets)
		t = pt.targets[pt.i]
		pt.i++

		if !t.isAvailable(pt.withJail) {
			continue
		}
		if isContainUnscannedMethod && len(t.supportedMethods) < len(pt.scannedMethodList)-1 { // take nodes that were once rpc
			continue
		}
		if isContainUnscannedMethod {
			isFound = true
			break
		}

		for _, method := range reqMethods {
			if _, ok := t.supportedMethods[method]; !ok {
				continue out
			}
		}

		isFound = true
		break
	}
	pt.endpointTargetsMutex.Unlock()
	if !isFound {
		return nil
	}

	return t
}

// Next returns an upstream target using round-robin technique.
func (pt *ProxyTransport) getNextFailoverTarget() (t *proxyTarget) {
	var isFound bool
	pt.failoverTargetsMutex.Lock()
	for i := 0; i < len(pt.failoverTargets); i++ {
		pt.fi = pt.fi % len(pt.failoverTargets)
		t = pt.failoverTargets[pt.fi]
		pt.fi++

		if !t.isAvailable(pt.withJail) {
			continue
		}

		isFound = true
		break
	}
	pt.failoverTargetsMutex.Unlock()
	if !isFound {
		return nil
	}

	return t
}

func (pt *ProxyTransport) NextAvailableTarget(reqMethods []string) (*proxyTarget, error) {
	target := pt.getNextTarget(reqMethods)
	if target != nil {
		return target, nil
	}

	target = pt.getNextFailoverTarget()
	if target != nil {
		return target, nil
	}

	return nil, fmt.Errorf("no available targets")
}

// AddTarget adds an upstream target to the list.
func (pt *ProxyTransport) AddTarget(urlWithMethods UrlWithMethods) bool {
	for _, t := range pt.targets {
		if strings.EqualFold(t.url.String(), urlWithMethods.Url.String()) {
			return false
		}
	}
	pt.endpointTargetsMutex.Lock()
	pt.targets = append(pt.targets, newProxyTarget(urlWithMethods, 0))
	pt.endpointTargetsMutex.Unlock()
	log.Logger.Proxy.Debugf("Transport added target: %s", urlWithMethods.Url.String())
	return true
}

// RemoveTarget removes an upstream target from the list.
func (pt *ProxyTransport) RemoveTarget(url *url.URL) bool {
	for i, t := range pt.targets {
		if strings.EqualFold(t.url.String(), url.String()) {
			pt.endpointTargetsMutex.Lock()
			pt.targets = append(pt.targets[:i], pt.targets[i+1:]...)
			pt.endpointTargetsMutex.Unlock()

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

	jailExpireTime   int64
	reqWindow        int64
	supportedMethods map[string]struct{}

	sync.Mutex
}

func newProxyTarget(urlWithMethods UrlWithMethods, reqLimit uint64) *proxyTarget {
	return &proxyTarget{
		url:              urlWithMethods.Url,
		reqLimit:         reqLimit,
		supportedMethods: urlWithMethods.SupportedMethods,
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
func (pt *ProxyTransport) UpdateTargets(urlsWithMethods []UrlWithMethods) {
	// Remove targets
	for _, t := range pt.targets {
		var found bool
		for _, u := range urlsWithMethods {
			if strings.EqualFold(t.url.String(), u.Url.String()) {
				found = true
				break
			}
		}

		if !found {
			pt.RemoveTarget(t.url)
		}
	}

	for _, u := range urlsWithMethods {
		pt.AddTarget(u)
	}
}

func getCurrentTimeWindow() int64 {
	return time.Now().Truncate(time.Second * limitWindowSeconds).Unix()
}
