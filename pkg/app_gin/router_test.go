package app_gin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DoWithLogic/golang-clean-architecture/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// newTestEngine builds the engine like New, without the prometheus middleware that can only be registered once.
func newTestEngine() *Engine {
	gin.SetMode(gin.TestMode)

	e := newEngine()
	e.Use(handleErrors, recoverMiddleware, corsWithConfig(defaultCORSConfig), cacheWithRevalidation)

	return e
}

func serve(e *Engine, method, target string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	return rec
}

func ok(c *gin.Context) { c.String(http.StatusOK, "ok") }

func TestEngine_NotFound(t *testing.T) {
	e := newTestEngine()
	e.Group("/api").GET("/ping", ok)

	for _, target := range []string{"/nope", "/api/ping/", "/API/PING", "/api//ping"} {
		rec := serve(e, http.MethodGet, target, nil)

		assert.Equal(t, http.StatusNotFound, rec.Code, target)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"), target)
		assert.Equal(t, "no-cache, max-age=120, must-revalidate", rec.Header().Get("Cache-Control"), target)
		assert.Equal(t, "Origin", rec.Header().Get("Vary"), target)
		assert.Equal(t, `{"code":404,"message":"not_found","error":"resource not found"}`+"\n", rec.Body.String(), target)
	}
}

func TestEngine_MethodNotAllowed(t *testing.T) {
	e := newTestEngine()
	e.Group("/api").GET("/ping", ok)

	rec := serve(e, http.MethodPost, "/api/ping", nil)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "OPTIONS, GET", rec.Header().Get("Allow"))
	assert.Equal(t, `{"code":500,"message":"internal_server_error","error":"code=405, message=Method Not Allowed"}`+"\n", rec.Body.String())
}

func TestEngine_Options(t *testing.T) {
	e := newTestEngine()
	api := e.Group("/api")
	api.GET("/ping", ok)
	api.PUT("/ping", ok)

	rec := serve(e, http.MethodOptions, "/api/ping", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "OPTIONS, GET, PUT", rec.Header().Get("Allow"))
	assert.Empty(t, rec.Body.String())

	rec = serve(e, http.MethodOptions, "/api/ping", map[string]string{
		"Origin":                         "http://example.com",
		"Access-Control-Request-Method":  http.MethodGet,
		"Access-Control-Request-Headers": "Authorization",
	})
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET,PUT,POST,DELETE", rec.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Authorization", rec.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"}, rec.Header().Values("Vary"))

	rec = serve(e, http.MethodOptions, "/nope", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Allow"))
}

func TestEngine_GroupMiddlewareRunsForUnmatchedRoutes(t *testing.T) {
	e := newTestEngine()
	api := e.Group("/api")
	api.Group("/user/public").POST("/login", ok)
	api.Group("/user", func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}).GET("/:id", ok)

	for _, tc := range []struct {
		method, target string
	}{
		{http.MethodGet, "/api/user"},
		{http.MethodGet, "/api/user/"},
		{http.MethodGet, "/api/user/1/unknown"},
		{http.MethodPost, "/api/user/1"},
		{http.MethodGet, "/api/user/public/login"},
	} {
		rec := serve(e, tc.method, tc.target, nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, tc.target)
		assert.Empty(t, rec.Header().Get("Allow"), tc.target)

		rec = serve(e, tc.method, tc.target, map[string]string{"Authorization": "token"})
		assert.Equal(t, http.StatusNotFound, rec.Code, tc.target)
		assert.Equal(t, `{"code":404,"message":"not_found","error":"resource not found"}`+"\n", rec.Body.String(), tc.target)
	}

	rec := serve(e, http.MethodGet, "/api/users", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = serve(e, http.MethodGet, "/api/user/1", map[string]string{"Authorization": "token"})
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestEngine_PathParamsAreNotUnescaped(t *testing.T) {
	e := newTestEngine()

	var value string
	e.Group("/api").GET("/contact/:value/detail", func(c *gin.Context) {
		value = c.Param("value")
		c.Status(http.StatusOK)
	})

	rec := serve(e, http.MethodGet, "/api/contact/john%40example.com/detail", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "john%40example.com", value)
}

func TestEngine_HandlerErrors(t *testing.T) {
	e := newTestEngine()
	api := e.Group("/api")
	api.GET("/panic", func(c *gin.Context) { panic(errors.New("boom")) })
	api.GET("/error", func(c *gin.Context) { _ = c.Error(response.Conflict(errors.New("already exists"))) })

	rec := serve(e, http.MethodGet, "/api/panic", nil)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Empty(t, rec.Header().Get("Cache-Control"))
	assert.Equal(t, `{"code":500,"message":"internal_server_error","error":"boom"}`+"\n", rec.Body.String())

	rec = serve(e, http.MethodGet, "/api/error", nil)
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "no-cache, max-age=120, must-revalidate", rec.Header().Get("Cache-Control"))
	assert.Equal(t, `{"code":409,"message":"conflict","error":"already exists"}`+"\n", rec.Body.String())
}
