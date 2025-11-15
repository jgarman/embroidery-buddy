package webui

import (
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

// UploadHandler handles file uploads
func (h *Handler) UploadHandler(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 100MB
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		http.Error(w, "File too large or invalid request", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Error retrieving file: %v", err)
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get the filename
	filename := header.Filename
	if filename == "" {
		http.Error(w, "Empty filename", http.StatusBadRequest)
		return
	}

	// Sanitize the filename to prevent path traversal
	filename = filepath.Base(filename)

	// Store the file in the root directory of the disk
	filePath := "/" + filename

	log.Printf("Uploading file: %s (size: %d bytes)", filename, header.Size)

	// Write the file using a transaction
	err = h.diskManager.BeginTransaction(func(tx *diskmanager.Transaction) error {
		return tx.WriteFile(filePath, file, header.Size)
	})

	if err != nil {
		log.Printf("Error writing file to disk: %v", err)
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "filename": "%s", "size": %d}`, filename, header.Size)

	log.Printf("Successfully uploaded: %s", filename)
}

// HealthHandler provides a health check endpoint
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `{"status": "ok"}`)
}
