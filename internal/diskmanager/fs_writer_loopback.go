package diskmanager

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// LoopbackFilesystemWriter uses loopback mounting for fast filesystem writes
type LoopbackFilesystemWriter struct {
	diskPath string
	mountDir string
}

// NewLoopbackFilesystemWriter creates a new loopback-based filesystem writer
func NewLoopbackFilesystemWriter(diskPath string) *LoopbackFilesystemWriter {
	return &LoopbackFilesystemWriter{
		diskPath: diskPath,
	}
}

// Begin mounts the disk image to a temporary directory using loopback mount
func (w *LoopbackFilesystemWriter) Begin() error {
	// Create temporary mount directory
	mountDir, err := os.MkdirTemp("", "bernina-mount-*")
	if err != nil {
		return fmt.Errorf("failed to create temp mount directory: %w", err)
	}

	// Mount the disk image loopback
	cmd := exec.Command("mount", "-o", "loop", w.diskPath, mountDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(mountDir)
		return fmt.Errorf("failed to mount loopback: %w (output: %s)", err, string(output))
	}

	w.mountDir = mountDir
	return nil
}

// WriteFile writes a file to the mounted filesystem using standard OS operations
func (w *LoopbackFilesystemWriter) WriteFile(filePath string, reader io.Reader, size int64) error {
	if w.mountDir == "" {
		return fmt.Errorf("filesystem not mounted")
	}

	// Normalize path
	filePath = normalizePath(filePath)

	// Convert to absolute path on mounted filesystem
	absPath := filepath.Join(w.mountDir, filePath)

	// Ensure parent directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Check if this is a disk full error
		if isOutOfSpaceError(err) {
			return ErrDiskFull
		}
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		// Check if this is a disk full error
		if isOutOfSpaceError(err) {
			return ErrDiskFull
		}
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Use a large buffered writer (1MB) for better performance
	bufferedWriter := bufio.NewWriterSize(file, 1024*1024)
	defer bufferedWriter.Flush()

	// Copy data - use io.Copy instead of io.CopyN to handle all data
	// The size parameter is provided for information but we'll copy everything
	_, err = io.Copy(bufferedWriter, reader)
	if err != nil {
		// Check if this is a disk full error
		if isOutOfSpaceError(err) {
			return ErrDiskFull
		}
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Ensure all data is flushed to disk
	if err := bufferedWriter.Flush(); err != nil {
		// Check if this is a disk full error
		if isOutOfSpaceError(err) {
			return ErrDiskFull
		}
		return fmt.Errorf("failed to flush file: %w", err)
	}

	return nil
}

// isOutOfSpaceError checks if an error is a "no space left on device" error
func isOutOfSpaceError(err error) bool {
	if err == nil {
		return false
	}
	// Check for ENOSPC errno
	return err.Error() == "no space left on device"
}

// End unmounts the loopback mount and removes the temporary directory
func (w *LoopbackFilesystemWriter) End() error {
	if w.mountDir == "" {
		return nil // Already unmounted or never mounted
	}

	// Unmount
	cmd := exec.Command("umount", w.mountDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unmount: %w (output: %s)", err, string(output))
	}

	// Remove temporary directory
	if err := os.RemoveAll(w.mountDir); err != nil {
		return fmt.Errorf("failed to remove temp directory: %w", err)
	}

	w.mountDir = ""
	return nil
}
