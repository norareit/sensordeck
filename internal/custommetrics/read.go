package custommetrics

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/hwmonchips"
)

func readFile(definition config.Metric) outcome {
	return readNumberFile(definition.File, definition)
}

// readHwmon reads the input of the chip at the place index of the chips of the name.
func readHwmon(sysfsRoot string, definition config.Metric) outcome {
	chips := hwmonchips.Named(sysfsRoot, definition.Hwmon)
	if len(chips) < definition.Index {
		return outcome{cause: fmt.Sprintf("%d hwmon chips named %s, fewer than index %d",
			len(chips), definition.Hwmon, definition.Index)}
	}
	return readNumberFile(filepath.Join(chips[definition.Index-1].Path, definition.Input), definition)
}

func readNumberFile(path string, definition config.Metric) outcome {
	content, err := os.ReadFile(path)
	if err != nil {
		return unreadable(path, err)
	}
	return fromText(string(content), path, definition)
}
