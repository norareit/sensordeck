// Command generate writes the background of minimal-dark as a PNG to the path of its one argument.
package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/norareit/sensordeck/internal/builtinthemes/background"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "generate: the output path is the one argument")
		os.Exit(1)
	}
	if err := write(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		os.Exit(1)
	}
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
