// Command sensordeck drives a SmartMonitor screen.
package main

import (
	"os"

	"github.com/norareit/sensordeck/internal/cli"
)

// version is set by the release build with -X main.version, and is empty for any other build.
var version string

func main() {
	os.Exit(cli.Main(version))
}
