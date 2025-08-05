# Static Site Hosting API Task

The goal of this assignment is to build a static site hosting platform backend that can manage, deploy, and serve these static websites. The server should support uploading a zip file with static assets (HTML, CSS, and JS) and serve them. The provided API should support tracking deployments and add metadata about the site, but you are encouraged to explore other areas such as preview URLs, triggering rollbacks, etc…

## Getting Started

To start the project, you need to have Go installed on your machine. You can download it from [the official Go website](https://golang.org/dl/).

1. Prerequisites:

    - Go 1.24.4 or higher

2. Install dependencies:

  ```bash
  go mod tidy
  ```

3. Start the development server:

  ```bash
  go run ./cmd/main.go
  ```

4. Navigate to `http://localhost:8080/hello-world` to see an starter API in action.

5. To run the tests:

  ```bash
  go test ./...
  ```

## Core Features

### File Upload & Deployment
- **Zip Upload**: Upload static sites as zip files via `POST /upload`
- **Automatic Extraction**: Extracts and deploys files to unique deployment directories
- **UUID Generation**: Each deployment gets a unique identifier for isolated hosting
- **File Validation**: Handles various static file types (HTML, CSS, JS, images, etc.)

### Static File Serving
- **Dynamic Routing**: Serves files at `/{deployment-id}/{file-path}`
- **Content Type Detection**: Automatically sets appropriate MIME types
- **Directory Structure Preservation**: Maintains original folder hierarchy from zip
- **404 Handling**: Proper error responses for missing files/deployments

### Deployment Management
- **List Deployments**: `GET /deployments` returns all deployments with metadata
- **Deployment History**: Persistent storage with timestamps and original filenames
- **Delete Deployments**: `DELETE /deployments/{id}` removes both database records and files
- **Delete All Deployments**: `DELETE /deployments` removes all deployments and files
- **Rollback Support**: `POST /rollback/{id}` creates new deployment from previous version
- **System Reset**: `POST /reset` completely clears all deployments (nuclear option)
- **Atomic Operations**: Database and filesystem stay in sync

### Data Persistence
- **SQLite Database**: Lightweight, file-based database for deployment metadata
- **Crash Recovery**: Deployments survive server restarts
- **Data Integrity**: Transactional operations ensure consistency

## Security & Reliability

### File Security
- **Path Traversal Protection**: Prevents `../` attacks in zip files
- **Sandboxed Deployments**: Each site isolated in its own directory
- **Filename Validation**: Rejects malicious file paths

### Error Handling
- **Graceful Failures**: Comprehensive error responses with appropriate HTTP status codes
- **Cleanup on Failure**: Failed uploads don't leave orphaned files
- **Input Validation**: Validates file uploads and request parameters

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/upload` | Upload a zip file containing static site |
| `GET` | `/deployments` | List all deployments with metadata |
| `DELETE` | `/deployments/{id}` | Delete a specific deployment |
| `DELETE` | `/deployments` | Delete ALL deployments and files |
| `POST` | `/rollback/{id}` | Create new deployment from previous version |
| `POST` | `/reset` | Reset entire system (nuclear option) |
| `GET` | `/{deployment-id}/{file-path}` | Serve static files |
| `GET` | `/hello-world` | Health check endpoint |

## Example Usage

```bash
# Upload a site
curl -X POST -F "file=@my-site.zip" http://localhost:8080/upload
# Returns: {"id":"abc123...","filename":"my-site.zip",timestamp, path...}

# List all deployments
curl http://localhost:8080/deployments

# Access your site (URL depends on your zip structure)
# For flat zip: my-site.zip/index.html
curl http://localhost:8080/abc123.../index.html

# For nested zip: my-site.zip/my-site/index.html  
curl http://localhost:8080/abc123.../my-site/index.html

# Rollback to a previous deployment
curl -X POST http://localhost:8080/rollback/abc123...
# Creates new deployment with same files as abc123...

# Delete a specific deployment
curl -X DELETE http://localhost:8080/deployments/abc123...

# Delete ALL deployments
curl -X DELETE http://localhost:8080/deployments

# Reset entire system (nuclear option)
curl -X POST http://localhost:8080/reset
```

## File Structure Note
Uploaded zip files preserve their internal directory structure. 

Example:
- Upload: `my-site.zip` containing `my-site/index.html`
- Access: `GET /{deployment-id}/my-site/index.html`

Breakdown:
- http://localhost:8080/ - Your server
- e99b140e-3866-42e9-be10-49640ed4bf2f/ - Your deployment ID
- file-name/ - The folder inside your zip file
- index.html - The file you want to access

## Run these E2E tests

  ```bash
    # Run all E2E tests
    go test ./cmd -v

    # Run just the main workflow
    go test ./cmd -v -run TestE2EStaticSiteHostingWorkflow

    # Run without performance tests (faster)
    go test ./cmd -v -short
  ```
## Technical Implementation

- **Language**: Go 1.24+
- **Database**: SQLite3 with persistent file storage
- **HTTP Router**: Go's built-in `net/http` multiplexer
- **File Handling**: Archive/zip package for extraction
- **Middleware**: Custom logging middleware for request tracking
- **Testing**: Comprehensive unit and E2E test coverage
