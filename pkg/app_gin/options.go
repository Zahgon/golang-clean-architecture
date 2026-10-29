package app_gin

import (
	"net/http"
)

type GinConfig struct {
	Port  string // The port on which the server will listen.
	Debug bool   // Indicates if debug mode is enabled.
}

// CORSConfig defines the config for CORS middleware.
type CORSConfig struct {
	// AllowOrigins determines the value of the Access-Control-Allow-Origin response header.
	// The wildcard characters '*' and '?' are supported. Default value []string{"*"}.
	AllowOrigins []string

	// AllowOriginFunc is a custom function to validate the origin. If set, AllowOrigins is ignored.
	AllowOriginFunc func(origin string) (bool, error)

	// AllowMethods determines the value of the Access-Control-Allow-Methods response header for preflight requests.
	// If left empty the methods of the routes matching the request path are used.
	AllowMethods []string

	// AllowHeaders determines the value of the Access-Control-Allow-Headers response header.
	// If left empty the Access-Control-Request-Headers request header is echoed.
	AllowHeaders []string

	// AllowCredentials determines the value of the Access-Control-Allow-Credentials response header.
	AllowCredentials bool

	// UnsafeWildcardOriginWithAllowCredentials UNSAFE/INSECURE: allows wildcard '*' origin to be used with
	// AllowCredentials flag, sending the request origin back in Access-Control-Allow-Origin.
	UnsafeWildcardOriginWithAllowCredentials bool

	// ExposeHeaders determines the value of the Access-Control-Expose-Headers response header.
	ExposeHeaders []string

	// MaxAge determines the value of the Access-Control-Max-Age response header, the header is set only if MaxAge != 0.
	MaxAge int
}

type ginRequest struct {
	IsObservabilityEnable bool
	ServiceName           *string
	CORSConfig            *CORSConfig
}

type GinOptionFn func(*ginRequest)

func WithTracing(serviceName string) GinOptionFn {
	return func(g *ginRequest) {
		g.IsObservabilityEnable = true
		g.ServiceName = &serviceName
	}
}
func WithCORS(c CORSConfig) GinOptionFn { return func(gr *ginRequest) { gr.CORSConfig = &c } }

var defaultCORSConfig = CORSConfig{
	AllowOrigins: []string{"*"},
	AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
}

func defaultGinRequest() *ginRequest {
	return &ginRequest{
		CORSConfig:            &defaultCORSConfig,
		IsObservabilityEnable: false,
	}
}
