package response

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DoWithLogic/golang-clean-architecture/pkg/types"
	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	// Keeps gin's default debug mode from logging; JSON indentation does not depend on it.
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestSendJSON(t *testing.T) {
	resp := ErrorResponse{
		Code:    http.StatusNotFound,
		Message: NotFoundMessage,
		Error:   "<not found> & gone",
	}

	t.Run("compact", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		if err := resp.Send(ctx); err != nil {
			t.Fatalf("Send() returned error: %v", err)
		}

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want %q", got, "application/json")
		}
		expected := "{\"code\":404,\"message\":\"not_found\",\"error\":\"\\u003cnot found\\u003e \\u0026 gone\"}\n"
		if rec.Body.String() != expected {
			t.Fatalf("body = %q, want %q", rec.Body.String(), expected)
		}
	})

	t.Run("pretty query param", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/?pretty", nil)

		if err := resp.Send(ctx); err != nil {
			t.Fatalf("Send() returned error: %v", err)
		}

		expected := "{\n  \"code\": 404,\n  \"message\": \"not_found\",\n  \"error\": \"\\u003cnot found\\u003e \\u0026 gone\"\n}\n"
		if rec.Body.String() != expected {
			t.Fatalf("body = %q, want %q", rec.Body.String(), expected)
		}
	})

	t.Run("gin debug mode alone does not indent", func(t *testing.T) {
		gin.SetMode(gin.DebugMode)
		defer gin.SetMode(gin.TestMode)

		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		if err := SuccessBuilder(nil).Send(ctx); err != nil {
			t.Fatalf("Send() returned error: %v", err)
		}

		expected := "{\"code\":200,\"message\":\"success\"}\n"
		if rec.Body.String() != expected {
			t.Fatalf("body = %q, want %q", rec.Body.String(), expected)
		}
	})

	t.Run("engine debug", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		ctx.Set(types.DebugContextKey.String(), true)

		if err := SuccessBuilder(nil).Send(ctx); err != nil {
			t.Fatalf("Send() returned error: %v", err)
		}

		expected := "{\n  \"code\": 200,\n  \"message\": \"success\"\n}\n"
		if rec.Body.String() != expected {
			t.Fatalf("body = %q, want %q", rec.Body.String(), expected)
		}
	})

	t.Run("keeps content type already set", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		ctx.Header("Content-Type", "text/plain; charset=utf-8")

		if err := resp.Send(ctx); err != nil {
			t.Fatalf("Send() returned error: %v", err)
		}

		if got := rec.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want %q", got, "text/plain; charset=utf-8")
		}
	})

	t.Run("encoding error is pushed to the context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		err := SuccessBuilder(make(chan int)).Send(ctx)
		if err == nil {
			t.Fatal("Send() should return the encoding error")
		}
		if last := ctx.Errors.Last(); last == nil || last.Err != err {
			t.Fatalf("ctx.Errors.Last() = %v, want %v", last, err)
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("body = %q, want empty", rec.Body.String())
		}
	})
}
