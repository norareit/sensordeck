//go:build !windows

package childprocess

import "os/exec"

// NoWindow does nothing on a system without console windows.
func NoWindow(command *exec.Cmd) {}
