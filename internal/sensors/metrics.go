// Package sensors is what every platform shares of reading hardware: the metric names and their units, the contract
// of a source, and the conversion of a reading to the 16 bits of a value update.
package sensors

import (
	"strconv"
	"strings"
)

// The eight metrics of a fixed name.
const (
	CPUTemperature = "cpu.temp"
	CPUUtilisation = "cpu.util"
	CPUClock       = "cpu.clock"
	GPUTemperature = "gpu.temp"
	GPUUtilisation = "gpu.util"
	GPUMemoryUsed  = "gpu.mem.used"
	GPUMemoryTotal = "gpu.mem.total"
	RAMUsed        = "ram.used"
	RAMTotal       = "ram.total"
)

// metricKind is one row of the table of metrics: a fixed name, or a numbered one made of prefix, number and suffix.
type metricKind struct {
	prefix, suffix string
	numbered       bool
	unit           string
}

// metricKinds is the one table of metrics, in the order in which readings are listed.
var metricKinds = []metricKind{
	{prefix: CPUTemperature, unit: "°C"},
	{prefix: "cpu.ccd", suffix: ".temp", numbered: true, unit: "°C"},
	{prefix: CPUUtilisation, unit: "%"},
	{prefix: CPUClock, unit: "MHz"},
	{prefix: GPUTemperature, unit: "°C"},
	{prefix: GPUUtilisation, unit: "%"},
	{prefix: GPUMemoryUsed, unit: "MiB"},
	{prefix: GPUMemoryTotal, unit: "MiB"},
	{prefix: RAMUsed, unit: "GiB"},
	{prefix: RAMTotal, unit: "GiB"},
	{prefix: "ram.", suffix: ".temp", numbered: true, unit: "°C"},
	{prefix: "nvme.", suffix: ".temp", numbered: true, unit: "°C"},
	{prefix: "commander.", suffix: ".temp", numbered: true, unit: "°C"},
}

// IsName tells whether name is a metric name: a fixed one, or a numbered one with a decimal number of 1 or more
// written without leading zeros. It does not tell whether a machine has a sensor for it.
func IsName(name string) bool {
	_, found := unitOf(name)
	return found
}

// Unit gives the unit of a metric name. For a string that is no metric name it gives the empty string.
func Unit(name string) string {
	unit, _ := unitOf(name)
	return unit
}

func unitOf(name string) (string, bool) {
	for _, kind := range metricKinds {
		if kind.matches(name) {
			return kind.unit, true
		}
	}
	return "", false
}

func (kind metricKind) matches(name string) bool {
	if !kind.numbered {
		return name == kind.prefix
	}
	number, found := strings.CutPrefix(name, kind.prefix)
	if !found {
		return false
	}
	number, found = strings.CutSuffix(number, kind.suffix)
	return found && isNumber(number)
}

// isNumber is true for one or more decimal digits, the first not 0.
func isNumber(text string) bool {
	if text == "" || text[0] == '0' {
		return false
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

// CCDTemperature is the name of the temperature of the CCD with the number.
func CCDTemperature(number int) string { return "cpu.ccd" + strconv.Itoa(number) + ".temp" }

// RAMTemperature is the name of the temperature of the memory module with the number.
func RAMTemperature(number int) string { return "ram." + strconv.Itoa(number) + ".temp" }

// NVMeTemperature is the name of the temperature of the drive with the number.
func NVMeTemperature(number int) string { return "nvme." + strconv.Itoa(number) + ".temp" }

// CommanderTemperature is the name of the temperature of the probe of the controller with the number, counted from 1.
func CommanderTemperature(number int) string { return "commander." + strconv.Itoa(number) + ".temp" }
