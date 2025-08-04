package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListDeploymentsHandler(t *testing.T) {
	// Save original deployments and restore after test
	originalDeployments := deployments
	defer func() { deployments = originalDeployments }()

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
			// Set up test data
			deployments = tt.deployments

			req := httptest.NewRequest(http.MethodGet, "/deployments", nil)
			rr := httptest.NewRecorder()

			ListDeploymentsHandler(rr, req)

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
			err := json.NewDecoder(rr.Body).Decode(&responseDeployments)
			if err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Check number of deployments
			if len(responseDeployments) != tt.expected {
				t.Errorf("expected %d deployments, got %d", tt.expected, len(responseDeployments))
			}

			// Verify deployment data matches
			for i, deployment := range responseDeployments {
				if i >= len(tt.deployments) {
					break
				}
				expected := tt.deployments[i]

				if deployment.ID != expected.ID {
					t.Errorf("deployment %d: expected ID %s, got %s", i, expected.ID, deployment.ID)
				}

				if deployment.Path != expected.Path {
					t.Errorf("deployment %d: expected Path %s, got %s", i, expected.Path, deployment.Path)
				}
			}
		})
	}
}
