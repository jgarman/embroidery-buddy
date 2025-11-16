# mDNS/Avahi Service Discovery

This package provides mDNS (multicast DNS) service discovery using Avahi on Linux systems. This allows devices to discover your embroidery server on the local network without needing to know the IP address.

## Overview

The embroidery-buddy server can advertise itself on the local network using mDNS/Avahi, making it discoverable as `embroidery-buddy.local` (or any custom name you configure).

Users can simply navigate to `http://embroidery-buddy.local:8080` instead of needing to know the IP address.

## How It Works

On Raspberry Pi and other Linux systems with Avahi installed, the server will:
1. Register itself with the Avahi daemon
2. Advertise an HTTP service (`_http._tcp`)
3. Include metadata in TXT records (path, version, etc.)
4. Be discoverable by mDNS clients (browsers, mobile apps, etc.)

## Methods

### DBus API (Recommended)

The DBus method communicates directly with the Avahi daemon via D-Bus. This is the most reliable method and doesn't require spawning external processes.

```go
publisher, err := mdns.PublishHTTPDBus("Embroidery Buddy", 8080, "path=/", "version=1.0")
if err != nil {
    log.Fatal(err)
}
defer publisher.Stop()
```

**Advantages:**
- More reliable than command-line
- Better error handling
- No external process management
- Works even if avahi-utils not installed

### Command-Line Method

Uses the `avahi-publish-service` command-line tool.

```go
publisher, err := mdns.PublishHTTP("Embroidery Buddy", 8080, "path=/")
if err != nil {
    log.Fatal(err)
}
defer publisher.Stop()
```

**Requirements:**
- `avahi-utils` package must be installed
- `avahi-publish-service` must be in PATH

## Installation

### Raspberry Pi / Debian / Ubuntu

```bash
# Install Avahi daemon and utilities
sudo apt-get update
sudo apt-get install avahi-daemon avahi-utils

# Enable and start the service
sudo systemctl enable avahi-daemon
sudo systemctl start avahi-daemon
```

### Checking Availability

```go
// Check if DBus API is available (recommended)
if mdns.IsAvahiDBusAvailable() {
    log.Println("Avahi DBus is available")
}

// Check if command-line tools are available
if mdns.IsAvahiAvailable() {
    log.Println("Avahi command-line tools are available")
}
```

## Configuration

In your config.json:

```json
{
  "mdns": {
    "enabled": true,
    "service_name": "Embroidery Buddy",
    "use_dbus": true,
    "txt_records": [
      "path=/",
      "version=1.0",
      "model=Raspberry Pi Zero W"
    ]
  }
}
```

### Configuration Options

- **enabled**: Enable/disable mDNS advertisement
- **service_name**: Name shown in service discovery (e.g., "Embroidery Buddy")
- **use_dbus**: Use DBus API instead of command-line (recommended)
- **txt_records**: Additional metadata as key=value pairs

## Usage Examples

### Basic HTTP Service

```go
package main

import (
    "log"
    "net/http"
    "github.com/jgarman/embroidery-buddy/internal/mdns"
)

func main() {
    // Start HTTP server
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello!"))
    })
    go http.ListenAndServe(":8080", nil)

    // Publish with Avahi
    publisher, err := mdns.PublishHTTPDBus("My Service", 8080)
    if err != nil {
        log.Fatal(err)
    }
    defer publisher.Stop()

    // Service is now available at http://my-service.local:8080
    select {} // Keep running
}
```

### With TXT Records

```go
publisher, err := mdns.PublishHTTPDBus(
    "Embroidery Buddy",
    8080,
    "path=/",
    "version=1.0",
    "api=v1",
    "secure=false",
)
```

### Custom Service Type

```go
service := &mdns.Service{
    Name:   "Custom Service",
    Type:   "_custom._tcp",
    Port:   9090,
    Domain: "local",
    TXTRecords: []string{"key=value"},
}

publisher := mdns.NewDBusPublisher()
err := publisher.PublishService(service)
```

## Service Discovery

Clients can discover your service using:

### Command Line (Linux)

```bash
# Browse for HTTP services
avahi-browse -t _http._tcp

# Resolve a specific service
avahi-resolve -n embroidery-buddy.local
```

### Web Browser

Modern browsers support mDNS:
- Navigate to `http://embroidery-buddy.local:8080`
- Works in Chrome, Firefox, Safari

### Mobile Apps

iOS and Android apps can use Bonjour/Zeroconf APIs:
- iOS: NSNetService / Bonjour
- Android: NsdManager

### Python

```python
from zeroconf import ServiceBrowser, Zeroconf

class MyListener:
    def add_service(self, zc, type_, name):
        info = zc.get_service_info(type_, name)
        print(f"Service {name} at {info.server}:{info.port}")

zeroconf = Zeroconf()
listener = MyListener()
browser = ServiceBrowser(zeroconf, "_http._tcp.local.", listener)
```

## Testing

Test mDNS functionality:

```bash
# Build test tool
go build ./cmd/test-mdns

# Check if Avahi is available
./test-mdns --check

# Publish a test service
./test-mdns -name "Test Service" -port 8080

# In another terminal, browse for the service
avahi-browse -t _http._tcp
```

## Troubleshooting

### Service Not Appearing

1. **Check Avahi is running:**
   ```bash
   systemctl status avahi-daemon
   ```

2. **Check firewall:**
   ```bash
   # mDNS uses UDP port 5353
   sudo ufw allow 5353/udp
   ```

3. **Test with avahi-browse:**
   ```bash
   avahi-browse -a  # Browse all services
   ```

### DBus Permission Errors

If you get permission errors with DBus:

```bash
# Add user to avahi group
sudo usermod -a -G avahi $USER

# Or run with appropriate permissions
sudo ./embroidery-usbd
```

### Multiple Network Interfaces

Avahi advertises on all network interfaces by default. To limit:

```bash
# Edit /etc/avahi/avahi-daemon.conf
[server]
allow-interfaces=wlan0
```

## Platform Support

- **Linux**: Full support via Avahi
- **macOS**: Bonjour (Apple's mDNS) - use different library
- **Windows**: Bonjour Service (if installed) - use different library

This package is Linux/Avahi specific. For cross-platform mDNS, consider:
- [github.com/hashicorp/mdns](https://github.com/hashicorp/mdns)
- [github.com/grandcat/zeroconf](https://github.com/grandcat/zeroconf)

## Security Considerations

- mDNS works on local network only (multicast packets don't route)
- Service names are public on the local network
- Consider using HTTPS for sensitive data
- TXT records are visible to all network clients
- No authentication in mDNS itself - implement at application level

## References

- [Avahi Documentation](https://www.avahi.org/)
- [RFC 6762 - mDNS](https://tools.ietf.org/html/rfc6762)
- [RFC 6763 - DNS-SD](https://tools.ietf.org/html/rfc6763)
- [Avahi DBus API](https://www.avahi.org/doxygen/html/)
