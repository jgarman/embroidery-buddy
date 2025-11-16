package diskmanager

import "github.com/diskfs/go-diskfs/filesystem"

// NewFilesystemWriter creates the appropriate FilesystemWriter based on the platform.
// On Linux, it returns a LoopbackFilesystemWriter for fast writes using kernel operations.
// On other platforms, it returns a DiskfsFilesystemWriter as a fallback.
func NewFilesystemWriter(diskPath string, fs filesystem.FileSystem) FilesystemWriter {
	return newFilesystemWriterPlatform(diskPath, fs)
}