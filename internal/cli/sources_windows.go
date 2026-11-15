package cli

import "github.com/norareit/sensordeck/internal/commandercore"

// This file and sources_linux.go do nothing but choose the sources of the readings. The sensors of the machine are
// not read on Windows yet, so the Commander is the only source, and the roots of the tests are not used.

func newSources(sysfsRoot, devRoot, procRoot string) (sourceSet, error) {
	return commanderSources(commandercore.Windows()), nil
}
