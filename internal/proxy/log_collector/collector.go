package log_collector

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"golang.org/x/crypto/blake2b"

	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/storage/clickhouse"
	solana2 "extrnode-be/internal/pkg/util/solana"
	"extrnode-be/internal/proxy/middlewares"
)

const (
	flushAmount   = 1000
	flushInterval = 10 * time.Second
)

type Collector struct {
	ctx       context.Context
	chStorage *clickhouse.Storage
	cache     []clickhouse.Stat
	mx        sync.Mutex
}

func NewCollector(ctx context.Context, chStorage *clickhouse.Storage) (c *Collector) {
	c = &Collector{
		ctx:       ctx,
		chStorage: chStorage,
		cache:     make([]clickhouse.Stat, 0, flushAmount),
	}

	if chStorage == nil {
		return
	}

	go c.startStatSaver()

	return
}

func (c *Collector) startStatSaver() {
	for {
		select {
		case <-c.ctx.Done():
			c.flushStats()
			return

		case <-time.After(flushInterval):
			c.flushStats()
		}
	}
}

func (c *Collector) AddStat(ip, requestId string, statusCode int, latency int64, endpoint string, attempts int, responseTime int64, rpcMethods []string, rpcErrorCodes []int, userAgent, reqBody string) {
	// if ch not set
	if c.chStorage == nil {
		return
	}

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

	c.addStatToCache(clickhouse.Stat{
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
	})
}

func (c *Collector) addStatToCache(s clickhouse.Stat) {
	c.mx.Lock()
	c.cache = append(c.cache, s)
	c.mx.Unlock()
}

func (c *Collector) getCachedStats() (s []clickhouse.Stat) {
	c.mx.Lock()
	s = c.cache
	c.cache = make([]clickhouse.Stat, 0, flushAmount)
	c.mx.Unlock()

	return
}

func (c *Collector) flushStats() {
	if c.chStorage == nil {
		return
	}
	entries := c.getCachedStats()
	if len(entries) == 0 {
		return
	}

	log.Logger.Proxy.Debugf("Collector: flushing %d logs to db", len(entries))

	timeNow := time.Now()
	err := c.chStorage.BatchInsertStats(entries)
	if err != nil {
		log.Logger.Proxy.Errorf("Collector: saver: flushStats: %s", err)
		return
	}

	log.Logger.Proxy.Debugf("Collector: fin flushing. Elapsed %s", time.Since(timeNow))
}

func getContextValueForRequest(rpcMethod, reqBody string) (res string) {
	if rpcMethod == "" || rpcMethod == solana2.MultipleValuesRequested || reqBody == "" {
		return
	}

	decoder := json.NewDecoder(bytes.NewBuffer([]byte(reqBody)))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var parsedJson middlewares.RPCRequest
	err := decoder.Decode(&parsedJson)
	if err != nil {
		log.Logger.Proxy.Errorf("Collector: getContextValueForRequest: json.Unmarshal: %s", err)
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
		err = tx.UnmarshalBase64(res)
		if err != nil {
			log.Logger.Proxy.Errorf("Collector: getContextValueForRequest: tx.Unmarshal: %s", err)
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
