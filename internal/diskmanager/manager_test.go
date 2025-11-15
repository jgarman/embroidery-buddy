package diskmanager

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestCreateDiskImage tests the creation of a disk image
func TestCreateDiskImage(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a 10MB disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Verify the file exists
	info, err := os.Stat(diskPath)
	if err != nil {
		t.Fatalf("Disk image file doesn't exist: %v", err)
	}

	// Verify file size is approximately correct (10MB = 10*1024*1024 bytes)
	expectedSize := int64(10 * 1024 * 1024)
	if info.Size() != expectedSize {
		t.Errorf("Expected disk size %d, got %d", expectedSize, info.Size())
	}
}

// TestNewManager tests creating a new manager with NoOp gadget
func TestNewManager(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image first
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget (no USB gadget setup)
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Verify manager was created
	if manager == nil {
		t.Fatal("Manager is nil")
	}

	if manager.disk == nil {
		t.Error("Manager disk is nil")
	}

	if manager.filesystem == nil {
		t.Error("Manager filesystem is nil")
	}
}

// TestWriteFile tests writing a file to the disk
func TestWriteFile(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test writing a simple file using a transaction
	content := []byte("Hello, World!")

	err = manager.BeginTransaction(func(tx *Transaction) error {
		reader := bytes.NewReader(content)
		return tx.WriteFile("test.txt", reader, int64(len(content)))
	})
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Verify by reading the file back
	readFile, err := manager.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	defer readFile.Close()

	readContent, err := io.ReadAll(readFile)
	if err != nil {
		t.Fatalf("Failed to read file contents: %v", err)
	}

	if !bytes.Equal(content, readContent) {
		t.Errorf("Content mismatch: expected %q, got %q", content, readContent)
	}
}

// TestWriteFileWithSubdirectory tests writing a file to a subdirectory
func TestWriteFileWithSubdirectory(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test writing a file in a subdirectory using a transaction
	content := []byte("File in subdirectory")

	err = manager.BeginTransaction(func(tx *Transaction) error {
		reader := bytes.NewReader(content)
		return tx.WriteFile("/subdir/nested/file.txt", reader, int64(len(content)))
	})
	if err != nil {
		t.Fatalf("Failed to write file in subdirectory: %v", err)
	}

	// Verify by reading the file back
	readFile, err := manager.ReadFile("/subdir/nested/file.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	defer readFile.Close()

	readContent, err := io.ReadAll(readFile)
	if err != nil {
		t.Fatalf("Failed to read file contents: %v", err)
	}

	if !bytes.Equal(content, readContent) {
		t.Errorf("Content mismatch: expected %q, got %q", content, readContent)
	}
}

// TestWriteMultipleFiles tests writing multiple files and verifies that
// a single transaction only calls Disconnect/Reconnect once
func TestWriteMultipleFiles(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Reset counts after initialization (Initialize calls Reconnect once)
	gadget.ResetCounts()

	// Write multiple files in a single transaction
	files := map[string]string{
		"/file1.txt":       "Content of file 1",
		"/file2.txt":       "Content of file 2",
		"/data/config.txt": "Configuration data",
		"/data/readme.md":  "# README\n\nThis is a test",
		"/images/test.dat": "Binary data here",
	}

	// Write all files in one transaction (single disconnect/reconnect cycle)
	err = manager.BeginTransaction(func(tx *Transaction) error {
		for path, content := range files {
			reader := bytes.NewReader([]byte(content))
			if err := tx.WriteFile(path, reader, int64(len(content))); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to write files in transaction: %v", err)
	}

	// Verify that Disconnect and Reconnect were each called exactly once
	// despite writing 5 files
	if gadget.GetDisconnectCalls() != 1 {
		t.Errorf("Expected Disconnect to be called 1 time for %d files, got %d", len(files), gadget.GetDisconnectCalls())
	}
	if gadget.GetReconnectCalls() != 1 {
		t.Errorf("Expected Reconnect to be called 1 time for %d files, got %d", len(files), gadget.GetReconnectCalls())
	}

	// Verify all files by reading them back
	for path, expectedContent := range files {
		readFile, err := manager.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", path, err)
			continue
		}

		readContent, err := io.ReadAll(readFile)
		readFile.Close()
		if err != nil {
			t.Errorf("Failed to read contents of %s: %v", path, err)
			continue
		}

		if string(readContent) != expectedContent {
			t.Errorf("Content mismatch for %s: expected %q, got %q", path, expectedContent, string(readContent))
		}
	}
}

// TestNormalizePath tests the path normalization function
func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test.txt", "/test.txt"},
		{"/test.txt", "/test.txt"},
		{"//test.txt", "/test.txt"},
		{"./test.txt", "/test.txt"},
		{"/dir/./file.txt", "/dir/file.txt"},
		{"/dir/../file.txt", "/file.txt"},
		{"dir/subdir/file.txt", "/dir/subdir/file.txt"},
	}

	for _, tt := range tests {
		result := normalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePath(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

// TestWriteFileWithoutDiskInitialization tests error handling when disk is not initialized
func TestWriteFileWithoutDiskInitialization(t *testing.T) {
	// Create a manager without properly initializing the filesystem
	manager := &Manager{}

	content := []byte("test")

	err := manager.BeginTransaction(func(tx *Transaction) error {
		reader := bytes.NewReader(content)
		return tx.WriteFile("/test.txt", reader, int64(len(content)))
	})
	if err != ErrDiskNotInitialized {
		t.Errorf("Expected ErrDiskNotInitialized, got %v", err)
	}
}

// TestReadFile tests reading a file from the disk
func TestReadFile(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Write a test file using a transaction
	expectedContent := []byte("This is test content for reading!")
	err = manager.BeginTransaction(func(tx *Transaction) error {
		reader := bytes.NewReader(expectedContent)
		return tx.WriteFile("/testread.txt", reader, int64(len(expectedContent)))
	})
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Read the file back
	readFile, err := manager.ReadFile("/testread.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	defer readFile.Close()

	actualContent, err := io.ReadAll(readFile)
	if err != nil {
		t.Fatalf("Failed to read file contents: %v", err)
	}

	if !bytes.Equal(expectedContent, actualContent) {
		t.Errorf("Content mismatch: expected %q, got %q", expectedContent, actualContent)
	}
}

// TestReadNonExistentFile tests error handling when reading a file that doesn't exist
func TestReadNonExistentFile(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Try to read a file that doesn't exist
	_, err = manager.ReadFile("/nonexistent.txt")
	if err != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %v", err)
	}
}

// TestReadFileWithoutDiskInitialization tests error handling when reading without initialization
func TestReadFileWithoutDiskInitialization(t *testing.T) {
	// Create a manager without properly initializing the filesystem
	manager := &Manager{}

	_, err := manager.ReadFile("/test.txt")
	if err != ErrDiskNotInitialized {
		t.Errorf("Expected ErrDiskNotInitialized, got %v", err)
	}
}

// TestReadWriteRoundTrip tests writing and reading various content types
func TestReadWriteRoundTrip(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test various content types
	testCases := []struct {
		path    string
		content []byte
	}{
		{"/empty.txt", []byte("")},
		{"/ascii.txt", []byte("Simple ASCII text")},
		{"/utf8.txt", []byte("UTF-8 content: 你好世界 🌍")},
		{"/binary.dat", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
		{"/multiline.txt", []byte("Line 1\nLine 2\r\nLine 3\n")},
		{"/large.txt", bytes.Repeat([]byte("A"), 1024)}, // 1KB of 'A's
	}

	for _, tc := range testCases {
		// Write the file using a transaction
		err = manager.BeginTransaction(func(tx *Transaction) error {
			reader := bytes.NewReader(tc.content)
			return tx.WriteFile(tc.path, reader, int64(len(tc.content)))
		})
		if err != nil {
			t.Errorf("Failed to write file %s: %v", tc.path, err)
			continue
		}

		// Read it back
		readFile, err := manager.ReadFile(tc.path)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", tc.path, err)
			continue
		}

		readContent, err := io.ReadAll(readFile)
		readFile.Close()
		if err != nil {
			t.Errorf("Failed to read contents of %s: %v", tc.path, err)
			continue
		}

		// Verify content matches
		if !bytes.Equal(tc.content, readContent) {
			t.Errorf("Content mismatch for %s: expected %d bytes, got %d bytes",
				tc.path, len(tc.content), len(readContent))
			if len(tc.content) < 50 && len(readContent) < 50 {
				t.Errorf("  Expected: %q", tc.content)
				t.Errorf("  Got:      %q", readContent)
			}
		}
	}
}

// TestUsbGadgetDisconnectReconnect tests disconnecting and reconnecting the USB gadget
func TestUsbGadgetDisconnectReconnect(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Check initial connection status
	if !manager.gadget.IsConnected() {
		t.Error("Expected gadget to be connected after initialization")
	}

	// Disconnect the gadget
	err = manager.gadget.Disconnect()
	if err != nil {
		t.Errorf("Failed to disconnect gadget: %v", err)
	}

	// Verify disconnected
	if manager.gadget.IsConnected() {
		t.Error("Expected gadget to be disconnected")
	}

	// Reconnect the gadget
	err = manager.gadget.Reconnect()
	if err != nil {
		t.Errorf("Failed to reconnect gadget: %v", err)
	}

	// Verify reconnected
	if !manager.gadget.IsConnected() {
		t.Error("Expected gadget to be reconnected")
	}

	// Test multiple disconnect/reconnect cycles
	for i := 0; i < 3; i++ {
		if err := manager.gadget.Disconnect(); err != nil {
			t.Errorf("Cycle %d: Failed to disconnect: %v", i, err)
		}
		if manager.gadget.IsConnected() {
			t.Errorf("Cycle %d: Expected disconnected state", i)
		}

		if err := manager.gadget.Reconnect(); err != nil {
			t.Errorf("Cycle %d: Failed to reconnect: %v", i, err)
		}
		if !manager.gadget.IsConnected() {
			t.Errorf("Cycle %d: Expected connected state", i)
		}
	}
}

// TestUsbGadgetIdempotentOperations tests that disconnect/reconnect are idempotent
func TestUsbGadgetIdempotentOperations(t *testing.T) {
	gadget := NewNoOpUsbGadget()

	// Initialize
	err := gadget.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Multiple disconnects should be safe
	for i := 0; i < 3; i++ {
		err = gadget.Disconnect()
		if err != nil {
			t.Errorf("Disconnect %d failed: %v", i, err)
		}
		if gadget.IsConnected() {
			t.Errorf("After disconnect %d, expected disconnected state", i)
		}
	}

	// Multiple reconnects should be safe
	for i := 0; i < 3; i++ {
		err = gadget.Reconnect()
		if err != nil {
			t.Errorf("Reconnect %d failed: %v", i, err)
		}
		if !gadget.IsConnected() {
			t.Errorf("After reconnect %d, expected connected state", i)
		}
	}
}

// TestManagerCloseDisconnects tests that Close properly disconnects the gadget
func TestManagerCloseDisconnects(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if !gadget.IsConnected() {
		t.Error("Expected connected state after initialization")
	}

	// Close should disconnect the gadget
	err = manager.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	if gadget.IsConnected() {
		t.Error("Expected disconnected state after Close")
	}
}

// TestManagerCloseTwice tests that calling Close multiple times is safe
func TestManagerCloseTwice(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	diskPath := filepath.Join(tempDir, "test.img")

	// Create a disk image
	err := CreateDiskImage(diskPath, 10)
	if err != nil {
		t.Fatalf("Failed to create disk image: %v", err)
	}

	// Create config
	config := Config{
		diskPath:           diskPath,
		gadgetShortName:    "test",
		gadgetVendorId:     0x1d6b,
		gadgetProductId:    0x0104,
		gadgetBcdDevice:    0x0100,
		gadgetBcdUsb:       0x0200,
		gadgetProductName:  "Test Product",
		gadgetManufacturer: "Test Manufacturer",
	}

	// Create manager with NoOp gadget
	gadget := NewNoOpUsbGadget()
	manager, err := New(config, gadget)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// First close
	err = manager.Close()
	if err != nil {
		t.Errorf("First Close returned error: %v", err)
	}

	// Second close should be safe (idempotent)
	err = manager.Close()
	if err != nil {
		t.Errorf("Second Close returned error: %v", err)
	}
}
