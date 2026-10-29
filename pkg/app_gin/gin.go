package app_gin

import (
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

	// Debug mode also pretty prints JSON responses (see response.Send).
	gin.SetMode(gin.ReleaseMode)
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	}

	e := newEngine()
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
