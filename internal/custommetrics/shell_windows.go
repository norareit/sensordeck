package custommetrics

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/norareit/sensordeck/internal/childprocess"
)

// shellCommand runs the text with the program of ComSpec, or cmd.exe. The command line is given as it is, because the
// quoting that Go applies to an argument is not the one cmd.exe reads, and /s keeps the quotes of the user.
func shellCommand(limited context.Context, text string) *exec.Cmd {
	program := os.Getenv("ComSpec")
	if program == "" {
		program = "cmd.exe"
	}
	command := exec.CommandContext(limited, program)
	command.SysProcAttr = &syscall.SysProcAttr{CmdLine: `"` + program + `" /d /s /c "` + text + `"`}
	childprocess.NoWindow(command)
	return command
}
