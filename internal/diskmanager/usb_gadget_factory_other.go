//go:build !linux

package diskmanager

import "log"

// newPlatformUsbGadget creates a NoOp USB gadget on non-Linux platforms
func newPlatformUsbGadget(config Config) UsbGadget {
	log.Printf("Warning: Linux USB gadget not available on this platform, using NoOp")
	return NewNoOpUsbGadget()
}
