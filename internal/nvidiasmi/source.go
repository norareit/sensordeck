// Package nvidiasmi reads the metrics of the first NVIDIA GPU from nvidia-smi, which the NVIDIA driver installs on
// Linux and on Windows.
package nvidiasmi

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

// TimeLimit leaves room for a slow first start of the driver and bounds a program that hangs.
const TimeLimit = 2 * time.Second

// pipeGrace is how long a run that was killed may keep a reading waiting for a child that holds its output open.
const pipeGrace = 500 * time.Millisecond

// gpuMetrics are the four fields that nvidia-smi is asked for, in the order of its output.
var gpuMetrics = [4]string{sensors.GPUTemperature, sensors.GPUUtilisation, sensors.GPUMemoryUsed, sensors.GPUMemoryTotal}

var gpuQueryArguments = []string{
	"--query-gpu=temperature.gpu,utilization.gpu,memory.used,memory.total",
	"--format=csv,noheader,nounits",
}

var gpuNumber = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// Source reads the metrics of the first NVIDIA GPU with one run of nvidia-smi for each reading.
type Source struct{ TimeLimit time.Duration }

func (source Source) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wantsAny(wanted) {
		return nil
	}
	limited, cancel := context.WithTimeout(stop, source.TimeLimit)
	defer cancel()
	// Made anew for every reading, so that PATH is searched anew and a program installed later is used.
	command := exec.CommandContext(limited, "nvidia-smi", gpuQueryArguments...)
	var output bytes.Buffer
	command.Stdout = &output
	command.WaitDelay = pipeGrace
	if err := command.Run(); err != nil {
		return nil
	}
	fields, ok := parseGPULine(output.String())
	if !ok {
		return nil
	}
	var readings []sensors.Reading
	for position, name := range gpuMetrics {
		if !wanted(name) || !gpuNumber.MatchString(fields[position]) {
			continue
		}
		if value, err := strconv.ParseFloat(fields[position], 64); err == nil {
			readings = append(readings, sensors.Reading{Name: name, Value: value})
		}
	}
	return readings
}

func wantsAny(wanted func(name string) bool) bool {
	for _, name := range gpuMetrics {
		if wanted(name) {
			return true
		}
	}
	return false
}

// parseGPULine splits the first line of the output at commas. The line ends at the first LF, and a CR right before it
// is not part of the line, because nvidia-smi on Windows may end its lines with CR LF. Any other count than four fields
// gives false.
func parseGPULine(output string) (fields [4]string, ok bool) {
	line, _, hasLineFeed := strings.Cut(output, "\n")
	if hasLineFeed {
		line = strings.TrimSuffix(line, "\r")
	}
	parts := strings.Split(line, ",")
	if len(parts) != len(fields) {
		return fields, false
	}
	for position, part := range parts {
		fields[position] = strings.Trim(part, " ")
	}
	return fields, true
}
