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

func TestDeleteAllDeploymentsHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll("deployments")

	// Create multiple test deployments
	testDeployments := []struct {
		id       string
		filename string
		path     string
	}{
		{"test-1", "site1.zip", "deployments/test-1"},
		{"test-2", "site2.zip", "deployments/test-2"},
		{"test-3", "site3.zip", "deployments/test-3"},
	}

	// Insert deployments into database and create directories
	for _, td := range testDeployments {
		// Create directory and test file
		err := os.MkdirAll(td.path, 0755)
		if err != nil {
			t.Fatalf("failed to create directory %s: %v", td.path, err)
		}

		testFile := filepath.Join(td.path, "index.html")
		err = os.WriteFile(testFile, []byte("<html>test</html>"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		// Insert into database
		_, err = db.Exec(
			"INSERT INTO deployments (id, filename, timestamp, path) VALUES (?, ?, ?, ?)",
			td.id, td.filename, time.Now(), td.path,
		)
		if err != nil {
			t.Fatalf("failed to insert test deployment: %v", err)
		}
	}

	// Verify deployments exist before deletion
	var countBefore int
	err := db.QueryRow("SELECT COUNT(*) FROM deployments").Scan(&countBefore)
	if err != nil {
		t.Fatalf("failed to count deployments: %v", err)
	}
	if countBefore != 3 {
		t.Errorf("expected 3 deployments before deletion, got %d", countBefore)
	}

	// Test delete all
	req := httptest.NewRequest(http.MethodDelete, "/deployments", nil)
	rr := httptest.NewRecorder()

	DeleteAllDeploymentsHandler(rr, req, db)

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

	// Check response content
	if response["message"] != "Bulk deletion completed" {
		t.Errorf("unexpected message: %v", response["message"])
	}

	deletedCount, ok := response["deleted_count"].(float64)
	if !ok || int(deletedCount) != 3 {
		t.Errorf("expected deleted_count 3, got %v", response["deleted_count"])
	}

	// Verify database is empty
	var countAfter int
	err = db.QueryRow("SELECT COUNT(*) FROM deployments").Scan(&countAfter)
	if err != nil {
		t.Fatalf("failed to count deployments after deletion: %v", err)
	}
	if countAfter != 0 {
		t.Errorf("expected 0 deployments after deletion, got %d", countAfter)
	}

	// Verify directories were deleted
	for _, td := range testDeployments {
		if _, err := os.Stat(td.path); !os.IsNotExist(err) {
			t.Errorf("directory %s should have been deleted", td.path)
		}
	}
}

func TestDeleteAllDeploymentsHandlerEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Test delete all when no deployments exist
	req := httptest.NewRequest(http.MethodDelete, "/deployments", nil)
	rr := httptest.NewRecorder()

	DeleteAllDeploymentsHandler(rr, req, db)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}

	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["message"] != "No deployments to delete" {
		t.Errorf("unexpected message: %v", response["message"])
	}

	deletedCount, ok := response["deleted_count"].(float64)
	if !ok || int(deletedCount) != 0 {
		t.Errorf("expected deleted_count 0, got %v", response["deleted_count"])
	}
}

func TestDeleteAllDeploymentsHandlerInvalidMethod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/deployments", nil)
	rr := httptest.NewRecorder()

	DeleteAllDeploymentsHandler(rr, req, db)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", status)
	}

	if !strings.Contains(rr.Body.String(), "DELETE required") {
		t.Error("expected 'DELETE required' error message")
	}
}

func TestResetSystemHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	defer os.RemoveAll("deployments")

	// Create test deployment
	testPath := "deployments/test-reset"
	err := os.MkdirAll(testPath, 0755)
	if err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	_, err = db.Exec(
		"INSERT INTO deployments (id, filename, timestamp, path) VALUES (?, ?, ?, ?)",
		"test-reset", "test.zip", time.Now(), testPath,
	)
	if err != nil {
		t.Fatalf("failed to insert test deployment: %v", err)
	}

	// Test system reset
	req := httptest.NewRequest(http.MethodPost, "/reset", nil)
	rr := httptest.NewRecorder()

	ResetSystemHandler(rr, req, db)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %d. Response: %s", status, rr.Body.String())
	}

	// Verify database is empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM deployments").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count deployments: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 deployments after reset, got %d", count)
	}

	// Verify deployments directory exists but is empty
	if _, err := os.Stat("deployments"); os.IsNotExist(err) {
		t.Error("deployments directory should exist after reset")
	}

	// Verify test deployment directory was removed
	if _, err := os.Stat(testPath); !os.IsNotExist(err) {
		t.Error("test deployment directory should have been removed")
	}
}
