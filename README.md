# Embroidery Buddy

A WiFi-enabled USB gadget for Raspberry Pi Zero W that emulates a USB flash drive for embroidery machines while providing a convenient web interface for file uploads.

## Overview

Embroidery Buddy eliminates the need to repeatedly swap USB drives when loading embroidery files onto your sewing machine. Simply plug the Raspberry Pi Zero W into your embroidery machine's USB port, and it will appear as a standard USB flash drive. Upload your embroidery files wirelessly through a web interface instead of physically removing and reinserting the USB drive.

### Use Case

Traditional workflow:
1. Save embroidery file to USB drive on computer
2. Remove USB drive from computer
3. Insert USB drive into embroidery machine
4. Load design
5. Remove USB drive from machine
6. Repeat for each new design

With Embroidery Buddy:
1. Plug Raspberry Pi Zero W into embroidery machine once
2. Upload files wirelessly via web browser
3. Load designs directly on the machine
4. No physical USB swapping required

## Features

- USB Mass Storage gadget emulation using Linux kernel's ConfigFS
- Web-based file upload interface accessible via WiFi
- Support for individual embroidery files or batch upload via ZIP files
- mDNS/Avahi service discovery for easy device access
- Automatic disk space management
- RESTful API for file operations
- Lightweight and optimized for Raspberry Pi Zero W

## Hardware Requirements

- Raspberry Pi Zero W (or Zero 2 W)
- microSD card (8GB or larger recommended)
- Micro USB cable (data-capable)
- Power source for the Raspberry Pi

## Software Stack

### Core Technologies

- **Language**: Go 1.25+
- **Operating System**: DietPi (Debian-based, lightweight Linux distribution)
- **USB Gadget**: Linux USB Gadget subsystem (ConfigFS)
- **Filesystem**: FAT32 (for embroidery machine compatibility)

### Key Dependencies

- **gorilla/mux**: HTTP router and URL matcher
- **rs/cors**: CORS middleware for cross-origin requests
- **diskfs/go-diskfs**: FAT32 filesystem manipulation
- **godbus/dbus**: D-Bus communication for Avahi/mDNS

### Components

- **USB Gadget Driver**: Configures the Raspberry Pi as a USB Mass Storage device
- **Disk Manager**: Manages the virtual disk image and filesystem operations
- **Web UI**: Provides the upload interface and API endpoints
- **mDNS Publisher**: Broadcasts the device on the local network for easy discovery

## Directory Structure

```
bernina-wifi/
├── cmd/
│   ├── embroidery-usbd/         # Main application entry point
│   ├── benchmark-copy/          # Performance testing utility
│   ├── get-mac/                 # Network MAC address utility
│   └── test-mdns/               # mDNS testing utility
├── internal/                    # Private application code
│   ├── config/                  # Configuration loading and parsing
│   ├── diskmanager/             # Virtual disk and USB gadget management
│   ├── mdns/                    # mDNS/Avahi service publishing
│   ├── system/                  # System utilities (network info)
│   └── webui/                   # Web interface and HTTP handlers
│       └── templates/           # HTML templates
├── scripts/                     # Deployment and setup scripts
│   ├── embroidery-usbd.service  # systemd service unit file
│   ├── usbgadget.sh             # USB gadget initialization script
│   ├── create-test-disk.sh      # Test disk image creation
│   ├── install-service.sh       # Service installation helper
│   └── run-benchmark.sh         # Benchmark runner
├── docs/                        # Documentation
│   └── configuration.md         # Configuration guide
├── build/                       # Build outputs (gitignored)
│   └── bin/
│       └── linux-arm/           # ARM binaries for Raspberry Pi
├── config.example.json          # Example configuration file
├── Makefile                     # Build automation
├── go.mod                       # Go module dependencies
└── README.md                    # This file
```

## Raspberry Pi Zero W Setup

Follow these steps to set up a Raspberry Pi Zero W with Embroidery Buddy:

### 1. Install DietPi

1. Download DietPi image for Raspberry Pi from [dietpi.com](https://dietpi.com)
2. Flash the image to your microSD card using Raspberry Pi Imager or Etcher
3. Boot the Raspberry Pi and complete initial setup

### 2. System Configuration

1. Configure time synchronization:
   ```bash
   dietpi-config
   # Select: Time sync option 4 - daemon + drift
   ```

2. Disable swap (to reduce SD card wear):
   ```bash
   dphys-swapfile swapoff
   dphys-swapfile uninstall
   systemctl disable dphys-swapfile
   ```

3. Enable Bluetooth (if needed):
   ```bash
   dietpi-config
   # Navigate to: Advanced Options -> Bluetooth -> Enable
   ```

### 3. Enable USB Gadget Support

1. Add USB gadget overlay to `/boot/config.txt`:
   ```bash
   echo "dtoverlay=dwc2,dr_mode=peripheral" >> /boot/config.txt
   ```

2. Load the `libcomposite` kernel module at boot:
   ```bash
   echo "libcomposite" > /etc/modules-load.d/libcomposite.conf
   ```

### 4. Install Embroidery Buddy

1. Create application directories:
   ```bash
   mkdir -p /opt/embroiderybuddy/bin
   mkdir -p /opt/embroiderybuddy/etc
   mkdir -p /var/lib/embroidery-usbd
   ```

2. Copy the systemd unit file:
   ```bash
   # From the scripts/ directory on your development machine
   scp scripts/embroidery-usbd.service root@dietpi.local:/etc/systemd/system/
   ```

3. Copy the example configuration:
   ```bash
   scp config.example.json root@dietpi.local:/opt/embroiderybuddy/etc/config.json
   ```

4. Edit the configuration as needed:
   ```bash
   nano /opt/embroiderybuddy/etc/config.json
   ```

5. Enable and start the service:
   ```bash
   systemctl daemon-reload
   systemctl enable embroidery-usbd.service
   systemctl start embroidery-usbd.service
   ```

6. Check service status:
   ```bash
   systemctl status embroidery-usbd.service
   ```

## Development

### Prerequisites

- Go 1.25 or later
- Make
- Cross-compilation tools for ARM (automatically handled by Go)

### Building

Build for all platforms:
```bash
make build-all
```

Build for Raspberry Pi only:
```bash
make build-rpi
```

Build and copy to Raspberry Pi (requires `dietpi.local` hostname):
```bash
make copy
```

This will:
1. Build the ARM binary
2. Copy it to `/opt/embroiderybuddy/bin/` on the Raspberry Pi via rsync

### Testing

Run unit tests:
```bash
make test
```

Run benchmarks:
```bash
make benchmark
```

### Running Locally (Development)

For development on macOS or Linux without USB gadget support:
```bash
make build
./build/bin/embroidery-usbd
```

The application will:
- Create a temporary disk image in `/tmp/embroidery.img`
- Use a no-op USB gadget implementation
- Start the web server on `http://localhost:80`

## Configuration

The application is configured via a JSON file. See [config.example.json](config.example.json) for all available options.

Key configuration sections:
- `server`: HTTP server settings (host, port, timeouts, CORS)
- `disk`: Virtual disk image settings (path, size, auto-creation)
- `usb_gadget`: USB device identification (vendor ID, product ID, device name)
- `mdns`: mDNS/Avahi service publishing settings
- `upload`: File upload limits

For detailed configuration documentation, see [docs/configuration.md](docs/configuration.md).

## Usage

### Accessing the Web Interface

Once the Raspberry Pi is running and connected to your WiFi network:

1. Open a web browser on any device connected to the same network
2. Navigate to `http://embroidery.local` (if mDNS is enabled)
3. Or use the Raspberry Pi's IP address: `http://192.168.1.xxx`

### Uploading Files

1. Click "Choose File" or drag and drop embroidery files
2. Supported formats: individual embroidery files or ZIP archives
3. ZIP files will be automatically extracted
4. Files appear immediately on the virtual USB drive

### Clearing Files

Use the "Clear All Files" button in the web interface to remove all files from the virtual disk.

## API Endpoints

- `GET /` - Web interface
- `POST /api/upload` - Upload embroidery files (accepts multipart/form-data)
- `POST /api/clear` - Clear all files from the disk
- `GET /api/health` - Health check endpoint

## Troubleshooting

### USB Gadget Not Appearing

1. Verify USB gadget kernel modules are loaded:
   ```bash
   lsmod | grep usb_f_mass_storage
   ```

2. Check systemd service logs:
   ```bash
   journalctl -u embroidery-usbd.service -f
   ```

3. Ensure the USB cable is data-capable (not power-only)

### Cannot Access Web Interface

1. Verify the service is running:
   ```bash
   systemctl status embroidery-usbd.service
   ```

2. Check network connectivity:
   ```bash
   ip addr show wlan0
   ```

3. Test mDNS resolution:
   ```bash
   avahi-browse -a
   ```

### Disk Full Errors

The virtual disk size is configured in `config.json`. To increase:
1. Stop the service
2. Delete the old disk image
3. Update `disk.size_mb` in the configuration
4. Restart the service (auto-create will make a new larger disk)

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## Acknowledgments

- Built with the excellent Go ecosystem
- Uses Linux USB Gadget subsystem for hardware emulation
- DietPi for providing a lightweight OS platform
