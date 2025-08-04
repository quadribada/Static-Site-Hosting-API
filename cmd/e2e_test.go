package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"static-site-hosting/handlers"
	"static-site-hosting/middleware"
)

func setupTestE2EDatabase(t *testing.T) *sql.DB {
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

func setupE2ERoutes(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()

	// API endpoints with database
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		handlers.UploadHandler(w, r, db)
	})
	mux.HandleFunc("/deployments", func(w http.ResponseWriter, r *http.Request) {
		handlers.ListDeploymentsHandler(w, r, db)
	})
	mux.HandleFunc("/deployments/", func(w http.ResponseWriter, r *http.Request) {
		handlers.DeleteDeploymentHandler(w, r, db)
	})
	mux.HandleFunc("/hello-world", handlers.HelloWorldHandler)

	// Static file serving - this should be last since it's a catch-all
	mux.Handle("/", handlers.StaticFileHandler())

	return mux
}

// E2E Test that simulates the complete user workflow
func TestE2EStaticSiteHostingWorkflow(t *testing.T) {
	// Setup: Clean state
	defer os.RemoveAll("deployments")

	// Create test database
	db := setupTestE2EDatabase(t)
	defer db.Close()

	// Create test server with database
	mux := setupE2ERoutes(db)
	server := httptest.NewServer(middleware.LoggingMiddleware(mux))
	defer server.Close()

	t.Run("Complete Workflow", func(t *testing.T) {
		// Step 1: Verify no deployments initially
		t.Log("Step 1: Check initial empty state")
		deployments := listDeployments(t, server.URL)
		if len(deployments) != 0 {
			t.Errorf("Expected 0 deployments initially, got %d", len(deployments))
		}

		// Step 2: Upload a site
		t.Log("Step 2: Upload test site")
		deployment := uploadTestSite(t, server.URL)
		if deployment.ID == "" {
			t.Fatal("Expected deployment ID to be set")
		}

		// Step 3: Verify deployment appears in list
		t.Log("Step 3: Verify deployment in list")
		deployments = listDeployments(t, server.URL)
		if len(deployments) != 1 {
			t.Errorf("Expected 1 deployment, got %d", len(deployments))
		}
		if deployments[0].ID != deployment.ID {
			t.Errorf("Expected deployment ID %s, got %s", deployment.ID, deployments[0].ID)
		}

		// Step 4: Access static files
		t.Log("Step 4: Test static file serving")
		testStaticFileAccess(t, server.URL, deployment.ID)

		// Step 5: Upload another site
		t.Log("Step 5: Upload second site")
		deployment2 := uploadTestSite(t, server.URL)
		if deployment2.ID == deployment.ID {
			t.Error("Second deployment should have different ID")
		}

		// Step 6: Verify both deployments exist
		t.Log("Step 6: Verify multiple deployments")
		deployments = listDeployments(t, server.URL)
		if len(deployments) != 2 {
			t.Errorf("Expected 2 deployments, got %d", len(deployments))
		}

		// Step 7: Test both sites are accessible independently
		t.Log("Step 7: Test independent site access")
		testStaticFileAccess(t, server.URL, deployment.ID)
		testStaticFileAccess(t, server.URL, deployment2.ID)

		// Step 8: Test deletion functionality
		t.Log("Step 8: Test deployment deletion")
		deleteURL := fmt.Sprintf("%s/deployments/%s", server.URL, deployment.ID)
		req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			t.Fatalf("Failed to create delete request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Delete request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 for delete, got %d", resp.StatusCode)
		}

		// Step 9: Verify deployment was deleted
		t.Log("Step 9: Verify deployment deleted")
		deployments = listDeployments(t, server.URL)
		if len(deployments) != 1 {
			t.Errorf("Expected 1 deployment after deletion, got %d", len(deployments))
		}

		// Verify the remaining deployment is the second one
		if len(deployments) > 0 && deployments[0].ID != deployment2.ID {
			t.Errorf("Expected remaining deployment to be %s, got %s", deployment2.ID, deployments[0].ID)
		}

		// Step 10: Verify deleted site is no longer accessible
		t.Log("Step 10: Verify deleted site inaccessible")
		resp, err = http.Get(server.URL + "/" + deployment.ID + "/index.html")
		if err != nil {
			t.Fatalf("Failed to test deleted site access: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 for deleted site, got %d", resp.StatusCode)
		}
	})
}

func TestE2EErrorScenarios(t *testing.T) {
	defer os.RemoveAll("deployments")

	db := setupTestE2EDatabase(t)
	defer db.Close()

	mux := setupE2ERoutes(db)
	server := httptest.NewServer(middleware.LoggingMiddleware(mux))
	defer server.Close()

	t.Run("Invalid Upload", func(t *testing.T) {
		// Test uploading non-zip file
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write([]byte("not a zip file"))
		writer.Close()

		resp, err := http.Post(
			server.URL+"/upload",
			writer.FormDataContentType(),
			body,
		)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected upload of invalid file to fail")
		}
	})

	t.Run("Access Non-existent Site", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/nonexistent-site/index.html")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 for non-existent site, got %d", resp.StatusCode)
		}
	})

	t.Run("Access Non-existent File", func(t *testing.T) {
		// First upload a site
		deployment := uploadTestSite(t, server.URL)

		// Try to access non-existent file
		resp, err := http.Get(server.URL + "/" + deployment.ID + "/nonexistent.html")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 for non-existent file, got %d", resp.StatusCode)
		}
	})

	t.Run("Delete Non-existent Deployment", func(t *testing.T) {
		deleteURL := fmt.Sprintf("%s/deployments/nonexistent-id", server.URL)
		req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
		if err != nil {
			t.Fatalf("Failed to create delete request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Delete request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 for deleting non-existent deployment, got %d", resp.StatusCode)
		}
	})
}

// Helper functions

func createTestSite() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	files := map[string]string{
		"index.html":    `<!DOCTYPE html><html><head><title>Test Site</title></head><body><h1>Welcome to Test Site</h1><p>This is a test deployment.</p></body></html>`,
		"about.html":    `<!DOCTYPE html><html><head><title>About</title></head><body><h1>About Page</h1><p>About our test site.</p></body></html>`,
		"css/style.css": `body { font-family: Arial, sans-serif; color: #333; } h1 { color: #007acc; }`,
		"js/main.js":    `console.log('Test site loaded successfully!'); document.addEventListener('DOMContentLoaded', function() { console.log('DOM ready'); });`,
		"robots.txt":    `User-agent: *\nDisallow:`,
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

	return buf, w.Close()
}

func uploadTestSite(t *testing.T, serverURL string) handlers.Deployment {
	zipBuffer, err := createTestSite()
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test-site.zip")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	io.Copy(part, zipBuffer)
	writer.Close()

	resp, err := http.Post(
		serverURL+"/upload",
		writer.FormDataContentType(),
		body,
	)
	if err != nil {
		t.Fatalf("Upload request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deployment handlers.Deployment
	if err := json.NewDecoder(resp.Body).Decode(&deployment); err != nil {
		t.Fatalf("Failed to decode upload response: %v", err)
	}

	return deployment
}

func listDeployments(t *testing.T, serverURL string) []handlers.Deployment {
	resp, err := http.Get(serverURL + "/deployments")
	if err != nil {
		t.Fatalf("Failed to list deployments: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List deployments failed with status %d", resp.StatusCode)
	}

	var deployments []handlers.Deployment
	if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
		t.Fatalf("Failed to decode deployments response: %v", err)
	}

	return deployments
}

func testStaticFileAccess(t *testing.T, serverURL, siteID string) {
	testCases := []struct {
		file         string
		expectedCode int
		contains     string
	}{
		{"index.html", 200, "Welcome to Test Site"},
		{"about.html", 200, "About Page"},
		{"css/style.css", 200, "font-family: Arial"},
		{"js/main.js", 200, "Test site loaded"},
		{"robots.txt", 200, "User-agent"},
		{"nonexistent.html", 404, ""},
	}

	for _, tc := range testCases {
		url := fmt.Sprintf("%s/%s/%s", serverURL, siteID, tc.file)
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("Failed to access %s: %v", tc.file, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != tc.expectedCode {
			t.Errorf("File %s: expected status %d, got %d", tc.file, tc.expectedCode, resp.StatusCode)
		}

		if tc.expectedCode == 200 && tc.contains != "" {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("Failed to read response for %s: %v", tc.file, err)
				continue
			}

			if !strings.Contains(string(body), tc.contains) {
				t.Errorf("File %s: expected content to contain %q", tc.file, tc.contains)
			}
		}
	}
}

func TestE2EPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	defer os.RemoveAll("deployments")

	db := setupTestE2EDatabase(t)
	defer db.Close()

	mux := setupE2ERoutes(db)
	server := httptest.NewServer(middleware.LoggingMiddleware(mux))
	defer server.Close()

	// Upload a site
	deployment := uploadTestSite(t, server.URL)

	// Test concurrent access
	t.Run("Concurrent File Access", func(t *testing.T) {
		const numRequests = 50
		done := make(chan bool, numRequests)

		start := time.Now()

		for i := 0; i < numRequests; i++ {
			go func() {
				resp, err := http.Get(server.URL + "/" + deployment.ID + "/index.html")
				if err != nil {
					t.Errorf("Request failed: %v", err)
				} else {
					resp.Body.Close()
					if resp.StatusCode != 200 {
						t.Errorf("Expected 200, got %d", resp.StatusCode)
					}
				}
				done <- true
			}()
		}

		for i := 0; i < numRequests; i++ {
			<-done
		}

		duration := time.Since(start)
		t.Logf("Completed %d concurrent requests in %v", numRequests, duration)

		if duration > 5*time.Second {
			t.Errorf("Performance test took too long: %v", duration)
		}
	})
}
