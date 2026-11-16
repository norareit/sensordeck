package sensors

import (
	"math"
	"testing"
)

func TestToValue(test *testing.T) {
	cases := []struct {
		value float64
		want  int16
	}{
		{51.5, 52}, {51.49, 51}, {0.5, 1}, {-0.4, 0}, {-0.5, -1}, {-5, -5}, {-51.5, -52}, {32767.4, 32767},
		{32767.5, 32767}, {40000, 32767}, {70000, 32767}, {-32768.4, -32768}, {-32768.5, -32768}, {-40000, -32768},
		{math.Inf(1), 32767}, {math.Inf(-1), -32768}, {0, 0},
	}
	for _, each := range cases {
		if got := ToValue(each.value); got != each.want {
			test.Errorf("ToValue(%v) = %d, want %d", each.value, got, each.want)
		}
	}
}
