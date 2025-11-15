package diskmanager

// UsbGadget defines the interface for managing USB gadgets
type UsbGadget interface {
	// Initialize sets up and activates the USB gadget
	Initialize() error

	// destroy deactivates and removes the USB gadget (private, called by Manager.Close)
	destroy()

	// Disconnect disconnects the USB gadget from the host without destroying the configuration
	Disconnect() error

	// Reconnect reconnects the USB gadget to the host
	Reconnect() error

	// IsConnected returns true if the USB gadget is currently connected to a host
	IsConnected() bool
}
