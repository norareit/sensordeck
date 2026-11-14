//go:build linux

package cli

import "github.com/norareit/sensordeck/internal/hidraw"

// This file and device_other.go do nothing but choose the transport: they list the supported screens and open one.

func listScreens(sysfsRoot, devRoot string) ([]string, error) {
	screens, err := hidraw.Discover(sysfsRoot, devRoot)
	if err != nil {
		return nil, err
	}
	lines := make([]string, len(screens))
	for position, screen := range screens {
		lines[position] = screen.Line()
	}
	return lines, nil
}

func openTransport(sysfsRoot, devRoot, device string) (transport, error) {
	var opened *hidraw.Transport
	var err error
	if device != "" {
		opened, err = hidraw.OpenPath(device)
	} else {
		opened, err = hidraw.OpenDiscovered(sysfsRoot, devRoot)
	}
	if err != nil {
		// Return a nil interface and not the nil pointer, so that the check of the error is the only check.
		return nil, err
	}
	return opened, nil
}
