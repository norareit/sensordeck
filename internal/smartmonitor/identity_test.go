package smartmonitor

import "testing"

func TestUSBIdentity(test *testing.T) {
	if VendorID != 0x0483 {
		test.Errorf("vendor ID 0x%04x", VendorID)
	}
	if ProductID != 0x0065 {
		test.Errorf("product ID 0x%04x", ProductID)
	}
}
