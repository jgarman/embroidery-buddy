package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	diskfs "github.com/diskfs/go-diskfs"
)

func main() {
	var (
		imagePath  = flag.String("image", "", "Path to FAT32 disk image (required)")
		sourceFile = flag.String("source", "", "Path to source file to copy (required)")
		destPath   = flag.String("dest", "/test.bin", "Destination path in image")
		iterations = flag.Int("iterations", 3, "Number of iterations to run")
	)
	flag.Parse()

	if *imagePath == "" || *sourceFile == "" {
		fmt.Println("Usage: benchmark-copy -image <disk.img> -source <file>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Check if files exist
	if _, err := os.Stat(*imagePath); os.IsNotExist(err) {
		fmt.Printf("Error: Image file not found: %s\n", *imagePath)
		os.Exit(1)
	}
	if _, err := os.Stat(*sourceFile); os.IsNotExist(err) {
		fmt.Printf("Error: Source file not found: %s\n", *sourceFile)
		os.Exit(1)
	}

	// Get source file size
	srcInfo, err := os.Stat(*sourceFile)
	if err != nil {
		fmt.Printf("Error: Failed to stat source file: %v\n", err)
		os.Exit(1)
	}
	fileSize := srcInfo.Size()

	fmt.Printf("Benchmark Configuration:\n")
	fmt.Printf("  Image: %s\n", *imagePath)
	fmt.Printf("  Source: %s (%d bytes / %.2f MB)\n", *sourceFile, fileSize, float64(fileSize)/1024/1024)
	fmt.Printf("  Destination: %s\n", *destPath)
	fmt.Printf("  Iterations: %d\n\n", *iterations)

	// Run benchmarks
	var durations []time.Duration
	var totalBytes int64

	for i := 0; i < *iterations; i++ {
		fmt.Printf("Iteration %d/%d...\n", i+1, *iterations)

		start := time.Now()
		bytesWritten, err := copyFileToImage(*imagePath, *sourceFile, *destPath)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		durations = append(durations, duration)
		totalBytes = bytesWritten

		fmt.Printf("  Duration: %v\n", duration)
		fmt.Printf("  Throughput: %.2f MB/s\n", float64(bytesWritten)/duration.Seconds()/1024/1024)
	}

	// Calculate statistics
	if len(durations) == 0 {
		fmt.Println("\nAll iterations failed!")
		os.Exit(1)
	}

	fmt.Println("\n=== Results ===")
	fmt.Printf("Successful iterations: %d/%d\n", len(durations), *iterations)
	fmt.Printf("Bytes written per iteration: %d (%.2f MB)\n\n", totalBytes, float64(totalBytes)/1024/1024)

	var sum time.Duration
	minDuration := durations[0]
	maxDuration := durations[0]

	for _, d := range durations {
		sum += d
		if d < minDuration {
			minDuration = d
		}
		if d > maxDuration {
			maxDuration = d
		}
	}

	avgDuration := sum / time.Duration(len(durations))
	avgThroughput := float64(totalBytes) / avgDuration.Seconds() / 1024 / 1024

	fmt.Printf("Min duration:  %v (%.2f MB/s)\n", minDuration, float64(totalBytes)/minDuration.Seconds()/1024/1024)
	fmt.Printf("Max duration:  %v (%.2f MB/s)\n", maxDuration, float64(totalBytes)/maxDuration.Seconds()/1024/1024)
	fmt.Printf("Avg duration:  %v (%.2f MB/s)\n", avgDuration, avgThroughput)
}

// copyFileToImage copies a file into a FAT32 disk image using go-diskfs
func copyFileToImage(imagePath, sourcePath, destPath string) (int64, error) {
	// Open the disk image
	disk, err := diskfs.Open(imagePath, diskfs.WithOpenMode(diskfs.ReadWriteExclusive))
	if err != nil {
		return 0, fmt.Errorf("failed to open disk image: %w", err)
	}

	// Get the filesystem
	fs, err := disk.GetFilesystem(0)
	if err != nil {
		return 0, fmt.Errorf("failed to get filesystem: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get file size
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat source file: %w", err)
	}
	fileSize := srcInfo.Size()

	// Create destination file in the image
	destFile, err := fs.OpenFile(destPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy the data
	bytesWritten, err := io.Copy(destFile, srcFile)
	if err != nil {
		return 0, fmt.Errorf("failed to copy data: %w", err)
	}

	if bytesWritten != fileSize {
		return bytesWritten, fmt.Errorf("incomplete copy: wrote %d bytes, expected %d", bytesWritten, fileSize)
	}

	return bytesWritten, nil
}
