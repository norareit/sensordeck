//go:build linux

package cli

import (
	"context"
	"time"

	"github.com/norareit/sensordeck/internal/hidraw"
)

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

func openTransport(sysfsRoot, devRoot, device string) (transport, screenRestarter, error) {
	var opened *hidraw.Transport
	var err error
	if device != "" {
		opened, err = hidraw.OpenPath(device)
	} else {
		opened, err = hidraw.OpenDiscovered(sysfsRoot, devRoot)
	}
	if err != nil {
		// Return a nil interface and not the nil pointer, so that the check of the error is the only check.
		return nil, nil, err
	}
	return opened, linuxRestarter{sysfsRoot: sysfsRoot, devRoot: devRoot, screen: opened}, nil
}

// linuxRestarter restarts the screen of the transport, through the USB device of the node that the transport has open
// at the call.
type linuxRestarter struct {
	sysfsRoot, devRoot string
	screen             *hidraw.Transport
}

func (restarter linuxRestarter) timedOut(err error) bool { return hidraw.TimedOut(err) }

func (restarter linuxRestarter) restart(stop context.Context) restartResult {
	result := hidraw.Restart{
		SysfsRoot: restarter.sysfsRoot,
		DevRoot:   restarter.devRoot,
		Node:      restarter.screen.Node(),
		Usbfs:     hidraw.SystemUsbfs{},
		Screen:    restarter.screen,
		Now:       time.Now,
		Wait:      waitFor,
	}.Run(stop)
	outcomes := map[hidraw.RestartOutcome]restartOutcome{
		hidraw.RestartNotPossible: restartNotPossible,
		hidraw.NotRestarted:       notRestarted,
		hidraw.RestartedOpen:      restartedOpen,
		hidraw.RestartedNotOpen:   restartedNotOpen,
	}
	return restartResult{outcome: outcomes[result.Outcome], index: result.Index, err: result.Err}
}
