//go:build !linux && !windows

package custommetrics

import (
	"context"
	"os/exec"
)

// shellCommand runs the text with /bin/sh. A system that is neither Linux nor Windows is built and not run.
func shellCommand(limited context.Context, text string) *exec.Cmd {
	return exec.CommandContext(limited, "/bin/sh", "-c", text)
}
