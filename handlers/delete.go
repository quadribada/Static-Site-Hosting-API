package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"static-site-hosting/models"
)

func DeleteDeploymentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodDelete {
		http.Error(w, "DELETE required", http.StatusMethodNotAllowed)
		return
	}

	// Extract deployment ID from URL path
	// Expected: DELETE /deployments/{id}
	path := strings.TrimPrefix(r.URL.Path, "/deployments/")
	if path == "" {
		http.Error(w, "Deployment ID required", http.StatusBadRequest)
		return
	}
	deploymentID := path

	// Get deployment info before deleting
	var deployment models.Deployment
	err := db.QueryRow("SELECT id, filename, timestamp, path FROM deployments WHERE id = ?", deploymentID).
		Scan(&deployment.ID, &deployment.Filename, &deployment.Timestamp, &deployment.Path)

	if err == sql.ErrNoRows {
		http.Error(w, "Deployment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to fetch deployment", http.StatusInternalServerError)
		return
	}

	// Delete from database
	_, err = db.Exec("DELETE FROM deployments WHERE id = ?", deploymentID)
	if err != nil {
		http.Error(w, "Failed to delete from database", http.StatusInternalServerError)
		return
	}

	// Delete files from filesystem
	if err := os.RemoveAll(deployment.Path); err != nil {
		// Log error but don't fail the request since DB deletion succeeded
		fmt.Printf("Warning: Failed to delete files at %s: %v\n", deployment.Path, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Deployment %s (%s) deleted successfully", deploymentID, deployment.Filename),
	})
}
