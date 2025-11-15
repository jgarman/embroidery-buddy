# Web UI Module

This module provides a web interface for uploading files to the embroidery disk manager.

## Features

- Modern drag-and-drop file upload interface
- Progress indication during upload
- Responsive design that works on desktop and mobile
- RESTful API endpoints

## API Endpoints

### `GET /`
Serves the main upload page with a beautiful drag-and-drop interface.

### `POST /api/upload`
Handles file uploads.

**Request:**
- Content-Type: `multipart/form-data`
- Form field: `file` (the file to upload)

**Response (Success):**
```json
{
  "success": true,
  "filename": "example.dst",
  "size": 12345
}
```

**Response (Error):**
HTTP status code 4xx or 5xx with error message in response body.

### `GET /api/health`
Health check endpoint.

**Response:**
```json
{
  "status": "ok"
}
```

## Usage

The web UI is automatically integrated into the main server. Simply start the server and navigate to `http://localhost:8080/` in your browser.

```bash
# Create a test disk image
go run scripts/create-test-disk.go

# Start the server
DISK_PATH=/tmp/embroidery.img go run cmd/embroidery-usbd/main.go
```

Then open your browser to `http://localhost:8080/` and you'll see the upload interface.

## Implementation Details

- Files are uploaded directly to the root directory of the disk image
- Each upload is wrapped in a transaction, ensuring the USB gadget is disconnected during the write operation
- Maximum upload size is 100MB (configurable in handler.go)
- Filenames are sanitized to prevent path traversal attacks
- HTML templates are embedded in the Go binary using `//go:embed` for easy deployment

## Development

The HTML template is located in [templates/index.html](templates/index.html) for easy editing during development. The template is automatically embedded into the binary at build time using Go's embed feature, so no external files are needed for deployment.
