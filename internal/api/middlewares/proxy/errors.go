package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"

	"extrnode-be/internal/api/middlewares"
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

	jsonMsgNullString = "null"
)

const bodyLimit = 1000

func (ptc *proxyTransportWithContext) decodeNodeResponse(httpResponse *http.Response) (errs []error) {
	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return append(errs, fmt.Errorf("ReadAll: %s", err))
	}

	httpResponse.Body = io.NopCloser(bytes.NewBuffer(body))
	decoder := json.NewDecoder(bytes.NewBuffer(body))
	decoder.UseNumber()

	// trim after cloning
	body = bytes.TrimSpace(body)
	// truncate body for context
	if len(body) > bodyLimit {
		body = body[:bodyLimit]
	}
	// save truncated body to context before handling it. Used in logger
	ptc.c.SetResBody(body)
	// clean possible old value
	ptc.c.SetRpcErrors(nil)

	if len(body) == 0 {
		return append(errs, errors.New("empty body"))
	}

	var errCodes []int
	switch fs := body[0]; {
	case fs == '{':
		var rpcResponse middlewares.RPCResponse
		rpcMethod := ptc.c.GetReqMethod()
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
			break
		}

		if string(rpcResponse.Result) == jsonMsgNullString && rpcMethod == "getBlock" {
			errs = append(errs, fmt.Errorf("empty response field"))
		}
	case fs == '[':
		var rpcResponse middlewares.RPCResponses
		rpcMethods := ptc.c.GetReqMethods()
		err = decoder.Decode(&rpcResponse)
		if err != nil {
			return append(errs, fmt.Errorf("error while parsing response: %s", err))
		}

		for key, r := range rpcResponse {
			if r == nil {
				errs = append(errs, fmt.Errorf("empty response"))
				continue
			}
			if r.JSONRPC == "" {
				errs = append(errs, fmt.Errorf("empty response body"))
				continue
			}
			if r.Error != nil {
				if r.Error.Code != 0 {
					errCodes = append(errCodes, r.Error.Code)
				}
				errs = append(errs, r.Error)
				continue
			}
			if len(rpcMethods) != len(rpcResponse) {
				continue
			}
			if string(r.Result) == jsonMsgNullString && rpcMethods[key] == "getBlock" {
				errs = append(errs, fmt.Errorf("empty response field"))
			}
		}
	default:
		return append(errs, fmt.Errorf("invalid json first symbol: %s", string(fs)))
	}

	if len(errCodes) != 0 {
		ptc.c.SetRpcErrors(errCodes)
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

	var joinedErr string
	for _, err := range errs {
		rpcErr, ok := err.(*jsonrpc.RPCError)
		if !ok {
			joinedErr = fmt.Sprintf("%s%s; ", joinedErr, err.Error())
			continue
		}

		switch rpcErr.Code {
		case SendTransactionPreflightFailureErrCode, TransactionSignatureVerificationFailureErrCode,
			TransactionPrecompileVerificationFailureErrCode, TransactionSignatureLenMismatchErrCode,
			UnsupportedTransactionVersionErrCode, ParseErrCode, InvalidRequestErrCode,
			InvalidParamsErrCode:
			if rpcErr.Code == InvalidParamsErrCode &&
				(strings.Contains(rpcErr.Message, "BigTable query failed (maybe timeout due to too large range") ||
					strings.Contains(rpcErr.Message, "blockstore error")) {
				break
			}

			return ErrInvalidRequest
		}

		joinedErr = fmt.Sprintf("%srpcErr: code %d %s; ", joinedErr, rpcErr.Code, rpcErr.Message)
	}

	return errors.New(joinedErr)
}

func (ptc *proxyTransportWithContext) getResponseError(httpResponse *http.Response) error {
	err := ptc.decodeNodeResponse(httpResponse)
	if err == nil {
		return nil
	}

	return rpcErrorAnalysis(err)
}
