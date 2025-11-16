package system

import (
	"fmt"
	"net"
	"strings"
)

// GetWiFiMACAddress returns the MAC address of the WiFi interface
// On Raspberry Pi Zero W, the WiFi interface is typically named wlan0
func GetWiFiMACAddress() (string, error) {
	return GetMACAddress("wlan0")
}

// GetMACAddress returns the MAC address for a specific network interface
func GetMACAddress(interfaceName string) (string, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return "", fmt.Errorf("failed to get interface %s: %w", interfaceName, err)
	}

	mac := iface.HardwareAddr.String()
	if mac == "" {
		return "", fmt.Errorf("no MAC address found for interface %s", interfaceName)
	}

	return mac, nil
}

// GetAllMACAddresses returns a map of interface names to MAC addresses
func GetAllMACAddresses() (map[string]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	result := make(map[string]string)
	for _, iface := range interfaces {
		mac := iface.HardwareAddr.String()
		if mac != "" {
			result[iface.Name] = mac
		}
	}

	return result, nil
}

// FindWiFiInterface finds the first WiFi interface by checking common names
// Returns the interface name and its MAC address
func FindWiFiInterface() (string, string, error) {
	// Common WiFi interface names on Raspberry Pi and Linux
	commonNames := []string{"wlan0", "wlan1", "wlp2s0", "wlp3s0"}

	for _, name := range commonNames {
		mac, err := GetMACAddress(name)
		if err == nil {
			return name, mac, nil
		}
	}

	// If common names don't work, search for any interface starting with 'wl'
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if strings.HasPrefix(iface.Name, "wl") {
			mac := iface.HardwareAddr.String()
			if mac != "" {
				return iface.Name, mac, nil
			}
		}
	}

	return "", "", fmt.Errorf("no WiFi interface found")
}

// FormatMACAddress formats a MAC address string in various formats
type MACFormat int

const (
	// MACFormatColon formats as aa:bb:cc:dd:ee:ff (default)
	MACFormatColon MACFormat = iota
	// MACFormatHyphen formats as aa-bb-cc-dd-ee-ff
	MACFormatHyphen
	// MACFormatNone formats as aabbccddeeff
	MACFormatNone
	// MACFormatUSBSerial formats as suitable for USB serial number (12 hex chars)
	MACFormatUSBSerial
)

// FormatMAC formats a MAC address string according to the specified format
func FormatMAC(mac string, format MACFormat) string {
	// Remove any existing separators
	cleaned := strings.ReplaceAll(mac, ":", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	switch format {
	case MACFormatHyphen:
		return strings.ReplaceAll(mac, ":", "-")
	case MACFormatNone:
		return cleaned
	case MACFormatUSBSerial:
		// USB serial numbers should be 12 hex characters
		return cleaned
	case MACFormatColon:
		fallthrough
	default:
		// Ensure colon format
		if len(cleaned) == 12 {
			return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
				cleaned[0:2], cleaned[2:4], cleaned[4:6],
				cleaned[6:8], cleaned[8:10], cleaned[10:12])
		}
		return mac
	}
}
