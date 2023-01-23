package middlewares

import (
	"encoding/json"
	"strings"
	"unicode"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	bodyLimit = 1000
)

type (
	RPCRequest struct {
		Method string `json:"method"`
		// unnecessary fields removed
	}

	RPCResponse struct {
		Error RPCError `json:"error,omitempty"`
	}
	RPCError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
)

type BodyDumpContextConfig struct {
	ReqMethodContextKey string
	ReqBodyContextKey   string
	ResBodyContextKey   string
	RpcErrorContextKey  string
}

func NewBodyDumpMiddleware(config BodyDumpContextConfig) echo.MiddlewareFunc {
	return middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		var (
			parsedReq RPCRequest
			parsedRes RPCResponse
		)
		_ = json.Unmarshal(reqBody, &parsedReq) // ignore err
		_ = json.Unmarshal(resBody, &parsedRes) // ignore err

		if len(reqBody) > bodyLimit {
			reqBody = reqBody[:bodyLimit]
		}
		if len(resBody) > bodyLimit {
			resBody = resBody[:bodyLimit]
		}

		reqBody = []byte(strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, string(reqBody)))
		resBody = []byte(strings.TrimSpace(string(resBody)))

		c.Set(config.ReqBodyContextKey, reqBody)
		c.Set(config.ResBodyContextKey, resBody)
		c.Set(config.ReqMethodContextKey, parsedReq.Method)
		if parsedRes.Error.Code != 0 {
			c.Set(config.RpcErrorContextKey, parsedRes.Error.Code)
		}
	})
}
