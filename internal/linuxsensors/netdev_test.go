//go:build linux

package linuxsensors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakesensors"
	"github.com/norareit/sensordeck/internal/sampling"
	"github.com/norareit/sensordeck/internal/sensors"
)

const netDevHeader = "Inter-|   Receive                                                |  Transmit\n" +
	" face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n"

const cardDevice = "pci0000:00/0000:00:02.2/0000:06:00.0"

// netDevLine writes a line of /proc/net/dev with the two byte counters given and 0 in every other field.
func netDevLine(name string, received, transmitted uint64) string {
	return fmt.Sprintf("%6s: %d 0 0 0 0 0 0 0 %d 0 0 0 0 0 0 0\n", name, received, transmitted)
}

func netDev(lines ...string) string { return netDevHeader + strings.Join(lines, "") }

func removeNetDev(test *testing.T, tree *fakesensors.Tree) {
	test.Helper()
	if err := os.Remove(filepath.Join(tree.ProcRoot, "net", "dev")); err != nil {
		test.Fatal(err)
	}
}

// networkRun makes the source of a tree with a wait that records its duration and writes the next content, and a
// clock that gives the moments the test set, one for each sample.
type networkRun struct {
	source  *sampling.Network[string]
	waits   []time.Duration
	changes []string
	moments []time.Time
	clocked int
}

// at is the moment that many seconds after an arbitrary start.
func at(seconds float64) time.Time {
	return time.Unix(1000, 0).Add(time.Duration(seconds * float64(time.Second)))
}

func newNetworkRun(test *testing.T, tree *fakesensors.Tree, first string, moments ...time.Time) *networkRun {
	test.Helper()
	tree.NetDev(first)
	run := &networkRun{moments: moments}
	run.source = newNetwork(tree.SysfsRoot, tree.ProcRoot, func(duration time.Duration) {
		run.waits = append(run.waits, duration)
		if len(run.changes) > 0 {
			tree.NetDev(run.changes[0])
			run.changes = run.changes[1:]
		}
	}, func() time.Time {
		if run.clocked >= len(run.moments) {
			test.Fatalf("the clock was asked %d times, for %d moments", run.clocked+1, len(run.moments))
		}
		run.clocked++
		return run.moments[run.clocked-1]
	})
	return run
}

func (run *networkRun) read() []sensors.Reading { return run.readWanted(everything) }

func (run *networkRun) readWanted(wanted func(string) bool) []sensors.Reading {
	return run.source.Read(context.Background(), wanted)
}

func rates(up, down float64) []sensors.Reading {
	return []sensors.Reading{{Name: "net.up", Value: up}, {Name: "net.down", Value: down}}
}

func countingTree(test *testing.T, names ...string) *fakesensors.Tree {
	tree := newTree(test)
	for _, name := range names {
		tree.NetInterface(name, cardDevice)
	}
	return tree
}

func TestFirstNetworkReadingTakesTwoSamplesAndWaitsOnce(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 1000, 5000)), at(0), at(1))
	run.changes = []string{netDev(netDevLine("enp6s0", 1000+10240, 5000+20480))}
	if got := run.read(); !reflect.DeepEqual(got, rates(20, 10)) {
		test.Errorf("got %v", got)
	}
	if !reflect.DeepEqual(run.waits, []time.Duration{200 * time.Millisecond}) || run.clocked != 2 {
		test.Errorf("waits %v, clock asked %d times", run.waits, run.clocked)
	}
}

func TestSecondNetworkReadingUsesTheKeptSampleAndDoesNotWait(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0)), at(0), at(1), at(3))
	run.changes = []string{netDev(netDevLine("enp6s0", 1024, 0))}
	run.read()
	tree.NetDev(netDev(netDevLine("enp6s0", 1024+2048, 4096)))
	if got := run.read(); !reflect.DeepEqual(got, rates(2, 1)) {
		test.Errorf("got %v", got)
	}
	if len(run.waits) != 1 || run.clocked != 3 {
		test.Errorf("waits %v, clock asked %d times", run.waits, run.clocked)
	}
}

func TestTimeBetweenSamplesIsThatOfTheClock(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0)), at(0), at(2.5))
	run.changes = []string{netDev(netDevLine("enp6s0", 10240, 0))}
	got := run.read()
	if !reflect.DeepEqual(got, rates(0, 4)) || !reflect.DeepEqual(run.waits, []time.Duration{200 * time.Millisecond}) {
		test.Errorf("got %v, waits %v", got, run.waits)
	}
}

func TestInterfacesThatCountAreSummed(test *testing.T) {
	tree := countingTree(test, "enp6s0", "wlp5s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0), netDevLine("wlp5s0", 0, 0)), at(0), at(1))
	run.changes = []string{netDev(netDevLine("enp6s0", 1024, 2048), netDevLine("wlp5s0", 3072, 4096))}
	if got := run.read(); !reflect.DeepEqual(got, rates(6, 4)) {
		test.Errorf("got %v", got)
	}
}

func TestInterfacesWithoutADeviceAddNothing(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	before := []string{netDevLine("enp6s0", 0, 0)}
	after := []string{netDevLine("enp6s0", 1024, 1024)}
	for _, name := range []string{"lo", "docker0", "veth1a2b3c", "br-0123456789ab", "tun0"} {
		tree.NetInterface(name, "")
		before = append(before, netDevLine(name, 0, 0))
		after = append(after, netDevLine(name, 1<<20, 1<<20))
	}
	run := newNetworkRun(test, tree, netDev(before...), at(0), at(1))
	run.changes = []string{netDev(after...)}
	if got := run.read(); !reflect.DeepEqual(got, rates(1, 1)) {
		test.Errorf("got %v", got)
	}
}

func TestDeviceEntryOfAnyKindCounts(test *testing.T) {
	tree := newTree(test)
	tree.NetInterface("usb0", "")
	if err := os.WriteFile(filepath.Join(tree.SysfsRoot, "class", "net", "usb0", "device"), nil, 0o644); err != nil {
		test.Fatal(err)
	}
	run := newNetworkRun(test, tree, netDev(netDevLine("usb0", 0, 0)), at(0), at(1))
	run.changes = []string{netDev(netDevLine("usb0", 1024, 0))}
	if got := run.read(); !reflect.DeepEqual(got, rates(0, 1)) {
		test.Errorf("got %v", got)
	}
}

func TestCounterWithNoSpaceAfterTheColonCounts(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0)), at(0), at(1))
	run.changes = []string{netDev("enp6s0:2048 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0\n")}
	if got := run.read(); !reflect.DeepEqual(got, rates(0, 2)) {
		test.Errorf("got %v", got)
	}
}

func TestShortOrMalformedLinesArePassedOver(test *testing.T) {
	tree := countingTree(test, "enp6s0", "short0", "bad0")
	short := "short0: 5 0 0 0 0 0 0 0 5 0 0 0 0 0 0\n"
	malformed := "bad0: 5 0 0 0 0 0 0 0 12x 0 0 0 0 0 0 0\n"
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0), short, malformed), at(0), at(1))
	run.changes = []string{netDev(netDevLine("enp6s0", 1024, 0), short, malformed)}
	if got := run.read(); !reflect.DeepEqual(got, rates(0, 1)) {
		test.Errorf("got %v", got)
	}
}

func TestInterfaceThatAppearsOrVanishesAddsNothing(test *testing.T) {
	tree := countingTree(test, "enp6s0", "wlp5s0", "usb0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0), netDevLine("wlp5s0", 100, 100)), at(0), at(1))
	run.changes = []string{netDev(netDevLine("enp6s0", 1024, 1024), netDevLine("usb0", 1<<30, 1<<30))}
	if got := run.read(); !reflect.DeepEqual(got, rates(1, 1)) {
		test.Errorf("got %v", got)
	}
}

func TestCounterThatWentBackAddsNothingThenAddsAgain(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 10000, 0)), at(0), at(1), at(2))
	run.changes = []string{netDev(netDevLine("enp6s0", 500, 2048))}
	if got := run.read(); !reflect.DeepEqual(got, rates(2, 0)) {
		test.Errorf("got %v", got)
	}
	tree.NetDev(netDev(netDevLine("enp6s0", 500+3072, 2048)))
	if got := run.read(); !reflect.DeepEqual(got, rates(0, 3)) {
		test.Errorf("next reading: %v", got)
	}
}

func TestMissingNetDevGivesNothingAndDoesNotWait(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, "")
	removeNetDev(test, tree)
	if got := run.read(); len(got) != 0 || len(run.waits) != 0 || run.clocked != 0 {
		test.Errorf("got %v, waits %v, clock asked %d times", got, run.waits, run.clocked)
	}
}

func TestNoInterfaceThatCountsGivesNothing(test *testing.T) {
	tree := newTree(test)
	tree.NetInterface("lo", "")
	run := newNetworkRun(test, tree, netDev(netDevLine("lo", 0, 0)), at(0), at(1))
	run.changes = []string{netDev(netDevLine("lo", 1024, 1024))}
	if got := run.read(); len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestNoTimeBetweenSamplesGivesNothingThenMeasuresFromTheLastSample(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0)), at(5), at(5), at(6))
	run.changes = []string{netDev(netDevLine("enp6s0", 1024, 0))}
	if got := run.read(); len(got) != 0 {
		test.Errorf("got %v", got)
	}
	tree.NetDev(netDev(netDevLine("enp6s0", 1024+4096, 0)))
	if got := run.read(); !reflect.DeepEqual(got, rates(0, 4)) {
		test.Errorf("next reading: %v", got)
	}
}

func TestFailedReadingKeepsThePreviousSample(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0)), at(0), at(1), at(3))
	run.changes = []string{netDev(netDevLine("enp6s0", 1024, 0))}
	run.read()
	removeNetDev(test, tree)
	if got := run.read(); len(got) != 0 {
		test.Errorf("second reading: %v", got)
	}
	tree.NetDev(netDev(netDevLine("enp6s0", 1024+6144, 0)))
	// measured from the sample of the first reading, taken at moment 1, and read at moment 3
	if got := run.read(); !reflect.DeepEqual(got, rates(0, 3)) {
		test.Errorf("third reading: %v", got)
	}
}

func TestOnlyDownWantedGivesOnlyDown(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0)), at(0), at(1))
	run.changes = []string{netDev(netDevLine("enp6s0", 1024, 2048))}
	got := run.readWanted(func(name string) bool { return name == "net.down" })
	if want := []sensors.Reading{{Name: "net.down", Value: 1}}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestNoNetworkMetricWantedReadsNoFile(test *testing.T) {
	tree := countingTree(test, "enp6s0")
	run := newNetworkRun(test, tree, netDev(netDevLine("enp6s0", 0, 0)))
	got := run.readWanted(func(name string) bool { return name == "cpu.util" })
	if len(got) != 0 || len(run.waits) != 0 || run.clocked != 0 {
		test.Errorf("got %v, waits %v, clock asked %d times", got, run.waits, run.clocked)
	}
}

func TestParseNetDev(test *testing.T) {
	content := netDev(netDevLine("lo", 11, 22), netDevLine("enp6s0", 33, 44))
	want := map[string]sampling.Counters{"lo": {Received: 11, Transmitted: 22}, "enp6s0": {Received: 33, Transmitted: 44}}
	if got := parseNetDev(content); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestParseNetDevPassesOverACounterBeyond64Bits(test *testing.T) {
	line := "enp6s0: 18446744073709551616 0 0 0 0 0 0 0 5 0 0 0 0 0 0 0\n"
	if got := parseNetDev(netDev(line)); len(got) != 0 {
		test.Errorf("got %v", got)
	}
}
