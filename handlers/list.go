package handlers

import (
	"encoding/json"
	"net/http"
)

func ListDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Handle potential JSON encoding errors
	if err := json.NewEncoder(w).Encode(deployments); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
