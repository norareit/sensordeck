package winhid

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

const testLimit = 2 * time.Second

func TestComplete(test *testing.T) {
	const other syscall.Errno = 31
	for _, testCase := range []struct {
		name      string
		kind      direction
		ended     result
		wantCount int
		wantText  string
		wantTime  bool // errors.Is with ErrReadTimeout
	}{
		{name: "read in time", kind: reading, ended: result{count: 65}, wantCount: 65},
		{name: "short read in time", kind: reading, ended: result{count: 10}, wantCount: 10},
		{name: "read of no bytes", kind: reading, ended: result{}, wantCount: 0},
		{name: "write in time", kind: writing, ended: result{count: 65}, wantCount: 65},
		{name: "short write in time", kind: writing, ended: result{count: 40}, wantText: "short write: 40 of 65 bytes"},
		{name: "read that completed after the wait ended", kind: reading, ended: result{cancelled: true, count: 65},
			wantCount: 65},
		{name: "cancelled read", kind: reading, ended: result{cancelled: true, code: errorOperationAborted},
			wantText: "no report arrived within the time limit after 2s", wantTime: true},
		{name: "aborted read that was not cancelled", kind: reading, ended: result{code: errorOperationAborted},
			wantText: "read failed: " + fmt.Sprint(errorOperationAborted)},
		{name: "cancelled write that wrote all", kind: writing, ended: result{cancelled: true, count: 65},
			wantCount: 65},
		{name: "cancelled write that wrote less", kind: writing, ended: result{cancelled: true, count: 3},
			wantText: "write timed out after 2s"},
		{name: "cancelled write with a code", kind: writing,
			ended: result{cancelled: true, code: errorOperationAborted}, wantText: "write timed out after 2s"},
		{name: "failed read", kind: reading, ended: result{code: other}, wantText: "read failed: " + fmt.Sprint(other)},
		{name: "failed write", kind: writing, ended: result{code: other},
			wantText: "write failed: " + fmt.Sprint(other)},
		{name: "cancelled read with another code", kind: reading, ended: result{cancelled: true, code: other},
			wantText: "read failed: " + fmt.Sprint(other)},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			count, err := complete(testCase.kind, 65, testCase.ended, testLimit)
			if testCase.wantText == "" {
				if err != nil || count != testCase.wantCount {
					test.Fatalf("got %d, %v, want %d", count, err, testCase.wantCount)
				}
				return
			}
			if err == nil || err.Error() != testCase.wantText {
				test.Fatalf("got %v, want %q", err, testCase.wantText)
			}
			if errors.Is(err, smartmonitor.ErrReadTimeout) != testCase.wantTime {
				test.Errorf("errors.Is with the timeout is %v", !testCase.wantTime)
			}
		})
	}
}

func TestCompleteCarriesTheCodeOfAFailure(test *testing.T) {
	for _, kind := range []direction{reading, writing} {
		_, err := complete(kind, 65, result{code: 31}, testLimit)
		var code syscall.Errno
		if !errors.As(err, &code) || code != 31 {
			test.Errorf("%v: the code is not reachable in %v", kind, err)
		}
	}
}

func TestMilliseconds(test *testing.T) {
	for _, testCase := range []struct {
		limit time.Duration
		want  uint32
	}{
		{0, 0}, {-time.Second, 0}, {1, 1}, {time.Millisecond, 1}, {1500 * time.Microsecond, 2},
		{time.Second, 1000}, {time.Duration(maximumWait+5) * time.Millisecond, maximumWait},
		{time.Duration(1 << 62), maximumWait},
	} {
		if got := milliseconds(testCase.limit); got != testCase.want {
			test.Errorf("milliseconds(%v) = %d, want %d", testCase.limit, got, testCase.want)
		}
	}
}
