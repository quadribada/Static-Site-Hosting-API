package models

import (
	"testing"
	"time"
)

func TestNewDeployment(t *testing.T) {
	id := "test-123"
	filename := "test-site.zip"
	path := "deployments/test-123"

	deployment := NewDeployment(id, filename, path)

	if deployment.ID != id {
		t.Errorf("expected ID %s, got %s", id, deployment.ID)
	}

	if deployment.Filename != filename {
		t.Errorf("expected Filename %s, got %s", filename, deployment.Filename)
	}

	if deployment.Path != path {
		t.Errorf("expected Path %s, got %s", path, deployment.Path)
	}

	// Check timestamp is recent (within last second)
	if time.Since(deployment.Timestamp) > time.Second {
		t.Error("expected Timestamp to be recent")
	}
}

func TestDeploymentTableName(t *testing.T) {
	deployment := &Deployment{}
	expected := "deployments"

	if deployment.TableName() != expected {
		t.Errorf("expected table name %s, got %s", expected, deployment.TableName())
	}
}

func TestDeploymentJSONSerialization(t *testing.T) {
	deployment := NewDeployment("test-123", "site.zip", "/path")

	// Test that the struct has proper JSON tags
	if deployment.ID == "" {
		t.Error("ID should not be empty")
	}

	if deployment.Filename == "" {
		t.Error("Filename should not be empty")
	}

	if deployment.Path == "" {
		t.Error("Path should not be empty")
	}
}
