package app_gin

import (
	"github.com/DoWithLogic/golang-clean-architecture/pkg/types"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// New creates and configures a new Gin server instance.
// Parameters:
//   - cfg: The application configuration.
//
// Returns:
//   - *Engine: A configured Gin server instance.
func (cfg GinConfig) New(opts ...GinOptionFn) *Engine {
	request := defaultGinRequest()
	for _, opt := range opts {
		opt(request)
	}

	// gin's mode is process-wide and here only selects gin's own debug logging. Responses do not depend on it:
	// debugJSON below ties Debug to this engine, like Echo's e.Debug.
	gin.SetMode(gin.ReleaseMode)
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	}

	e := newEngine()
	if cfg.Debug {
		e.Use(debugJSON)
	}
	e.Use(handleErrors)
	e.Use(recoverMiddleware)
	e.Use(corsWithConfig(*request.CORSConfig))
	e.Use(prometheusMiddleware("http"))
	e.Use(cacheWithRevalidation)

	if request.IsObservabilityEnable {
		e.Use(otelgin.Middleware(*request.ServiceName))
	}

	return e
}

// debugJSON marks the requests of an engine built with Debug, so response.Send indents their JSON whatever other
// engines in the process use. It runs before handleErrors, so rendered errors are indented too.
func debugJSON(c *gin.Context) {
	c.Set(types.DebugContextKey.String(), true)
	c.Next()
}
