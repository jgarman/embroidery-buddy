// +build linux

package diskmanager

import "github.com/diskfs/go-diskfs/filesystem"

// newFilesystemWriterPlatform returns a LoopbackFilesystemWriter on Linux
func newFilesystemWriterPlatform(diskPath string, fs filesystem.FileSystem) FilesystemWriter {
	return NewLoopbackFilesystemWriter(diskPath)
}