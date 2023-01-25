package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

var ErrInvalidRequest = errors.New("invalid request")

const (
	BlockCleanedUpErrCode                           = -32001
	SendTransactionPreflightFailureErrCode          = -32002
	TransactionSignatureVerificationFailureErrCode  = -32003
	BlockNotAvailableErrCode                        = -32004
	NodeUnhealthyErrCode                            = -32005
	TransactionPrecompileVerificationFailureErrCode = -32006
	SlotSkippedErrCode                              = -32007
	NoSnapshotErrCode                               = -32008
	LongTermStorageSlotSkippedErrCode               = -32009
	KeyExcludedFromSecondaryIndexErrCode            = -32010
	TransactionHistoryNotAvailableErrCode           = -32011
	ScanErrCode                                     = -32012
	TransactionSignatureLenMismatchErrCode          = -32013
	BlockStatusNotAvailableYetErrCode               = -32014
	UnsupportedTransactionVersionErrCode            = -32015
	MinContextSlotNotReachedErrCode                 = -32016
	ParseErrCode                                    = -32700
	InvalidRequestErrCode                           = -32600
	MethodNotFoundErrCode                           = -32601
	InvalidParamsErrCode                            = -32602
	InternalErrorErrCode                            = -32603
)

const bodyLimit = 1000

func (ptc *proxyTransportWithContext) decodeNodeResponse(httpResponse *http.Response) (errs []error) {
	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return append(errs, fmt.Errorf("ReadAll: %s", err))
	}

	httpResponse.Body = io.NopCloser(bytes.NewBuffer(body))
	decoder := json.NewDecoder(bytes.NewBuffer(body))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()

	// trim after cloning
	body = bytes.TrimSpace(body)
	// truncate body for context
	if len(body) > bodyLimit {
		body = body[:bodyLimit]
	}
	// save truncated body to context before handling it. Used in logger
	ptc.c.Set(ptc.config.ResBodyContextKey, body)

	if len(body) == 0 {
		return append(errs, errors.New("empty body"))
	}

	var errCodes []int
	switch fs := body[0]; {
	case fs == '{':
		var rpcResponse jsonrpc.RPCResponse
		err = decoder.Decode(&rpcResponse)
		if err != nil {
			return append(errs, fmt.Errorf("error while parsing response: %s", err))
		}

		if rpcResponse.JSONRPC == "" {
			errs = append(errs, fmt.Errorf("empty response body"))
			break
		}
		if rpcResponse.Error != nil {
			if rpcResponse.Error.Code != 0 {
				errCodes = append(errCodes, rpcResponse.Error.Code)
			}
			errs = append(errs, rpcResponse.Error)
		}
	case fs == '[':
		var rpcResponse jsonrpc.RPCResponses
		err = decoder.Decode(&rpcResponse)
		if err != nil {
			return append(errs, fmt.Errorf("error while parsing response: %s", err))
		}

		for _, r := range rpcResponse {
			if r.JSONRPC == "" {
				errs = append(errs, fmt.Errorf("empty response body"))
				continue
			}
			if r.Error != nil {
				if r.Error.Code != 0 {
					errCodes = append(errCodes, r.Error.Code)
				}
				errs = append(errs, r.Error)
			}
		}
	default:
		return append(errs, fmt.Errorf("invalid json first symbol: %s", string(fs)))
	}

	if len(errCodes) != 0 {
		ptc.c.Set(ptc.config.RpcErrorContextKey, errCodes)
	}
	if len(errs) != 0 {
		return errs
	}

	return nil
}

func rpcErrorAnalysis(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	var joinedErr error
	for _, err := range errs {
		rpcErr, ok := err.(*jsonrpc.RPCError)
		if !ok {
			joinedErr = fmt.Errorf("%s; %s", joinedErr, err)
			continue
		}

		switch rpcErr.Code {
		case SendTransactionPreflightFailureErrCode, TransactionSignatureVerificationFailureErrCode,
			TransactionPrecompileVerificationFailureErrCode, TransactionSignatureLenMismatchErrCode,
			UnsupportedTransactionVersionErrCode, ParseErrCode, InvalidRequestErrCode,
			InvalidParamsErrCode:
			return ErrInvalidRequest
		default:
			joinedErr = fmt.Errorf("%s; %s", joinedErr, err)
			continue
		}
	}

	return joinedErr
}

func (ptc *proxyTransportWithContext) getResponseError(httpResponse *http.Response) error {
	err := ptc.decodeNodeResponse(httpResponse)
	if err == nil {
		return nil
	}

	return rpcErrorAnalysis(err)
}
