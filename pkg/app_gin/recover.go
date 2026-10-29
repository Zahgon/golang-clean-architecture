package app_gin

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/log"
)

const recoverStackSize = 4 << 10 // 4 KB

// recoverMiddleware recovers from panics anywhere in the chain, logs the stack of the panicking goroutine
// and hands the error to errorHandler.
func recoverMiddleware(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			if r == http.ErrAbortHandler {
				panic(r)
			}
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}

			stack := make([]byte, recoverStackSize)
			length := runtime.Stack(stack, false)
			log.Print(fmt.Sprintf("[PANIC RECOVER] %v %s\n", err, stack[:length]))

			errorHandler(err, c)
			c.Abort()
		}
	}()

	c.Next()
}
