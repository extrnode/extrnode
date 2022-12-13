package solana

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

type TopRpcMethod string

const (
	getAccountInfo          TopRpcMethod = "getAccountInfo"
	sendTransaction                      = "sendTransaction"
	getSignaturesForAddress              = "getSignaturesForAddress"
	getLatestBlockhash                   = "getLatestBlockhash"
	getSlot                              = "getSlot"
	getTransaction                       = "getTransaction"
	getInflationReward                   = "getInflationReward"
	//getProgramAccounts                            = "getProgramAccounts"           rpc.MainNetBeta_RPC don`t implement this method
	getSignatureStatuses             = "getSignatureStatuses"
	getTokenAccountBalance           = "getTokenAccountBalance"
	getMultipleAccounts              = "getMultipleAccounts"
	getEpochInfo                     = "getEpochInfo"
	getBalance                       = "getBalance"
	getRecentPerformanceSamples      = "getRecentPerformanceSamples"
	getVoteAccounts                  = "getVoteAccounts"
	getInflationRate                 = "getInflationRate"
	getSupply                        = "getSupply"
	getBlockTime                     = "getBlockTime"
	getBlockHeight                   = "getBlockHeight"
	getMinimumBalanceForRentExemptio = "getMinimumBalanceForRentExemptio"
	isBlockhashValid                 = "isBlockhashValid"
	getTransactionCount              = "getTransactionCount"
	getTokenAccountsByOwner          = "getTokenAccountsByOwner"
)

const (
	sendTxSanitizeErr = -32602
)

func checkRpcMethod(method TopRpcMethod, rpcClient *rpc.Client, ctx context.Context) (out bool, responseTime time.Duration, code int, IsMethodCorrect error) {

	testKey := solana.SystemProgramID
	testKey2 := solana.MustPublicKeyFromBase58("EverSFw9uN5t1V8kS3ficHUcKffSjwpGzUSGd7mgmSks")
	testKey3 := solana.MustPublicKeyFromBase58("9qGSDWfWn5a7JkvPbuwvkSohMz4VDH6ck7BRJxZFTMbQ")
	testMint := solana.MustPublicKeyFromBase58("Hg35Vd8K3BS2pLB3xwC2WqQV8pmpCm3oNRGYP1PEpmCM")
	testTokenAccount := solana.MustPublicKeyFromBase58("7rEjmuTevAyiY7iUDWT6ucBNHXT2XqjcfQqKvshYrVsh")

	start := time.Now()
	out = false
	code = http.StatusOK
	var err error
	switch method {
	case getAccountInfo:
		if _, err = rpcClient.GetAccountInfo(ctx, testKey); err == nil {
			out = true
		}
	case getBalance:
		if _, err = rpcClient.GetBalance(ctx, testKey, rpc.CommitmentProcessed); err == nil {
			out = true
		}
	case getBlockHeight:
		if _, err = rpcClient.GetBlockHeight(ctx, rpc.CommitmentProcessed); err == nil {
			out = true
		}
	case getBlockTime:
		var block uint64
		block, err = rpcClient.GetFirstAvailableBlock(ctx)
		if err == nil {
			if _, err = rpcClient.GetBlockTime(ctx, block); err == nil {
				out = true
			}
		}
	case getEpochInfo:
		if _, err = rpcClient.GetEpochInfo(ctx, rpc.CommitmentProcessed); err == nil {
			out = true
		}
	case getInflationRate:
		if _, err = rpcClient.GetInflationRate(ctx); err == nil {
			out = true
		}
	case getInflationReward:
		if _, err = rpcClient.GetInflationReward(ctx, []solana.PublicKey{testKey}, nil); err == nil {
			out = true
		}
	case getLatestBlockhash:
		if _, err = rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentProcessed); err == nil {
			out = true
		}
	case getMinimumBalanceForRentExemptio:
		if _, err = rpcClient.GetMinimumBalanceForRentExemption(ctx, 100, rpc.CommitmentProcessed); err == nil {
			out = true
		}
	case getMultipleAccounts:
		if _, err = rpcClient.GetMultipleAccounts(ctx, testKey); err == nil {
			out = true
		}
	//case getProgramAccounts:
	//	if _, err = rpcClient.GetProgramAccounts(ctx, testKey2); err == nil {
	//		out = true
	//	}
	case getRecentPerformanceSamples:
		if _, err = rpcClient.GetRecentPerformanceSamples(ctx, nil); err == nil {
			out = true
		}
	case getSignaturesForAddress:
		if _, err = rpcClient.GetSignaturesForAddress(ctx, testKey2); err == nil {
			out = true
		}
	case getSignatureStatuses:
		var signatures []*rpc.TransactionSignature
		signatures, err = rpcClient.GetSignaturesForAddress(ctx, testKey3)
		if len(signatures) > 0 {
			if _, err = rpcClient.GetSignatureStatuses(ctx, false, signatures[0].Signature); err == nil {
				out = true
			}
		}
	case getSlot:
		if _, err = rpcClient.GetSlot(ctx, rpc.CommitmentProcessed); err == nil {
			out = true
		}
	case getSupply:
		if _, err = rpcClient.GetSupply(ctx, rpc.CommitmentProcessed); err == nil {
			out = true
		}
	case getTokenAccountBalance:
		if _, err = rpcClient.GetTokenAccountBalance(ctx, testTokenAccount, rpc.CommitmentProcessed); err == nil {
			out = true
		}
	case getTokenAccountsByOwner:
		conf := rpc.GetTokenAccountsConfig{
			Mint: &testMint,
		}
		if _, err = rpcClient.GetTokenAccountsByOwner(ctx, testKey2, &conf, nil); err == nil {
			out = true
		}
	case getTransaction:
		var signatures []*rpc.TransactionSignature
		signatures, err = rpcClient.GetSignaturesForAddress(ctx, testKey3)
		if len(signatures) > 0 {
			if _, err = rpcClient.GetTransaction(ctx, signatures[0].Signature, nil); err == nil {
				out = true
			}
		}
	case getTransactionCount:
		if _, err = rpcClient.GetTransactionCount(ctx, rpc.CommitmentProcessed); err == nil {
			out = true
		}
	case getVoteAccounts:
		if _, err = rpcClient.GetVoteAccounts(ctx, nil); err == nil {
			out = true
		}
	case isBlockhashValid:
		var blockhash *rpc.GetLatestBlockhashResult
		blockhash, err = rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
		if err == nil || blockhash != nil {
			if _, err := rpcClient.IsBlockhashValid(ctx, (*blockhash).Value.Blockhash, rpc.CommitmentProcessed); err == nil {
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
			if _, err = rpcClient.SendTransaction(ctx, tx); err == nil || err.(*jsonrpc.RPCError).Code == sendTxSanitizeErr {
				out = true
			}
		}
	default:
		return out, responseTime, code, fmt.Errorf("wrong method send to processing: %s", method)
	}

	responseTime = time.Since(start)
	if err != nil {
		if _, ok := err.(*jsonrpc.RPCError); ok {
			code = http.StatusInternalServerError
		} else if parseErr, ok := err.(*jsonrpc.HTTPError); ok {
			code = parseErr.Code
		} else {
			return out, responseTime, code, err
		}
	}

	return out, responseTime, code, nil
}
