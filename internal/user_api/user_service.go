package user_api

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"extrnode-be/internal/pkg/log"
	echo2 "extrnode-be/internal/pkg/util/echo"
)

const longTermCache = 1 * time.Hour

func (a *userApi) apiTokenHandler(ctx echo.Context) error {
	cc := ctx.(*echo2.CustomContext)
	user := cc.GetUser()
	if user == nil {
		log.Logger.UserApi.Errorf("apiTokenHandler: fail to get user from context")
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if !user.EmailVerified {
		return ctx.JSON(http.StatusBadRequest, ErrNeedEmailVerification)
	}

	cacheValue, ok := a.cache.Get(user.UID)
	if ok {
		return ctx.JSON(http.StatusOK, cacheValue.(uuid.UUID))
	}

	t, err := a.pgStorage.GetOrCreateUser(user.UID)
	if err != nil {
		log.Logger.UserApi.Errorf("storage.GetOrCreateUser: %s", err)
		return err
	}

	a.cache.Set(user.UID, t.ApiToken, longTermCache)

	return ctx.JSON(http.StatusOK, t.ApiToken)
}
