package diskmanager

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/diskfs/go-diskfs/filesystem"
)

// DiskfsFilesystemWriter uses go-diskfs for filesystem writes (fallback for non-Linux systems)
type DiskfsFilesystemWriter struct {
	filesystem filesystem.FileSystem
}

// NewDiskfsFilesystemWriter creates a new go-diskfs based filesystem writer
func NewDiskfsFilesystemWriter(fs filesystem.FileSystem) *DiskfsFilesystemWriter {
	return &DiskfsFilesystemWriter{
		filesystem: fs,
	}
}

// Begin prepares the filesystem for writing (no-op for diskfs)
func (w *DiskfsFilesystemWriter) Begin() error {
	if w.filesystem == nil {
		return ErrDiskNotInitialized
	}
	return nil
}

// ensureDir creates all parent directories for a path
func (w *DiskfsFilesystemWriter) ensureDir(dirPath string) error {
	parts := splitPath(dirPath)
	currentPath := "/"

	for _, part := range parts {
		if part == "" {
			continue
		}

		currentPath = path.Join(currentPath, part)

		// Try to create directory - if it already exists, Mkdir will return an error
		// which we can safely ignore
		if err := w.filesystem.Mkdir(currentPath); err != nil && !os.IsExist(err) {
			// Check if this is a disk full error
			if isOutOfSpaceError(err) {
				return ErrDiskFull
			}
			return fmt.Errorf("failed to create directory %s: %w", currentPath, err)
		}
	}

	return nil
}

// splitPath splits a path into its components
func splitPath(p string) []string {
	var parts []string
	for {
		dir, file := path.Split(p)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		if dir == "" || dir == "/" {
			break
		}
		p = path.Clean(dir)
	}
	return parts
}

// WriteFile writes a file to the filesystem using go-diskfs
func (w *DiskfsFilesystemWriter) WriteFile(filePath string, reader io.Reader, size int64) error {
	if w.filesystem == nil {
		return ErrDiskNotInitialized
	}

	// Normalize path
	filePath = normalizePath(filePath)

	// Ensure parent directory exists
	dir := path.Dir(filePath)
	if dir != "/" && dir != "." {
		if err := w.ensureDir(dir); err != nil {
			return err // Already formatted with proper error type
		}
	}

	// Create file
	file, err := w.filesystem.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		// Check if this is a disk full error
		if isOutOfSpaceError(err) {
			return ErrDiskFull
		}
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data
	_, err = io.CopyN(file, reader, size)
	if err != nil && err != io.EOF {
		// Check if this is a disk full error
		if isOutOfSpaceError(err) {
			return ErrDiskFull
		}
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// End finalizes the filesystem writes (no-op for diskfs)
func (w *DiskfsFilesystemWriter) End() error {
	return nil
}