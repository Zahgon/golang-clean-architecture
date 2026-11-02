package v1

import (
	"github.com/DoWithLogic/golang-clean-architecture/pkg/app_gin"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/middleware"
)

func (h *handlers) MapRoutes(router *app_gin.RouterGroup, mw *middleware.Middleware) {
	h.registerPublicRoutes(router.Group("/user/public"))
	h.registerPrivateRoutes(router.Group("/user", mw.JWTMiddleware()))
}

func (h *handlers) registerPublicRoutes(router *app_gin.RouterGroup) {
	router.POST("/login", h.LoginHandler)
	router.POST("/sign-up", h.SignUpHandler)
}

func (h *handlers) registerPrivateRoutes(router *app_gin.RouterGroup) {
	router.GET("/:id/detail", h.UserDetailByIDHandler)
	router.GET("/contact/:contact_value/detail", h.UserDetailByContactValueHandler)
	router.PATCH("/:id/update", h.UpdateUserHandler)
	router.PUT("/:id/status/transition", h.TransitionUserStatusHandler)
}
