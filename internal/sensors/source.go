package sensors

import "context"

// Reading is a metric that has a value now. Its unit is Unit(Name).
type Reading struct {
	Name  string
	Value float64
}

// Source reads the metrics of one source at the moment it is called. It reads a metric only when wanted gives true
// for its name, and gives a reading for each wanted metric that is not unavailable, in the order of the table of
// metrics. Being unavailable is never an error. stop ends a read that waits for a program early.
type Source interface {
	Read(stop context.Context, wanted func(name string) bool) []Reading
}

// ReadAll asks each source in turn and joins what they give.
func ReadAll(stop context.Context, sources []Source, wanted func(name string) bool) []Reading {
	var readings []Reading
	for _, source := range sources {
		readings = append(readings, source.Read(stop, wanted)...)
	}
	return readings
}
