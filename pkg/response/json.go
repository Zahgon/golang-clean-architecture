package response

import (
	"encoding/json"

	"github.com/DoWithLogic/golang-clean-architecture/pkg/types"
	"github.com/gin-gonic/gin"
)

const jsonIndent = "  "

// sendJSON writes i as a JSON response: "application/json" content type (unless one is already set), newline
// terminated, and indented when the engine serving the request has Debug set or the request has a `pretty` query param.
// An encoding error is pushed to c.Errors so the error handler middleware can report it, like gin's own renderers do.
func sendJSON(c *gin.Context, code int, i any) error {
	indent := ""
	if _, pretty := c.Request.URL.Query()["pretty"]; c.GetBool(types.DebugContextKey.String()) || pretty {
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
