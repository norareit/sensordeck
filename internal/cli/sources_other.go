//go:build !linux

package cli

import "github.com/norareit/sensordeck/internal/sensors"

func newSources(sysfsRoot, procRoot string) ([]sensors.Source, error) {
	return nil, unsupported()
}
