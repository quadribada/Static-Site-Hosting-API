package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"static-site-hosting/handlers"
	"static-site-hosting/middleware"
)

func main() {
	// Ensure necessary directories exist
	if err := os.MkdirAll("deployments", 0755); err != nil {
		log.Fatalf("Error creating deployments directory: %v", err)
	}

	if err := os.MkdirAll("db", 0755); err != nil {
		log.Fatalf("Error creating db directory: %v", err)
	}

	// Setup and connect to the database
	db, err := setupDatabase()
	if err != nil {
		log.Fatalf("Database setup failed: %v", err)
	}
	defer db.Close()

	// Setup HTTP routes
	mux := setupRoutes()

	// Apply middleware
	wrappedMux := middleware.LoggingMiddleware(mux)

	log.Println("Server starting on :8080")
	log.Println("Endpoints available:")
	log.Println("  POST /upload - Upload a zip file")
	log.Println("  GET /deployments - List all deployments")
	log.Println("  GET /{site-id}/{file-path} - Serve static files")
	log.Println("  GET /hello-world - Test endpoint")

	log.Fatal(http.ListenAndServe(":8080", wrappedMux))
}

func setupDatabase() (*sql.DB, error) {
	// Remove existing database for fresh start (development only!)
	// TODO: Remove this in production
	err := os.Remove("./db/database.db")
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	db, err := sql.Open("sqlite3", "./db/database.db")
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// Create tables
	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	// Create deployments table for persistent storage
	// (Currently you're using in-memory slice, but this prepares for DB storage)
	createDeploymentsTable := `
	CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(createDeploymentsTable); err != nil {
		return err
	}

	// Keep the example table for now
	createExampleTable := `
	CREATE TABLE IF NOT EXISTS example (
		id INTEGER PRIMARY KEY, 
		name TEXT
	)`

	if _, err := db.Exec(createExampleTable); err != nil {
		return err
	}

	return nil
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/upload", handlers.UploadHandler)
	mux.HandleFunc("/deployments", handlers.ListDeploymentsHandler)
	mux.HandleFunc("/hello-world", handlers.HelloWorldHandler)

	// Static file serving - this should be last since it's a catch-all
	mux.Handle("/", handlers.StaticFileHandler())

	return mux
}
