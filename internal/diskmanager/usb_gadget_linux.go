package diskmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	err := os.WriteFile(path, []byte(value), 0664)
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
	if err := os.MkdirAll(gadgetBase, desiredPermissions); err != nil {
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

	// TODO: Generate or retrieve actual serial number instead of hardcoded value
	if err := writeSysfs(filepath.Join(stringsDir, "serialnumber"), "b827ebcc658d"); err != nil {
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

	// Get UDC name for activation
	udcList, err := os.ReadFile("/sys/class/udc")
	if err != nil {
		return fmt.Errorf("failed to read UDC list: %w", err)
	}

	udcName := strings.TrimSpace(string(udcList))
	if udcName == "" {
		return fmt.Errorf("no UDC available")
	}

	// Get first UDC if multiple are available
	if idx := strings.Index(udcName, "\n"); idx != -1 {
		udcName = udcName[:idx]
	}

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
	if err := writeSysfs(udcPath, ""); err != nil {
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
