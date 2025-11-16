package diskmanager

// NewUsbGadget creates the appropriate USB gadget implementation based on the platform
// and configuration. If useNoOp is true, it returns a NoOpUsbGadget regardless of platform.
func NewUsbGadget(config Config, useNoOp bool) UsbGadget {
	if useNoOp {
		return NewNoOpUsbGadget()
	}
	return newPlatformUsbGadget(config)
}
