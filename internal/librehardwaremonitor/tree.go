// Package librehardwaremonitor reads the metrics that Windows does not give a program without a kernel driver from the
// JSON tree that the web server of LibreHardwareMonitor serves. It is not limited to a platform.
package librehardwaremonitor

import (
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/norareit/sensordeck/internal/sensors"
)

// MaxNumber is the highest number of a CCD, a module or a drive that is given, so that a reading can tell with a
// bounded number of questions to wanted whether it needs the tree.
const MaxNumber = 64

// The two failures of a body that is no tree.
var (
	ErrNotJSON = errors.New("the answer is not JSON")
	ErrNotTree = errors.New("the answer is not the tree of data.json")
)

// sensorNode is a node of the document that is a sensor, with the fields the mapping relies on.
type sensorNode struct {
	id, hardware, kind, text, value string
}

var (
	// valuePattern is a number and a unit, as LibreHardwareMonitor formats a value with the culture of the user. The
	// group separator of a culture is not read, because the format of a sensor writes none.
	valuePattern   = regexp.MustCompile(`^([-−]?)([0-9]+)(?:[.,]([0-9]+))? +(°C|°F|MHz)$`)
	ccdPattern     = regexp.MustCompile(`^CCD([1-9][0-9]*) \(Tdie\)$`)
	coreClock      = regexp.MustCompile(`^(?:CPU Core|(?:P-|E-)?Core #[0-9]+)$`)
	driveHardware  = regexp.MustCompile(`^/nvme/([0-9]+)$`)
	moduleHardware = regexp.MustCompile(`^/memory/dimm/([0-9]+)$`)
	decimalDigits  = regexp.MustCompile(`^[0-9]+$`)
)

// Map gives the readings of the monitor metrics that a body of data.json holds, in the order of the table of metrics
// and, within a numbered row, by number. A body that is no tree is a failure; a tree that holds none of the metrics
// gives no reading and no error. It does no input and output.
func Map(body []byte) ([]sensors.Reading, error) {
	var document any
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, ErrNotJSON
	}
	root, isObject := document.(map[string]any)
	if !isObject {
		return nil, ErrNotTree
	}
	children, isArray := root["Children"].([]any)
	if !isArray {
		return nil, ErrNotTree
	}
	var found []sensorNode
	for _, child := range children {
		found = collectSensors(child, found)
	}
	return readingsOf(found), nil
}

// collectSensors walks a node and its children depth first, in the order of the document. A node of an unexpected
// shape is passed over, and its children are still searched, so that a field that a later version changes costs that
// field only.
func collectSensors(node any, found []sensorNode) []sensorNode {
	object, isObject := node.(map[string]any)
	if !isObject {
		return found
	}
	if sensor, isSensor := asSensor(object); isSensor {
		found = append(found, sensor)
	}
	children, _ := object["Children"].([]any)
	for _, child := range children {
		found = collectSensors(child, found)
	}
	return found
}

func asSensor(object map[string]any) (sensorNode, bool) {
	id, idIsText := object["SensorId"].(string)
	kind, kindIsText := object["Type"].(string)
	text, textIsText := object["Text"].(string)
	value, valueIsText := object["Value"].(string)
	if !idIsText || !kindIsText || !textIsText || !valueIsText {
		return sensorNode{}, false
	}
	segments := strings.Split(id, "/")
	// The first segment is the empty text before the leading slash.
	if len(segments) < 4 || segments[0] != "" || !decimalDigits.MatchString(segments[len(segments)-1]) {
		return sensorNode{}, false
	}
	hardware := strings.Join(segments[:len(segments)-2], "/")
	return sensorNode{id: id, hardware: hardware, kind: kind, text: text, value: value}, true
}

// valueOf reads the value of a sensor by its type. A temperature is in °C or °F, and is given in °C. A clock is in MHz.
func (sensor sensorNode) valueOf() (float64, bool) {
	parts := valuePattern.FindStringSubmatch(strings.Trim(sensor.value, " "))
	if parts == nil {
		return 0, false
	}
	number := parts[2]
	if parts[3] != "" {
		number += "." + parts[3]
	}
	value, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, false
	}
	if parts[1] != "" {
		value = -value
	}
	switch {
	case sensor.kind == "Temperature" && parts[4] == "°C":
		return value, true
	case sensor.kind == "Temperature" && parts[4] == "°F":
		return (value - 32) / 1.8, true
	case sensor.kind == "Clock" && parts[4] == "MHz":
		return value, true
	}
	return 0, false
}

func readingsOf(found []sensorNode) []sensors.Reading {
	var readings []sensors.Reading
	readings = appendControlTemperature(readings, found)
	readings = appendCCDTemperatures(readings, found)
	readings = appendClock(readings, found)
	readings = appendModuleTemperatures(readings, found)
	readings = appendDriveTemperatures(readings, found)
	return readings
}

// cpuSensors are the sensors of the CPU with the type. The CPU is the AMD one when the document has a sensor of it and
// otherwise the Intel one.
func cpuSensors(found []sensorNode, kind string) []sensorNode {
	cpu := "/intelcpu/0"
	if slices.ContainsFunc(found, func(sensor sensorNode) bool { return sensor.hardware == "/amdcpu/0" }) {
		cpu = "/amdcpu/0"
	}
	var result []sensorNode
	for _, sensor := range found {
		if sensor.hardware == cpu && sensor.kind == kind {
			result = append(result, sensor)
		}
	}
	return result
}

// appendControlTemperature adds cpu.temp, which is Tctl as on Linux: the sensor with the first of the three names that
// the CPU has. The sensor is chosen by its name and not by its value, so that a missing value does not fall through to
// another sensor.
func appendControlTemperature(readings []sensors.Reading, found []sensorNode) []sensors.Reading {
	temperatures := cpuSensors(found, "Temperature")
	for _, name := range []string{"Core (Tctl/Tdie)", "Core (Tctl)", "CPU Package"} {
		position := slices.IndexFunc(temperatures, func(sensor sensorNode) bool { return sensor.text == name })
		if position < 0 {
			continue
		}
		if value, ok := temperatures[position].valueOf(); ok {
			readings = append(readings, sensors.Reading{Name: sensors.CPUTemperature, Value: value})
		}
		break
	}
	return readings
}

func appendCCDTemperatures(readings []sensors.Reading, found []sensorNode) []sensors.Reading {
	first := map[int]sensorNode{}
	for _, sensor := range cpuSensors(found, "Temperature") {
		parts := ccdPattern.FindStringSubmatch(sensor.text)
		if parts == nil {
			continue
		}
		number, err := strconv.Atoi(parts[1])
		if _, taken := first[number]; err != nil || number > MaxNumber || taken {
			continue
		}
		first[number] = sensor
	}
	numbers := make([]int, 0, len(first))
	for number := range first {
		numbers = append(numbers, number)
	}
	slices.Sort(numbers)
	for _, number := range numbers {
		if value, ok := first[number].valueOf(); ok {
			readings = append(readings, sensors.Reading{Name: sensors.CCDTemperature(number), Value: value})
		}
	}
	return readings
}

// appendClock adds cpu.clock as the highest clock of a core. The effective clocks, the averages and the bus are not
// the clock of a core.
func appendClock(readings []sensors.Reading, found []sensorNode) []sensors.Reading {
	var highest float64
	hasClock := false
	for _, sensor := range cpuSensors(found, "Clock") {
		if !coreClock.MatchString(sensor.text) {
			continue
		}
		if value, ok := sensor.valueOf(); ok && (!hasClock || value > highest) {
			highest, hasClock = value, true
		}
	}
	if !hasClock {
		return readings
	}
	return append(readings, sensors.Reading{Name: sensors.CPUClock, Value: highest})
}

// appendModuleTemperatures numbers the modules that have a thermal sensor by their index, so that a module without one
// takes no number.
func appendModuleTemperatures(readings []sensors.Reading, found []sensorNode) []sensors.Reading {
	for position, sensor := range numberedSensors(found, moduleHardware, true) {
		if position+1 > MaxNumber {
			break
		}
		if value, ok := sensor.valueOf(); ok {
			readings = append(readings, sensors.Reading{Name: sensors.RAMTemperature(position + 1), Value: value})
		}
	}
	return readings
}

// appendDriveTemperatures numbers the drives by their number whether or not they have a temperature, so that a drive
// whose temperature is missing does not renumber the others.
func appendDriveTemperatures(readings []sensors.Reading, found []sensorNode) []sensors.Reading {
	for position, sensor := range numberedSensors(found, driveHardware, false) {
		if position+1 > MaxNumber {
			break
		}
		if sensor.id == "" {
			continue
		}
		if value, ok := sensor.valueOf(); ok {
			readings = append(readings, sensors.Reading{Name: sensors.NVMeTemperature(position + 1), Value: value})
		}
	}
	return readings
}

// numberedSensors gives, for each hardware of the pattern in the order of its number, its sensor temperature/0, or an
// empty node when it has none. With onlyWithSensor the hardware without that sensor is left out.
func numberedSensors(found []sensorNode, pattern *regexp.Regexp, onlyWithSensor bool) []sensorNode {
	var hardware []string
	for _, sensor := range found {
		if pattern.MatchString(sensor.hardware) && !slices.Contains(hardware, sensor.hardware) {
			hardware = append(hardware, sensor.hardware)
		}
	}
	slices.SortFunc(hardware, func(first, second string) int {
		return compareNumbers(pattern.FindStringSubmatch(first)[1], pattern.FindStringSubmatch(second)[1], first, second)
	})
	var result []sensorNode
	for _, each := range hardware {
		wanted := each + "/temperature/0"
		position := slices.IndexFunc(found, func(sensor sensorNode) bool { return sensor.id == wanted })
		switch {
		case position >= 0:
			result = append(result, found[position])
		case !onlyWithSensor:
			result = append(result, sensorNode{})
		}
	}
	return result
}

// compareNumbers orders two numbers written as digits by their value, without converting them, so that none overflows.
// The whole texts break a tie between spellings of one value, such as 7 and 07.
func compareNumbers(first, second, firstText, secondText string) int {
	first, second = strings.TrimLeft(first, "0"), strings.TrimLeft(second, "0")
	if len(first) != len(second) {
		return len(first) - len(second)
	}
	if order := strings.Compare(first, second); order != 0 {
		return order
	}
	return strings.Compare(firstText, secondText)
}
