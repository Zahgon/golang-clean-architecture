package app_gin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func cacheWithRevalidation(c *gin.Context) {
	// Call the next handler
	c.Next()

	// Set Cache-Control header to enable caching and revalidation with a maximum age of 120 seconds
	c.Writer.Header().Set("Cache-Control", "no-cache, max-age=120, must-revalidate")

	// Set Expires header to a future time to indicate the expiration time
	expiresTime := time.Now().Add(120 * time.Second)
	c.Writer.Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))

	// Set Last-Modified header to the current time
	c.Writer.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
}
