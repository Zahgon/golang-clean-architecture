package app_gin

import (
	"github.com/gin-gonic/gin"
)

// Blob sends b with the given status code and content type (unless a content type is already set).
// Unlike gin's Context.Data it does not set Content-Length, net/http sends bodies larger than its 2KB buffer chunked.
func Blob(c *gin.Context, code int, contentType string, b []byte) {
	if c.Writer.Header().Get("Content-Type") == "" {
		c.Header("Content-Type", contentType)
	}
	c.Status(code)

	if _, err := c.Writer.Write(b); err != nil {
		_ = c.Error(err)
	}
}
