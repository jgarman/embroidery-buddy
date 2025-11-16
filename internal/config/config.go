package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server"`

	// Disk configuration
	Disk DiskConfig `json:"disk"`

	// USB Gadget configuration
	USBGadget USBGadgetConfig `json:"usb_gadget"`

	// Upload configuration
	Upload UploadConfig `json:"upload"`

	// mDNS/Avahi configuration
	MDNS MDNSConfig `json:"mdns"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`

	// Timeout settings in seconds
	ReadTimeout  int `json:"read_timeout"`
	WriteTimeout int `json:"write_timeout"`
	IdleTimeout  int `json:"idle_timeout"`

	// CORS settings
	CORS CORSConfig `json:"cors"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
}

// DiskConfig contains disk image settings
type DiskConfig struct {
	Path   string `json:"path"`
	SizeMB int64  `json:"size_mb"`

	// Auto-create the disk image if it doesn't exist
	AutoCreate bool `json:"auto_create"`
}

// USBGadgetConfig contains USB gadget settings
type USBGadgetConfig struct {
	ShortName    string `json:"short_name"`
	VendorID     string `json:"vendor_id"`
	ProductID    string `json:"product_id"`
	BCDDevice    string `json:"bcd_device"`
	BCDUSB       string `json:"bcd_usb"`
	ProductName  string `json:"product_name"`
	Manufacturer string `json:"manufacturer"`

	// Use NoOp gadget for development/testing
	UseNoOp bool `json:"use_noop"`
}

// UploadConfig contains file upload settings
type UploadConfig struct {
	// Maximum upload size in MB
	MaxSizeMB int64 `json:"max_size_mb"`
}

// MDNSConfig contains mDNS/Avahi service discovery settings
type MDNSConfig struct {
	// Enable mDNS service advertisement
	Enabled bool `json:"enabled"`

	// Service name (e.g., "Embroidery Buddy")
	ServiceName string `json:"service_name"`

	// Use DBus API (more reliable than command-line)
	UseDBus bool `json:"use_dbus"`

	// Additional TXT records (key=value pairs)
	TXTRecords []string `json:"txt_records"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  15,
			WriteTimeout: 15,
			IdleTimeout:  60,
			CORS: CORSConfig{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"*"},
				AllowCredentials: true,
			},
		},
		Disk: DiskConfig{
			Path:       "/var/lib/embroidery-buddy/disk.img",
			SizeMB:     100,
			AutoCreate: true,
		},
		USBGadget: USBGadgetConfig{
			ShortName:    "embroidery",
			VendorID:     "0x1d6b",
			ProductID:    "0x0104",
			BCDDevice:    "0x0100",
			BCDUSB:       "0x0200",
			ProductName:  "Embroidery USB Storage",
			Manufacturer: "Embroidery Buddy",
			UseNoOp:      false,
		},
		Upload: UploadConfig{
			MaxSizeMB: 100,
		},
		MDNS: MDNSConfig{
			Enabled:     true,
			ServiceName: "Embroidery Buddy",
			UseDBus:     true,
			TXTRecords: []string{
				"path=/",
				"version=1.0",
			},
		},
	}
}

// Load loads configuration from a JSON file
// If the file doesn't exist, it returns the default configuration
func Load(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Default(), nil
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	config := Default() // Start with defaults
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// Save writes the configuration to a JSON file
func (c *Config) Save(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ParseHex converts a hex string (like "0x1d6b") to an integer
func ParseHex(s string) (int, error) {
	var val int
	_, err := fmt.Sscanf(s, "0x%x", &val)
	if err != nil {
		return 0, fmt.Errorf("invalid hex value %s: %w", s, err)
	}
	return val, nil
}
