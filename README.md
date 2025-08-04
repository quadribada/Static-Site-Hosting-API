# Static Site Hosting API

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

Provided in the `/examples` directory are a few example static sites that you can use to test the API you create. There are also a few extra folders (`/models` and `/services`) that you can use as a reference for how to structure your code but feel free to modify or ignore them as you see fit.

## Run these E2E tests

  ```bash
    # Run all E2E tests
    go test ./cmd -v

    # Run just the main workflow test
    go test ./cmd -v -run TestE2EStaticSiteHostingWorkflow

    # Run without performance tests (faster)
    go test ./cmd -v -short

    # Test with race condition detection
    go test ./cmd -v -race
  ```

## File Structure
Uploaded zip files preserve their internal directory structure. 

Example:
- Upload: `my-site.zip` containing `my-site/index.html`
- Access: `GET /{site-id}/my-site/index.html`