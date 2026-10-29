package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	userV1 "github.com/DoWithLogic/golang-clean-architecture/internal/app/users/delivery/http/v1"
	userRepository "github.com/DoWithLogic/golang-clean-architecture/internal/app/users/repository"
	userUseCase "github.com/DoWithLogic/golang-clean-architecture/internal/app/users/usecase"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/app_gin"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/binder"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/encryptions"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/jwt"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/logging"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/middleware"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/observability"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/redis"
	"github.com/gin-gonic/gin"
)

type routeMapper interface {
	MapRoutes(api *app_gin.RouterGroup, mw *middleware.Middleware)
}

func (s *Server) setup() error {
	s.setupMiddleware()

	api := s.gin.Group("/api/v1")

	s.registerUtilityRoutes(api)

	middleware, handlers := s.buildHandlers()

	for _, handler := range handlers {
		handler.MapRoutes(api, middleware)
	}

	return nil
}

func (s *Server) setupMiddleware() {
	logger := observability.NewZeroLogHook().Z()
	s.gin.Use(logging.Middleware(logging.WithLogger(logger), logging.WithMaskedKeys("password", "token")))
}

func (s *Server) registerUtilityRoutes(api *app_gin.RouterGroup) {
	api.GET("/ping", s.ping)
	api.GET("/swagger/*any", s.swagger)
}

func (s *Server) ping(c *gin.Context) {
	app_gin.Blob(c, http.StatusOK, "text/plain; charset=UTF-8", []byte("Hello World 👋"))
}

// swagger serves the generated swagger document on /swagger/doc.json and the swagger UI on the other paths,
// gin does not allow a static route next to the /swagger/*any wildcard.
func (s *Server) swagger(c *gin.Context) {
	if c.Param("any") == "/doc.json" {
		s.swaggerDoc(c)
		return
	}

	app_gin.SwaggerHandler(c)
}

func (s *Server) swaggerDoc(c *gin.Context) {
	data, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		_ = c.Error(binder.NewHTTPError(http.StatusInternalServerError, "failed to read swagger document"))
		return
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		_ = c.Error(binder.NewHTTPError(http.StatusInternalServerError, "failed to parse swagger document"))
		return
	}

	doc["host"] = s.cfg.App.Host
	doc["schemes"] = strings.Split(s.cfg.App.Scheme, ",")

	body, err := json.Marshal(doc)
	if err != nil {
		_ = c.Error(binder.NewHTTPError(http.StatusInternalServerError, "failed to generate swagger document"))
		return
	}

	app_gin.Blob(c, http.StatusOK, gin.MIMEJSON, body)
}

func (s *Server) buildHandlers() (*middleware.Middleware, []routeMapper) {
	redisManager := redis.NewRedisManager(nil)

	jwtFactory := jwt.NewJWTFactory(s.cfg.JWT, redisManager)
	crypto := encryptions.NewCrypto(s.cfg.Authentication.Key)

	mw := middleware.New(jwtFactory)

	userRepo := userRepository.NewRepository(s.db)

	userUC := userUseCase.NewUseCase(userUseCase.Dependencies{
		Repositories: userUseCase.Repositories{
			Repo: userRepo,
		},
		Pkgs: userUseCase.Pkgs{
			AppJwt: jwtFactory,
			Crypto: crypto,
		},
	})

	handlers := []routeMapper{
		userV1.NewHandlers(userUC),
	}

	return mw, handlers
}
