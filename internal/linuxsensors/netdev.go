//go:build linux

package linuxsensors

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/norareit/sensordeck/internal/sampling"
)

// parseNetDev gives the counters of each line that has a colon and 16 or more fields after it, the first and the
// ninth of which are whole numbers of 64 bits. The two header lines are passed over by the same rule.
func parseNetDev(content string) map[string]sampling.Counters {
	interfaces := map[string]sampling.Counters{}
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
		interfaces[strings.TrimSpace(name)] = sampling.Counters{Received: received, Transmitted: transmitted}
	}
	return interfaces
}

// hasDevice tells whether the kernel ties the interface to a device, which a virtual interface is not.
func hasDevice(sysfsRoot, name string) bool {
	_, err := os.Lstat(filepath.Join(sysfsRoot, "class", "net", name, "device"))
	return err == nil
}

// newNetwork reads the interfaces of /proc/net/dev under procRoot that belong to a device, deciding at every sample
// which ones do, with the rates computed by internal/sampling. The moment is taken right after the read.
func newNetwork(sysfsRoot, procRoot string, wait func(time.Duration), now func() time.Time) *sampling.Network[string] {
	return &sampling.Network[string]{
		Take: func() (map[string]sampling.Counters, bool) {
			content, err := os.ReadFile(filepath.Join(procRoot, "net", "dev"))
			if err != nil {
				return nil, false
			}
			counting := map[string]sampling.Counters{}
			for name, counters := range parseNetDev(string(content)) {
				if hasDevice(sysfsRoot, name) {
					counting[name] = counters
				}
			}
			return counting, true
		},
		Wait: wait,
		Now:  now,
	}
}
