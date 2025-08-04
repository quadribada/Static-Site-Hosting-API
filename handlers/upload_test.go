package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestZip() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Add test files to zip
	files := map[string]string{
		"index.html": "<html><body>Test Site</body></html>",
		"style.css":  "body { color: blue; }",
		"script.js":  "console.log('hello world');",
	}

	for filename, content := range files {
		f, err := w.Create(filename)
		if err != nil {
			return nil, err
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			return nil, err
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func TestUploadHandler(t *testing.T) {
	// Clean up any existing deployments
	defer os.RemoveAll("deployments")
	deployments = []Deployment{} // Reset the deployments slice

	// Create test zip dynamically
	zipBuffer, err := createTestZip()
	if err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test-site.zip")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	_, err = io.Copy(part, zipBuffer)
	if err != nil {
		t.Fatalf("failed to copy zip to form: %v", err)
	}

	writer.Close()

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	rr := httptest.NewRecorder()
	UploadHandler(rr, req)

	// Check response status
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d. Response: %s", status, rr.Body.String())
	}

	// Parse response
	var deployment Deployment
	err = json.NewDecoder(rr.Body).Decode(&deployment)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify deployment was created
	if deployment.ID == "" {
		t.Error("expected deployment ID to be set")
	}

	if deployment.Path == "" {
		t.Error("expected deployment path to be set")
	}

	// Verify files were extracted
	expectedFiles := []string{"index.html", "style.css", "script.js"}
	for _, filename := range expectedFiles {
		filePath := filepath.Join(deployment.Path, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist in deployment", filename)
		}
	}

	// Verify file content
	indexPath := filepath.Join(deployment.Path, "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}

	expectedContent := "<html><body>Test Site</body></html>"
	if string(content) != expectedContent {
		t.Errorf("expected file content %q, got %q", expectedContent, string(content))
	}
}

func TestUploadHandlerInvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/upload", nil)
	rr := httptest.NewRecorder()

	UploadHandler(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", status)
	}
}

func TestUploadHandlerNoFile(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	UploadHandler(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "Invalid file") {
		t.Error("expected 'Invalid file' error message")
	}
}
