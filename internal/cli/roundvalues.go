package cli

import (
	"slices"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// roundUnits gives the units of a round, in the order of their lowest tag. A metric with a reading gives a unit of one
// value for its plain entry, and a unit of two for each of its pairs: the value to the normal tag and the hiding value
// to the alarm tag in the normal state, and the reverse in the alarm state. The two of a pair are one unit so that they
// are sent in one report. A metric with no reading gives no unit, so that the screen keeps what it showed.
func roundUnits(settings config.Config, readings map[string]float64) [][]smartmonitor.Value {
	var units [][]smartmonitor.Value
	for _, mapping := range settings.Mappings {
		if reading, found := readings[mapping.Name]; found {
			units = append(units, []smartmonitor.Value{{Tag: mapping.Tag, Value: sensors.ToValue(reading)}})
		}
	}
	for _, pair := range settings.Alarms {
		if reading, found := readings[pair.Name]; found {
			units = append(units, pairUnit(pair, sensors.ToValue(reading)))
		}
	}
	slices.SortStableFunc(units, func(first, second []smartmonitor.Value) int { return int(lowestTag(first)) - int(lowestTag(second)) })
	return units
}

func pairUnit(pair config.AlarmPair, value int16) []smartmonitor.Value {
	normalValue, alarmValue := value, config.HidingValue
	if pair.Threshold.Alarm(value) {
		normalValue, alarmValue = config.HidingValue, value
	}
	return []smartmonitor.Value{{Tag: pair.Normal, Value: normalValue}, {Tag: pair.Alarm, Value: alarmValue}}
}

func lowestTag(unit []smartmonitor.Value) uint8 {
	lowest := unit[0].Tag
	for _, value := range unit[1:] {
		lowest = min(lowest, value.Tag)
	}
	return lowest
}
