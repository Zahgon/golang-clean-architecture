package app_gin

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// corsWithConfig returns a Cross-Origin Resource Sharing (CORS) middleware.
// Preflight (OPTIONS) requests are always answered here with 204 No Content, without calling the next handlers.
func corsWithConfig(config CORSConfig) gin.HandlerFunc {
	// Defaults
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = []string{"*"}
	}
	hasCustomAllowMethods := true
	if len(config.AllowMethods) == 0 {
		hasCustomAllowMethods = false
		config.AllowMethods = []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete}
	}

	allowOriginPatterns := []string{}
	for _, origin := range config.AllowOrigins {
		pattern := regexp.QuoteMeta(origin)
		pattern = strings.ReplaceAll(pattern, "\\*", ".*")
		pattern = strings.ReplaceAll(pattern, "\\?", ".")
		pattern = "^" + pattern + "$"
		allowOriginPatterns = append(allowOriginPatterns, pattern)
	}

	allowMethods := strings.Join(config.AllowMethods, ",")
	allowHeaders := strings.Join(config.AllowHeaders, ",")
	exposeHeaders := strings.Join(config.ExposeHeaders, ",")

	maxAge := "0"
	if config.MaxAge > 0 {
		maxAge = strconv.Itoa(config.MaxAge)
	}

	return func(c *gin.Context) {
		req := c.Request
		header := c.Writer.Header()
		origin := req.Header.Get("Origin")
		allowOrigin := ""

		header.Add("Vary", "Origin")

		// Preflight request is an OPTIONS request, using three HTTP request headers: Access-Control-Request-Method,
		// Access-Control-Request-Headers, and the Origin header. See: https://developer.mozilla.org/en-US/docs/Glossary/Preflight_request
		// For simplicity we just consider method type and later `Origin` header.
		preflight := req.Method == http.MethodOptions

		// We avoid calling next for OPTIONS in this middleware as CORS requests do not have cookies / authentication
		// headers by default, so we could get stuck in auth middlewares by calling next.
		// But we still want to send `Allow` header as response in case of Non-CORS OPTIONS request.
		routerAllowMethods := ""
		if preflight {
			if allow := c.GetString(contextKeyHeaderAllow); allow != "" {
				routerAllowMethods = allow
				header.Set(headerAllow, routerAllowMethods)
			}
		}

		// No Origin provided. This is (probably) not request from actual browser - proceed executing middleware chain
		if origin == "" {
			if !preflight {
				c.Next()
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		if config.AllowOriginFunc != nil {
			allowed, err := config.AllowOriginFunc(origin)
			if err != nil {
				_ = c.Error(err)
				c.Abort()
				return
			}
			if allowed {
				allowOrigin = origin
			}
		} else {
			// Check allowed origins
			for _, o := range config.AllowOrigins {
				if o == "*" && config.AllowCredentials && config.UnsafeWildcardOriginWithAllowCredentials {
					allowOrigin = origin
					break
				}
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
		}

		// Origin not allowed
		if allowOrigin == "" {
			if !preflight {
				c.Next()
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		header.Set("Access-Control-Allow-Origin", allowOrigin)
		if config.AllowCredentials {
			header.Set("Access-Control-Allow-Credentials", "true")
		}

		// Simple request
		if !preflight {
			if exposeHeaders != "" {
				header.Set("Access-Control-Expose-Headers", exposeHeaders)
			}
			c.Next()
			return
		}

		// Preflight request
		header.Add("Vary", "Access-Control-Request-Method")
		header.Add("Vary", "Access-Control-Request-Headers")

		if !hasCustomAllowMethods && routerAllowMethods != "" {
			header.Set("Access-Control-Allow-Methods", routerAllowMethods)
		} else {
			header.Set("Access-Control-Allow-Methods", allowMethods)
		}

		if allowHeaders != "" {
			header.Set("Access-Control-Allow-Headers", allowHeaders)
		} else {
			if h := req.Header.Get("Access-Control-Request-Headers"); h != "" {
				header.Set("Access-Control-Allow-Headers", h)
			}
		}
		if config.MaxAge != 0 {
			header.Set("Access-Control-Max-Age", maxAge)
		}
		c.AbortWithStatus(http.StatusNoContent)
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
