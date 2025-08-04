# Static Site Hosting API Task

Static site hosting is a way to serve websites made entirely of pre-built files—typically HTML, CSS, and JavaScript—directly to users’ browsers, without any server-side processing or database queries. This approach is ideal for sites that don’t require dynamic content generation on the server, such as portfolios, documentation, landing pages, or marketing sites. Because there’s no backend logic or databases involved, static site hosting platforms focus on efficiently storing, deploying, and serving these static assets, often leveraging features like automated deployments, version tracking, and preview URLs to streamline the workflow for developers and content creators.

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
| `GET` | `/{deployment-id}/{file-path}` | Serve static files |
| `GET` | `/hello-world` | Health check endpoint |

## Example Usage

```bash
# Upload a site
curl -X POST -F "file=@my-site.zip" http://localhost:8080/upload

# List all deployments
curl http://localhost:8080/deployments

# Access your site
curl http://localhost:8080/{deployment-id}/index.html

# Delete a deployment
curl -X DELETE http://localhost:8080/deployments/{deployment-id}
```

## Run these E2E tests

  ```bash
    # Run all E2E tests
    go test ./cmd -v

    # Run just the main workflow
    go test ./cmd -v -run TestE2EStaticSiteHostingWorkflow

    # Run without performance tests (faster)
    go test ./cmd -v -short
  ```

## File Structure
Uploaded zip files preserve their internal directory structure. 

Example:
- Upload: `my-site.zip` containing `my-site/index.html`
- Access: `GET /{site-id}/my-site/index.html`

## Technical Implementation

- **Language**: Go 1.24+
- **Database**: SQLite3 with persistent file storage
- **HTTP Router**: Go's built-in `net/http` multiplexer
- **File Handling**: Archive/zip package for extraction
- **Middleware**: Custom logging middleware for request tracking
- **Testing**: Comprehensive unit and E2E test coverage
