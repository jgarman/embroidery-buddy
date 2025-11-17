# Claude Code Development Guide

This document provides insights for AI assistants (like Claude) working on the Embroidery Buddy project.

## Project Overview

This is a Go-based USB gadget application that transforms a Raspberry Pi Zero W into a WiFi-enabled virtual USB flash drive for embroidery machines. The project uses platform-specific code to handle USB gadget configuration on Linux while providing no-op implementations for development on other platforms.

## Key Architecture Insights

### Platform-Specific Build Pattern

The project uses Go's build tags extensively to handle platform differences:

- **Linux-specific files**: `*_linux.go` - Actual USB gadget and filesystem implementations
- **Non-Linux files**: `*_other.go` - No-op implementations for development on macOS/Windows
- **Factory pattern**: `*_factory.go` files abstract platform differences

This allows development and testing on any platform while maintaining production code for Raspberry Pi.

### Core Components

1. **diskmanager** ([internal/diskmanager/](internal/diskmanager/))
   - Manages the virtual FAT32 disk image
   - Handles USB gadget configuration via Linux ConfigFS
   - Provides transaction-based file operations for consistency
   - Key files:
     - `manager.go` - Core disk management logic
     - `usb_gadget_linux.go` - Linux USB gadget implementation
     - `usb_gadget_noop.go` - Development no-op implementation
     - `fs_writer_*.go` - Different filesystem writer strategies

2. **webui** ([internal/webui/](internal/webui/))
   - HTTP handlers for file upload and management
   - Embedded HTML templates (see `template.go`)
   - Streaming multipart upload handling
   - ZIP file extraction support
   - Important: Uses streaming to handle large files without loading entirely into memory

3. **config** ([internal/config/](internal/config/))
   - JSON-based configuration
   - See `config.example.json` for schema
   - Hex value parsing for USB device IDs

4. **mdns** ([internal/mdns/](internal/mdns/))
   - Service discovery via Avahi
   - Two implementations: D-Bus and CLI-based
   - Platform-specific code for Linux only

### Transaction Pattern

The disk manager uses a transaction pattern for file operations:

```go
err := diskManager.BeginTransaction(func(tx *diskmanager.Transaction) error {
    return tx.WriteFile(path, reader, size)
})
```

This ensures filesystem consistency and proper cleanup on errors.

## Build System - IMPORTANT

### Always Use the Makefile

**DO NOT** use `go build` directly. Always use the provided Makefile targets:

```bash
# Build for current platform (development)
make build

# Build for Raspberry Pi (ARM)
make build-rpi

# Build for all platforms
make build-all

# Run tests
make test

# Build and copy to Raspberry Pi
make copy

# Run benchmarks
make benchmark
```

### Why Use the Makefile?

1. **Correct build flags**: The Makefile includes `-ldflags="-s -w"` to strip debug symbols and reduce binary size
2. **Cross-compilation**: Properly sets `GOOS=linux GOARCH=arm GOARM=6` for Raspberry Pi Zero W
3. **Consistent output paths**: Binaries go to `build/bin/` and `build/bin/linux-arm/`
4. **Multiple binaries**: Builds both the main application and utilities in one command

## Testing and Development Workflow

### Local Development (macOS/Linux/Windows)

```bash
# Build and run locally (uses no-op USB gadget)
make build
./build/bin/embroidery-usbd
```

The application will:
- Create a test disk at `/tmp/embroidery.img`
- Use no-op USB gadget implementation
- Start web server on port 80 (may require sudo)

### Raspberry Pi Development

```bash
# Build and deploy to Raspberry Pi at dietpi.local
make copy

# Then SSH to the Pi and restart the service
ssh dietpi@dietpi.local
sudo systemctl restart embroidery-usbd.service
```

### Running Tests

```bash
# Run all tests
make test

# Test specific package
go test -v ./internal/diskmanager/
```

## Configuration

The application uses `config.example.json` as a template. Key configuration points:

- **disk.path**: Location of the virtual disk image
- **disk.size_mb**: Size of the FAT32 filesystem (typical: 256MB)
- **usb_gadget.use_noop**: Set to `true` for non-Linux development
- **server.port**: HTTP server port (default: 80)
- **mdns.enabled**: Enable mDNS service discovery

## Common Development Tasks

### Adding a New API Endpoint

1. Add handler function in [internal/webui/handler.go](internal/webui/handler.go)
2. Register route in [cmd/embroidery-usbd/main.go](cmd/embroidery-usbd/main.go)
3. Update API documentation in README.md

### Modifying Disk Operations

1. Edit [internal/diskmanager/manager.go](internal/diskmanager/manager.go)
2. Consider transaction safety
3. Check if changes affect both `fs_writer_loopback.go` and `fs_writer_diskfs.go`
4. Run tests: `make test`

### Updating the Web UI

1. Edit HTML in [internal/webui/templates/index.html](internal/webui/templates/index.html)
2. The template is embedded via [internal/webui/template.go](internal/webui/template.go)
3. No need to rebuild for template changes in development if using `go:embed`
4. For production, rebuild: `make build-rpi`

### Working with USB Gadget Code

- Linux-specific code is in `usb_gadget_linux.go`
- Uses ConfigFS at `/sys/kernel/config/usb_gadget/`
- Requires root privileges and `libcomposite` module
- Development uses `usb_gadget_noop.go` - doesn't require root

## File Organization Conventions

- `cmd/`: Executable entry points (main packages)
- `internal/`: Private application code (cannot be imported by other projects)
- `scripts/`: Shell scripts for deployment and testing
- `build/`: Build artifacts (gitignored)
- `docs/`: Additional documentation

## Dependencies

Major dependencies and their purposes:

- `github.com/gorilla/mux` - HTTP routing with better pattern matching than stdlib
- `github.com/rs/cors` - CORS middleware for web API
- `github.com/diskfs/go-diskfs` - FAT32 filesystem manipulation without mounting
- `github.com/godbus/dbus/v5` - D-Bus communication for Avahi mDNS

## Error Handling Patterns

The codebase uses Go 1.13+ error wrapping:

```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

Sentinel errors for specific conditions:
- `diskmanager.ErrDiskFull` - Disk has insufficient space
- `diskmanager.ErrDiskNotInitialized` - Disk image not created

Check errors with `errors.Is()`:

```go
if errors.Is(err, diskmanager.ErrDiskFull) {
    // Handle disk full specifically
}
```

## Logging

- Uses standard `log` package
- Logs go to stdout/stderr (captured by systemd journal)
- Log levels indicated by prefix: "Error:", "Warning:", or informational

## Security Considerations

1. **Path traversal prevention**: All uploaded filenames are sanitized with `filepath.Base()` and `filepath.Clean()`
2. **File size limits**: Upload size capped at 200MB (see `handler.go`)
3. **ZIP extraction**: Validates paths don't contain ".." to prevent directory traversal
4. **No authentication**: This is designed for isolated WiFi networks; add auth if exposed to untrusted networks

## Performance Notes

1. **Streaming uploads**: Files are streamed to disk, not buffered entirely in memory
2. **Buffer sizes**: Uses 1MB buffers for optimal throughput on Raspberry Pi
3. **ZIP handling**: ZIP files must be buffered in memory due to format requirements (central directory at end)
4. **Transaction overhead**: Each file write is a full filesystem transaction for safety

## Systemd Service

The service runs as root (required for USB gadget access):
- Service file: [scripts/embroidery-usbd.service](scripts/embroidery-usbd.service)
- Logs: `journalctl -u embroidery-usbd.service -f`
- Working directory: `/var/lib/embroidery-usbd`
- Binary location: `/opt/embroiderybuddy/bin/embroidery-usbd-linux-arm`

## Troubleshooting Development Issues

### "Permission denied" errors on Linux
- USB gadget operations require root: `sudo ./build/bin/embroidery-usbd`
- Or use development mode with `use_noop: true` in config

### Cross-compilation issues
- Ensure you're using the Makefile, not direct `go build`
- The Makefile sets the correct `GOARCH=arm GOARM=6` for Pi Zero W

### Tests failing
- Some tests may require Linux-specific features
- Build tags ensure platform-specific tests only run on appropriate platforms

### Can't connect to web interface
- Check if port 80 requires sudo on your platform
- Try changing `server.port` to 8080 in config for development
- Verify firewall settings

## Quick Reference Commands

```bash
# Full build and test cycle
make clean && make test && make build-all

# Deploy to Raspberry Pi
make copy

# View logs on Raspberry Pi
ssh dietpi@dietpi.local "journalctl -u embroidery-usbd.service -f"

# Restart service after deployment
ssh dietpi@dietpi.local "sudo systemctl restart embroidery-usbd.service"

# Check USB gadget status on Pi
ssh dietpi@dietpi.local "lsmod | grep usb_f_mass_storage"
```

## When Making Changes

1. **Always run tests first**: `make test`
2. **Use the Makefile**: Don't bypass it with manual `go build` commands
3. **Test on target platform**: Deploy to actual Raspberry Pi for integration testing
4. **Check logs**: Monitor systemd journal for runtime issues
5. **Update README.md**: Document any user-facing changes
6. **Update this file**: Add insights for future AI assistants
