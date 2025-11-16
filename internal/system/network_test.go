package system

import (
	"testing"
)

func TestGetMACAddress(t *testing.T) {
	// This test will work on any system with a loopback interface
	interfaces := []string{"lo", "lo0"} // lo on Linux, lo0 on macOS

	var mac string
	var err error
	for _, iface := range interfaces {
		mac, err = GetMACAddress(iface)
		if err == nil {
			break
		}
	}

	// Some systems don't have MAC addresses for loopback
	// So we just test that the function doesn't panic
	if err != nil {
		t.Logf("Loopback interface has no MAC (expected on some systems): %v", err)
	} else {
		t.Logf("Loopback MAC: %s", mac)
	}
}

func TestGetAllMACAddresses(t *testing.T) {
	macs, err := GetAllMACAddresses()
	if err != nil {
		t.Fatalf("Failed to get MAC addresses: %v", err)
	}

	if len(macs) == 0 {
		t.Error("Expected at least one interface with MAC address")
	}

	for iface, mac := range macs {
		t.Logf("Interface %s: %s", iface, mac)
	}
}

func TestFindWiFiInterface(t *testing.T) {
	// This test may not find a WiFi interface on all systems
	// So we just ensure it doesn't panic
	name, mac, err := FindWiFiInterface()
	if err != nil {
		t.Logf("No WiFi interface found (may be expected): %v", err)
	} else {
		t.Logf("WiFi interface found: %s with MAC %s", name, mac)
	}
}

func TestFormatMAC(t *testing.T) {
	testMAC := "aa:bb:cc:dd:ee:ff"

	tests := []struct {
		format   MACFormat
		expected string
	}{
		{MACFormatColon, "aa:bb:cc:dd:ee:ff"},
		{MACFormatHyphen, "aa-bb-cc-dd-ee-ff"},
		{MACFormatNone, "aabbccddeeff"},
		{MACFormatUSBSerial, "aabbccddeeff"},
	}

	for _, tt := range tests {
		result := FormatMAC(testMAC, tt.format)
		if result != tt.expected {
			t.Errorf("FormatMAC(%s, %v) = %s, want %s", testMAC, tt.format, result, tt.expected)
		}
	}
}

func TestFormatMACFromDifferentFormats(t *testing.T) {
	inputs := []string{
		"aa:bb:cc:dd:ee:ff",
		"aa-bb-cc-dd-ee-ff",
		"aabbccddeeff",
	}

	expected := "aabbccddeeff"

	for _, input := range inputs {
		result := FormatMAC(input, MACFormatNone)
		if result != expected {
			t.Errorf("FormatMAC(%s, MACFormatNone) = %s, want %s", input, result, expected)
		}
	}
}
