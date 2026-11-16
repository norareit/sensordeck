package custommetrics

import (
	"math"
	"regexp"
	"strconv"

	"github.com/norareit/sensordeck/internal/config"
)

// numberPattern is an optional sign, digits and optionally a point and digits. Exponents, a comma as the decimal mark
// and the names NaN and Inf are not numbers.
var numberPattern = regexp.MustCompile(`[-+]?[0-9]+(\.[0-9]+)?`)

// Number gives the first number of a text, and false when there is none or when it is not finite.
func Number(text string) (float64, bool) {
	match := numberPattern.FindString(text)
	if match == "" {
		return 0, false
	}
	number, err := strconv.ParseFloat(match, 64)
	if err != nil || math.IsInf(number, 0) || math.IsNaN(number) {
		return 0, false
	}
	return number, true
}

// Value gives the value of a number for a definition: the number multiplied by the scale, plus the offset.
func Value(number float64, definition config.Metric) float64 {
	return number*definition.Scale + definition.Offset
}
