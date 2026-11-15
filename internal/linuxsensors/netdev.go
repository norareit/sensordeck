//go:build linux

package linuxsensors

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

// bytesPerKibibyte turns bytes a second into KiB/s.
const bytesPerKibibyte = 1024

// interfaceCounters is what one line of /proc/net/dev gives that matters: the bytes received and transmitted.
type interfaceCounters struct{ received, transmitted uint64 }

// networkSample is the counters of the interfaces that count, and the moment they were read.
type networkSample struct {
	moment     time.Time
	interfaces map[string]interfaceCounters
}

// network reads net.up and net.down as the bytes of the interfaces that belong to a device between two samples. It
// keeps its previous sample from one reading to the next, so that only the first reading waits.
type network struct {
	sysfsRoot, procRoot string
	wait                func(time.Duration)
	now                 func() time.Time
	previous            *networkSample
}

// parseNetDev gives the counters of each line that has a colon and 16 or more fields after it, the first and the
// ninth of which are whole numbers of 64 bits. The two header lines are passed over by the same rule.
func parseNetDev(content string) map[string]interfaceCounters {
	interfaces := map[string]interfaceCounters{}
	for _, line := range strings.Split(content, "\n") {
		name, rest, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) < 16 || !isDigits(fields[0]) || !isDigits(fields[8]) {
			continue
		}
		received, receivedErr := strconv.ParseUint(fields[0], 10, 64)
		transmitted, transmittedErr := strconv.ParseUint(fields[8], 10, 64)
		if receivedErr != nil || transmittedErr != nil {
			continue
		}
		interfaces[strings.TrimSpace(name)] = interfaceCounters{received: received, transmitted: transmitted}
	}
	return interfaces
}

// hasDevice tells whether the kernel ties the interface to a device, which a virtual interface is not.
func hasDevice(sysfsRoot, name string) bool {
	_, err := os.Lstat(filepath.Join(sysfsRoot, "class", "net", name, "device"))
	return err == nil
}

// take reads a sample, deciding at every call which interfaces count. The moment is taken right after the read.
func (source *network) take() (networkSample, bool) {
	content, err := os.ReadFile(filepath.Join(source.procRoot, "net", "dev"))
	if err != nil {
		return networkSample{}, false
	}
	moment := source.now()
	counting := map[string]interfaceCounters{}
	for name, counters := range parseNetDev(string(content)) {
		if hasDevice(source.sysfsRoot, name) {
			counting[name] = counters
		}
	}
	return networkSample{moment: moment, interfaces: counting}, true
}

func (source *network) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wanted(sensors.NetworkUp) && !wanted(sensors.NetworkDown) {
		return nil
	}
	if source.previous == nil {
		first, ok := source.take()
		if !ok {
			return nil
		}
		source.previous = &first
		source.wait(sampleInterval)
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
func growth(previous, current networkSample) (transmitted, received uint64, shared int) {
	for name, now := range current.interfaces {
		before, found := previous.interfaces[name]
		if !found {
			continue
		}
		shared++
		if now.transmitted >= before.transmitted {
			transmitted += now.transmitted - before.transmitted
		}
		if now.received >= before.received {
			received += now.received - before.received
		}
	}
	return transmitted, received, shared
}
