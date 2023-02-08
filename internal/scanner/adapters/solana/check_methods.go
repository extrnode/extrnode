package solana

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"

	solana2 "extrnode-be/internal/pkg/util/solana"
)

const (
	sendTxSanitizeErr = -32602
)

var (
	solanaProgramOwner = solana.MustPublicKeyFromBase58("NativeLoader1111111111111111111111111111111")
	testKey2           = solana.MustPublicKeyFromBase58("EverSFw9uN5t1V8kS3ficHUcKffSjwpGzUSGd7mgmSks")
	testKey3           = solana.MustPublicKeyFromBase58("9qGSDWfWn5a7JkvPbuwvkSohMz4VDH6ck7BRJxZFTMbQ")
	testKey4           = solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111111")
	testMint           = solana.MustPublicKeyFromBase58("Hg35Vd8K3BS2pLB3xwC2WqQV8pmpCm3oNRGYP1PEpmCM")
	testTokenAccount   = solana.MustPublicKeyFromBase58("7rEjmuTevAyiY7iUDWT6ucBNHXT2XqjcfQqKvshYrVsh")

	limit = 1
)

func (a *SolanaAdapter) checkRpcMethod(method string, rpcClient *rpc.Client) (out bool, responseTime time.Duration, code int, err error) {
	code = http.StatusOK
	start := time.Now()

	switch method {
	case solana2.GetAccountInfo:
		var resp *rpc.GetAccountInfoResult
		resp, err = rpcClient.GetAccountInfo(a.ctx, solana.SystemProgramID)
		if err == nil && resp != nil && resp.Value != nil && resp.Value.Owner == solanaProgramOwner {
			out = true
		}
	case solana2.GetBalance:
		var resp *rpc.GetBalanceResult
		resp, err = rpcClient.GetBalance(a.ctx, solana.SystemProgramID, rpc.CommitmentProcessed)
		if err == nil && resp != nil && resp.Value == 1 {
			out = true
		}
	case solana2.GetBlockHeight:
		var resp uint64
		resp, err = rpcClient.GetBlockHeight(a.ctx, rpc.CommitmentProcessed)
		if err == nil && resp > 0 {
			out = true
		}
	case solana2.GetBlockTime:
		var block uint64
		block, err = rpcClient.GetSlot(a.ctx, rpc.CommitmentProcessed)
		if err == nil {
			var resp *solana.UnixTimeSeconds
			resp, err = rpcClient.GetBlockTime(a.ctx, block-100)
			if err == nil && resp != nil && resp.Time().Unix() > 0 {
				out = true
			}
		}
	case solana2.GetEpochInfo:
		var resp *rpc.GetEpochInfoResult
		resp, err = rpcClient.GetEpochInfo(a.ctx, rpc.CommitmentProcessed)
		if err == nil && resp != nil && resp.TransactionCount != nil {
			out = true
		}
	case solana2.GetInflationRate:
		var resp *rpc.GetInflationRateResult
		resp, err = rpcClient.GetInflationRate(a.ctx)
		if err == nil && resp != nil &&
			(resp.Validator+resp.Total+resp.Foundation+resp.Epoch) > 0 {
			out = true
		}
	case solana2.GetInflationReward:
		// TODO: temporary remove this check
		//var resp []*rpc.GetInflationRewardResult
		//resp, err = rpcClient.GetInflationReward(a.ctx, []solana.PublicKey{solana.SystemProgramID}, nil)
		//if err == nil && len(resp) == 1 {
		out = true
		//}
	case solana2.GetLatestBlockhash:
		var resp *rpc.GetLatestBlockhashResult
		resp, err = rpcClient.GetLatestBlockhash(a.ctx, rpc.CommitmentProcessed)
		if err == nil && resp != nil && resp.Value != nil && !resp.Value.Blockhash.IsZero() {
			out = true
		}
	case solana2.GetMinimumBalanceForRentExemption:
		var resp uint64
		resp, err = rpcClient.GetMinimumBalanceForRentExemption(a.ctx, 100, rpc.CommitmentProcessed)
		if err == nil && resp > 0 {
			out = true
		}
	case solana2.GetMultipleAccounts:
		var resp *rpc.GetMultipleAccountsResult
		resp, err = rpcClient.GetMultipleAccounts(a.ctx, solana.SystemProgramID)
		if err == nil && resp != nil && resp.Value != nil &&
			len(resp.Value) > 0 && resp.Value[0].Owner == solanaProgramOwner {
			out = true
		}
	case solana2.GetProgramAccounts:
		var resp rpc.GetProgramAccountsResult
		resp, err = rpcClient.GetProgramAccounts(a.ctx, testKey2)
		if err == nil && len(resp) > 0 {
			out = true
		}
	case solana2.GetRecentPerformanceSamples:
		var resp []*rpc.GetRecentPerformanceSamplesResult
		resp, err = rpcClient.GetRecentPerformanceSamples(a.ctx, nil)
		if err == nil && len(resp) > 0 {
			out = true
		}
	case solana2.GetSignaturesForAddress:
		var resp []*rpc.TransactionSignature
		resp, err = rpcClient.GetSignaturesForAddressWithOpts(a.ctx, testKey4, &rpc.GetSignaturesForAddressOpts{Limit: &limit})
		if err == nil && len(resp) > 0 {
			out = true
		}
	case solana2.GetSignatureStatuses:
		var resp *rpc.GetSignatureStatusesResult
		resp, err = rpcClient.GetSignatureStatuses(a.ctx, true, a.signatureForAddress)
		if err == nil && len(resp.Value) > 0 && resp.Value[0] != nil {
			out = true
		}
	case solana2.GetSlot:
		var resp uint64
		resp, err = rpcClient.GetSlot(a.ctx, rpc.CommitmentProcessed)
		if err == nil && resp > 0 {
			out = true
		}
	case solana2.GetSupply:
		var resp *rpc.GetSupplyResult
		resp, err = rpcClient.GetSupply(a.ctx, rpc.CommitmentProcessed)
		if err == nil && resp != nil && resp.Value != nil && len(resp.Value.NonCirculatingAccounts) > 0 {
			out = true
		}
	case solana2.GetTokenAccountBalance:
		var resp *rpc.GetTokenAccountBalanceResult
		resp, err = rpcClient.GetTokenAccountBalance(a.ctx, testTokenAccount, rpc.CommitmentProcessed)
		if err == nil && resp != nil && resp.Value != nil && resp.Value.Decimals > 0 {
			out = true
		}
	case solana2.GetTokenAccountsByOwner:
		conf := rpc.GetTokenAccountsConfig{
			Mint: &testMint,
		}
		var resp *rpc.GetTokenAccountsResult
		resp, err = rpcClient.GetTokenAccountsByOwner(a.ctx, testKey3, &conf, nil)
		if err == nil && resp != nil && len(resp.Value) > 0 &&
			resp.Value[0] != nil && resp.Value[0].Account.Owner == solana.TokenProgramID {
			out = true
		}
	case solana2.GetTransaction:
		var resp *rpc.GetTransactionResult
		ops := rpc.GetTransactionOpts{
			MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
		}
		resp, err = rpcClient.GetTransaction(a.ctx, a.signatureForAddress, &ops)
		if err == nil && resp.BlockTime != nil && resp.BlockTime.Time().Unix() > 0 {
			out = true
		}
	case solana2.GetTransactionCount:
		var resp uint64
		resp, err = rpcClient.GetTransactionCount(a.ctx, rpc.CommitmentProcessed)
		if err == nil && resp > 0 {
			out = true
		}
	case solana2.GetVoteAccounts:
		var resp *rpc.GetVoteAccountsResult
		resp, err = rpcClient.GetVoteAccounts(a.ctx, nil)
		if err == nil && resp != nil && len(resp.Current) > 0 {
			out = true
		}
	case solana2.IsBlockhashValid:
		var blockhash *rpc.GetLatestBlockhashResult
		blockhash, err = rpcClient.GetLatestBlockhash(a.ctx, rpc.CommitmentFinalized)
		if err == nil && blockhash != nil && blockhash.Value != nil {
			var resp *rpc.IsValidBlockhashResult
			resp, err = rpcClient.IsBlockhashValid(a.ctx, blockhash.Value.Blockhash, rpc.CommitmentFinalized)
			if err == nil && resp != nil && resp.Value {
				out = true
			}
		}
	case solana2.GetBlock:
		var resp *rpc.GetBlockResult
		slot := a.slot
		for j := 0; j < getBlockTries; j++ {
			resp, err = rpcClient.GetBlockWithOpts(a.ctx, slot, &rpc.GetBlockOpts{
				MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
			})
			if typedErr, ok := err.(*jsonrpc.RPCError); ok && typedErr.Code == slotSkipperErrCode {
				slot = slot + 10
				continue
			}
			if err == nil && !resp.Blockhash.IsZero() {
				out = true
			}
			break
		}
	case solana2.GetVersion:
		out = true // if alghorithm reach this point, then getVersion is working method on node
	case solana2.SendTransaction:
		var blockhash *rpc.GetRecentBlockhashResult
		blockhash, err = rpcClient.GetRecentBlockhash(a.ctx, rpc.CommitmentFinalized)
		if err == nil {
			var tx *solana.Transaction
			tx, _ = solana.NewTransaction(
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

			_, err = rpcClient.SendTransaction(a.ctx, tx)
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
				method == solana2.GetInflationReward && (typedErr.Code == -32004 || typedErr.Code == -32001) ||
				method == solana2.GetTokenAccountsByOwner && typedErr.Code == -32010 || method == solana2.GetProgramAccounts && typedErr.Code == -32010 ||
				method == solana2.GetBlockTime && typedErr.Code == -32004 {
				err = nil
			} else {
				err = reformatSolanaRpcError(err)
			}
			code = http.StatusInternalServerError
		} else if parseErr, ok := err.(*jsonrpc.HTTPError); ok {
			code = parseErr.Code
			if code == http.StatusTooManyRequests {
				err = nil // usually contains multiple line html
			}
		} else if strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "use of closed network connection") {
			code = http.StatusRequestTimeout
			err = nil
		}
	}

	return out, responseTime, code, err
}
