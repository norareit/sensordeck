package cli

import (
	"github.com/norareit/sensordeck/internal/winhid"
)

// This file and device_linux.go do nothing but choose the transport: they list the supported screens and open one.
// On Windows the roots of the tests are not used.

func listScreens(sysfsRoot, devRoot string) ([]string, []string, error) {
	collections, err := winhid.List(winhid.Windows())
	if err != nil {
		return nil, nil, err
	}
	screens, nearMisses := winhid.Screens(collections)
	lines := make([]string, len(screens))
	for position, screen := range screens {
		lines[position] = screen.Line()
	}
	return lines, winhid.NearMissLines(nearMisses), nil
}

func openTransport(sysfsRoot, devRoot, device string) (transport, error) {
	var opened *winhid.Transport
	var err error
	if device != "" {
		opened, err = winhid.OpenPath(winhid.Windows(), device)
	} else {
		opened, err = winhid.OpenDiscovered(winhid.Windows())
	}
	if err != nil {
		// Return a nil interface and not the nil pointer, so that the check of the error is the only check.
		return nil, err
	}
	return opened, nil
}
