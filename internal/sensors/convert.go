package sensors

import "math"

// ToValue gives the 16-bit value of a value update for a reading: rounded to the nearest whole number with halves
// away from zero, then clamped to 0 to 65535.
func ToValue(value float64) uint16 {
	rounded := math.Round(value)
	if rounded <= 0 {
		return 0
	}
	if rounded >= math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(rounded)
}
