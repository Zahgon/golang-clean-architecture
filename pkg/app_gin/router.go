package app_gin

import (
	"net/http"
	"slices"
	"strings"

	"github.com/DoWithLogic/golang-clean-architecture/pkg/binder"
	"github.com/gin-gonic/gin"
)

const (
	headerAllow = "Allow"

	// contextKeyHeaderAllow holds the Allow header value when the request path only matches routes of other methods.
	contextKeyHeaderAllow = "app_gin.header_allow"
)

// allowMethodsOrder is the order methods are listed in the Allow header, after OPTIONS.
var allowMethodsOrder = []string{
	http.MethodConnect,
	http.MethodDelete,
	http.MethodGet,
	http.MethodHead,
	http.MethodPatch,
	http.MethodPost,
	"PROPFIND",
	http.MethodPut,
	http.MethodTrace,
	"REPORT",
}

// Engine is a gin.Engine with the routing rules of the API:
//   - routes are matched on the raw (escaped) path, path params are not unescaped and trailing slashes are not redirected,
//   - a path that only matches routes of other methods fails with 405 and an Allow header like "OPTIONS, GET",
//   - a path under a group with middleware that matches no route of that group still runs the group middleware
//     before failing with 404 (see RouterGroup.Group),
//   - not found and method not allowed errors are rendered by errorHandler.
type Engine struct {
	*gin.Engine
	notFoundGroups []*RouterGroup
}

// RouterGroup is a gin.RouterGroup that keeps track of its group level middleware.
type RouterGroup struct {
	*gin.RouterGroup
	engine     *Engine
	middleware gin.HandlersChain
}

func newEngine() *Engine {
	e := &Engine{Engine: gin.New()}
	e.RedirectTrailingSlash = false
	e.HandleMethodNotAllowed = true
	e.UseRawPath = true
	e.UnescapePathValues = false

	e.Use(e.allowedMethods)
	e.NoRoute(e.routeNotFound)
	e.NoMethod(e.methodNotAllowed)

	return e
}

// Group creates a new router group, see RouterGroup.Group.
func (e *Engine) Group(relativePath string, middleware ...gin.HandlerFunc) *RouterGroup {
	root := &RouterGroup{RouterGroup: &e.RouterGroup, engine: e}
	return root.Group(relativePath, middleware...)
}

// Group creates a new router group. When the group has middleware (its own or from a parent group), requests under
// the group prefix that match none of its routes, whatever the method, run through that middleware before failing
// with 404, so e.g. unknown paths under an authenticated group answer 401 to anonymous clients.
func (g *RouterGroup) Group(relativePath string, middleware ...gin.HandlerFunc) *RouterGroup {
	group := &RouterGroup{
		RouterGroup: g.RouterGroup.Group(relativePath, middleware...),
		engine:      g.engine,
		middleware:  append(slices.Clip(g.middleware), middleware...),
	}
	if len(group.middleware) > 0 {
		g.engine.notFoundGroups = append(g.engine.notFoundGroups, group)
	}

	return group
}

// allowedMethods runs before any other middleware. Gin sets the Allow header up front when the path only matches
// routes of other methods; the header is removed here and its value, OPTIONS first, kept in the context for
// methodNotAllowed and CORS preflight requests. Paths handled by a group are reported as not found instead.
func (e *Engine) allowedMethods(c *gin.Context) {
	header := c.Writer.Header()
	if allow := header.Get(headerAllow); allow != "" {
		header.Del(headerAllow)
		if e.notFoundGroup(c) == nil {
			c.Set(contextKeyHeaderAllow, allowHeader(strings.Split(allow, ", ")))
		}
	}

	c.Next()
}

func (e *Engine) routeNotFound(c *gin.Context) {
	if group := e.notFoundGroup(c); group != nil {
		for _, handler := range group.middleware {
			handler(c)
			if c.IsAborted() {
				return
			}
		}
	}

	_ = c.Error(binder.ErrNotFound)
}

func (e *Engine) methodNotAllowed(c *gin.Context) {
	allow := c.GetString(contextKeyHeaderAllow)
	if allow == "" {
		e.routeNotFound(c)
		return
	}

	c.Header(headerAllow, allow)
	_ = c.Error(binder.ErrMethodNotAllowed)
}

// notFoundGroup returns the innermost group with middleware whose prefix contains the request path.
func (e *Engine) notFoundGroup(c *gin.Context) *RouterGroup {
	path := c.Request.URL.RawPath
	if path == "" {
		path = c.Request.URL.Path
	}

	var found *RouterGroup
	for _, group := range e.notFoundGroups {
		prefix := group.BasePath()
		if path != prefix && !strings.HasPrefix(path, prefix+"/") {
			continue
		}
		if found == nil || len(prefix) > len(found.BasePath()) {
			found = group
		}
	}

	return found
}

// allowHeader lists OPTIONS followed by the given methods in allowMethodsOrder, then any other method.
func allowHeader(methods []string) string {
	allow := []string{http.MethodOptions}
	for _, method := range allowMethodsOrder {
		if slices.Contains(methods, method) {
			allow = append(allow, method)
		}
	}
	for _, method := range methods {
		if method != http.MethodOptions && !slices.Contains(allowMethodsOrder, method) {
			allow = append(allow, method)
		}
	}

	return strings.Join(allow, ", ")
}
