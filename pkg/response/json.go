package response

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

const jsonIndent = "  "

// sendJSON writes i as a JSON response: "application/json" content type (unless one is already set), newline
// terminated, and indented in debug mode or when the request has a `pretty` query param.
// An encoding error is pushed to c.Errors so the error handler middleware can report it, like gin's own renderers do.
func sendJSON(c *gin.Context, code int, i any) error {
	indent := ""
	if _, pretty := c.Request.URL.Query()["pretty"]; gin.IsDebugging() || pretty {
		indent = jsonIndent
	}

	header := c.Writer.Header()
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", gin.MIMEJSON)
	}
	c.Status(code)

	enc := json.NewEncoder(c.Writer)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	if err := enc.Encode(i); err != nil {
		_ = c.Error(err)
		return err
	}
	return nil
}
