package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHelloWorldHandler(t *testing.T) {
	t.Run("Returns Hello World", func(t *testing.T) {
		// Create request to test endpoint
		req, err := http.NewRequest("GET", "/hello-world", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Create response recorder
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(HelloWorldHandler)

		// Execute handler
		handler.ServeHTTP(rr, req)

		// Verify status code
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		// Verify response body
		expected := "Hello, World!"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %q want %q",
				rr.Body.String(), expected)
		}
	})
}
