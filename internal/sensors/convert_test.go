package sensors

import (
	"math"
	"testing"
)

func TestToValue(test *testing.T) {
	cases := []struct {
		value float64
		want  uint16
	}{
		{51.5, 52}, {51.49, 51}, {0.5, 1}, {-5, 0}, {-0.4, 0}, {70000, 65535}, {65535.4, 65535}, {65535, 65535},
		{math.Inf(1), 65535}, {math.Inf(-1), 0}, {0, 0},
	}
	for _, each := range cases {
		if got := ToValue(each.value); got != each.want {
			test.Errorf("ToValue(%v) = %d, want %d", each.value, got, each.want)
		}
	}
}
