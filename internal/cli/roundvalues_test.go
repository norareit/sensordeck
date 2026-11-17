package cli

import (
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

func value(tag uint8, number int16) smartmonitor.Value {
	return smartmonitor.Value{Tag: tag, Value: number}
}

func pairOf(metric string, threshold config.Threshold, normal, alarm uint8) config.AlarmPair {
	return config.AlarmPair{Name: metric, Threshold: threshold, Normal: normal, Alarm: alarm}
}

func TestRoundUnitsSwitchesAPairAcrossAnAboveThreshold(test *testing.T) {
	settings := config.Config{Alarms: []config.AlarmPair{pairOf("gpu.temp", config.Threshold{Above: 80, HasAbove: true}, 1, 2)}}
	cases := []struct {
		reading float64
		want    []smartmonitor.Value
	}{
		{79, []smartmonitor.Value{value(1, 79), value(2, -1)}},
		{80, []smartmonitor.Value{value(1, 80), value(2, -1)}},
		{80.4, []smartmonitor.Value{value(1, 80), value(2, -1)}},
		{80.5, []smartmonitor.Value{value(1, -1), value(2, 81)}},
		{81, []smartmonitor.Value{value(1, -1), value(2, 81)}},
		{79, []smartmonitor.Value{value(1, 79), value(2, -1)}},
	}
	for _, each := range cases {
		units := roundUnits(settings, map[string]float64{"gpu.temp": each.reading})
		if !reflect.DeepEqual(units, [][]smartmonitor.Value{each.want}) {
			test.Errorf("reading %v: %v, want %v", each.reading, units, each.want)
		}
	}
}

func TestRoundUnitsSwitchesAPairAcrossABelowThreshold(test *testing.T) {
	settings := config.Config{Alarms: []config.AlarmPair{pairOf("fan.pump", config.Threshold{Below: 1000, HasBelow: true}, 3, 4)}}
	cases := []struct {
		reading float64
		want    []smartmonitor.Value
	}{
		{1001, []smartmonitor.Value{value(3, 1001), value(4, -1)}},
		{1000, []smartmonitor.Value{value(3, 1000), value(4, -1)}},
		{999, []smartmonitor.Value{value(3, -1), value(4, 999)}},
		{1000, []smartmonitor.Value{value(3, 1000), value(4, -1)}},
	}
	for _, each := range cases {
		units := roundUnits(settings, map[string]float64{"fan.pump": each.reading})
		if !reflect.DeepEqual(units, [][]smartmonitor.Value{each.want}) {
			test.Errorf("reading %v: %v, want %v", each.reading, units, each.want)
		}
	}
}

func TestRoundUnitsSendsNothingForAMetricWithNoReading(test *testing.T) {
	settings := config.Config{
		Mappings: []config.Mapping{{Name: "cpu.temp", Tag: 1}},
		Alarms:   []config.AlarmPair{pairOf("gpu.temp", config.Threshold{Above: 80, HasAbove: true}, 2, 3)},
	}
	if units := roundUnits(settings, map[string]float64{"cpu.temp": 5}); !reflect.DeepEqual(units, [][]smartmonitor.Value{{value(1, 5)}}) {
		test.Errorf("units %v", units)
	}
	if units := roundUnits(settings, nil); len(units) != 0 {
		test.Errorf("units %v", units)
	}
}

func TestRoundUnitsHoldsAPlainEntryAndAPairOfOneMetric(test *testing.T) {
	settings := config.Config{
		Mappings: []config.Mapping{{Name: "gpu.temp", Tag: 1}},
		Alarms:   []config.AlarmPair{pairOf("gpu.temp", config.Threshold{Above: 80, HasAbove: true}, 2, 3)},
	}
	want := [][]smartmonitor.Value{{value(1, 90)}, {value(2, -1), value(3, 90)}}
	if units := roundUnits(settings, map[string]float64{"gpu.temp": 90}); !reflect.DeepEqual(units, want) {
		test.Errorf("units %v, want %v", units, want)
	}
}

func TestRoundUnitsAreInTheOrderOfTheirLowestTag(test *testing.T) {
	settings := config.Config{
		Mappings: []config.Mapping{{Name: "cpu.temp", Tag: 3}},
		Alarms:   []config.AlarmPair{pairOf("gpu.temp", config.Threshold{Above: 80, HasAbove: true}, 2, 5)},
	}
	want := [][]smartmonitor.Value{{value(2, 10), value(5, -1)}, {value(3, 7)}}
	if units := roundUnits(settings, map[string]float64{"gpu.temp": 10, "cpu.temp": 7}); !reflect.DeepEqual(units, want) {
		test.Errorf("units %v, want %v", units, want)
	}
}

func TestRoundUnitsGivesTheNormalTagFirstEvenWhenItIsTheHigherTag(test *testing.T) {
	settings := config.Config{Alarms: []config.AlarmPair{pairOf("gpu.temp", config.Threshold{Above: 80, HasAbove: true}, 5, 2)}}
	want := [][]smartmonitor.Value{{value(5, 10), value(2, -1)}}
	if units := roundUnits(settings, map[string]float64{"gpu.temp": 10}); !reflect.DeepEqual(units, want) {
		test.Errorf("units %v, want %v", units, want)
	}
}
