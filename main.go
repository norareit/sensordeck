// Command sensordeck drives a SmartMonitor screen.
package main

import (
	"os"

	"github.com/norareit/sensordeck/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
