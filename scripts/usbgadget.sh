#!/bin/bash

# Wait for system initialization
sleep 5

# Remove existing gadget
if [ -d /sys/kernel/config/usb_gadget/embroiderybuddy ]; then
    echo "" > /sys/kernel/config/usb_gadget/embroiderybuddy/UDC 2>/dev/null
    rm -rf /sys/kernel/config/usb_gadget/embroiderybuddy
fi

modprobe libcomposite

cd /sys/kernel/config/usb_gadget/
mkdir -p embroiderybuddy
cd embroiderybuddy

# Configure gadget
echo 0x1d6b > idVendor
echo 0x0104 > idProduct
echo 0x0100 > bcdDevice
echo 0x0200 > bcdUSB

mkdir -p strings/0x409
echo "b827ebcc658d" > strings/0x409/serialnumber
echo "garman.group" > strings/0x409/manufacturer
echo "embroideryBuddy" > strings/0x409/product

mkdir -p configs/c.1/strings/0x409
echo "Mass Storage" > configs/c.1/strings/0x409/configuration
echo 250 > configs/c.1/MaxPower

# Mass Storage Function
mkdir -p functions/mass_storage.usb0
echo 1 > functions/mass_storage.usb0/stall
echo 0 > functions/mass_storage.usb0/lun.0/cdrom
echo 0 > functions/mass_storage.usb0/lun.0/ro
echo 0 > functions/mass_storage.usb0/lun.0/nofua
echo /home/dietpi/usbdiskimg.img > functions/mass_storage.usb0/lun.0/file

ln -s functions/mass_storage.usb0 configs/c.1/

# Activate gadget
ls /sys/class/udc > UDC 2>/dev/null || true