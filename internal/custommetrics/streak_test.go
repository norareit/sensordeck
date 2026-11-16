package custommetrics

import (
	"strings"
	"testing"
)

func TestAStreakWritesAtItsStartAndAtItsEnd(test *testing.T) {
	var stderr strings.Builder
	var current streak
	failure := outcome{cause: "no number in x"}
	for _, result := range []outcome{failure, failure, {value: 1}, {value: 2}, failure} {
		current.observe(&stderr, "a.b", result)
	}
	want := "metric a.b: no number in x\nmetric a.b: read again\nmetric a.b: no number in x\n"
	if stderr.String() != want {
		test.Errorf("written %q, want %q", stderr.String(), want)
	}
}
