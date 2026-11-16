package childprocess

import (
	"os/exec"
	"syscall"
)

// createNoWindow is the process creation flag CREATE_NO_WINDOW.
const createNoWindow = 0x08000000

// NoWindow makes the command start with no console window. The background run detaches from its console, and a
// console program started then would otherwise get a window of its own.
func NoWindow(command *exec.Cmd) {
	if command.SysProcAttr == nil {
		command.SysProcAttr = &syscall.SysProcAttr{}
	}
	command.SysProcAttr.CreationFlags |= createNoWindow
}
