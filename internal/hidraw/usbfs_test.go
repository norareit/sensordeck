//go:build linux

package hidraw

import (
	"errors"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUsbdevfsControlIsTheNumberOfTheKernel(test *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		test.Skip("the number is that of the 64-bit layout of the struct")
	}
	if usbdevfsControl != 0xc0185500 {
		test.Fatalf("got %#x", uint(usbdevfsControl))
	}
}

func TestSystemUsbfsOpenGivesAnErrorThatHoldsThePath(test *testing.T) {
	path := filepath.Join(test.TempDir(), "missing")
	_, err := SystemUsbfs{}.Open(path)
	if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), path) {
		test.Fatalf("got %v", err)
	}
}
