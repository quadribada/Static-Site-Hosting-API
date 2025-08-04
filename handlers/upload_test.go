package handlers

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create tables
	createDeploymentsTable := `
	CREATE TABLE deployments (
		id TEXT PRIMARY KEY,
		filename TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(createDeploymentsTable); err != nil {
		t.Fatalf("Failed to create deployments table: %v", err)
	}

	return db
}

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
	db := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll("deployments")

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

	// Execute request with database
	rr := httptest.NewRecorder()
	UploadHandler(rr, req, db)

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

	if deployment.Filename != "test-site.zip" {
		t.Errorf("expected filename 'test-site.zip', got %s", deployment.Filename)
	}

	if deployment.Path == "" {
		t.Error("expected deployment path to be set")
	}

	// Verify it was saved to database
	var dbCount int
	err = db.QueryRow("SELECT COUNT(*) FROM deployments WHERE id = ?", deployment.ID).Scan(&dbCount)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if dbCount != 1 {
		t.Errorf("expected 1 deployment in database, got %d", dbCount)
	}

	// Verify files were extracted
	expectedFiles := []string{"index.html", "style.css", "script.js"}
	for _, filename := range expectedFiles {
		filePath := filepath.Join(deployment.Path, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist in deployment", filename)
		}
	}
}

func TestUploadHandlerInvalidMethod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/upload", nil)
	rr := httptest.NewRecorder()

	UploadHandler(rr, req, db)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", status)
	}
}

func TestUploadHandlerNoFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	UploadHandler(rr, req, db)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "Invalid file") {
		t.Error("expected 'Invalid file' error message")
	}
}

func TestUploadHandlerWithFilename(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll("deployments")

	zipBuffer, err := createTestZip()
	if err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Use a specific filename to test
	testFilename := "my-awesome-site.zip"
	part, err := writer.CreateFormFile("file", testFilename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	_, err = io.Copy(part, zipBuffer)
	if err != nil {
		t.Fatalf("failed to copy zip to form: %v", err)
	}

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	UploadHandler(rr, req, db)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d. Response: %s", status, rr.Body.String())
	}

	var deployment Deployment
	err = json.NewDecoder(rr.Body).Decode(&deployment)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Test the new filename field
	if deployment.Filename != testFilename {
		t.Errorf("expected filename %s, got %s", testFilename, deployment.Filename)
	}

	if deployment.ID == "" {
		t.Error("expected deployment ID to be set")
	}

	if deployment.Path == "" {
		t.Error("expected deployment path to be set")
	}
}

func TestUploadHandlerEmptyFilename(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll("deployments")

	zipBuffer, err := createTestZip()
	if err != nil {
		t.Fatalf("failed to create test zip: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Use empty filename - this should cause the upload to fail
	part, err := writer.CreateFormFile("file", "")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	_, err = io.Copy(part, zipBuffer)
	if err != nil {
		t.Fatalf("failed to copy zip to form: %v", err)
	}

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	UploadHandler(rr, req, db)

	// Empty filename should cause a 400 Bad Request
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty filename, got %d", status)
	}

	// Verify we get the expected error message
	if !strings.Contains(rr.Body.String(), "Invalid file") {
		t.Errorf("expected 'Invalid file' error message, got: %s", rr.Body.String())
	}
}
