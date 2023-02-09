package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/labstack/echo/v4"

	"extrnode-be/internal/pkg/util/solana"
)

const (
	bodyLimit      = 1000
	jsonrpcVersion = "2.0"
)

func NewValidatorMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := c.(*CustomContext)
			if c.Request().Header.Get(echo.HeaderContentType) != echo.MIMEApplicationJSON {
				return echo.NewHTTPError(http.StatusUnsupportedMediaType, invalidContentTypeErrorResponse)
			}

			// Request
			reqBody := []byte{}
			if c.Request().Body != nil { // Read
				reqBody, _ = io.ReadAll(c.Request().Body)
			}
			c.Request().Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Reset

			reqBody = []byte(strings.Map(func(r rune) rune {
				if unicode.IsSpace(r) {
					return -1
				}
				return r
			}, string(reqBody)))
			if len(reqBody) == 0 {
				return echo.NewHTTPError(http.StatusOK, ParseErrorResponse) // solana mainnet return parse err in this case
			}

			decoder := json.NewDecoder(bytes.NewBuffer(reqBody))
			decoder.DisallowUnknownFields()
			decoder.UseNumber()

			cc.SetReqBody(reqBody)

			var methodArray []string
			switch fs := reqBody[0]; {
			case fs == '{':
				parsedJson := RPCRequest{}
				err := decoder.Decode(&parsedJson)
				if err != nil {
					return echo.NewHTTPError(http.StatusOK, ParseErrorResponse)
				}

				rpcErr := checkJsonRpcBody(parsedJson)
				if rpcErr != nil {
					return echo.NewHTTPError(http.StatusOK, RPCResponse{
						Error:   rpcErr,
						JSONRPC: jsonrpcVersion,
						ID:      parsedJson.ID,
					})
				}

				methodArray = append(methodArray, parsedJson.Method)
			case fs == '[':
				parsedJson := RPCRequests{}
				err := decoder.Decode(&parsedJson)
				if err != nil {
					return echo.NewHTTPError(http.StatusOK, ParseErrorResponse)
				}

				for _, r := range parsedJson {
					if r == nil {
						continue
					}
					rpcErr := checkJsonRpcBody(*r)
					if rpcErr != nil {
						return echo.NewHTTPError(http.StatusOK, RPCResponse{
							Error:   rpcErr,
							JSONRPC: jsonrpcVersion,
							ID:      r.ID,
						})
					}
					methodArray = append(methodArray, r.Method)
				}
			default:
				return echo.NewHTTPError(http.StatusOK, ParseErrorResponse)
			}
			cc.SetReqMethods(methodArray)

			return next(c)
		}
	}
}

func checkJsonRpcBody(req RPCRequest) *jsonrpc.RPCError {
	if req.JSONRPC != jsonrpcVersion {
		return invalidReqError
	}
	_, ok := solana.FullMethodList[req.Method]
	if !ok {
		return methodNotFoundError
	}

	return nil
}
