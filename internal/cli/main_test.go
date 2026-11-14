package cli

import (
	"context"
	"testing"
	"time"
)

func TestWaitForPassesWithALiveContext(test *testing.T) {
	if !waitFor(context.Background(), 10*time.Millisecond) {
		test.Error("the wait was cut short")
	}
}

func TestWaitForEndsAtOnceWhenStopped(test *testing.T) {
	stop, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	if waitFor(stop, 10*time.Second) {
		test.Error("the wait was not cut short")
	}
	if time.Since(started) > time.Second {
		test.Error("the wait did not end at once")
	}
}
