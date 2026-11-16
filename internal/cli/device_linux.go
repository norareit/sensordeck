//go:build linux

package cli

import "github.com/norareit/sensordeck/internal/hidraw"

// This file and device_other.go do nothing but choose the transport: they list the supported screens and open one.

func listScreens(sysfsRoot, devRoot string) ([]string, []string, error) {
	screens, err := hidraw.Discover(sysfsRoot, devRoot)
	if err != nil {
		return nil, nil, err
	}
	lines := make([]string, len(screens))
	for position, screen := range screens {
		lines[position] = screen.Line()
	}
	return lines, nil, nil
}

func openTransport(sysfsRoot, devRoot, device string) (transport, screenPower, error) {
	var opened *hidraw.Transport
	var err error
	if device != "" {
		opened, err = hidraw.OpenPath(sysfsRoot, devRoot, device)
	} else {
		opened, err = hidraw.OpenDiscovered(sysfsRoot, devRoot)
	}
	if err != nil {
		// Return a nil interface and not the nil pointer, so that the check of the error is the only check.
		return nil, nil, err
	}
	return opened, linuxPower{sysfsRoot: sysfsRoot, devRoot: devRoot, node: opened.Node}, nil
}

// linuxPower reads the power files of the USB device of the node that the transport has open now, at each call.
type linuxPower struct {
	sysfsRoot, devRoot string
	node               func() string
}

func (power linuxPower) timedOut(err error) bool { return hidraw.TimedOut(err) }

func (power linuxPower) recovery() (hangRecovery, bool) {
	device, found := hidraw.USBDevice(power.sysfsRoot, power.devRoot, power.node())
	if !found {
		return hangRecovery{}, false
	}
	delay, possible := hidraw.Recovery(device)
	if !possible {
		return hangRecovery{}, false
	}
	return hangRecovery{delay: delay, suspended: func() bool { return hidraw.Suspended(device) }}, true
}
