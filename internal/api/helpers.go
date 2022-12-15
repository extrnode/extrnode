package api

import (
	"time"

	"extrnode-be/internal/models"
)

const (
	shortTermCache = 1 * time.Minute

	statsCacheKey = "stats"
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
