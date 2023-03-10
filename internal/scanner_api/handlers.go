package scanner_api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"extrnode-be/internal/models"
	"extrnode-be/internal/pkg/log"
	"extrnode-be/internal/pkg/util/solana"
)

const (
	maxLimit         = 1000
	minLimit         = 1
	defaultLimit     = 50
	asnMaxCount      = 264
	arrMaxLen        = 100
	solanaBlockchain = "solana"
)

func (a *scannerApi) statsHandler(ctx echo.Context) error {
	res, err := a.getStats()
	if err != nil {
		log.Logger.ScannerApi.Errorf("getStats: %s", err)
		return err
	}

	return ctx.JSON(http.StatusOK, res)
}

func (a *scannerApi) endpointsHandler(ctx echo.Context) error {
	var (
		err error
		ok  bool

		limit            = defaultLimit
		format           = jsonOutputFormat
		isRpc            *bool
		isValidator      *bool
		asnCountries     []string
		versions         []string
		supportedMethods []string
	)

	// TODO: in future will be as a query param
	blockchainID, ok := a.blockchainIDs[solanaBlockchain]
	if !ok {
		err = fmt.Errorf("endpointsHandler: fail to get blockchainID")
		log.Logger.ScannerApi.Errorf(err.Error())
		return err
	}
	if paramString := ctx.QueryParam("limit"); paramString != "" {
		limit, err = strconv.Atoi(paramString)
		if err != nil || limit > maxLimit || limit < minLimit {
			return echo.NewHTTPError(http.StatusBadRequest, "limit")
		}
	}
	if paramString := ctx.QueryParam("format"); paramString != "" {
		if _, ok = a.supportedOutputFormats[paramString]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "format")
		}

		format = paramString
	}
	if paramString := ctx.QueryParam("is_rpc"); paramString != "" {
		paramLocal, err := strconv.ParseBool(paramString)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "is_rpc")
		}

		isRpc = &paramLocal
	}
	if paramString := ctx.QueryParam("is_validator"); paramString != "" {
		paramLocal, err := strconv.ParseBool(paramString)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "is_validator")
		}

		isValidator = &paramLocal
	}
	if paramString := ctx.QueryParam("asn_country"); paramString != "" {
		asnCountries = strings.Split(paramString, ",")
		if len(asnCountries) > asnMaxCount {
			return echo.NewHTTPError(http.StatusBadRequest, "asn_country length")
		}
	}
	if paramString := ctx.QueryParam("version"); paramString != "" {
		versions = strings.Split(paramString, ",")
		if len(versions) > arrMaxLen {
			return echo.NewHTTPError(http.StatusBadRequest, "version length")
		}
	}
	if paramString := ctx.QueryParam("supported_method"); paramString != "" {
		supportedMethods = strings.Split(paramString, ",")
		if len(supportedMethods) > arrMaxLen {
			return echo.NewHTTPError(http.StatusBadRequest, "supported_method length")
		}
		for _, method := range supportedMethods {
			if _, ok := solana.FullMethodList[method]; !ok {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unknown method: %s", method))
			}
		}
	}

	res, err := a.slStorage.GetEndpoints(blockchainID, limit, isRpc, isValidator, asnCountries, versions, supportedMethods)
	if err != nil {
		log.Logger.ScannerApi.Errorf("endpointsHandler: GetEndpoints: %s", err)
		return err
	}

	if format == csvOutputFormat {
		resCsv := make([]models.EndpointCsv, len(res))
		for i, r := range res {
			resCsv[i] = models.EndpointCsv{
				Endpoint:    r.Endpoint,
				Version:     r.Version,
				As:          r.AsnInfo.As,
				Network:     r.AsnInfo.Network,
				Country:     r.AsnInfo.Country.Name,
				Isp:         r.AsnInfo.Isp,
				IsRpc:       r.IsRpc,
				IsValidator: r.IsValidator,
			}
		}
		return csvResp(ctx, resCsv, "")
	}

	if format == haproxyOutputFormat {
		var resString []byte
		for _, r := range res {
			resString = fmt.Appendln(resString, r.Endpoint)
		}
		return textResp(ctx, resString)
	}

	return ctx.JSON(http.StatusOK, res)
}
