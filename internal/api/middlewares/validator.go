package middlewares

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/labstack/echo/v4"
)

type ValidatorContextConfig struct {
	ReqMethodContextKey string
	ReqBodyContextKey   string
}

const (
	bodyLimit      = 1000
	jsonrpcVersion = "2.0"
)

func NewValidatorMiddleware(config ValidatorContextConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get(echo.HeaderContentType) != echo.MIMEApplicationJSON {
				return echo.NewHTTPError(http.StatusUnsupportedMediaType, "Invalid content-type, this application only supports application/json")
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
				return fmt.Errorf("empty body")
			}

			decoder := json.NewDecoder(bytes.NewBuffer(reqBody))
			decoder.DisallowUnknownFields()
			decoder.UseNumber()

			// save body before handling
			if len(reqBody) > bodyLimit {
				reqBody = reqBody[:bodyLimit]
			}
			c.Set(config.ReqBodyContextKey, reqBody)

			var methodArray []string
			switch fs := reqBody[0]; {
			case fs == '{':
				parsedJson := RPCRequest{}
				err := decoder.Decode(&parsedJson)
				if err != nil {
					// TODO: return 200 and {
					//    "jsonrpc": "2.0",
					//    "error": {
					//        "code": -32600,
					//        "message": "Invalid request"
					//    },
					//    "id": 1
					//}
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("unmarshal: %s", err))
				}

				err = checkJsonRpcBody(parsedJson)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid request: %s", err))
				}

				methodArray = append(methodArray, parsedJson.Method)
			case fs == '[':
				parsedJson := RPCRequests{}
				err := decoder.Decode(&parsedJson)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("unmarshal: %s", err))
				}

				for _, r := range parsedJson {
					if r == nil {
						continue
					}
					err = checkJsonRpcBody(*r)
					if err != nil {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid request: %s", err))
					}
					methodArray = append(methodArray, r.Method)
				}
			default:
				return fmt.Errorf("invalid json first symbol: %s", string(fs))
			}

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

	// deprecated methods, but works now on solana mainnet
	"getConfirmedBlock":                 {},
	"getConfirmedBlocks":                {},
	"getConfirmedBlocksWithLimit":       {},
	"getConfirmedSignaturesForAddress2": {},
	"getConfirmedTransaction":           {},
	"getFeeCalculatorForBlockhash":      {},
	"getFeeRateGovernor":                {},
	"getFees":                           {},
	"getRecentBlockhash":                {},
	"getSnapshotSlot":                   {},
}

func checkJsonRpcBody(req RPCRequest) error {
	if req.JSONRPC != jsonrpcVersion {
		return errors.New("invalid version")
	}
	_, ok := fullMethodList[req.Method]
	if !ok {
		// return understandable error for user
		return fmt.Errorf("invalid method: %s", req.Method)
	}

	return nil
}
