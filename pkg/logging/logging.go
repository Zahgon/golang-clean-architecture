// Package logging provides a reusable Gin middleware for structured HTTP
// request/response logging via zerolog. It supports sensitive-field masking,
// path exclusion, and X-Request-ID propagation.
//
// # Quick Start
//
//	import (
//	    logging "github.com/golang-clean-architecture/pkg/logging"
//	)
//
//	r := gin.New()
//	r.Use(logging.Middleware(
//	    logging.WithLogger(&logger),
//	    logging.WithMaskedKeys("password", "token", "secret"),
//	))
package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	xRequestIDHeader = "X-Request-ID"
)

// contextKey is an unexported type for context keys owned by this package,
// preventing collisions with keys defined elsewhere.
type contextKey string

const (
	// RequestIDContextKey is the context key under which the X-Request-ID is stored.
	RequestIDContextKey contextKey = xRequestIDHeader
)

// Middleware returns a Gin middleware function that logs every HTTP request
// and its response as a structured zerolog event.
//
// Paths matching opts.IgnoredPatterns are passed through without logging.
// Sensitive fields listed in opts.MaskedKeys are redacted in both request
// and response bodies before the log entry is written.
//
// The middleware injects (or forwards) an X-Request-ID header and stores it
// in the request context under RequestIDContextKey.
func Middleware(options ...LoggingOption) gin.HandlerFunc {
	opts := defaultConfig()
	for _, opt := range options {
		opt(&opts)
	}

	return func(c *gin.Context) {
		if opts.IsPathIgnored(c.Request.RequestURI) {
			c.Next()
			return
		}

		startTime := time.Now()

		// Propagate or generate X-Request-ID.
		requestID := resolveRequestID(c)
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), RequestIDContextKey, requestID),
		)
		c.Writer.Header().Set(xRequestIDHeader, requestID)

		// Wrap the response writer so we can read the body after the handler runs.
		crw := New(c.Writer)
		c.Writer = &ginResponseWriter{ResponseWriter: c.Writer, crw: crw}

		// Buffer the request body so we can log it without consuming it.
		requestBodyJSON := readRequestBody(c, opts)

		// Execute the handler chain.
		c.Next()

		// Emit the log entry.
		logCtx := &loggingContext{
			Request:         c.Request,
			RequestBodyJSON: requestBodyJSON,
			StartTime:       startTime,
			Options:         opts,
			Logger:          opts.logger,
			Ctx:             c.Request.Context(),
		}
		emitLogEntry(logCtx, crw)
	}
}

// resolveRequestID returns the X-Request-ID from the incoming request header,
// or generates a new UUID if the header is absent or empty.
func resolveRequestID(c *gin.Context) string {
	if id := c.Request.Header.Get(xRequestIDHeader); id != "" {
		return id
	}
	return uuid.New().String()
}

// readRequestBody reads, unmarshals, and masks the JSON request body.
// The body is restored to the request so downstream handlers can still read it.
func readRequestBody(c *gin.Context, opts loggingConfig) map[string]any {
	if c.Request.Body == nil {
		return nil
	}

	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil
	}
	// Always restore the body for subsequent handlers.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))

	var bodyJSON map[string]any
	if err := json.Unmarshal(raw, &bodyJSON); err != nil {
		return nil
	}

	return opts.MaskSensitiveFields(bodyJSON)
}

// emitLogEntry builds and fires the zerolog event for a completed request.
func emitLogEntry(data *loggingContext, crw *CustomResponseWriter) {
	stop := time.Now()
	latency := stop.Sub(data.StartTime)

	contentLength := data.Request.Header.Get("Content-Length")
	if contentLength == "" {
		contentLength = "0"
	}

	event := data.Logger.Info().
		Ctx(data.Ctx).
		Str("remote_ip", data.Request.RemoteAddr).
		Str("host", data.Request.Host).
		Str("method", data.Request.Method).
		Str("uri", data.Request.RequestURI).
		Str("user_agent", data.Request.UserAgent())

	headers := make(map[string]any)
	for k, v := range data.Request.Header {
		switch k {
		case "Authorization", "Cookie", "Set-Cookie":
			headers[k] = "***"
		default:
			if len(v) == 1 {
				headers[k] = v[0]
			} else {
				headers[k] = v
			}
		}
	}

	event = event.
		Interface("headers", headers).
		Int("status", crw.Status()).
		Str("referer", data.Request.Referer()).
		Dur("latency", latency).
		Str("latency_human", latency.String()).
		Str("bytes_in", contentLength).
		Int("bytes_out", crw.Size()).
		Interface("request_body", data.RequestBodyJSON)

	// Attach the response body only when the content type is JSON.
	if strings.Contains(crw.Header().Get("Content-Type"), gin.MIMEJSON) {
		var responseBodyJSON map[string]any
		if err := json.Unmarshal(crw.Body(), &responseBodyJSON); err == nil {
			responseBodyJSON = data.Options.MaskSensitiveFields(responseBodyJSON)
			event = event.Interface("response_body", responseBodyJSON)
		}
	}

	event.Msg("[Received HTTP request]")
}
