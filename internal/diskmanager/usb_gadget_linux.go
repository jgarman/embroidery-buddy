//go:build linux

package diskmanager

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jgarman/embroidery-buddy/internal/system"
)

// LinuxUsbGadget implements UsbGadget for Linux systems using configfs
type LinuxUsbGadget struct {
	config    Config
	connected bool
	udcName   string // Store the UDC name for reconnection
}

// NewLinuxUsbGadget creates a new Linux USB gadget implementation
func NewLinuxUsbGadget(config Config) *LinuxUsbGadget {
	return &LinuxUsbGadget{
		config: config,
	}
}

// writeSysfs writes a value to a sysfs file
func writeSysfs(path, value string) error {
	err := os.WriteFile(path, []byte(value), 0644)
	if err != nil {
		return fmt.Errorf("failed to write '%s' to %s: %w", value, path, err)
	}
	return nil
}

// Initialize sets up and activates the USB gadget
func (g *LinuxUsbGadget) Initialize() error {
	desiredPermissions := os.FileMode(0775)
	gadgetBase := filepath.Join("/sys/kernel/config/usb_gadget", g.config.GadgetShortName)

	// Check if USB gadget directory already exists
	_, err := os.Stat(gadgetBase)
	if err == nil {
		return fmt.Errorf("gadget %s already configured", g.config.GadgetShortName)
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("other error for gadget %s: %w", g.config.GadgetShortName, err)
	}

	// Create USB gadget directory
	if err := os.Mkdir(gadgetBase, desiredPermissions); err != nil {
		return fmt.Errorf("could not create USB gadget directory: %w", err)
	}

	// Configure gadget - write vendor/product IDs and USB versions
	if err := writeSysfs(filepath.Join(gadgetBase, "idVendor"), fmt.Sprintf("0x%04x", g.config.GadgetVendorId)); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(gadgetBase, "idProduct"), fmt.Sprintf("0x%04x", g.config.GadgetProductId)); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(gadgetBase, "bcdDevice"), fmt.Sprintf("0x%04x", g.config.GadgetBcdDevice)); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(gadgetBase, "bcdUSB"), fmt.Sprintf("0x%04x", g.config.GadgetBcdUsb)); err != nil {
		return err
	}

	// Create and configure strings directory (0x409 = English US)
	stringsDir := filepath.Join(gadgetBase, "strings/0x409")
	if err := os.MkdirAll(stringsDir, desiredPermissions); err != nil {
		return fmt.Errorf("failed to create strings directory: %w", err)
	}

	// Get WiFi MAC address for serial number
	serialNumber := getSerialNumber()
	if err := writeSysfs(filepath.Join(stringsDir, "serialnumber"), serialNumber); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(stringsDir, "manufacturer"), g.config.GadgetManufacturer); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(stringsDir, "product"), g.config.GadgetProductName); err != nil {
		return err
	}

	// Create and configure config
	configStringsDir := filepath.Join(gadgetBase, "configs/c.1/strings/0x409")
	if err := os.MkdirAll(configStringsDir, desiredPermissions); err != nil {
		return fmt.Errorf("failed to create config strings directory: %w", err)
	}

	if err := writeSysfs(filepath.Join(gadgetBase, "configs/c.1/strings/0x409/configuration"), "Mass Storage"); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(gadgetBase, "configs/c.1/MaxPower"), "250"); err != nil {
		return err
	}

	// Create and configure mass storage function
	massStorageDir := filepath.Join(gadgetBase, "functions/mass_storage.usb0")
	if err := os.MkdirAll(massStorageDir, desiredPermissions); err != nil {
		return fmt.Errorf("failed to create mass storage function directory: %w", err)
	}

	if err := writeSysfs(filepath.Join(massStorageDir, "stall"), "1"); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(massStorageDir, "lun.0/cdrom"), "0"); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(massStorageDir, "lun.0/ro"), "0"); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(massStorageDir, "lun.0/nofua"), "0"); err != nil {
		return err
	}
	if err := writeSysfs(filepath.Join(massStorageDir, "lun.0/file"), g.config.DiskPath); err != nil {
		return err
	}

	// Link function to config
	functionLink := filepath.Join(gadgetBase, "configs/c.1/mass_storage.usb0")
	functionTarget := filepath.Join(gadgetBase, "functions/mass_storage.usb0")
	if err := os.Symlink(functionTarget, functionLink); err != nil {
		return fmt.Errorf("failed to link function to config: %w", err)
	}

	// Get UDC name for activation by reading directory entries
	udcEntries, err := os.ReadDir("/sys/class/udc")
	if err != nil {
		return fmt.Errorf("failed to read UDC directory: %w", err)
	}

	if len(udcEntries) == 0 {
		return fmt.Errorf("no UDC available")
	}

	// Use the first UDC found
	udcName := udcEntries[0].Name()

	// Store UDC name for later use
	g.udcName = udcName

	// Connect the gadget to the host
	return g.Reconnect()
}

// Disconnect disconnects the USB gadget from the host without destroying the configuration
func (g *LinuxUsbGadget) Disconnect() error {
	if !g.connected {
		return nil // Already disconnected
	}

	gadgetBase := filepath.Join("/sys/kernel/config/usb_gadget", g.config.GadgetShortName)
	udcPath := filepath.Join(gadgetBase, "UDC")

	// Disconnect by writing empty string to UDC
	if err := writeSysfs(udcPath, "\n"); err != nil {
		return fmt.Errorf("failed to disconnect gadget: %w", err)
	}

	g.connected = false
	return nil
}

// Reconnect reconnects the USB gadget to the host
func (g *LinuxUsbGadget) Reconnect() error {
	if g.connected {
		return nil // Already connected
	}

	if g.udcName == "" {
		return fmt.Errorf("no UDC name available, gadget may not have been initialized")
	}

	gadgetBase := filepath.Join("/sys/kernel/config/usb_gadget", g.config.GadgetShortName)
	udcPath := filepath.Join(gadgetBase, "UDC")

	// Reconnect by writing UDC name back
	if err := writeSysfs(udcPath, g.udcName); err != nil {
		return fmt.Errorf("failed to reconnect gadget: %w", err)
	}

	g.connected = true
	return nil
}

// IsConnected returns true if the USB gadget is currently connected to a host
func (g *LinuxUsbGadget) IsConnected() bool {
	return g.connected
}

// destroy deactivates and removes the USB gadget (private method)
func (g *LinuxUsbGadget) destroy() {
	gadgetBase := filepath.Join("/sys/kernel/config/usb_gadget", g.config.GadgetShortName)

	// Check if gadget exists
	if _, err := os.Stat(gadgetBase); os.IsNotExist(err) {
		// Gadget doesn't exist, nothing to destroy
		return
	}

	// Disconnect first
	_ = g.Disconnect()

	// Remove the symlink from config to function
	functionLink := filepath.Join(gadgetBase, "configs/c.1/mass_storage.usb0")
	_ = os.Remove(functionLink)

	// Remove directories in reverse order of creation
	// Note: We ignore errors since some directories may not exist if setup was incomplete
	_ = os.RemoveAll(filepath.Join(gadgetBase, "configs/c.1/strings/0x409"))
	_ = os.RemoveAll(filepath.Join(gadgetBase, "configs/c.1"))
	_ = os.RemoveAll(filepath.Join(gadgetBase, "functions/mass_storage.usb0"))
	_ = os.RemoveAll(filepath.Join(gadgetBase, "strings/0x409"))

	// Finally remove the gadget directory itself
	_ = os.RemoveAll(gadgetBase)
}

// getSerialNumber returns a serial number based on the WiFi MAC address
// Falls back to a default if MAC cannot be determined
func getSerialNumber() string {
	// Try to get WiFi MAC address
	_, mac, err := system.FindWiFiInterface()
	if err != nil {
		log.Printf("Warning: Could not get WiFi MAC address: %v, using default serial number", err)
		return "000000000000"
	}

	// Format as USB serial (12 hex characters, no separators)
	serialNumber := system.FormatMAC(mac, system.MACFormatUSBSerial)
	return serialNumber
}
