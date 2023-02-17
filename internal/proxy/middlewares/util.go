package middlewares

import (
	"bytes"
	"encoding/json"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/labstack/echo/v4"
)

type (
	// copied from jsonrpc, 'id' field changed to interface (can be int, string, null)
	RPCRequest struct {
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
		ID      interface{} `json:"id"`
		JSONRPC string      `json:"jsonrpc"`
	}
	RPCRequests []*RPCRequest

	RPCResponse struct {
		JSONRPC string            `json:"jsonrpc"`
		Result  json.RawMessage   `json:"result,omitempty"`
		Error   *jsonrpc.RPCError `json:"error,omitempty"`
		ID      interface{}       `json:"id"`
	}
	RPCResponses []*RPCResponse
)

func errMsg(err error) string {
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr == nil {
		return ""
	}
	rpcResponse, ok := httpErr.Message.(*RPCResponse)
	if !ok || rpcResponse == nil || rpcResponse.Error == nil {
		return ""
	}

	return rpcResponse.Error.Message
}

var (
	parseErrorResponse = &RPCResponse{
		Error: &jsonrpc.RPCError{
			Code:    -32700,
			Message: "Parse error",
		},
		JSONRPC: jsonrpcVersion,
	}
	extraNodeNoAvailableTargetsErrorResponse = &RPCResponse{
		Error: &jsonrpc.RPCError{
			Code:    2000,
			Message: "No available targets",
		},
		JSONRPC: jsonrpcVersion,
	}
	invalidContentTypeErrorResponse = &RPCResponse{
		Error: &jsonrpc.RPCError{
			Code:    415,
			Message: "Invalid content-type, this application only supports application/json",
		},
		JSONRPC: jsonrpcVersion,
	}
	invalidReqError = &jsonrpc.RPCError{
		Code:    -32600,
		Message: "Invalid request",
	}
	methodNotFoundError = &jsonrpc.RPCError{
		Code:    -32601,
		Message: "Method not found",
	}
)

func newJsonDecoder(data []byte, disallowUnknownFields bool) (decoder *json.Decoder) {
	decoder = json.NewDecoder(bytes.NewBuffer(data))
	decoder.UseNumber()
	if disallowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	return
}
