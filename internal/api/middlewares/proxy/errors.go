package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

var ErrInvalidRequest = fmt.Errorf("error: invalid request")

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

func decodeNodeResponse(httpResponse *http.Response) (err error) {
	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return fmt.Errorf("ReadAll: %s", err)
	}

	httpResponse.Body = io.NopCloser(bytes.NewBuffer(body))
	decoder := json.NewDecoder(bytes.NewBuffer(body))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()

	var rpcResponse jsonrpc.RPCResponse
	err = decoder.Decode(&rpcResponse)
	if err != nil {
		return fmt.Errorf("error while parsing response: %s", err.Error())
	}

	if rpcResponse.JSONRPC == "" {
		return fmt.Errorf("empty response body")
	}

	if rpcResponse.Error != nil {
		return rpcResponse.Error
	}

	return nil
}

func rpcErrorAnalysis(err error) error {
	rpcErr, ok := err.(*jsonrpc.RPCError)
	if !ok {
		return err
	}

	switch rpcErr.Code {
	case SendTransactionPreflightFailureErrCode, TransactionSignatureVerificationFailureErrCode,
		TransactionPrecompileVerificationFailureErrCode, TransactionSignatureLenMismatchErrCode,
		UnsupportedTransactionVersionErrCode, ParseErrCode, InvalidRequestErrCode,
		InvalidParamsErrCode:
		return ErrInvalidRequest
	default:
		return err
	}
}

func getResponseError(httpResponse *http.Response) error {
	err := decodeNodeResponse(httpResponse)
	if err == nil {
		return nil
	}

	return rpcErrorAnalysis(err)
}
