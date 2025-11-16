// +build !linux

package diskmanager

import "github.com/diskfs/go-diskfs/filesystem"

// newFilesystemWriterPlatform returns a DiskfsFilesystemWriter on non-Linux platforms
func newFilesystemWriterPlatform(diskPath string, fs filesystem.FileSystem) FilesystemWriter {
	return NewDiskfsFilesystemWriter(fs)
}