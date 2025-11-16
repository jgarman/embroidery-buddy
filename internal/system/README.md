# System Package

This package provides system-level utilities for the embroidery-buddy application.

## Network Utilities

### Getting WiFi MAC Address

The network utilities provide functions to retrieve MAC addresses from network interfaces, particularly useful for getting the WiFi adapter's MAC address on Raspberry Pi devices.

#### Basic Usage

```go
import "github.com/jgarman/embroidery-buddy/internal/system"

// Get WiFi MAC address (looks for wlan0 by default)
mac, err := system.GetWiFiMACAddress()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("WiFi MAC: %s\n", mac)
```

#### Finding WiFi Interface

```go
// Automatically find WiFi interface
interfaceName, mac, err := system.FindWiFiInterface()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("WiFi interface %s has MAC: %s\n", interfaceName, mac)
```

#### Get Specific Interface

```go
// Get MAC address for a specific interface
mac, err := system.GetMACAddress("wlan0")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("wlan0 MAC: %s\n", mac)
```

#### List All Interfaces

```go
// Get all network interfaces and their MAC addresses
macs, err := system.GetAllMACAddresses()
if err != nil {
    log.Fatal(err)
}

for iface, mac := range macs {
    fmt.Printf("%s: %s\n", iface, mac)
}
```

### MAC Address Formatting

The package supports multiple MAC address formats:

```go
mac := "aa:bb:cc:dd:ee:ff"

// Colon-separated (default): aa:bb:cc:dd:ee:ff
formatted := system.FormatMAC(mac, system.MACFormatColon)

// Hyphen-separated: aa-bb-cc-dd-ee-ff
formatted = system.FormatMAC(mac, system.MACFormatHyphen)

// No separators: aabbccddeeff
formatted = system.FormatMAC(mac, system.MACFormatNone)

// USB serial format (12 hex chars): aabbccddeeff
formatted = system.FormatMAC(mac, system.MACFormatUSBSerial)
```

### Command-Line Tool

A command-line tool is provided to query MAC addresses:

```bash
# Find WiFi interface and show MAC
./get-mac

# List all interfaces
./get-mac --all

# Get specific interface
./get-mac -interface wlan0

# Format as USB serial number (no separators)
./get-mac -interface wlan0 -format usb

# Available formats: colon, hyphen, none, usb
./get-mac -format hyphen
```

## Raspberry Pi Zero W

On Raspberry Pi Zero W, the built-in WiFi adapter is typically named `wlan0`. The `FindWiFiInterface()` function will automatically detect it.

### USB Gadget Serial Number

The Linux USB gadget implementation automatically uses the WiFi MAC address as the USB serial number. This provides a unique identifier for each device:

- On Raspberry Pi Zero W with built-in WiFi, the MAC address from `wlan0` is used
- If no WiFi interface is found, a default serial number is used
- The MAC address is formatted as 12 hexadecimal characters (e.g., `b827ebcc658d`)

This is particularly useful for:
- Uniquely identifying devices when multiple are connected to the same host
- Device management and tracking
- Ensuring consistent device identification across reboots

## Error Handling

All functions return appropriate errors when:
- Interface does not exist
- Interface has no MAC address
- System permissions prevent access to network information

Always check error returns:

```go
mac, err := system.GetWiFiMACAddress()
if err != nil {
    // Handle error appropriately
    log.Printf("Could not get WiFi MAC: %v", err)
    // Use fallback or exit
}
```

## Platform Compatibility

The network utilities work on all platforms supported by Go's `net` package:
- Linux (including Raspberry Pi)
- macOS
- Windows
- BSD variants

Interface names vary by platform:
- Linux: `wlan0`, `wlan1`, `wlp2s0`, etc.
- macOS: `en0`, `en1`, etc.
- Windows: Network adapter names

The `FindWiFiInterface()` function handles common naming patterns automatically.
