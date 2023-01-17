package solana

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

type TopRpcMethod string

const (
	getAccountInfo                    TopRpcMethod = "getAccountInfo"
	sendTransaction                                = "sendTransaction"
	getSignaturesForAddress                        = "getSignaturesForAddress"
	getLatestBlockhash                             = "getLatestBlockhash"
	getSlot                                        = "getSlot"
	getTransaction                                 = "getTransaction"
	getInflationReward                             = "getInflationReward"
	getProgramAccounts                             = "getProgramAccounts"
	getSignatureStatuses                           = "getSignatureStatuses"
	getTokenAccountBalance                         = "getTokenAccountBalance"
	getMultipleAccounts                            = "getMultipleAccounts"
	getEpochInfo                                   = "getEpochInfo"
	getBalance                                     = "getBalance"
	getRecentPerformanceSamples                    = "getRecentPerformanceSamples"
	getVoteAccounts                                = "getVoteAccounts"
	getInflationRate                               = "getInflationRate"
	getSupply                                      = "getSupply"
	getBlockTime                                   = "getBlockTime"
	getBlockHeight                                 = "getBlockHeight"
	getMinimumBalanceForRentExemption              = "getMinimumBalanceForRentExemption"
	isBlockhashValid                               = "isBlockhashValid"
	getTransactionCount                            = "getTransactionCount"
	getTokenAccountsByOwner                        = "getTokenAccountsByOwner"
)

const (
	sendTxSanitizeErr = -32602
)

var (
	solanaProgramOwner = solana.MustPublicKeyFromBase58("NativeLoader1111111111111111111111111111111")
	testKey2           = solana.MustPublicKeyFromBase58("EverSFw9uN5t1V8kS3ficHUcKffSjwpGzUSGd7mgmSks")
	testKey3           = solana.MustPublicKeyFromBase58("9qGSDWfWn5a7JkvPbuwvkSohMz4VDH6ck7BRJxZFTMbQ")
	testMint           = solana.MustPublicKeyFromBase58("Hg35Vd8K3BS2pLB3xwC2WqQV8pmpCm3oNRGYP1PEpmCM")
	testTokenAccount   = solana.MustPublicKeyFromBase58("7rEjmuTevAyiY7iUDWT6ucBNHXT2XqjcfQqKvshYrVsh")
)

func checkRpcMethod(method TopRpcMethod, rpcClient *rpc.Client, ctx context.Context) (out bool, responseTime time.Duration, code int, err error) {
	code = http.StatusOK
	start := time.Now()

	switch method {
	case getAccountInfo:
		var resp *rpc.GetAccountInfoResult
		if resp, err = rpcClient.GetAccountInfo(ctx, solana.SystemProgramID); err == nil && resp != nil && resp.Value != nil && resp.Value.Owner == solanaProgramOwner {
			out = true
		}
	case getBalance:
		var resp *rpc.GetBalanceResult
		if resp, err = rpcClient.GetBalance(ctx, solana.SystemProgramID, rpc.CommitmentProcessed); err == nil && resp != nil && resp.Value == 1 {
			out = true
		}
	case getBlockHeight:
		var resp uint64
		if resp, err = rpcClient.GetBlockHeight(ctx, rpc.CommitmentProcessed); err == nil && resp > 0 {
			out = true
		}
	case getBlockTime:
		var block uint64
		block, err = rpcClient.GetSlot(ctx, rpc.CommitmentProcessed)
		if err == nil {
			var resp *solana.UnixTimeSeconds
			if resp, err = rpcClient.GetBlockTime(ctx, block-100); err == nil && resp != nil && resp.Time().Unix() > 0 {
				out = true
			}
		}
	case getEpochInfo:
		var resp *rpc.GetEpochInfoResult
		if resp, err = rpcClient.GetEpochInfo(ctx, rpc.CommitmentProcessed); err == nil && resp != nil && resp.TransactionCount != nil {
			out = true
		}
	case getInflationRate:
		var resp *rpc.GetInflationRateResult
		if resp, err = rpcClient.GetInflationRate(ctx); err == nil && resp != nil && (resp.Validator+resp.Total+resp.Foundation+resp.Epoch) > 0 {
			out = true
		}
	case getInflationReward:
		var resp []*rpc.GetInflationRewardResult
		if resp, err = rpcClient.GetInflationReward(ctx, []solana.PublicKey{solana.SystemProgramID}, nil); err == nil && len(resp) == 1 {
			out = true
		}
	case getLatestBlockhash:
		var resp *rpc.GetLatestBlockhashResult
		if resp, err = rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentProcessed); err == nil && resp != nil && resp.Value != nil && !resp.Value.Blockhash.IsZero() {
			out = true
		}
	case getMinimumBalanceForRentExemption:
		var resp uint64
		if resp, err = rpcClient.GetMinimumBalanceForRentExemption(ctx, 100, rpc.CommitmentProcessed); err == nil && resp > 0 {
			out = true
		}
	case getMultipleAccounts:
		var resp *rpc.GetMultipleAccountsResult
		if resp, err = rpcClient.GetMultipleAccounts(ctx, solana.SystemProgramID); err == nil && resp != nil && resp.Value != nil && len(resp.Value) > 0 && resp.Value[0].Owner == solanaProgramOwner {
			out = true
		}
	case getProgramAccounts:
		var resp rpc.GetProgramAccountsResult
		if resp, err = rpcClient.GetProgramAccounts(ctx, testKey2); err == nil && len(resp) > 0 {
			out = true
		}
	case getRecentPerformanceSamples:
		var resp []*rpc.GetRecentPerformanceSamplesResult
		if resp, err = rpcClient.GetRecentPerformanceSamples(ctx, nil); err == nil && len(resp) > 0 {
			out = true
		}
	case getSignaturesForAddress:
		var resp []*rpc.TransactionSignature
		if resp, err = rpcClient.GetSignaturesForAddress(ctx, testKey2); err == nil && len(resp) > 0 {
			out = true
		}
	case getSignatureStatuses:
		var signatures []*rpc.TransactionSignature
		signatures, err = rpcClient.GetSignaturesForAddress(ctx, testKey3)
		if len(signatures) > 0 {
			var resp *rpc.GetSignatureStatusesResult
			if resp, err = rpcClient.GetSignatureStatuses(ctx, true, signatures[0].Signature); err == nil && len(resp.Value) > 0 && resp.Value[0] != nil {
				out = true
			}
		}
	case getSlot:
		var resp uint64
		if resp, err = rpcClient.GetSlot(ctx, rpc.CommitmentProcessed); err == nil && resp > 0 {
			out = true
		}
	case getSupply:
		var resp *rpc.GetSupplyResult
		if resp, err = rpcClient.GetSupply(ctx, rpc.CommitmentProcessed); err == nil && resp != nil && resp.Value != nil && len(resp.Value.NonCirculatingAccounts) > 0 {
			out = true
		}
	case getTokenAccountBalance:
		var resp *rpc.GetTokenAccountBalanceResult
		if resp, err = rpcClient.GetTokenAccountBalance(ctx, testTokenAccount, rpc.CommitmentProcessed); err == nil && resp != nil && resp.Value != nil && resp.Value.Decimals > 0 {
			out = true
		}
	case getTokenAccountsByOwner:
		conf := rpc.GetTokenAccountsConfig{
			Mint: &testMint,
		}
		var resp *rpc.GetTokenAccountsResult
		if resp, err = rpcClient.GetTokenAccountsByOwner(ctx, testKey3, &conf, nil); err == nil && resp != nil && len(resp.Value) > 0 && resp.Value[0] != nil && resp.Value[0].Account.Owner == solana.TokenProgramID {
			out = true
		}
	case getTransaction:
		var signatures []*rpc.TransactionSignature
		signatures, err = rpcClient.GetSignaturesForAddress(ctx, testKey3)
		if len(signatures) > 0 {
			var resp *rpc.GetTransactionResult
			if resp, err = rpcClient.GetTransaction(ctx, signatures[0].Signature, nil); err == nil && resp.BlockTime != nil && resp.BlockTime.Time().Unix() > 0 {
				out = true
			}
		}
	case getTransactionCount:
		var resp uint64
		if resp, err = rpcClient.GetTransactionCount(ctx, rpc.CommitmentProcessed); err == nil && resp > 0 {
			out = true
		}
	case getVoteAccounts:
		var resp *rpc.GetVoteAccountsResult
		if resp, err = rpcClient.GetVoteAccounts(ctx, nil); err == nil && resp != nil && len(resp.Current) > 0 {
			out = true
		}
	case isBlockhashValid:
		var blockhash *rpc.GetLatestBlockhashResult
		blockhash, err = rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err == nil && blockhash != nil && blockhash.Value != nil {
			var resp *rpc.IsValidBlockhashResult
			if resp, err = rpcClient.IsBlockhashValid(ctx, blockhash.Value.Blockhash, rpc.CommitmentFinalized); err == nil && resp != nil && resp.Value {
				out = true
			}
		}
	case sendTransaction:
		var blockhash *rpc.GetRecentBlockhashResult
		blockhash, err = rpcClient.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
		if err == nil {
			var tx *solana.Transaction
			tx, err = solana.NewTransaction(
				[]solana.Instruction{
					system.NewTransferInstruction(
						1,
						testKey3,
						testKey3,
					).Build(),
				},
				blockhash.Value.Blockhash,
				solana.TransactionPayer(testKey3),
			)

			_, err = rpcClient.SendTransaction(ctx, tx)
			if err != nil {
				if rpcErr, ok := err.(*jsonrpc.RPCError); ok && rpcErr.Code == sendTxSanitizeErr {
					out = true
					err = nil // reset err for err check below switch
				}
			}
		}
	default:
		return out, responseTime, code, fmt.Errorf("wrong method send to processing: %s", method)
	}

	responseTime = time.Since(start)
	if err != nil {
		// make sure 'out' is set to false in err case
		out = false

		if typedErr, ok := err.(*jsonrpc.RPCError); ok {
			// rm popular errors
			if typedErr.Code == -32601 || typedErr.Code == -32011 ||
				method == getInflationReward && (typedErr.Code == -32004 || typedErr.Code == -32001) ||
				method == getTokenAccountsByOwner && typedErr.Code == -32010 || method == getProgramAccounts && typedErr.Code == -32010 ||
				method == getBlockTime && typedErr.Code == -32004 {
				err = nil
			}
			code = http.StatusInternalServerError
		} else if parseErr, ok := err.(*jsonrpc.HTTPError); ok {
			code = parseErr.Code
		} else if strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "use of closed network connection") {
			code = http.StatusRequestTimeout
			err = nil
		}
	}

	return out, responseTime, code, err
}
