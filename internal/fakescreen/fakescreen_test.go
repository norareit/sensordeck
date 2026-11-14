//go:build linux

package fakescreen

import (
	"os"
	"testing"
	"time"
)

// writeToNode writes the bytes to the node of the screen, as a transport does.
func writeToNode(test *testing.T, screen *Screen, data []byte) {
	test.Helper()
	node, err := os.OpenFile(screen.Node(), os.O_WRONLY, 0)
	if err != nil {
		test.Fatal(err)
	}
	defer node.Close()
	if _, err := node.Write(data); err != nil {
		test.Fatal(err)
	}
}

func writtenReport(first byte) []byte {
	data := make([]byte, writtenReportSize)
	data[1] = first
	return data
}

func TestWaitReportsGivesTheReportsWithoutTheReportID(test *testing.T) {
	screen := New(test).Add(3, "name", "")
	writeToNode(test, screen, writtenReport(0x0A))
	writeToNode(test, screen, writtenReport(0x0B))
	reports, reached := screen.WaitReports(2, 5*time.Second)
	if !reached || len(reports) != 2 || reports[0][0] != 0x0A || reports[1][0] != 0x0B {
		test.Errorf("reached %v, reports %d", reached, len(reports))
	}
}

func TestWaitReportsWithTooFewGivesWhatArrivedAndFalse(test *testing.T) {
	screen := New(test).Add(3, "name", "")
	writeToNode(test, screen, writtenReport(1))
	writeToNode(test, screen, writtenReport(2))
	started := time.Now()
	reports, reached := screen.WaitReports(3, 100*time.Millisecond)
	if reached || len(reports) != 2 {
		test.Errorf("reached %v, reports %d", reached, len(reports))
	}
	if time.Since(started) < 100*time.Millisecond {
		test.Error("it did not wait for the limit")
	}
}

func TestReportsLeavesOutAReportCutShort(test *testing.T) {
	screen := New(test).Add(3, "name", "")
	writeToNode(test, screen, writtenReport(1))
	writeToNode(test, screen, writtenReport(2)[:30])
	screen.Received() // waits until every byte has arrived
	if got := screen.Reports(); len(got) != 1 {
		test.Errorf("reports %d", len(got))
	}
}
