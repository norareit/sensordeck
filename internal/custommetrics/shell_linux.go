package custommetrics

import (
	"context"
	"os/exec"
	"syscall"
)

// shellCommand runs the text with /bin/sh, by its absolute path because the PATH of a service may be short. The
// command is the leader of a process group that the time limit kills whole, so that the programs of a pipeline end
// with the shell.
func shellCommand(limited context.Context, text string) *exec.Cmd {
	command := exec.CommandContext(limited, "/bin/sh", "-c", text)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return syscall.Kill(-command.Process.Pid, syscall.SIGKILL) }
	return command
}
