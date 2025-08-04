package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(next)

	req := httptest.NewRequest("GET", "/test-path", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Expected next handler to be called")
	}

	logged := buf.String()
	if !strings.Contains(logged, "GET /test-path") {
		t.Errorf("Expected log to contain 'GET /test-path', got %q", logged)
	}
}
