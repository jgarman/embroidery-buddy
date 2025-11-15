#!/bin/bash
# Script to create a test disk image for development

set -e

DISK_PATH="${DISK_PATH:-/tmp/embroidery.img}"
DISK_SIZE_MB="${DISK_SIZE_MB:-100}"

echo "Creating test disk image at: $DISK_PATH"
echo "Size: ${DISK_SIZE_MB}MB"

# Build the helper tool
go build -o /tmp/create-disk ./cmd/create-disk 2>/dev/null || {
    echo "Building inline disk creator..."
    cat > /tmp/create-disk.go <<'EOF'
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jgarman/embroidery-buddy/internal/diskmanager"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <disk-path> <size-mb>\n", os.Args[0])
		os.Exit(1)
	}

	diskPath := os.Args[1]
	sizeMb, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid size: %v\n", err)
		os.Exit(1)
	}

	err = diskmanager.CreateDiskImage(diskPath, sizeMb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create disk: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created disk image: %s (%dMB)\n", diskPath, sizeMb)
}
EOF
    go run /tmp/create-disk.go "$DISK_PATH" "$DISK_SIZE_MB"
    rm -f /tmp/create-disk.go
    exit 0
}

/tmp/create-disk "$DISK_PATH" "$DISK_SIZE_MB"
rm -f /tmp/create-disk

echo "Test disk created successfully!"
echo "You can now run the server with:"
echo "  DISK_PATH=$DISK_PATH ./embroidery-usbd"
