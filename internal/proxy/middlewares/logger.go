package middlewares

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/blake2b"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage/clickhouse"
	echo2 "extrnode-be/internal/pkg/util/echo"
	solana2 "extrnode-be/internal/pkg/util/solana"
)

func NewLoggerMiddleware(saveLog func(s clickhouse.Stat)) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:    true,
		LogMethod:    true,
		LogRequestID: true,
		LogLatency:   true,
		LogError:     true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogURI:       true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			cc := c.(*echo2.CustomContext)
			reqBody := cc.GetReqBody()

			saveLog(buildStatStruct(v.RemoteIP, v.RequestID, v.Status, v.Latency.Milliseconds(), cc.GetProxyEndpoint(),
				cc.GetProxyAttempts(), cc.GetProxyResponseTime(), cc.GetReqMethods(), cc.GetRpcErrors(), v.UserAgent, reqBody))

			// truncate before log
			if len(reqBody) > bodyLimit {
				reqBody = reqBody[:bodyLimit]
			}

			if v.Error != nil || len(cc.GetRpcErrors()) != 0 || v.Status >= http.StatusBadRequest {
				log.Logger.Proxy.Errorf("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %v, attempts: %d, node_response_time: %dms, "+
					"rpc_error_code: %v, error: %s, request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s, path: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), cc.GetProxyEndpoint(), cc.GetReqMethods(), cc.GetProxyAttempts(), cc.GetProxyResponseTime(),
					cc.GetRpcErrors(), errMsg(v.Error), reqBody, cc.GetResBody(), v.RemoteIP, v.UserAgent, v.URI)
			} else {
				log.Logger.Proxy.Infof("%d %s, id: %s, latency: %d, endpoint: %s, rpc_method: %v, attempts: %d, node_response_time: %dms, "+
					"request_body: %s, response_body: %s, remote_ip: %s, user_agent: %s, path: %s",
					v.Status, v.Method, v.RequestID, v.Latency.Milliseconds(), cc.GetProxyEndpoint(), cc.GetReqMethods(), cc.GetProxyAttempts(), cc.GetProxyResponseTime(),
					reqBody, cc.GetResBody(), v.RemoteIP, v.UserAgent, v.URI)
			}

			return nil
		},
	})
}

func buildStatStruct(ip, requestId string, statusCode int, latency int64, endpoint string, attempts int, responseTime int64,
	rpcMethods []string, rpcErrorCodes []int, userAgent, reqBody string) clickhouse.Stat {
	var rpcMethod string
	if len(rpcMethods) > 1 {
		rpcMethod = solana2.MultipleValuesRequested
	} else if len(rpcMethods) == 1 {
		rpcMethod = rpcMethods[0]
	}
	var rpcErrorCodeString string
	if len(rpcErrorCodes) > 1 {
		rpcErrorCodeString = solana2.MultipleValuesRequested
	} else if len(rpcErrorCodes) == 1 {
		rpcErrorCodeString = fmt.Sprintf("%d", rpcErrorCodes[0])
	}

	userUUidHash := blake2b.Sum256([]byte(ip))

	return clickhouse.Stat{
		UserUUID:       hex.EncodeToString(userUUidHash[:]),
		RequestID:      requestId,
		Status:         uint16(statusCode),
		ExecutionTime:  latency,
		Endpoint:       endpoint,
		Attempts:       uint8(attempts),
		ResponseTime:   responseTime,
		RpcErrorCode:   rpcErrorCodeString,
		UserAgent:      userAgent,
		RpcMethod:      rpcMethod,
		RpcRequestData: getContextValueForRequest(rpcMethod, reqBody),
		Timestamp:      time.Now().UTC(),
	}
}

func getContextValueForRequest(rpcMethod, reqBody string) (res string) {
	if rpcMethod == "" || rpcMethod == solana2.MultipleValuesRequested || reqBody == "" {
		return
	}

	var parsedJson RPCRequest
	switch fs := reqBody[0]; {
	case fs == '{':
		err := newJsonDecoder([]byte(reqBody), false).Decode(&parsedJson)
		if err != nil {
			log.Logger.Proxy.Errorf("getContextValueForRequest: json.Unmarshal: %s", err)
			return
		}
	case fs == '[':
		var parsedJsons []RPCRequest
		err := newJsonDecoder([]byte(reqBody), false).Decode(&parsedJsons)
		if err != nil {
			log.Logger.Proxy.Errorf("getContextValueForRequest: json.Unmarshal: %s", err)
			return
		}
		if len(parsedJsons) == 0 {
			log.Logger.Proxy.Errorf("getContextValueForRequest: invalid reqBody: %s", reqBody)
			return
		}

		parsedJson = parsedJsons[0]
	default:
		log.Logger.Proxy.Errorf("invalid json first symbol: %s", string(fs))
		return
	}

	switch parsedJson.Method {
	case solana2.GetSignaturesForAddress, solana2.GetTokenAccountsByOwner, solana2.GetAccountInfo, solana2.GetProgramAccounts, solana2.SendTransaction,
		solana2.GetStakeActivation, solana2.GetTokenAccountBalance, solana2.GetTokenAccountsByDelegate, solana2.GetTokenLargestAccounts,
		solana2.GetTokenSupply, solana2.IsBlockhashValid, solana2.GetTransaction, solana2.GetBalance:
		if paramsArr, ok := parsedJson.Params.([]interface{}); ok && len(paramsArr) > 0 {
			res, _ = paramsArr[0].(string)
		}
	case solana2.GetBlock, solana2.GetBlocks, solana2.GetBlockCommitment, solana2.GetBlocksWithLimit, solana2.GetBlockTime:
		if paramsArr, ok := parsedJson.Params.([]interface{}); ok && len(paramsArr) > 0 {
			resNumber, _ := paramsArr[0].(json.Number)
			res = resNumber.String()
		}
	}

	if parsedJson.Method == solana2.SendTransaction {
		var tx solana.Transaction
		err := tx.UnmarshalBase64(res)
		if err != nil {
			log.Logger.Proxy.Errorf("getContextValueForRequest: tx.Unmarshal: %s", err)
			return ""
		}
		// unset raw tx
		res = ""

		if len(tx.Message.Instructions) > 0 {
			res = tx.Message.AccountKeys[tx.Message.Instructions[0].ProgramIDIndex].String()
		}
	}

	return
}
