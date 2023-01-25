package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"github.com/labstack/echo/v4"
)

type ValidatorContextConfig struct {
	ReqMethodContextKey string
	ReqBodyContextKey   string
}

const bodyLimit = 1000

func NewValidatorMiddleware(config ValidatorContextConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
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
				return fmt.Errorf("empty body")
			}

			var methodArray []string
			switch fs := reqBody[0]; {
			case fs == '{':
				parsedJson := jsonrpc.RPCRequest{}
				err := json.Unmarshal(reqBody, &parsedJson)
				if err != nil {
					return fmt.Errorf("unmarshal: %s", err)
				}

				methodArray = append(methodArray, parsedJson.Method)
			case fs == '[':
				parsedJson := jsonrpc.RPCRequests{}
				err := json.Unmarshal(reqBody, &parsedJson)
				if err != nil {
					return fmt.Errorf("unmarshal: %s", err)
				}

				for _, r := range parsedJson {
					methodArray = append(methodArray, r.Method)
				}
			default:
				return fmt.Errorf("invalid json first symbol: %s", string(fs))
			}

			for _, m := range methodArray {
				_, ok := fullMethodList[m]
				if !ok {
					// return understandable error for user
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid method: %s", m))
				}
			}

			// truncate body
			if len(reqBody) > bodyLimit {
				reqBody = reqBody[:bodyLimit]
			}
			c.Set(config.ReqBodyContextKey, reqBody)
			c.Set(config.ReqMethodContextKey, methodArray)

			return next(c)
		}
	}
}

var fullMethodList = map[string]struct{}{
	"getAccountInfo":                    {},
	"getBalance":                        {},
	"getBlock":                          {},
	"getBlockHeight":                    {},
	"getBlockProduction":                {},
	"getBlockCommitment":                {},
	"getBlocks":                         {},
	"getBlocksWithLimit":                {},
	"getBlockTime":                      {},
	"getClusterNodes":                   {},
	"getEpochInfo":                      {},
	"getEpochSchedule":                  {},
	"getFeeForMessage":                  {},
	"getFirstAvailableBlock":            {},
	"getGenesisHash":                    {},
	"getHealth":                         {},
	"getHighestSnapshotSlot":            {},
	"getIdentity":                       {},
	"getInflationGovernor":              {},
	"getInflationRate":                  {},
	"getInflationReward":                {},
	"getLargestAccounts":                {},
	"getLatestBlockhash":                {},
	"getLeaderSchedule":                 {},
	"getMaxRetransmitSlot":              {},
	"getMaxShredInsertSlot":             {},
	"getMinimumBalanceForRentExemption": {},
	"getMultipleAccounts":               {},
	"getProgramAccounts":                {},
	"getRecentPerformanceSamples":       {},
	"getRecentPrioritizationFees":       {},
	"getSignaturesForAddress":           {},
	"getSignatureStatuses":              {},
	"getSlot":                           {},
	"getSlotLeader":                     {},
	"getSlotLeaders":                    {},
	"getStakeActivation":                {},
	"getStakeMinimumDelegation":         {},
	"getSupply":                         {},
	"getTokenAccountBalance":            {},
	"getTokenAccountsByDelegate":        {},
	"getTokenAccountsByOwner":           {},
	"getTokenLargestAccounts":           {},
	"getTokenSupply":                    {},
	"getTransaction":                    {},
	"getTransactionCount":               {},
	"getVersion":                        {},
	"getVoteAccounts":                   {},
	"isBlockhashValid":                  {},
	"minimumLedgerSlot":                 {},
	"requestAirdrop":                    {},
	"sendTransaction":                   {},
	"simulateTransaction":               {},
}
