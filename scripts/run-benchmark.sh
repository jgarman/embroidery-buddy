#!/bin/bash
set -e

# Script to run copy benchmarks with different file sizes

# Configuration
IMAGE_PATH="${IMAGE_PATH:-/tmp/benchmark-test.img}"
IMAGE_SIZE_MB="${IMAGE_SIZE_MB:-100}"
ITERATIONS="${ITERATIONS:-5}"

echo "=== Disk Copy Benchmark Suite ==="
echo ""

# Build the benchmark tool
echo "Building benchmark tool..."
go build -o /tmp/benchmark-copy ./cmd/benchmark-copy

# Create a test disk image if it doesn't exist
if [ ! -f "$IMAGE_PATH" ]; then
    echo "Creating test disk image: $IMAGE_PATH (${IMAGE_SIZE_MB}MB)"

    # Create image using the embroidery-usbd binary if available
    if [ -f "./embroidery-usbd" ]; then
        # Use the CreateDiskImage functionality
        cat > /tmp/create-disk.go << 'EOF'
package main

import (
    "fmt"
    "os"
    "strconv"
    "github.com/jgarman/embroidery-buddy/internal/diskmanager"
)

func main() {
    if len(os.Args) != 3 {
        fmt.Println("Usage: create-disk <path> <size_mb>")
        os.Exit(1)
    }
    path := os.Args[1]
    sizeMB, _ := strconv.ParseInt(os.Args[2], 10, 64)

    if err := diskmanager.CreateDiskImage(path, sizeMB); err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("Created disk image: %s\n", path)
}
EOF
        go run /tmp/create-disk.go "$IMAGE_PATH" "$IMAGE_SIZE_MB"
        rm /tmp/create-disk.go
    else
        echo "Error: embroidery-usbd not built. Run 'make build' first."
        exit 1
    fi
else
    echo "Using existing disk image: $IMAGE_PATH"
fi

echo ""
echo "Creating test files..."

# Create test files of various sizes
TEST_FILES=(
    "1KB:/tmp/test-1kb.bin:1024"
    "10KB:/tmp/test-10kb.bin:10240"
    "100KB:/tmp/test-100kb.bin:102400"
    "1MB:/tmp/test-1mb.bin:1048576"
    "10MB:/tmp/test-10mb.bin:10485760"
)

for spec in "${TEST_FILES[@]}"; do
    IFS=':' read -r name path size <<< "$spec"
    if [ ! -f "$path" ]; then
        echo "  Creating $name test file..."
        dd if=/dev/urandom of="$path" bs=1 count="$size" status=none
    fi
done

echo ""
echo "Running benchmarks..."
echo "======================================"

# Run benchmarks for each file size
for spec in "${TEST_FILES[@]}"; do
    IFS=':' read -r name path size <<< "$spec"
    echo ""
    echo "=== Testing $name file ==="
    /tmp/benchmark-copy \
        -image "$IMAGE_PATH" \
        -source "$path" \
        -dest "/benchmark-$name.bin" \
        -iterations "$ITERATIONS"
    echo ""
done

echo ""
echo "======================================"
echo "Benchmark complete!"
echo ""
echo "To run a custom benchmark:"
echo "  /tmp/benchmark-copy -image $IMAGE_PATH -source <your-file> -iterations 5"
echo ""
echo "To clean up test files:"
echo "  rm -f /tmp/test-*.bin /tmp/benchmark-copy"
echo "  rm -f $IMAGE_PATH"
