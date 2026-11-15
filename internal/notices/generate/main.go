// Command generate writes the notices file of the module in the working directory to the path of its one argument.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/norareit/sensordeck/internal/notices"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// run gives the exit status of the command for its arguments, and writes its error to stderr.
func run(arguments []string, stderr io.Writer) int {
	if len(arguments) != 1 {
		_, _ = fmt.Fprintln(stderr, "generate: the output path is the one argument")
		return 1
	}
	if err := write(arguments[0]); err != nil {
		_, _ = fmt.Fprintln(stderr, "generate:", err)
		return 1
	}
	return 0
}

// write makes the text before it touches the path, so that a failure leaves an existing file as it was.
func write(path string) error {
	text, err := notices.Generate(".")
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}
