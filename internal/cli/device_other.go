//go:build !linux

package cli

import (
	"fmt"
	"runtime"
)

func listScreens(sysfsRoot, devRoot string) ([]string, error) {
	return nil, unsupported()
}

func openTransport(sysfsRoot, devRoot, device string) (transport, error) {
	return nil, unsupported()
}

func unsupported() error {
	return fmt.Errorf("%s is not supported yet", runtime.GOOS)
}
