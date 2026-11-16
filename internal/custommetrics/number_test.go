package custommetrics

import (
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/sensors"
)

func TestNumber(test *testing.T) {
	cases := []struct {
		text string
		want float64
		ok   bool
	}{
		{"42000\n", 42000, true},
		{"temp: 41.5 C", 41.5, true},
		{"-3", -3, true},
		{"+7", 7, true},
		{".5", 5, true},
		{"1.2.3", 1.2, true},
		{"12 34", 12, true},
		{"1e3", 1, true},
		{"1,5", 1, true},
		{"NaN", 0, false},
		{"Inf", 0, false},
		{"", 0, false},
		{strings.Repeat("9", 400), 0, false},
	}
	for _, each := range cases {
		got, ok := Number(each.text)
		if got != each.want || ok != each.ok {
			test.Errorf("Number(%.20q) = %v, %v, want %v, %v", each.text, got, ok, each.want, each.ok)
		}
	}
}

func TestValueScalesAndShifts(test *testing.T) {
	definition := config.Metric{Scale: 0.001, Offset: -273.15}
	if got := Value(300000, definition); got < 26.84 || got > 26.86 {
		test.Errorf("value = %v", got)
	}
}

func TestValuesAreSentAsTheScreenTakesThem(test *testing.T) {
	cases := map[float64]uint16{-5: 0, 0.4: 0, 51.5: 52, 70000: 65535}
	for value, want := range cases {
		if got := sensors.ToValue(Value(value, config.Metric{Scale: 1})); got != want {
			test.Errorf("ToValue(%v) = %d, want %d", value, got, want)
		}
	}
}

func TestAValueThatOverflowsGivesNoValue(test *testing.T) {
	result := fromText("1"+strings.Repeat("0", 300), "x", config.Metric{Scale: 1e300})
	if !result.failed() {
		test.Errorf("value = %v", result.value)
	}
}
