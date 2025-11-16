package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jgarman/embroidery-buddy/internal/mdns"
)

func main() {
	var (
		name       = flag.String("name", "Test HTTP Service", "Service name")
		port       = flag.Int("port", 8080, "Port number")
		useDBus    = flag.Bool("dbus", true, "Use DBus API (more reliable)")
		checkAvahi = flag.Bool("check", false, "Check if Avahi is available and exit")
	)
	flag.Parse()

	// Check Avahi availability
	if *checkAvahi {
		if mdns.IsAvahiDBusAvailable() {
			fmt.Println("Avahi is available via DBus")
		} else if mdns.IsAvahiAvailable() {
			fmt.Println("Avahi command-line tools are available")
		} else {
			fmt.Println("Avahi is NOT available")
			os.Exit(1)
		}
		return
	}

	// Start a simple HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from mDNS-advertised service!\n")
		fmt.Fprintf(w, "Service name: %s\n", *name)
		fmt.Fprintf(w, "Port: %d\n", *port)
	})

	go func() {
		log.Printf("Starting HTTP server on port %d", *port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
			log.Fatal(err)
		}
	}()

	// Publish service with Avahi
	var publisher interface{ Stop() error }
	var err error

	if *useDBus && mdns.IsAvahiDBusAvailable() {
		log.Println("Publishing service using Avahi DBus API...")
		publisher, err = mdns.PublishHTTPDBus(
			*name,
			*port,
			"path=/",
			"version=1.0",
		)
	} else if mdns.IsAvahiAvailable() {
		log.Println("Publishing service using avahi-publish-service command...")
		publisher, err = mdns.PublishHTTP(
			*name,
			*port,
			"path=/",
			"version=1.0",
		)
	} else {
		log.Fatal("Avahi is not available on this system")
	}

	if err != nil {
		log.Fatalf("Failed to publish service: %v", err)
	}
	defer publisher.Stop()

	log.Printf("Service '%s' published successfully", *name)
	log.Printf("Available at: http://<hostname>.local:%d/", *port)
	log.Println("Press Ctrl+C to stop...")

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Stopping service...")
}
