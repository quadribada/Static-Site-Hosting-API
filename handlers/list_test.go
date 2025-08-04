package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestListDeploymentsHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name        string
		deployments []Deployment
		expected    int
	}{
		{
			name:        "empty deployments",
			deployments: []Deployment{},
			expected:    0,
		},
		{
			name: "single deployment",
			deployments: []Deployment{
				{
					ID:        "test-123",
					Filename:  "my-site.zip",
					Timestamp: time.Now(),
					Path:      "deployments/test-123",
				},
			},
			expected: 1,
		},
		{
			name: "multiple deployments",
			deployments: []Deployment{
				{
					ID:        "test-123",
					Filename:  "my-site.zip",
					Timestamp: time.Now(),
					Path:      "deployments/test-123",
				},
				{
					ID:        "test-456",
					Filename:  "another-site.zip",
					Timestamp: time.Now(),
					Path:      "deployments/test-456",
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear database and insert test data
			_, err := db.Exec("DELETE FROM deployments")
			if err != nil {
				t.Fatalf("failed to clear database: %v", err)
			}

			// Insert test deployments into database
			for _, deployment := range tt.deployments {
				_, err := db.Exec(
					"INSERT INTO deployments (id, filename, timestamp, path) VALUES (?, ?, ?, ?)",
					deployment.ID, deployment.Filename, deployment.Timestamp, deployment.Path,
				)
				if err != nil {
					t.Fatalf("failed to insert test deployment: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodGet, "/deployments", nil)
			rr := httptest.NewRecorder()

			ListDeploymentsHandler(rr, req, db)

			// Check status code
			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}

			// Check content type
			expectedContentType := "application/json"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("expected Content-Type %s, got %s", expectedContentType, contentType)
			}

			// Parse response
			var responseDeployments []Deployment
			err = json.NewDecoder(rr.Body).Decode(&responseDeployments)
			if err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Check number of deployments
			if len(responseDeployments) != tt.expected {
				t.Errorf("expected %d deployments, got %d", tt.expected, len(responseDeployments))
			}

			// Verify deployment data matches
			// Note: We just check that all expected deployments are present,
			// regardless of order since database ordering may vary
			if len(responseDeployments) > 0 {
				// Create maps for easier comparison
				expectedIDs := make(map[string]Deployment)
				for _, d := range tt.deployments {
					expectedIDs[d.ID] = d
				}

				responseIDs := make(map[string]Deployment)
				for _, d := range responseDeployments {
					responseIDs[d.ID] = d
				}

				// Check that all expected deployments are present
				for expectedID, expectedDeployment := range expectedIDs {
					responseDeployment, exists := responseIDs[expectedID]
					if !exists {
						t.Errorf("expected deployment %s not found in response", expectedID)
						continue
					}

					if responseDeployment.Filename != expectedDeployment.Filename {
						t.Errorf("deployment %s: expected Filename %s, got %s",
							expectedID, expectedDeployment.Filename, responseDeployment.Filename)
					}

					if responseDeployment.Path != expectedDeployment.Path {
						t.Errorf("deployment %s: expected Path %s, got %s",
							expectedID, expectedDeployment.Path, responseDeployment.Path)
					}
				}
			}
		})
	}
}

func TestListDeploymentsHandlerInvalidMethod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/deployments", nil)
	rr := httptest.NewRecorder()

	ListDeploymentsHandler(rr, req, db)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", status)
	}
}
