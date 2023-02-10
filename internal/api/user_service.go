package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"extrnode-be/internal/pkg/log"
	echo2 "extrnode-be/internal/pkg/util/echo"
)

func (a *api) apiTokenHandler(ctx echo.Context) error {
	cc := ctx.(*echo2.CustomContext)
	user := cc.GetUser()
	if user == nil {
		log.Logger.Api.Errorf("apiTokenHandler: fail to get user from context")
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if !user.EmailVerified {
		return ctx.JSON(http.StatusBadRequest, ErrNeedEmailVerification)
	}

	t, err := a.pgStorage.GetOrCreateUser(user.UID)
	if err != nil {
		log.Logger.Api.Errorf("storage.GetOrCreateUser: %s", err)
		return err
	}

	return ctx.JSON(http.StatusOK, t.ApiToken)
}
