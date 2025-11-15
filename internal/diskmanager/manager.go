package diskmanager

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
)

var (
	ErrDiskNotInitialized = errors.New("disk not initialized")
	ErrFileNotFound       = errors.New("file not found")
	ErrPathExists         = errors.New("path already exists")
	ErrInvalidPath        = errors.New("invalid path")
	ErrDiskFull           = errors.New("disk full")
	ErrOperationFailed    = errors.New("operation failed")
	ErrTransactionActive  = errors.New("transaction already active")
)

type Config struct {
	diskPath string

	// USB gadget information
	gadgetShortName string
	gadgetVendorId  int
	gadgetProductId int
	gadgetBcdDevice int
	gadgetBcdUsb    int

	gadgetProductName  string
	gadgetManufacturer string
}

type Manager struct {
	// external state
	// configuration
	config Config

	// internal state
	// sync so multiple threads won't step on each other
	mu sync.RWMutex

	// information for the virtual disk presented to the USB host
	disk       *disk.Disk
	filesystem filesystem.FileSystem

	// is the disk writable by the host?
	writable bool

	// USB gadget handler (injected dependency)
	gadget UsbGadget
}

func (m *Manager) openDisk() error {
	// Open disk in read-write mode
	disk, err := diskfs.Open(m.config.diskPath, diskfs.WithOpenMode(diskfs.ReadWriteExclusive))
	if err != nil {
		return fmt.Errorf("failed to open disk: %w", err)
	}

	fs, err := disk.GetFilesystem(0)
	if err != nil {
		return fmt.Errorf("failed to get filesystem: %w", err)
	}

	m.disk = disk
	m.filesystem = fs

	return nil
}

// New creates a new disk manager with the given configuration and USB gadget implementation.
// The caller is responsible for calling Close() when done to clean up resources.
//
// Example usage:
//
//	config := diskmanager.Config{...}
//	gadget := diskmanager.NewLinuxUsbGadget(config)
//	manager, err := diskmanager.New(config, gadget)
//	if err != nil {
//	    return err
//	}
//	defer manager.Close()
//
//	// Use manager...
func New(config Config, gadget UsbGadget) (*Manager, error) {
	m := &Manager{
		config:   config,
		writable: false,
		gadget:   gadget,
	}

	// Check if disk image exists
	_, err := os.Stat(m.config.diskPath)
	if err != nil {
		return nil, fmt.Errorf("disk image %s doesn't exist: %w", m.config.diskPath, err)
	}
	if err := m.openDisk(); err != nil {
		return nil, err
	}

	// Initialize the USB gadget
	if err = m.gadget.Initialize(); err != nil {
		// Clean up on error
		_ = m.Close()
		return nil, err
	}

	return m, nil
}

func CreateDiskImage(diskPath string, diskSizeMb int64) error {
	mydisk, err := diskfs.Create(diskPath, diskSizeMb*1024*1024,
		diskfs.SectorSizeDefault)

	if err != nil {
		return fmt.Errorf("failed to create disk: %w", err)
	}

	fmt.Println("Created disk")

	_, err = mydisk.CreateFilesystem(disk.FilesystemSpec{
		Partition:   0,
		FSType:      filesystem.TypeFat32,
		VolumeLabel: "EMBROIDERY",
	})
	if err != nil {
		return fmt.Errorf("failed to create filesystem: %w", err)
	}

	return nil
}

// normalizePath normalizes a file path
func normalizePath(p string) string {
	// Ensure path starts with /
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	// Clean the path
	p = path.Clean(p)

	return p
}

func (m *Manager) ensureDir(dirPath string) error {
	parts := strings.Split(dirPath, "/")
	currentPath := "/"

	for _, part := range parts {
		if part == "" {
			continue
		}

		currentPath = path.Join(currentPath, part)

		// Try to create directory - if it already exists, Mkdir will return an error
		// which we can safely ignore
		if err := m.filesystem.Mkdir(currentPath); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to create directory %s: %w", currentPath, err)
		}
	}

	return nil
}

// Transaction represents a batch of write operations that are performed
// with the USB gadget disconnected to prevent host access during modifications.
type Transaction struct {
	manager *Manager
}

// WriteFile writes a file to the disk within the transaction.
// The file path is normalized and parent directories are created automatically.
func (t *Transaction) WriteFile(filePath string, reader io.Reader, size int64) error {
	if t.manager.filesystem == nil {
		return ErrDiskNotInitialized
	}

	// Normalize path
	filePath = normalizePath(filePath)

	// Ensure parent directory exists
	dir := path.Dir(filePath)
	if dir != "/" && dir != "." {
		if err := t.manager.ensureDir(dir); err != nil {
			return fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	// Create file
	file, err := t.manager.filesystem.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data
	_, err = io.CopyN(file, reader, size)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// BeginTransaction starts a new transaction for batch write operations.
// The USB gadget is disconnected before the transaction function runs and
// reconnected after it completes (or panics).
//
// Example usage:
//
//	err := manager.BeginTransaction(func(tx *diskmanager.Transaction) error {
//	    if err := tx.WriteFile("/file1.txt", reader1, size1); err != nil {
//	        return err
//	    }
//	    if err := tx.WriteFile("/file2.txt", reader2, size2); err != nil {
//	        return err
//	    }
//	    return nil
//	})
func (m *Manager) BeginTransaction(fn func(*Transaction) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.filesystem == nil {
		return ErrDiskNotInitialized
	}

	// Disconnect the USB gadget before transaction
	if err := m.gadget.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect USB gadget: %w", err)
	}

	// Ensure we reconnect even if there's an error or panic
	defer func() {
		if err := m.gadget.Reconnect(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to reconnect USB gadget: %v\n", err)
		}
	}()

	// Create transaction and execute user function
	tx := &Transaction{manager: m}
	return fn(tx)
}

// ReadFile reads a file from the disk
func (m *Manager) ReadFile(filePath string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.filesystem == nil {
		return nil, ErrDiskNotInitialized
	}

	// Normalize path
	filePath = normalizePath(filePath)

	file, err := m.filesystem.OpenFile(filePath, os.O_RDONLY)
	if err != nil {
		// Check for file not found error (diskfs returns specific error messages)
		errStr := err.Error()
		if os.IsNotExist(err) || strings.Contains(errStr, "does not exist") {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Close cleans up resources held by the Manager
// It implements the io.Closer interface
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Destroy the USB gadget if it exists
	if m.gadget != nil {
		m.gadget.destroy()
	}

	// Note: diskfs doesn't require explicit close of the disk
	// but we clear the references to allow GC
	m.disk = nil
	m.filesystem = nil

	return nil
}
