package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/labstack/echo/v4"

	"extrnode-be/internal/models"
)

const (
	mimeTextCSV    = "text/csv"
	shortTermCache = 1 * time.Minute
	statsCacheKey  = "stats"
)

func csvResp(ctx echo.Context, res interface{}, fileName string) error {
	ctx.Response().Header().Set(echo.HeaderContentType, mimeTextCSV)
	if fileName != "" {
		ctx.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	}
	ctx.Response().WriteHeader(http.StatusOK)

	return gocsv.Marshal(res, ctx.Response())
}

func textResp(ctx echo.Context, res []byte) error {
	ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)
	ctx.Response().WriteHeader(http.StatusOK)
	_, err := ctx.Response().Write(res)

	return err
}

func (a *api) getStats() (res models.Stat, err error) {
	cacheValue, ok := a.cache.Get(statsCacheKey)
	if ok {
		return cacheValue.(models.Stat), nil
	}

	res, err = a.slStorage.GetStats()
	if err != nil {
		return res, err
	}

	a.cache.Set(statsCacheKey, res, shortTermCache)

	return res, nil
}
