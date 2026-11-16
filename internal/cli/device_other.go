//go:build !linux && !windows

package cli

import (
	"fmt"
	"runtime"
)

func listScreens(sysfsRoot, devRoot string) ([]string, []string, error) {
	return nil, nil, unsupported()
}

func openTransport(sysfsRoot, devRoot, device string) (transport, screenPower, error) {
	return nil, nil, unsupported()
}

func unsupported() error {
	return fmt.Errorf("%s is not supported yet", runtime.GOOS)
}
