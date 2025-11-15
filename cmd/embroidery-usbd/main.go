package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jgarman/embroidery-buddy/internal/diskmanager"
	"github.com/jgarman/embroidery-buddy/internal/webui"
	"github.com/rs/cors"
)

func main() {
	serverHost := "0.0.0.0"
	serverPort := 8080

	// TODO: Make these configurable via flags or environment variables
	diskPath := os.Getenv("DISK_PATH")
	if diskPath == "" {
		diskPath = "/tmp/embroidery.img"
		log.Printf("DISK_PATH not set, using default: %s", diskPath)
		// create new disk since we're testing
		diskmanager.CreateDiskImage(diskPath, 100)
	}

	// Create disk manager configuration
	config := diskmanager.Config{
		DiskPath:           diskPath,
		GadgetShortName:    "embroidery",
		GadgetVendorId:     0x1d6b, // Linux Foundation
		GadgetProductId:    0x0104, // Multifunction Composite Gadget
		GadgetBcdDevice:    0x0100, // Device version 1.0
		GadgetBcdUsb:       0x0200, // USB 2.0
		GadgetProductName:  "Embroidery USB Storage",
		GadgetManufacturer: "Embroidery Buddy",
	}

	// Initialize disk manager with NoOp gadget for development
	// In production, this would use LinuxUsbGadget
	gadget := diskmanager.NewNoOpUsbGadget()
	dm, err := diskmanager.New(config, gadget)
	if err != nil {
		log.Fatalf("Failed to initialize disk manager: %v", err)
	}
	defer dm.Close()

	log.Printf("Disk manager initialized with disk: %s", diskPath)

	// Create web UI handler
	webHandler, err := webui.New(dm)
	if err != nil {
		log.Fatalf("Failed to initialize web UI: %v", err)
	}

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Setup routes
	r := mux.NewRouter()
	r.HandleFunc("/", webHandler.IndexHandler).Methods("GET")
	r.HandleFunc("/api/upload", webHandler.UploadHandler).Methods("POST")
	r.HandleFunc("/api/health", webHandler.HealthHandler).Methods("GET")

	handler := c.Handler(r)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", serverHost, serverPort),
		Handler:      handler,
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server at %s:%d\n", serverHost, serverPort)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicf("Failed to start server: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %s", err)
	}

	log.Println("Server exited")

}
