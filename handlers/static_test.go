package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStaticFileHandler(t *testing.T) {
	// Setup test directory
	siteID := "test123"
	deployPath := filepath.Join("deployments", siteID)
	err := os.MkdirAll(deployPath, 0755)
	if err != nil {
		t.Fatalf("failed to create deployments dir: %v", err)
	}
	defer os.RemoveAll("deployments")

	// Create test file
	testContent := "<html><body>Hello World</body></html>"
	indexPath := filepath.Join(deployPath, "index.html")
	err = os.WriteFile(indexPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "serve existing file",
			path:           "/test123/index.html",
			expectedStatus: http.StatusOK,
			expectedBody:   testContent,
		},
		{
			name:           "file not found",
			path:           "/test123/nonexistent.html",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "invalid path - no site ID",
			path:           "/index.html",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "invalid path - only site ID",
			path:           "/test123/",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
	}

	handler := StaticFileHandler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK && rr.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}
