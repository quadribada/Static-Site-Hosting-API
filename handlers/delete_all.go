package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"static-site-hosting/models"
)

func DeleteAllDeploymentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodDelete {
		http.Error(w, "DELETE required", http.StatusMethodNotAllowed)
		return
	}

	// Get all deployments before deleting
	rows, err := db.Query("SELECT id, filename, timestamp, path FROM deployments")
	if err != nil {
		http.Error(w, "Failed to fetch deployments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var deployments []models.Deployment
	var pathsToDelete []string

	for rows.Next() {
		var d models.Deployment
		err := rows.Scan(&d.ID, &d.Filename, &d.Timestamp, &d.Path)
		if err != nil {
			http.Error(w, "Failed to scan deployment", http.StatusInternalServerError)
			return
		}
		deployments = append(deployments, d)
		pathsToDelete = append(pathsToDelete, d.Path)
	}

	// If no deployments exist, return early
	if len(deployments) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":          "No deployments to delete",
			"deleted_count":    0,
			"failed_deletions": []string{},
		})
		return
	}

	// Delete all deployments from database first
	result, err := db.Exec("DELETE FROM deployments")
	if err != nil {
		http.Error(w, "Failed to delete deployments from database", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to get deletion count", http.StatusInternalServerError)
		return
	}

	// Delete all deployment directories from filesystem
	var failedDeletions []string

	for _, path := range pathsToDelete {
		if err := os.RemoveAll(path); err != nil {
			// Log error but continue with other deletions
			fmt.Printf("Warning: Failed to delete directory %s: %v\n", path, err)
			failedDeletions = append(failedDeletions, path)
		}
	}

	// Also try to remove the entire deployments directory if it's empty
	// This will fail silently if there are other files/directories
	os.Remove("deployments")

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":             "Bulk deletion completed",
		"deleted_count":       int(rowsAffected),
		"deleted_deployments": deployments,
		"failed_deletions":    failedDeletions,
	}

	if len(failedDeletions) > 0 {
		response["warning"] = "Some files could not be deleted from filesystem"
	}

	json.NewEncoder(w).Encode(response)
}

// Alternative: Delete all deployments and reset the entire system
func ResetSystemHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	// Get count before deletion
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM deployments").Scan(&count)
	if err != nil {
		http.Error(w, "Failed to count deployments", http.StatusInternalServerError)
		return
	}

	// Delete all from database
	_, err = db.Exec("DELETE FROM deployments")
	if err != nil {
		http.Error(w, "Failed to clear database", http.StatusInternalServerError)
		return
	}

	// Remove entire deployments directory
	err = os.RemoveAll("deployments")
	if err != nil {
		fmt.Printf("Warning: Failed to remove deployments directory: %v\n", err)
	}

	// Recreate empty deployments directory
	err = os.MkdirAll("deployments", 0755)
	if err != nil {
		http.Error(w, "Failed to recreate deployments directory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "System reset completed",
		"deleted_count": count,
		"status":        "All deployments and files removed",
	})
}
