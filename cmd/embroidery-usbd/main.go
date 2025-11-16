package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jgarman/embroidery-buddy/internal/config"
	"github.com/jgarman/embroidery-buddy/internal/diskmanager"
	"github.com/jgarman/embroidery-buddy/internal/mdns"
	"github.com/jgarman/embroidery-buddy/internal/webui"
	"github.com/rs/cors"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file (default: use built-in defaults)")
	generateConfig := flag.Bool("generate-config", false, "Generate example configuration file and exit")
	flag.Parse()

	// Handle config generation
	if *generateConfig {
		cfg := config.Default()
		path := "config.json"
		if *configPath != "" {
			path = *configPath
		}
		if err := cfg.Save(path); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		log.Printf("Configuration file generated: %s", path)
		return
	}

	// Load configuration
	var cfg *config.Config
	var err error
	if *configPath != "" {
		log.Printf("Loading configuration from: %s", *configPath)
		cfg, err = config.Load(*configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		log.Printf("No config file specified, using defaults")
		cfg = config.Default()
		// For development, use temp directory
		cfg.Disk.Path = "/tmp/embroidery.img"
		cfg.USBGadget.UseNoOp = true
	}

	// Create disk image if it doesn't exist and auto-create is enabled
	if cfg.Disk.AutoCreate {
		if _, err := os.Stat(cfg.Disk.Path); os.IsNotExist(err) {
			log.Printf("Creating disk image: %s (%dMB)", cfg.Disk.Path, cfg.Disk.SizeMB)
			if err := diskmanager.CreateDiskImage(cfg.Disk.Path, cfg.Disk.SizeMB); err != nil {
				log.Fatalf("Failed to create disk image: %v", err)
			}
		}
	}

	// Parse USB gadget hex values
	vendorId, err := config.ParseHex(cfg.USBGadget.VendorID)
	if err != nil {
		log.Fatalf("Invalid vendor ID: %v", err)
	}
	productId, err := config.ParseHex(cfg.USBGadget.ProductID)
	if err != nil {
		log.Fatalf("Invalid product ID: %v", err)
	}
	bcdDevice, err := config.ParseHex(cfg.USBGadget.BCDDevice)
	if err != nil {
		log.Fatalf("Invalid BCD device: %v", err)
	}
	bcdUsb, err := config.ParseHex(cfg.USBGadget.BCDUSB)
	if err != nil {
		log.Fatalf("Invalid BCD USB: %v", err)
	}

	// Create disk manager configuration
	dmConfig := diskmanager.Config{
		DiskPath:           cfg.Disk.Path,
		GadgetShortName:    cfg.USBGadget.ShortName,
		GadgetVendorId:     vendorId,
		GadgetProductId:    productId,
		GadgetBcdDevice:    bcdDevice,
		GadgetBcdUsb:       bcdUsb,
		GadgetProductName:  cfg.USBGadget.ProductName,
		GadgetManufacturer: cfg.USBGadget.Manufacturer,
	}

	// Initialize disk manager with appropriate gadget implementation
	gadget := diskmanager.NewUsbGadget(dmConfig, cfg.USBGadget.UseNoOp)
	dm, err := diskmanager.New(dmConfig, gadget)
	if err != nil {
		log.Fatalf("Failed to initialize disk manager: %v", err)
	}
	defer dm.Close()

	log.Printf("Disk manager initialized with disk: %s", cfg.Disk.Path)

	// Create web UI handler
	webHandler, err := webui.New(dm)
	if err != nil {
		log.Fatalf("Failed to initialize web UI: %v", err)
	}

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.Server.CORS.AllowedOrigins,
		AllowedMethods:   cfg.Server.CORS.AllowedMethods,
		AllowedHeaders:   cfg.Server.CORS.AllowedHeaders,
		AllowCredentials: cfg.Server.CORS.AllowCredentials,
	})

	// Setup routes
	r := mux.NewRouter()
	r.HandleFunc("/", webHandler.IndexHandler).Methods("GET")
	r.HandleFunc("/api/upload", webHandler.UploadHandler).Methods("POST")
	r.HandleFunc("/api/health", webHandler.HealthHandler).Methods("GET")
	r.HandleFunc("/api/clear", webHandler.ClearFilesHandler).Methods("POST")

	handler := c.Handler(r)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  time.Second * time.Duration(cfg.Server.ReadTimeout),
		WriteTimeout: time.Second * time.Duration(cfg.Server.WriteTimeout),
		IdleTimeout:  time.Second * time.Duration(cfg.Server.IdleTimeout),
	}

	// Publish mDNS service if enabled
	var mdnsPublisher interface{ Stop() error }
	if cfg.MDNS.Enabled {
		if cfg.MDNS.UseDBus && mdns.IsAvahiDBusAvailable() {
			log.Printf("Publishing mDNS service '%s' via DBus...", cfg.MDNS.ServiceName)
			mdnsPublisher, err = mdns.PublishHTTPDBus(
				cfg.MDNS.ServiceName,
				cfg.Server.Port,
				cfg.MDNS.TXTRecords...,
			)
			if err != nil {
				log.Printf("Warning: Failed to publish mDNS service via DBus: %v", err)
			} else {
				log.Printf("mDNS service published: %s.local:%d", cfg.MDNS.ServiceName, cfg.Server.Port)
				defer mdnsPublisher.Stop()
			}
		} else if mdns.IsAvahiAvailable() {
			log.Printf("Publishing mDNS service '%s' via avahi-publish...", cfg.MDNS.ServiceName)
			mdnsPublisher, err = mdns.PublishHTTP(
				cfg.MDNS.ServiceName,
				cfg.Server.Port,
				cfg.MDNS.TXTRecords...,
			)
			if err != nil {
				log.Printf("Warning: Failed to publish mDNS service: %v", err)
			} else {
				log.Printf("mDNS service published: %s.local:%d", cfg.MDNS.ServiceName, cfg.Server.Port)
				defer mdnsPublisher.Stop()
			}
		} else {
			log.Println("Warning: mDNS enabled but Avahi not available")
		}
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server at %s:%d\n", cfg.Server.Host, cfg.Server.Port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicf("Failed to start server: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop mDNS publishing
	if mdnsPublisher != nil {
		log.Println("Stopping mDNS service...")
		mdnsPublisher.Stop()
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %s", err)
	}

	log.Println("Server exited")

}
