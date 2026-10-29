package middleware

import (
	"errors"
	"strings"

	"github.com/DoWithLogic/golang-clean-architecture/pkg/jwt"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/response"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/types"
	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidAuthenticationCredentials = errors.New("invalid authentication credentials")
)

type Middleware struct {
	jwtFactory *jwt.JWTFactory
}

func New(jwtFactory *jwt.JWTFactory) *Middleware {
	return &Middleware{jwtFactory: jwtFactory}
}

func (m *Middleware) JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.Request.Header.Get(types.AuthorizationHeaderKey.String())
		if tokenString == "" {
			response.ErrorBuilder(response.Unauthorized(ErrInvalidAuthenticationCredentials)).Send(c)
			c.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		claims, err := m.jwtFactory.VerifyJWT(c.Request.Context(), tokenString)
		if err != nil {
			response.ErrorBuilder(response.Unauthorized(ErrInvalidAuthenticationCredentials)).Send(c)
			c.Abort()
			return
		}

		embedClaimedDataIntoContext(c, embedClaimedDataIntoContextOpts{claimedData: claims})

		c.Next()
	}
}
