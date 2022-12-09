package api

import (
	"fmt"
	"net/http"

	"github.com/gocarina/gocsv"
	"github.com/labstack/echo/v4"
)

const (
	mimeTextCSV = "text/csv"
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
