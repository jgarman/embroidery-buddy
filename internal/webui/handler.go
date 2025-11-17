package webui

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

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
	// Process each part in the multipart form
	var filename string
	var fileSize int64
	var filesExtracted int
	var err error
	var part *multipart.Part
	var reader *multipart.Reader

	reader, err = r.MultipartReader()
	if err != nil {
		log.Printf("Error creating multipart reader: %v", err)
		http.Error(w, "Invalid multipart request", http.StatusBadRequest)
		return
	}

	log.Printf("In multipart reader")

	for {
		part, err = reader.NextPart()
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

		// Check if this is a zip file

		if isZipFile(filename) {
			log.Printf("Detected zip file, extracting contents...")

			// Use a large buffer (1MB) for better performance
			bufferedReader := bufio.NewReaderSize(part, 1024*1024)

			// Extract zip contents
			filesExtracted, fileSize, err = h.extractZipStream(bufferedReader, 200*1024*1024)
			part.Close()

			if err != nil {
				log.Printf("Error extracting zip file: %v", err)

				// Check for specific error types and provide user-friendly messages
				var statusCode int
				var errorMessage string

				if errors.Is(err, diskmanager.ErrDiskFull) {
					statusCode = http.StatusInsufficientStorage
					errorMessage = "Disk is full. Please clear some files and try again."
				} else if errors.Is(err, diskmanager.ErrDiskNotInitialized) {
					statusCode = http.StatusInternalServerError
					errorMessage = "Disk not initialized. Please contact support."
				} else {
					statusCode = http.StatusInternalServerError
					errorMessage = fmt.Sprintf("Failed to extract zip file: %v", err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				fmt.Fprintf(w, `{"success": false, "error": "%s"}`, errorMessage)
				return
			}

			log.Printf("Successfully extracted %d files from %s", filesExtracted, filename)
		} else {
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

				// Check for specific error types and provide user-friendly messages
				var statusCode int
				var errorMessage string

				if errors.Is(err, diskmanager.ErrDiskFull) {
					statusCode = http.StatusInsufficientStorage
					errorMessage = "Disk is full. Please clear some files and try again."
				} else if errors.Is(err, diskmanager.ErrDiskNotInitialized) {
					statusCode = http.StatusInternalServerError
					errorMessage = "Disk not initialized. Please contact support."
				} else {
					statusCode = http.StatusInternalServerError
					errorMessage = fmt.Sprintf("Failed to save file: %v", err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				fmt.Fprintf(w, `{"success": false, "error": "%s"}`, errorMessage)
				return
			}
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
	if filesExtracted > 0 {
		fmt.Fprintf(w, `{"success": true, "filename": "%s", "size": %d, "filesExtracted": %d}`, filename, fileSize, filesExtracted)
		log.Printf("Successfully extracted %d files from %s (%d bytes total)", filesExtracted, filename, fileSize)
	} else {
		fmt.Fprintf(w, `{"success": true, "filename": "%s", "size": %d}`, filename, fileSize)
		log.Printf("Successfully uploaded: %s (%d bytes)", filename, fileSize)
	}
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

// isZipFile checks if a filename has a .zip extension
func isZipFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".zip")
}

// extractZipStream extracts a zip file from a reader and writes all files to the disk
// Returns the number of files extracted and any error
func (h *Handler) extractZipStream(reader io.Reader, maxSize int64) (int, int64, error) {
	// Since zip files need random access to read the central directory,
	// we need to buffer the entire file in memory
	// For very large files, this could be memory-intensive
	buf := &bytes.Buffer{}
	written, err := io.CopyN(buf, reader, maxSize)
	if err != nil && err != io.EOF {
		return 0, written, fmt.Errorf("failed to buffer zip file: %w", err)
	}

	// Create a zip reader from the buffered data
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return 0, written, fmt.Errorf("failed to open zip file: %w", err)
	}

	filesExtracted := 0
	totalSize := int64(0)

	// Extract all files in a single transaction
	err = h.diskManager.BeginTransaction(func(tx *diskmanager.Transaction) error {
		for _, zipFile := range zipReader.File {
			// Skip directories
			if zipFile.FileInfo().IsDir() {
				continue
			}

			// Sanitize the file path to prevent directory traversal
			cleanPath := filepath.Clean("/" + zipFile.Name)
			if strings.Contains(cleanPath, "..") {
				log.Printf("Skipping potentially malicious path in zip: %s", zipFile.Name)
				continue
			}

			// Open the file in the zip
			rc, err := zipFile.Open()
			if err != nil {
				return fmt.Errorf("failed to open file %s in zip: %w", zipFile.Name, err)
			}

			// Write the file to disk
			if err := tx.WriteFile(cleanPath, rc, int64(zipFile.UncompressedSize64)); err != nil {
				rc.Close()
				return fmt.Errorf("failed to write file %s: %w", zipFile.Name, err)
			}

			rc.Close()
			filesExtracted++
			totalSize += int64(zipFile.UncompressedSize64)
			log.Printf("Extracted: %s (%d bytes)", zipFile.Name, zipFile.UncompressedSize64)
		}

		return nil
	})

	if err != nil {
		return filesExtracted, totalSize, err
	}

	return filesExtracted, totalSize, nil
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
