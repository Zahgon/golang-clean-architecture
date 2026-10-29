package binder_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DoWithLogic/golang-clean-architecture/pkg/binder"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type user struct {
	Name string `json:"name"`
}

type request struct {
	ID    int64  `param:"id"`
	Query string `query:"q"`
	user
}

func newContext(method, target, contentType string, body io.Reader, params gin.Params) *gin.Context {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest(method, target, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Params = params

	return c
}

func TestBind(t *testing.T) {
	idParam := gin.Params{{Key: "id", Value: "7"}}

	t.Run("binds path params, query params and body", func(t *testing.T) {
		var r request
		c := newContext(http.MethodGet, "/?q=search&ID=9", gin.MIMEJSON, strings.NewReader(`{"name":"john"}`), idParam)

		assert.NoError(t, binder.Bind(c, &r))
		assert.Equal(t, request{ID: 7, Query: "search", user: user{Name: "john"}}, r)
	})

	t.Run("query params are only bound for GET, DELETE and HEAD", func(t *testing.T) {
		var r request
		c := newContext(http.MethodPost, "/?q=search", gin.MIMEJSON, strings.NewReader(`{"name":"john"}`), idParam)

		assert.NoError(t, binder.Bind(c, &r))
		assert.Equal(t, request{ID: 7, user: user{Name: "john"}}, r)
	})

	t.Run("empty body is not decoded", func(t *testing.T) {
		var r request
		c := newContext(http.MethodPost, "/", "text/plain", nil, idParam)

		assert.NoError(t, binder.Bind(c, &r))
		assert.Equal(t, request{ID: 7}, r)
	})

	t.Run("invalid path param", func(t *testing.T) {
		var r request
		c := newContext(http.MethodGet, "/", "", nil, gin.Params{{Key: "id", Value: "abc"}})

		err := binder.Bind(c, &r)
		assert.EqualError(t, err, `code=400, message=strconv.ParseInt: parsing "abc": invalid syntax, internal=strconv.ParseInt: parsing "abc": invalid syntax`)
	})

	t.Run("JSON syntax error", func(t *testing.T) {
		var r request
		c := newContext(http.MethodPost, "/", gin.MIMEJSON, strings.NewReader(`{bad}`), nil)

		err := binder.Bind(c, &r)
		assert.EqualError(t, err, `code=400, message=Syntax error: offset=2, error=invalid character 'b' looking for beginning of object key string, internal=invalid character 'b' looking for beginning of object key string`)
	})

	t.Run("JSON type error", func(t *testing.T) {
		var r request
		c := newContext(http.MethodPost, "/", gin.MIMEJSON, strings.NewReader(`{"name":1}`), nil)

		err := binder.Bind(c, &r)
		assert.EqualError(t, err, `code=400, message=Unmarshal type error: expected=string, got=number, field=user.name, offset=9, internal=json: cannot unmarshal number into Go struct field request.user.name of type string`)
	})

	t.Run("truncated JSON", func(t *testing.T) {
		var r request
		c := newContext(http.MethodPost, "/", gin.MIMEJSON, strings.NewReader(`{"name":`), nil)

		err := binder.Bind(c, &r)
		assert.EqualError(t, err, `code=400, message=unexpected EOF, internal=unexpected EOF`)
	})

	t.Run("unsupported media type", func(t *testing.T) {
		var r request
		c := newContext(http.MethodPost, "/", "text/plain", strings.NewReader(`name`), nil)

		err := binder.Bind(c, &r)
		assert.Equal(t, binder.ErrUnsupportedMediaType, err)
		assert.EqualError(t, err, `code=415, message=Unsupported Media Type`)
	})

	t.Run("form fields need a form tag", func(t *testing.T) {
		var r struct {
			Name  string `form:"name"`
			Other string
		}
		c := newContext(http.MethodPost, "/", gin.MIMEPOSTForm, strings.NewReader(`name=john&Other=x`), nil)

		assert.NoError(t, binder.Bind(c, &r))
		assert.Equal(t, "john", r.Name)
		assert.Empty(t, r.Other)
	})
}
