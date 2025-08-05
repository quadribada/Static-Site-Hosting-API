package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestRollbackHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll("deployments")

	// Create source deployment in database and filesystem
	sourceID := "source-deployment-123"
	sourceFilename := "original-site.zip"
	sourcePath := filepath.Join("deployments", sourceID)

	// Create test directory and files
	err := os.MkdirAll(sourcePath, 0755)
	if err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	// Create test files
	testFiles := map[string]string{
		"index.html": "<html><body>Original Site</body></html>",
		"style.css":  "body { color: red; }",
		"README.txt": "This is the original site",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(sourcePath, filename)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", filename, err)
		}
	}

	// Insert source deployment into database
	_, err = db.Exec(
		"INSERT INTO deployments (id, filename, timestamp, path) VALUES (?, ?, ?, ?)",
		sourceID, sourceFilename, time.Now(), sourcePath,
	)
	if err != nil {
		t.Fatalf("failed to insert source deployment: %v", err)
	}

	// Test successful rollback
	req := httptest.NewRequest(http.MethodPost, "/rollback/"+sourceID, nil)
	rr := httptest.NewRecorder()

	RollbackHandler(rr, req, db)

	// Check response status
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d. Response: %s", status, rr.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check response structure
	if response["message"] != "Rollback successful" {
		t.Errorf("expected success message, got: %v", response["message"])
	}

	// Extract new deployment info
	newDeploymentData, ok := response["new_deployment"].(map[string]interface{})
	if !ok {
		t.Fatal("new_deployment not found in response")
	}

	newDeploymentID, ok := newDeploymentData["id"].(string)
	if !ok || newDeploymentID == "" {
		t.Fatal("new deployment ID not found or empty")
	}

	newDeploymentPath, ok := newDeploymentData["path"].(string)
	if !ok || newDeploymentPath == "" {
		t.Fatal("new deployment path not found or empty")
	}

	// Verify new deployment was saved to database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM deployments WHERE id = ?", newDeploymentID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 new deployment in database, got %d", count)
	}

	// Verify files were copied correctly
	for filename, expectedContent := range testFiles {
		newFilePath := filepath.Join(newDeploymentPath, filename)
		content, err := os.ReadFile(newFilePath)
		if err != nil {
			t.Errorf("failed to read copied file %s: %v", filename, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("file %s: expected content %q, got %q", filename, expectedContent, string(content))
		}
	}

	// Verify filename has [ROLLBACK] prefix
	expectedFilename := "[ROLLBACK] " + sourceFilename
	actualFilename, ok := newDeploymentData["filename"].(string)
	if !ok || actualFilename != expectedFilename {
		t.Errorf("expected filename %q, got %q", expectedFilename, actualFilename)
	}
}

func TestRollbackHandlerNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Try to rollback non-existent deployment
	req := httptest.NewRequest(http.MethodPost, "/rollback/nonexistent-id", nil)
	rr := httptest.NewRecorder()

	RollbackHandler(rr, req, db)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "Source deployment not found") {
		t.Error("expected 'Source deployment not found' error message")
	}
}

func TestRollbackHandlerInvalidMethod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/rollback/some-id", nil)
	rr := httptest.NewRecorder()

	RollbackHandler(rr, req, db)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "POST required") {
		t.Error("expected 'POST required' error message")
	}
}

func TestRollbackHandlerMissingID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/rollback/", nil)
	rr := httptest.NewRecorder()

	RollbackHandler(rr, req, db)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "Deployment ID required") {
		t.Error("expected 'Deployment ID required' error message")
	}
}

func TestRollbackHandlerFilesNotExist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll("deployments")

	// Create deployment in database but don't create files
	sourceID := "source-without-files"
	sourcePath := filepath.Join("deployments", sourceID)

	_, err := db.Exec(
		"INSERT INTO deployments (id, filename, timestamp, path) VALUES (?, ?, ?, ?)",
		sourceID, "test.zip", time.Now(), sourcePath,
	)
	if err != nil {
		t.Fatalf("failed to insert source deployment: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/rollback/"+sourceID, nil)
	rr := httptest.NewRecorder()

	RollbackHandler(rr, req, db)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "Source deployment files no longer exist") {
		t.Error("expected 'files no longer exist' error message")
	}
}
