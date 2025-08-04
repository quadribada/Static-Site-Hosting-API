package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func ListDeploymentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, filename, timestamp, path FROM deployments ORDER BY timestamp DESC")
	if err != nil {
		http.Error(w, "Failed to fetch deployments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var deployments []Deployment
	for rows.Next() {
		var d Deployment
		err := rows.Scan(&d.ID, &d.Filename, &d.Timestamp, &d.Path)
		if err != nil {
			http.Error(w, "Failed to scan deployment", http.StatusInternalServerError)
			return
		}
		deployments = append(deployments, d)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deployments); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
