package diskmanager

// NoOpUsbGadget is a no-op implementation of UsbGadget for testing
type NoOpUsbGadget struct {
	connected bool
}

// NewNoOpUsbGadget creates a new no-op USB gadget implementation
func NewNoOpUsbGadget() *NoOpUsbGadget {
	return &NoOpUsbGadget{
		connected: false,
	}
}

// Initialize does nothing and always succeeds
func (g *NoOpUsbGadget) Initialize() error {
	g.connected = true
	return nil
}

// destroy does nothing (private method)
// Safe to call multiple times (idempotent)
func (g *NoOpUsbGadget) destroy() {
	if g.connected {
		g.connected = false
	}
}

// Disconnect simulates disconnecting the gadget
func (g *NoOpUsbGadget) Disconnect() error {
	g.connected = false
	return nil
}

// Reconnect simulates reconnecting the gadget
func (g *NoOpUsbGadget) Reconnect() error {
	g.connected = true
	return nil
}

// IsConnected returns the connection status
func (g *NoOpUsbGadget) IsConnected() bool {
	return g.connected
}
