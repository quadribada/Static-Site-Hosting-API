package handlers

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Deployment struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
}

var deployments []Deployment

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(20 << 20) // 20 MB max
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	siteID := uuid.New().String()
	tempZip := fmt.Sprintf("temp-%s.zip", siteID) // Use unique temp file name
	dst, err := os.Create(tempZip)
	if err != nil {
		http.Error(w, "Could not create temp file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	defer os.Remove(tempZip) // Ensure cleanup even if unzip fails

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	dst.Close() // Close before reading

	destDir := filepath.Join("deployments", siteID)
	if err := unzip(tempZip, destDir); err != nil {
		http.Error(w, "Failed to unzip", http.StatusInternalServerError)
		return
	}

	d := Deployment{
		ID:        siteID,
		Timestamp: time.Now(),
		Path:      destDir,
	}
	deployments = append(deployments, d)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(dest, 0755)

	for _, f := range r.File {
		// Prevent path traversal attacks
		if strings.Contains(f.Name, "..") {
			continue // Skip files with .. in path
		}

		fPath := filepath.Join(dest, f.Name)

		// Ensure the file path is within dest directory
		if !strings.HasPrefix(fPath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fPath, f.Mode())
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(fPath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// ResetDeployments clears the deployments slice for testing
func ResetDeployments() {
	deployments = []Deployment{}
}
