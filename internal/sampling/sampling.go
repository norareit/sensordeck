// Package sampling computes the share of busy time of the CPU and the rates of the network from two samples. Linux and
// Windows both take their samples in their own way and share this decision, so that it is held once.
package sampling

import (
	"context"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

// Interval is how long a first reading waits between its two samples.
const Interval = 200 * time.Millisecond

// bytesPerKibibyte turns bytes a second into KiB/s.
const bytesPerKibibyte = 1024

// CPUSample is the counters of the CPU that matter: all time, and the idle time.
type CPUSample struct{ Total, Idle uint64 }

// CPU reads cpu.util as the share of non-idle time between two samples. It keeps its previous sample from one reading
// to the next, so that only the first reading waits.
type CPU struct {
	Take     func() (CPUSample, bool)
	Wait     func(time.Duration)
	previous *CPUSample
}

func (source *CPU) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wanted(sensors.CPUUtilisation) {
		return nil
	}
	if source.previous == nil {
		first, ok := source.Take()
		if !ok {
			return nil
		}
		source.previous = &first
		source.Wait(Interval)
	}
	current, ok := source.Take()
	if !ok {
		return nil
	}
	previous := *source.previous
	source.previous = &current
	// Differences as floats, so that a counter that went back cannot wrap.
	growth := float64(current.Total) - float64(previous.Total)
	if growth <= 0 {
		return nil
	}
	busy := growth - (float64(current.Idle) - float64(previous.Idle))
	return []sensors.Reading{{Name: sensors.CPUUtilisation, Value: min(max(100*busy/growth, 0), 100)}}
}

// Counters is what an interface gives that matters: the bytes received and transmitted.
type Counters struct{ Received, Transmitted uint64 }

// networkSample is the counters of the interfaces that count, and the moment they were read.
type networkSample[Key comparable] struct {
	moment     time.Time
	interfaces map[Key]Counters
}

// Network reads net.up and net.down as the bytes of the interfaces that count between two samples. Take gives the
// counters of the interfaces that count at that call, so that which ones count is decided at every sample. It keeps its
// previous sample from one reading to the next, so that only the first reading waits.
type Network[Key comparable] struct {
	Take     func() (map[Key]Counters, bool)
	Wait     func(time.Duration)
	Now      func() time.Time
	previous *networkSample[Key]
}

// take reads a sample. The moment is taken right after a read that succeeded, and never after one that failed.
func (source *Network[Key]) take() (networkSample[Key], bool) {
	interfaces, ok := source.Take()
	if !ok {
		return networkSample[Key]{}, false
	}
	return networkSample[Key]{moment: source.Now(), interfaces: interfaces}, true
}

func (source *Network[Key]) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wanted(sensors.NetworkUp) && !wanted(sensors.NetworkDown) {
		return nil
	}
	if source.previous == nil {
		first, ok := source.take()
		if !ok {
			return nil
		}
		source.previous = &first
		source.Wait(Interval)
	}
	current, ok := source.take()
	if !ok {
		return nil
	}
	previous := *source.previous
	source.previous = &current
	seconds := current.moment.Sub(previous.moment).Seconds()
	transmitted, received, shared := growth(previous, current)
	if shared == 0 || seconds <= 0 {
		return nil
	}
	var readings []sensors.Reading
	if wanted(sensors.NetworkUp) {
		readings = append(readings, sensors.Reading{Name: sensors.NetworkUp, Value: float64(transmitted) / seconds / bytesPerKibibyte})
	}
	if wanted(sensors.NetworkDown) {
		readings = append(readings, sensors.Reading{Name: sensors.NetworkDown, Value: float64(received) / seconds / bytesPerKibibyte})
	}
	return readings
}

// growth sums the bytes transmitted and received between two samples over the interfaces that are in both, and tells
// how many those are. A counter that went back adds nothing to its direction, and is not subtracted as unsigned.
func growth[Key comparable](previous, current networkSample[Key]) (transmitted, received uint64, shared int) {
	for key, now := range current.interfaces {
		before, found := previous.interfaces[key]
		if !found {
			continue
		}
		shared++
		if now.Transmitted >= before.Transmitted {
			transmitted += now.Transmitted - before.Transmitted
		}
		if now.Received >= before.Received {
			received += now.Received - before.Received
		}
	}
	return transmitted, received, shared
}
