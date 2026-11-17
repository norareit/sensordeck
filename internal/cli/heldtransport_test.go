package cli

import (
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

type recordingTransport struct {
	calls *[]string
}

func (inner recordingTransport) Write(smartmonitor.Report) error {
	*inner.calls = append(*inner.calls, "write")
	return nil
}
func (inner recordingTransport) Read(time.Duration) (smartmonitor.Report, error) {
	*inner.calls = append(*inner.calls, "read")
	return smartmonitor.Report{}, nil
}
func (inner recordingTransport) Reopen() error {
	*inner.calls = append(*inner.calls, "reopen")
	return nil
}
func (inner recordingTransport) Close() error {
	*inner.calls = append(*inner.calls, "close")
	return nil
}

func TestHeldTransportReleasesOnceAfterTheClose(test *testing.T) {
	var calls []string
	held := &heldTransport{transport: recordingTransport{calls: &calls}, release: func() { calls = append(calls, "release") }}
	_ = held.Write(smartmonitor.Report{})
	_, _ = held.Read(time.Second)
	_ = held.Reopen()
	_ = held.Close()
	_ = held.Close()
	want := []string{"write", "read", "reopen", "close", "release"}
	if len(calls) != len(want) {
		test.Fatalf("got %v, want %v", calls, want)
	}
	for position := range want {
		if calls[position] != want[position] {
			test.Fatalf("got %v, want %v", calls, want)
		}
	}
}
