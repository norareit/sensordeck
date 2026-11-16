package sensors

import "math"

// ToValue gives the 16-bit value of a value update for a reading: rounded to the nearest whole number with halves
// away from zero, then clamped to -32768 to 32767, the range of the signed number that the screen reads.
func ToValue(value float64) int16 {
	rounded := math.Round(value)
	if rounded <= math.MinInt16 {
		return math.MinInt16
	}
	if rounded >= math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(rounded)
}
