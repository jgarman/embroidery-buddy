package diskmanager

import "io"

// FilesystemWriter is an interface for writing files to the disk image.
// Different implementations can use different methods (loopback mount, go-diskfs, etc.)
type FilesystemWriter interface {
	// Begin prepares the filesystem for writing (e.g., mounting)
	Begin() error

	// WriteFile writes a file to the filesystem at the given path
	WriteFile(filePath string, reader io.Reader, size int64) error

	// End finalizes the filesystem writes (e.g., unmounting)
	End() error
}
