package echo

import (
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"

	"extrnode-be/internal/pkg/util/solana"
)

type CustomContext struct {
	echo.Context

	reqMethods        []string
	reqBody           []byte
	resBody           string
	rpcErrors         []int
	proxyEndpoint     string
	proxyAttempts     int
	proxyResponseTime int64
	proxyUserError    bool
	proxyHasError     bool
	reqDuration       time.Time
	user              *auth.UserRecord
}

func (c *CustomContext) SetReqMethods(reqMethods []string) {
	c.reqMethods = reqMethods
}

func (c *CustomContext) GetReqMethods() []string {
	return c.reqMethods
}

func (c *CustomContext) GetReqMethod() string {
	if len(c.reqMethods) == 1 {
		return c.reqMethods[0]
	}
	if len(c.reqMethods) > 1 {
		return solana.MultipleValuesRequested
	}
	return ""
}

func (c *CustomContext) SetReqBody(reqBody []byte) {
	c.reqBody = reqBody
}

func (c *CustomContext) GetReqBody() string {
	return string(c.reqBody)
}

func (c *CustomContext) SetResBody(resBody string) {
	c.resBody = resBody
}

func (c *CustomContext) GetResBody() string {
	return c.resBody
}

func (c *CustomContext) SetRpcErrors(rpcErrors []int) {
	c.rpcErrors = rpcErrors
}

func (c *CustomContext) GetRpcErrors() []int {
	return c.rpcErrors
}

func (c *CustomContext) SetProxyEndpoint(proxyEndpoint string) {
	c.proxyEndpoint = proxyEndpoint
}

func (c *CustomContext) GetProxyEndpoint() string {
	return c.proxyEndpoint
}

func (c *CustomContext) SetProxyAttempts(proxyAttempts int) {
	c.proxyAttempts = proxyAttempts
}

func (c *CustomContext) GetProxyAttempts() int {
	return c.proxyAttempts
}

func (c *CustomContext) SetProxyResponseTime(proxyResponseTime int64) {
	c.proxyResponseTime = proxyResponseTime
}

func (c *CustomContext) GetProxyResponseTime() int64 {
	return c.proxyResponseTime
}

func (c *CustomContext) SetProxyUserError(proxyUserError bool) {
	c.proxyUserError = proxyUserError
}

func (c *CustomContext) GetProxyUserError() bool {
	return c.proxyUserError
}

func (c *CustomContext) SetProxyHasError(proxyHasError bool) {
	c.proxyHasError = proxyHasError
}

func (c *CustomContext) GetProxyHasError() bool {
	return c.proxyHasError
}

func (c *CustomContext) SetReqDuration(reqDuration time.Time) {
	c.reqDuration = reqDuration
}

func (c *CustomContext) GetReqDuration() time.Time {
	return c.reqDuration
}

func (c *CustomContext) SetUser(u *auth.UserRecord) {
	c.user = u
}

func (c *CustomContext) GetUser() *auth.UserRecord {
	return c.user
}
