package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/echo/v4"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/option"

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
)

type AuthMiddleware struct {
	ctx          context.Context
	authProvider *auth.Client
	cache        *cache.Cache
}

const (
	cacheTTL             = 15 * time.Minute
	cacheCleanup         = 10 * time.Second
	tokenCacheDefaultTTL = 10 * time.Minute
	bearerAuthSchema     = "bearer"
)

var (
	ErrEmptyAuthHeaders = echo.NewHTTPError(http.StatusUnauthorized, "Empty authorization headers")
	ErrTokenIsExpired   = echo.NewHTTPError(http.StatusUnauthorized, "Auth token already expired")
	ErrTokenRevoked     = echo.NewHTTPError(http.StatusUnauthorized, "Auth token already revoked")
	ErrTokenInvalid     = echo.NewHTTPError(http.StatusUnauthorized, "Auth token invalid")
	ErrUserNotFound     = echo.NewHTTPError(http.StatusUnauthorized, "User not found")
	ErrAuthUnknown      = echo.NewHTTPError(http.StatusUnauthorized, "Auth unknown error")
)

func NewAuthMiddleware(ctx context.Context, conf config.ApiConfig) (a AuthMiddleware, err error) {
	opt := option.WithCredentialsFile(conf.FirebaseFilePath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return a, fmt.Errorf("NewApp: %s", err)
	}

	authProvider, err := app.Auth(context.Background())
	if err != nil {
		return a, fmt.Errorf("Auth: %s", err)
	}

	return AuthMiddleware{
		ctx:          ctx,
		authProvider: authProvider,
		cache:        cache.New(cacheTTL, cacheCleanup),
	}, nil
}

func (a *AuthMiddleware) getTokenInfo(authToken string) (tokenInfo *auth.Token, err error) {
	// try get from cache
	if cachedToken, ok := a.cache.Get(authToken); ok {
		if tokenInfo, ok = cachedToken.(*auth.Token); ok {
			return tokenInfo, nil
		}
	}

	tokenInfo, err = a.authProvider.VerifyIDTokenAndCheckRevoked(a.ctx, authToken)
	if err != nil {
		switch {
		case auth.IsIDTokenExpired(err):
			return tokenInfo, ErrTokenIsExpired
		case auth.IsIDTokenRevoked(err):
			return tokenInfo, ErrTokenRevoked
		}

		log.Logger.Api.Errorf("getTokenInfo: VerifyIDTokenAndCheckRevoked: %s", err)
		return tokenInfo, ErrAuthUnknown
	}
	if tokenInfo == nil {
		return tokenInfo, ErrTokenInvalid
	}

	// use ttl not greater than tokenCacheDefaultTTL
	ttl := time.Duration(tokenInfo.Expires-time.Now().Unix()) * time.Second
	if ttl > tokenCacheDefaultTTL {
		ttl = tokenCacheDefaultTTL
	}
	a.cache.Set(authToken, tokenInfo, ttl)

	return tokenInfo, nil
}

func (a *AuthMiddleware) getAuthToken(c echo.Context) (res string, err error) {
	splittedToken := strings.SplitN(c.Request().Header.Get(echo.HeaderAuthorization), " ", 2)
	if len(splittedToken) != 2 {
		return res, ErrEmptyAuthHeaders
	}
	if strings.ToLower(splittedToken[0]) != bearerAuthSchema || splittedToken[1] == "" {
		return res, ErrTokenInvalid
	}

	return splittedToken[1], nil
}

func (a *AuthMiddleware) LoadUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authToken, err := a.getAuthToken(c)
		if err != nil {
			return err
		}
		tokenInfo, err := a.getTokenInfo(authToken)
		if err != nil {
			return err
		}

		user, err := a.authProvider.GetUser(a.ctx, tokenInfo.UID)
		if err != nil {
			return ErrUserNotFound
		}

		c.(*CustomContext).SetUser(user)

		return next(c)
	}
}
