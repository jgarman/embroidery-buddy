//go:build linux

package diskmanager

// newPlatformUsbGadget creates a Linux USB gadget implementation
func newPlatformUsbGadget(config Config) UsbGadget {
	return NewLinuxUsbGadget(config)
}
