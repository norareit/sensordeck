package childprocess

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestNoWindowSetsTheFlagOnACommandWithoutAttributes(test *testing.T) {
	command := exec.Command("cmd.exe")
	NoWindow(command)
	if command.SysProcAttr == nil || command.SysProcAttr.CreationFlags != createNoWindow {
		test.Fatalf("attributes = %+v", command.SysProcAttr)
	}
}

func TestNoWindowKeepsTheFlagsThatAreSet(test *testing.T) {
	command := exec.Command("cmd.exe")
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x200}
	NoWindow(command)
	if command.SysProcAttr.CreationFlags != 0x200|createNoWindow {
		test.Fatalf("flags = %#x", command.SysProcAttr.CreationFlags)
	}
}
