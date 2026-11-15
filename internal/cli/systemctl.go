package cli

import (
	"bytes"
	"os/exec"
)

// runProgram runs the program with the arguments and gives what it wrote to standard output and standard error, with
// the error of Run.
func runProgram(name string, arguments ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	command := exec.Command(name, arguments...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return stdout.String(), stderr.String(), err
}

// runSystemctl runs systemctl --user with the arguments.
func runSystemctl(arguments ...string) (string, string, error) {
	return runProgram("systemctl", append([]string{"--user"}, arguments...)...)
}
