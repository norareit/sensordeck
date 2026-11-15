package cli

import (
	"bytes"
	"os/exec"
)

// runSystemctl runs systemctl --user with the arguments and gives what it wrote to standard output and standard error,
// with the error of Run.
func runSystemctl(arguments ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	command := exec.Command("systemctl", append([]string{"--user"}, arguments...)...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return stdout.String(), stderr.String(), err
}
