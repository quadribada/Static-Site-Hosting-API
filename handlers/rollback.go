package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"static-site-hosting/models"

	"github.com/google/uuid"
)

func RollbackHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	// Extract deployment ID from URL path
	// Expected: POST /rollback/{deployment-id}
	path := strings.TrimPrefix(r.URL.Path, "/rollback/")
	if path == "" {
		http.Error(w, "Deployment ID required", http.StatusBadRequest)
		return
	}
	sourceDeploymentID := path

	// Get the source deployment info
	var sourceDeployment models.Deployment
	err := db.QueryRow("SELECT id, filename, timestamp, path FROM deployments WHERE id = ?", sourceDeploymentID).
		Scan(&sourceDeployment.ID, &sourceDeployment.Filename, &sourceDeployment.Timestamp, &sourceDeployment.Path)

	if err == sql.ErrNoRows {
		http.Error(w, "Source deployment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to fetch source deployment", http.StatusInternalServerError)
		return
	}

	// Check if source deployment files still exist
	if _, err := os.Stat(sourceDeployment.Path); os.IsNotExist(err) {
		http.Error(w, "Source deployment files no longer exist", http.StatusNotFound)
		return
	}

	// Create new deployment ID for the rollback
	newDeploymentID := uuid.New().String()
	newDeploymentPath := filepath.Join("deployments", newDeploymentID)

	// Copy files from source deployment to new deployment
	if err := copyDir(sourceDeployment.Path, newDeploymentPath); err != nil {
		http.Error(w, "Failed to copy deployment files", http.StatusInternalServerError)
		return
	}

	// Create new deployment record in database
	newFilename := fmt.Sprintf("[ROLLBACK] %s", sourceDeployment.Filename)
	newDeployment := models.NewDeployment(newDeploymentID, newFilename, newDeploymentPath)

	_, err = db.Exec(
		"INSERT INTO deployments (id, filename, timestamp, path) VALUES (?, ?, ?, ?)",
		newDeployment.ID, newDeployment.Filename, newDeployment.Timestamp, newDeployment.Path,
	)
	if err != nil {
		// Clean up files if DB insert fails
		os.RemoveAll(newDeploymentPath)
		http.Error(w, "Failed to save rollback deployment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":           "Rollback successful",
		"source_deployment": sourceDeployment,
		"new_deployment":    newDeployment,
	}
	json.NewEncoder(w).Encode(response)
}

// copyDir recursively copies a directory tree
func copyDir(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}
