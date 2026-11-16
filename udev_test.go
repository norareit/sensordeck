package main

import (
	"os"
	"strings"
	"testing"
)

const udevRulesPath = "udev/70-sensordeck.rules"

// TestTheUdevRuleHoldsTheHidrawRulesAndTheUsbfsRule checks the lines whole, so that a rule loosened by an edit
// fails here and not on a user's machine.
func TestTheUdevRuleHoldsTheHidrawRulesAndTheUsbfsRule(test *testing.T) {
	content, err := os.ReadFile(udevRulesPath)
	if err != nil {
		test.Fatal(err)
	}
	lines := strings.Split(string(content), "\n")
	wanted := []string{
		`SUBSYSTEM=="hidraw", ATTRS{idVendor}=="0483", ATTRS{idProduct}=="0065", MODE="0660", TAG+="uaccess"`,
		`SUBSYSTEM=="hidraw", ATTRS{idVendor}=="1b1c", ATTRS{idProduct}=="0c2a", MODE="0660", TAG+="uaccess"`,
		`SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTR{idVendor}=="0483", ATTR{idProduct}=="0065", MODE="0660", TAG+="uaccess"`,
	}
	for _, wantedLine := range wanted {
		if !containsLine(lines, wantedLine) {
			test.Errorf("%s has no line %q", udevRulesPath, wantedLine)
		}
	}
	for _, line := range lines {
		for _, forbidden := range []string{"power/control", "power/autosuspend_delay_ms"} {
			if strings.Contains(line, forbidden) {
				test.Errorf("%s holds a line that sets %s: %q", udevRulesPath, forbidden, line)
			}
		}
	}
}

func containsLine(lines []string, wanted string) bool {
	for _, line := range lines {
		if line == wanted {
			return true
		}
	}
	return false
}
