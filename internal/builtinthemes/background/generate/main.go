// Command generate writes the background of minimal-dark as a PNG to the path of its one argument.
package main

import (
	"fmt"
	"image/png"
	"io"
	"os"

	"github.com/norareit/sensordeck/internal/builtinthemes/background"
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

func write(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, background.MinimalDark()); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
