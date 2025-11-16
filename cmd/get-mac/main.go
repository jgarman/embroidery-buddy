package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jgarman/embroidery-buddy/internal/system"
)

func main() {
	var (
		interfaceName = flag.String("interface", "", "Specific interface name (e.g., wlan0)")
		listAll       = flag.Bool("all", false, "List all interfaces and their MAC addresses")
		formatType    = flag.String("format", "colon", "Output format: colon, hyphen, none, usb")
	)
	flag.Parse()

	// Determine MAC format
	var format system.MACFormat
	switch *formatType {
	case "colon":
		format = system.MACFormatColon
	case "hyphen":
		format = system.MACFormatHyphen
	case "none":
		format = system.MACFormatNone
	case "usb":
		format = system.MACFormatUSBSerial
	default:
		log.Fatalf("Invalid format: %s (use: colon, hyphen, none, usb)", *formatType)
	}

	// List all interfaces
	if *listAll {
		macs, err := system.GetAllMACAddresses()
		if err != nil {
			log.Fatalf("Failed to get MAC addresses: %v", err)
		}

		fmt.Println("Network Interfaces:")
		for iface, mac := range macs {
			formatted := system.FormatMAC(mac, format)
			fmt.Printf("  %-15s %s\n", iface+":", formatted)
		}
		return
	}

	// Get specific interface
	if *interfaceName != "" {
		mac, err := system.GetMACAddress(*interfaceName)
		if err != nil {
			log.Fatalf("Failed to get MAC address for %s: %v", *interfaceName, err)
		}
		fmt.Println(system.FormatMAC(mac, format))
		return
	}

	// Find WiFi interface (default behavior)
	name, mac, err := system.FindWiFiInterface()
	if err != nil {
		// Try to get all and show what's available
		macs, listErr := system.GetAllMACAddresses()
		if listErr != nil {
			log.Fatalf("No WiFi interface found and failed to list interfaces: %v", err)
		}

		fmt.Fprintf(os.Stderr, "No WiFi interface found. Available interfaces:\n")
		for iface := range macs {
			fmt.Fprintf(os.Stderr, "  - %s\n", iface)
		}
		fmt.Fprintf(os.Stderr, "\nUse -interface <name> to specify an interface\n")
		os.Exit(1)
	}

	formatted := system.FormatMAC(mac, format)
	fmt.Printf("WiFi interface: %s\n", name)
	fmt.Printf("MAC address: %s\n", formatted)
}
