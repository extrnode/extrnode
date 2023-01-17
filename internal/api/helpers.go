package api

import (
	"time"

	"extrnode-be/internal/models"
)

const (
	shortTermCache = 1 * time.Minute

	statsCacheKey = "stats"
)

type (
	RPCRequest struct {
		Method string `json:"method"`
		// unnecessary fields removed
	}

	RPCResponse struct {
		Error RPCError `json:"error,omitempty"`
	}
	RPCError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
)

func (a *api) getStats() (res models.Stat, err error) {
	cacheValue, ok := a.cache.Get(statsCacheKey)
	if ok {
		return cacheValue.(models.Stat), nil
	}

	res, err = a.storage.GetStats()
	if err != nil {
		return res, err
	}

	a.cache.Set(statsCacheKey, res, shortTermCache)

	return res, nil
}
