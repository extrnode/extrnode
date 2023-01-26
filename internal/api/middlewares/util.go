package middlewares

import (
	"encoding/json"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
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
