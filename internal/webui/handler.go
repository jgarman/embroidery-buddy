package webui

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/jgarman/embroidery-buddy/internal/diskmanager"
)

// Handler manages HTTP requests for the web UI
type Handler struct {
	diskManager *diskmanager.Manager
	templates   *template.Template
}

// New creates a new web UI handler
func New(dm *diskmanager.Manager) (*Handler, error) {
	// Parse embedded templates
	tmpl, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Handler{
		diskManager: dm,
		templates:   tmpl,
	}, nil
}

// IndexHandler serves the main upload page
func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.templates.ExecuteTemplate(w, "index", nil); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// UploadHandler handles file uploads using streaming multipart reader
func (h *Handler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 200MB
	r.Body = http.MaxBytesReader(w, r.Body, 200*1024*1024)

	// Get the multipart reader for streaming
	reader, err := r.MultipartReader()
	if err != nil {
		log.Printf("Error creating multipart reader: %v", err)
		http.Error(w, "Invalid multipart request", http.StatusBadRequest)
		return
	}

	log.Printf("In multipart reader")

	// Process each part in the multipart form
	var filename string
	var fileSize int64

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading multipart part: %v", err)
			http.Error(w, "Error reading upload", http.StatusBadRequest)
			return
		}

		// Only process file parts
		if part.FormName() != "file" {
			part.Close()
			continue
		}

		// Get the filename
		filename = part.FileName()
		if filename == "" {
			part.Close()
			http.Error(w, "Empty filename", http.StatusBadRequest)
			return
		}

		// Sanitize the filename to prevent path traversal
		filename = filepath.Base(filename)

		// Store the file in the root directory of the disk
		filePath := "/" + filename

		log.Printf("Uploading file: %s", filename)

		// Write the file using a transaction with streaming
		// Use a large buffer (1MB) for better performance
		var byteCounter int64
		bufferedReader := bufio.NewReaderSize(part, 1024*1024)
		countingReader := &countingReader{reader: bufferedReader, count: &byteCounter}

		err = h.diskManager.BeginTransaction(func(tx *diskmanager.Transaction) error {
			// Pass a large size since we're streaming - the actual size will be determined by EOF
			return tx.WriteFile(filePath, countingReader, 100*1024*1024)
		})

		fileSize = byteCounter
		part.Close()

		if err != nil {
			log.Printf("Error writing file to disk: %v", err)
			http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
			return
		}

		// Break after processing the first file
		break
	}

	if filename == "" {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "filename": "%s", "size": %d}`, filename, fileSize)

	log.Printf("Successfully uploaded: %s (%d bytes)", filename, fileSize)
}

// countingReader wraps an io.Reader and counts bytes read
type countingReader struct {
	reader io.Reader
	count  *int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.reader.Read(p)
	*cr.count += int64(n)
	return n, err
}

// HealthHandler provides a health check endpoint
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `{"status": "ok"}`)
}

// ClearFilesHandler clears all files from the disk by recreating the filesystem
func (h *Handler) ClearFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Clearing all files from disk")

	// Clear all files by recreating the filesystem
	if err := h.diskManager.ClearFiles(); err != nil {
		log.Printf("Failed to clear files: %v", err)
		http.Error(w, fmt.Sprintf("Failed to clear files: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully cleared all files from disk")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `{"success": true}`)
}
