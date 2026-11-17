package cli

import (
	"github.com/norareit/sensordeck/internal/devicelock"
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

// screenLockName is the lock file of the screen: one lock serves every screen of the user.
const screenLockName = "screen.lock"

func openTransport(sysfsRoot, devRoot, procRoot, device string) (transport, screenRestarter, error) {
	release, err := devicelock.Acquire(screenLockName)
	if err != nil {
		return nil, nil, err
	}
	var opened *winhid.Transport
	if device != "" {
		opened, err = winhid.OpenPath(winhid.Windows(), device)
	} else {
		opened, err = winhid.OpenDiscovered(winhid.Windows())
	}
	if err != nil {
		release()
		// Return a nil interface and not the nil pointer, so that the check of the error is the only check.
		return nil, nil, err
	}
	return &heldTransport{transport: opened, release: release}, noScreenRestarter{}, nil
}
