package binder

import (
	"fmt"
	"net/http"
)

// HTTPError represents an error that occurred while handling a request.
// Its format matches the errors the API returned before the move to gin, e.g. "code=404, message=Not Found".
type HTTPError struct {
	Code     int
	Message  any
	Internal error // Stores the error returned by an external dependency
}

var (
	ErrNotFound             = NewHTTPError(http.StatusNotFound)             // HTTP 404 Not Found
	ErrMethodNotAllowed     = NewHTTPError(http.StatusMethodNotAllowed)     // HTTP 405 Method Not Allowed
	ErrUnsupportedMediaType = NewHTTPError(http.StatusUnsupportedMediaType) // HTTP 415 Unsupported Media Type
)

// NewHTTPError creates a new HTTPError instance.
func NewHTTPError(code int, message ...any) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		he.Message = message[0]
	}
	return he
}

// Error makes it compatible with `error` interface.
func (he *HTTPError) Error() string {
	if he.Internal == nil {
		return fmt.Sprintf("code=%d, message=%v", he.Code, he.Message)
	}
	return fmt.Sprintf("code=%d, message=%v, internal=%v", he.Code, he.Message, he.Internal)
}

// SetInternal sets error to HTTPError.Internal
func (he *HTTPError) SetInternal(err error) *HTTPError {
	he.Internal = err
	return he
}

// Unwrap satisfies the Go 1.13 error wrapper interface.
func (he *HTTPError) Unwrap() error {
	return he.Internal
}
