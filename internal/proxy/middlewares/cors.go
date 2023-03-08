package middlewares

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

type (
	// CORSConfig defines the config for CORS middleware.
	CORSConfig struct {
		// AllowOrigins determines the value of the Access-Control-Allow-Origin
		// response header.  This header defines a list of origins that may access the
		// resource.  The wildcard characters '*' and '?' are supported and are
		// converted to regex fragments '.*' and '.' accordingly.
		//
		// Security: use extreme caution when handling the origin, and carefully
		// validate any logic. Remember that attackers may register hostile domain names.
		// See https://blog.portswigger.net/2016/10/exploiting-cors-misconfigurations-for.html
		//
		// Optional. Default value []string{"*"}.
		//
		// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin
		AllowOrigins []string `yaml:"allow_origins"`

		// AllowMethods determines the value of the Access-Control-Allow-Methods
		// response header.  This header specified the list of methods allowed when
		// accessing the resource.  This is used in response to a preflight request.
		//
		// Optional. Default value DefaultCORSConfig.AllowMethods.
		// If `allowMethods` is left empty, this middleware will fill for preflight
		// request `Access-Control-Allow-Methods` header value
		// from `Allow` header that echo.Router set into context.
		//
		// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Methods
		AllowMethods []string `yaml:"allow_methods"`

		// AllowHeaders determines the value of the Access-Control-Allow-Headers
		// response header.  This header is used in response to a preflight request to
		// indicate which HTTP headers can be used when making the actual request.
		//
		// Optional. Default value []string{}.
		//
		// See also: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Headers
		AllowHeaders []string `yaml:"allow_headers"`
	}
)

var (
	// DefaultCORSConfig is the default CORS middleware config.
	DefaultCORSConfig = CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}
)

// CORSWithConfig returns a CORS middleware with config.
// See: [CORS].
func CORSWithConfig(config CORSConfig) echo.MiddlewareFunc {
	// Defaults
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = DefaultCORSConfig.AllowOrigins
	}
	hasCustomAllowMethods := true
	if len(config.AllowMethods) == 0 {
		hasCustomAllowMethods = false
		config.AllowMethods = DefaultCORSConfig.AllowMethods
	}

	allowOriginPatterns := []string{}
	for _, origin := range config.AllowOrigins {
		pattern := regexp.QuoteMeta(origin)
		pattern = strings.Replace(pattern, "\\*", ".*", -1)
		pattern = strings.Replace(pattern, "\\?", ".", -1)
		pattern = "^" + pattern + "$"
		allowOriginPatterns = append(allowOriginPatterns, pattern)
	}

	allowMethods := strings.Join(config.AllowMethods, ",")
	allowHeaders := strings.Join(config.AllowHeaders, ",")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			origin := req.Header.Get(echo.HeaderOrigin)
			allowOrigin := ""

			res.Header().Add(echo.HeaderVary, echo.HeaderOrigin)

			// Preflight request is an OPTIONS request, using three HTTP request headers: Access-Control-Request-Method,
			// Access-Control-Request-Headers, and the Origin header. See: https://developer.mozilla.org/en-US/docs/Glossary/Preflight_request
			// For simplicity we just consider method type and later `Origin` header.
			preflight := req.Method == http.MethodOptions

			// Although router adds special handler in case of OPTIONS method we avoid calling next for OPTIONS in this middleware
			// as CORS requests do not have cookies / authentication headers by default, so we could get stuck in auth
			// middlewares by calling next(c).
			// But we still want to send `Allow` header as response in case of Non-CORS OPTIONS request as router default
			// handler does.
			routerAllowMethods := ""
			if preflight {
				tmpAllowMethods, ok := c.Get(echo.ContextKeyHeaderAllow).(string)
				if ok && tmpAllowMethods != "" {
					routerAllowMethods = tmpAllowMethods
					c.Response().Header().Set(echo.HeaderAllow, routerAllowMethods)
				}
			}

			// No Origin provided. This is (probably) not request from actual browser - proceed executing middleware chain
			if origin == "" {
				if !preflight {
					return next(c)
				}
				return c.NoContent(http.StatusNoContent)
			}

			// Check allowed origins
			for _, o := range config.AllowOrigins {
				if o == "*" || o == origin {
					allowOrigin = o
					break
				}
				if matchSubdomain(origin, o) {
					allowOrigin = origin
					break
				}
			}

			checkPatterns := false
			if allowOrigin == "" {
				// to avoid regex cost by invalid (long) domains (253 is domain name max limit)
				if len(origin) <= (253+3+5) && strings.Contains(origin, "://") {
					checkPatterns = true
				}
			}
			if checkPatterns {
				for _, re := range allowOriginPatterns {
					if match, _ := regexp.MatchString(re, origin); match {
						allowOrigin = origin
						break
					}
				}
			}

			// Origin not allowed
			if allowOrigin == "" {
				if !preflight {
					return next(c)
				}
				return c.NoContent(http.StatusNoContent)
			}

			res.Header().Set(echo.HeaderAccessControlAllowOrigin, allowOrigin)

			// Simple request
			if !preflight {
				err := next(c)

				// reassign header after proxy handling
				res.Header().Set(echo.HeaderAccessControlAllowOrigin, allowOrigin)

				return err
			}

			// Preflight request
			res.Header().Add(echo.HeaderVary, echo.HeaderAccessControlRequestMethod)
			res.Header().Add(echo.HeaderVary, echo.HeaderAccessControlRequestHeaders)

			if !hasCustomAllowMethods && routerAllowMethods != "" {
				res.Header().Set(echo.HeaderAccessControlAllowMethods, routerAllowMethods)
			} else {
				res.Header().Set(echo.HeaderAccessControlAllowMethods, allowMethods)
			}

			if allowHeaders != "" {
				res.Header().Set(echo.HeaderAccessControlAllowHeaders, allowHeaders)
			} else {
				h := req.Header.Get(echo.HeaderAccessControlRequestHeaders)
				if h != "" {
					res.Header().Set(echo.HeaderAccessControlAllowHeaders, h)
				}
			}

			return c.NoContent(http.StatusNoContent)
		}
	}
}

func matchScheme(domain, pattern string) bool {
	didx := strings.Index(domain, ":")
	pidx := strings.Index(pattern, ":")
	return didx != -1 && pidx != -1 && domain[:didx] == pattern[:pidx]
}

// matchSubdomain compares authority with wildcard
func matchSubdomain(domain, pattern string) bool {
	if !matchScheme(domain, pattern) {
		return false
	}
	didx := strings.Index(domain, "://")
	pidx := strings.Index(pattern, "://")
	if didx == -1 || pidx == -1 {
		return false
	}
	domAuth := domain[didx+3:]
	// to avoid long loop by invalid long domain
	if len(domAuth) > 253 {
		return false
	}
	patAuth := pattern[pidx+3:]

	domComp := strings.Split(domAuth, ".")
	patComp := strings.Split(patAuth, ".")
	for i := len(domComp)/2 - 1; i >= 0; i-- {
		opp := len(domComp) - 1 - i
		domComp[i], domComp[opp] = domComp[opp], domComp[i]
	}
	for i := len(patComp)/2 - 1; i >= 0; i-- {
		opp := len(patComp) - 1 - i
		patComp[i], patComp[opp] = patComp[opp], patComp[i]
	}

	for i, v := range domComp {
		if len(patComp) <= i {
			return false
		}
		p := patComp[i]
		if p == "*" {
			return true
		}
		if p != v {
			return false
		}
	}
	return false
}
