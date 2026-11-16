//go:build linux

// Package linuxsensors reads hardware readings on Linux: hwmon, cpufreq and /proc. Each source reads its
// files with os.ReadFile and hands the content to a function that only parses it.
package linuxsensors

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/norareit/sensordeck/internal/hwmonchips"
	"github.com/norareit/sensordeck/internal/sensors"
)

// entriesNamed gives the chips with the driver name that have a device, ordered by the resolved device path and,
// between equal paths, by the hwmon number. The numbers follow the order in which the drivers load and can change
// after a reboot, and the device paths follow the hardware, which is what keeps numbered metrics stable.
func entriesNamed(sysfsRoot, driver string) []hwmonchips.Chip {
	var entries []hwmonchips.Chip
	for _, chip := range hwmonchips.Named(sysfsRoot, driver) {
		if chip.Device != "" {
			entries = append(entries, chip)
		}
	}
	return entries
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

// labelled is a temperature sensor of an entry that has a label.
type labelled struct {
	number int
	label  string
}

// labelledSensors gives the sensors of the entry that have a readable label, ordered by their number.
func labelledSensors(entryPath string) []labelled {
	files, err := filepath.Glob(filepath.Join(entryPath, "temp*_label"))
	if err != nil {
		return nil
	}
	var sensorsFound []labelled
	for _, file := range files {
		digits := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(file), "temp"), "_label")
		number, err := strconv.Atoi(digits)
		content, readErr := os.ReadFile(file)
		if !isDigits(digits) || err != nil || readErr != nil {
			continue
		}
		sensorsFound = append(sensorsFound, labelled{number: number, label: strings.TrimSuffix(string(content), "\n")})
	}
	slices.SortFunc(sensorsFound, func(first, second labelled) int { return first.number - second.number })
	return sensorsFound
}

func inputPath(entryPath string, number int) string {
	return filepath.Join(entryPath, "temp"+strconv.Itoa(number)+"_input")
}

// readTemperature reads the input of a sensor in °C. An empty path, a missing file and a malformed input give false.
func readTemperature(path string) (float64, bool) {
	if path == "" {
		return 0, false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	return parseMillidegrees(string(content))
}

// parseMillidegrees reads a decimal integer, possibly negative, in millidegrees. A plus sign is refused.
func parseMillidegrees(content string) (float64, bool) {
	text := strings.TrimSuffix(content, "\n")
	digits := strings.TrimPrefix(text, "-")
	if !isDigits(digits) {
		return 0, false
	}
	millidegrees, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, false
	}
	return float64(millidegrees) / 1000, true
}

// cpuTemperatures reads the first k10temp entry: Tctl as cpu.temp and each Tccd<n> as cpu.ccd<n>.temp.
type cpuTemperatures struct{ sysfsRoot string }

func (source cpuTemperatures) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	entries := entriesNamed(source.sysfsRoot, "k10temp")
	if len(entries) == 0 {
		return nil
	}
	var control float64
	var haveControl bool
	ccds := map[int]float64{}
	seen := map[string]bool{}
	for _, sensor := range labelledSensors(entries[0].Path) {
		// The sensor with the lower number counts, even when its input cannot be read.
		if seen[sensor.label] {
			continue
		}
		seen[sensor.label] = true
		isControl := sensor.label == "Tctl" && wanted(sensors.CPUTemperature)
		number, isCCD := ccdNumber(sensor.label)
		isCCD = isCCD && wanted(sensors.CCDTemperature(number))
		if !isControl && !isCCD {
			continue
		}
		value, ok := readTemperature(inputPath(entries[0].Path, sensor.number))
		if !ok {
			continue
		}
		if isControl {
			control, haveControl = value, true
		}
		if isCCD {
			ccds[number] = value
		}
	}
	var readings []sensors.Reading
	if haveControl {
		readings = append(readings, sensors.Reading{Name: sensors.CPUTemperature, Value: control})
	}
	numbers := make([]int, 0, len(ccds))
	for number := range ccds {
		numbers = append(numbers, number)
	}
	slices.Sort(numbers)
	for _, number := range numbers {
		readings = append(readings, sensors.Reading{Name: sensors.CCDTemperature(number), Value: ccds[number]})
	}
	return readings
}

// numberedTemperatures reads the entries of one driver, each as a metric numbered by its position among them.
type numberedTemperatures struct {
	sysfsRoot string
	driver    string
	name      func(number int) string
	input     func(entryPath string) string // the path of the input of the entry, or empty when it has none
}

func (source numberedTemperatures) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	var readings []sensors.Reading
	// The number is the position before any input is read, so that a drive whose sensor fails keeps its number.
	for position, each := range entriesNamed(source.sysfsRoot, source.driver) {
		name := source.name(position + 1)
		if !wanted(name) {
			continue
		}
		if value, ok := readTemperature(source.input(each.Path)); ok {
			readings = append(readings, sensors.Reading{Name: name, Value: value})
		}
	}
	return readings
}

func memoryModules(sysfsRoot string) numberedTemperatures {
	return numberedTemperatures{
		sysfsRoot: sysfsRoot,
		driver:    "spd5118",
		name:      sensors.RAMTemperature,
		input:     func(entryPath string) string { return inputPath(entryPath, 1) },
	}
}

func drives(sysfsRoot string) numberedTemperatures {
	return numberedTemperatures{
		sysfsRoot: sysfsRoot,
		driver:    "nvme",
		name:      sensors.NVMeTemperature,
		input: func(entryPath string) string {
			for _, sensor := range labelledSensors(entryPath) {
				if sensor.label == "Composite" {
					return inputPath(entryPath, sensor.number)
				}
			}
			return ""
		},
	}
}

// ccdNumber gives n of a label Tccd<n> when cpu.ccd<n>.temp is a metric name.
func ccdNumber(label string) (int, bool) {
	digits, found := strings.CutPrefix(label, "Tccd")
	if !found || !isDigits(digits) {
		return 0, false
	}
	number, err := strconv.Atoi(digits)
	return number, err == nil && sensors.IsName(sensors.CCDTemperature(number))
}
