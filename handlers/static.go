package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func StaticFileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Requested path:", r.URL.Path)

		// Remove leading slash and split
		path := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.SplitN(path, "/", 2)

		if len(parts) < 2 || parts[1] == "" {
			http.NotFound(w, r)
			return
		}

		siteID := parts[0]
		filePath := parts[1]

		// Construct and clean the full path
		fullPath := filepath.Join("deployments", siteID, filePath)

		// Security check: ensure we're not going outside deployments directory
		absDeployments, _ := filepath.Abs("deployments")
		absFullPath, _ := filepath.Abs(fullPath)
		if !strings.HasPrefix(absFullPath, absDeployments) {
			http.NotFound(w, r)
			return
		}

		// Check if file exists and is not a directory
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if info.IsDir() {
			http.NotFound(w, r)
			return
		}

		// Instead of ServeFile, read and serve manually to avoid 301 redirects
		file, err := os.Open(fullPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		// Set appropriate content type
		http.ServeContent(w, r, filepath.Base(fullPath), info.ModTime(), file)
	})
}
