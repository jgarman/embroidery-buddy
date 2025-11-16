# Configuration Guide

The embroidery-usbd server can be configured using a JSON configuration file or command-line flags.

## Command-Line Flags

```bash
./embroidery-usbd [flags]
```

### Flags

- `-config <path>` - Path to JSON configuration file (optional)
- `-generate-config` - Generate an example configuration file and exit

## Configuration File

### Generating a Configuration File

To generate an example configuration file:

```bash
./embroidery-usbd --generate-config -config myconfig.json
```

This creates a file with all default values that you can customize.

### Configuration Structure

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "read_timeout": 15,
    "write_timeout": 15,
    "idle_timeout": 60,
    "cors": {
      "allowed_origins": ["*"],
      "allowed_methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
      "allowed_headers": ["*"],
      "allow_credentials": true
    }
  },
  "disk": {
    "path": "/var/lib/embroidery-buddy/disk.img",
    "size_mb": 100,
    "auto_create": true
  },
  "usb_gadget": {
    "short_name": "embroidery",
    "vendor_id": "0x1d6b",
    "product_id": "0x0104",
    "bcd_device": "0x0100",
    "bcd_usb": "0x0200",
    "product_name": "Embroidery USB Storage",
    "manufacturer": "Embroidery Buddy",
    "use_noop": false
  },
  "upload": {
    "max_size_mb": 100
  }
}
```

### Configuration Options

#### Server Configuration

- **host** - Network interface to bind to (default: `0.0.0.0` for all interfaces)
- **port** - HTTP port to listen on (default: `8080`)
- **read_timeout** - Maximum duration for reading entire request in seconds (default: `15`)
- **write_timeout** - Maximum duration before timing out writes in seconds (default: `15`)
- **idle_timeout** - Maximum amount of time to wait for next request in seconds (default: `60`)

#### CORS Configuration

- **allowed_origins** - List of allowed origins (default: `["*"]` for all)
- **allowed_methods** - Allowed HTTP methods
- **allowed_headers** - Allowed headers
- **allow_credentials** - Whether to allow credentials (default: `true`)

#### Disk Configuration

- **path** - Path to the disk image file
- **size_mb** - Size of the disk image in megabytes (used when creating new disk)
- **auto_create** - Automatically create disk image if it doesn't exist (default: `true`)

#### USB Gadget Configuration

- **short_name** - Short name for the USB gadget (used in configfs path)
- **vendor_id** - USB vendor ID in hex format (e.g., `"0x1d6b"`)
- **product_id** - USB product ID in hex format (e.g., `"0x0104"`)
- **bcd_device** - Device version in BCD format (e.g., `"0x0100"` for version 1.0)
- **bcd_usb** - USB specification version (e.g., `"0x0200"` for USB 2.0)
- **product_name** - Product name string shown to USB host
- **manufacturer** - Manufacturer name string shown to USB host
- **use_noop** - Use No-Op gadget for testing/development (default: `false`)

#### Upload Configuration

- **max_size_mb** - Maximum upload size in megabytes (default: `100`)

## Examples

### Development Configuration

```json
{
  "server": {
    "host": "127.0.0.1",
    "port": 8080
  },
  "disk": {
    "path": "/tmp/embroidery-dev.img",
    "size_mb": 50,
    "auto_create": true
  },
  "usb_gadget": {
    "use_noop": true
  }
}
```

### Production Configuration

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 80,
    "cors": {
      "allowed_origins": ["https://my-embroidery-app.com"]
    }
  },
  "disk": {
    "path": "/var/lib/embroidery-buddy/disk.img",
    "size_mb": 500,
    "auto_create": true
  },
  "usb_gadget": {
    "short_name": "embroidery",
    "vendor_id": "0x1d6b",
    "product_id": "0x0104",
    "product_name": "Embroidery Storage",
    "manufacturer": "My Company",
    "use_noop": false
  },
  "upload": {
    "max_size_mb": 200
  }
}
```

## Running with Configuration

```bash
# Use a specific configuration file
./embroidery-usbd -config /etc/embroidery-buddy/config.json

# Use default configuration
./embroidery-usbd
```

## Platform Notes

### Linux
On Linux systems, the server can use the real USB gadget implementation via configfs. Ensure you have:
- USB gadget support in the kernel
- configfs mounted at `/sys/kernel/config`
- Appropriate permissions to create USB gadgets

### Other Platforms
On non-Linux platforms (macOS, Windows), the `use_noop` option is automatically enabled, and the USB gadget functionality is simulated. This is useful for development and testing the web interface.

## Default Behavior

When no configuration file is specified:
- Server binds to `0.0.0.0:8080`
- Disk image is created at `/tmp/embroidery.img` (development mode)
- USB gadget uses No-Op mode for compatibility
- All other settings use documented defaults
