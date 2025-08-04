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

func TestDeleteDeploymentHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll("deployments")

	// Create test deployment in database and filesystem
	testID := "test-delete-123"
	testFilename := "test-site.zip"
	testPath := filepath.Join("deployments", testID)

	// Create test directory and file
	err := os.MkdirAll(testPath, 0755)
	if err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	testFile := filepath.Join(testPath, "index.html")
	err = os.WriteFile(testFile, []byte("<html>test</html>"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Insert into database
	_, err = db.Exec(
		"INSERT INTO deployments (id, filename, timestamp, path) VALUES (?, ?, ?, ?)",
		testID, testFilename, time.Now(), testPath,
	)
	if err != nil {
		t.Fatalf("failed to insert test deployment: %v", err)
	}

	// Test successful deletion
	req := httptest.NewRequest(http.MethodDelete, "/deployments/"+testID, nil)
	rr := httptest.NewRecorder()

	DeleteDeploymentHandler(rr, req, db)

	// Check response status
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d. Response: %s", status, rr.Body.String())
	}

	// Check response content
	var response map[string]string
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedMessage := "Deployment " + testID + " (" + testFilename + ") deleted successfully"
	if response["message"] != expectedMessage {
		t.Errorf("expected message %q, got %q", expectedMessage, response["message"])
	}

	// Verify deployment was removed from database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM deployments WHERE id = ?", testID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if count != 0 {
		t.Errorf("expected deployment to be deleted from database, but still exists")
	}

	// Verify files were removed from filesystem
	if _, err := os.Stat(testPath); !os.IsNotExist(err) {
		t.Error("expected deployment directory to be deleted from filesystem")
	}
}

func TestDeleteDeploymentHandlerNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Try to delete non-existent deployment
	req := httptest.NewRequest(http.MethodDelete, "/deployments/nonexistent-id", nil)
	rr := httptest.NewRecorder()

	DeleteDeploymentHandler(rr, req, db)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "Deployment not found") {
		t.Error("expected 'Deployment not found' error message")
	}
}

func TestDeleteDeploymentHandlerInvalidMethod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/deployments/some-id", nil)
	rr := httptest.NewRecorder()

	DeleteDeploymentHandler(rr, req, db)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "DELETE required") {
		t.Error("expected 'DELETE required' error message")
	}
}

func TestDeleteDeploymentHandlerMissingID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Try to delete without providing ID
	req := httptest.NewRequest(http.MethodDelete, "/deployments/", nil)
	rr := httptest.NewRecorder()

	DeleteDeploymentHandler(rr, req, db)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "Deployment ID required") {
		t.Error("expected 'Deployment ID required' error message")
	}
}
