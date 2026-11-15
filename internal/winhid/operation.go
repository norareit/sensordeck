package winhid

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// WriteTimeLimit is how long a write waits for its completion. On Linux the kernel ends a write of an output report
// to hidraw after 5 seconds, which this keeps.
const WriteTimeLimit = 5 * time.Second

// maximumWait is the longest wait in milliseconds. 0xFFFFFFFF is the infinite wait, which no limit may ask for.
const maximumWait = 0xFFFFFFFE

type direction int

const (
	reading direction = iota
	writing
)

func (kind direction) String() string {
	if kind == writing {
		return "write"
	}
	return "read"
}

// result is how an operation ended: whether the transport cancelled it after the wait ended without completion, and
// the code and the byte count of its completion.
type result struct {
	cancelled bool
	code      syscall.Errno
	count     uint32
}

// complete classifies the end of an operation on a buffer of the length. It gives the count of bytes of a read, or of
// a write that wrote all, and otherwise the timeout of the read, the failure of the write, or the failure with its
// code. Only an operation that the transport cancelled itself is a timeout, since a removal may abort one too.
func complete(kind direction, length int, ended result, limit time.Duration) (int, error) {
	count := int(ended.count)
	if kind == writing && ended.cancelled && (ended.code != 0 || count < length) {
		return 0, fmt.Errorf("write timed out after %s", limit)
	}
	if kind == reading && ended.cancelled && ended.code == errorOperationAborted {
		return 0, fmt.Errorf("%w after %s", smartmonitor.ErrReadTimeout, limit)
	}
	if ended.code != 0 {
		return 0, fmt.Errorf("%s failed: %w", kind, ended.code)
	}
	if kind == writing && count < length {
		return 0, fmt.Errorf("short write: %d of %d bytes", count, length)
	}
	return count, nil
}

// milliseconds rounds the limit up to a whole millisecond, so that a limit is never cut short, and keeps it from the
// infinite wait.
func milliseconds(limit time.Duration) uint32 {
	if limit <= 0 {
		return 0
	}
	whole := (limit + time.Millisecond - 1) / time.Millisecond
	return uint32(min(whole, maximumWait))
}

// run starts the operation on the buffer and waits at most the limit for it. When the wait ends without completion
// it cancels the operation. It always reaps, so that it returns with nothing pending and the buffer free.
func run(seam Seam, kind direction, handle Handle, buffer []byte, limit time.Duration) (int, error) {
	start := seam.StartRead
	if kind == writing {
		start = seam.StartWrite
	}
	operation, err := start(handle, buffer)
	if err != nil {
		var code syscall.Errno
		if !errors.As(err, &code) {
			return 0, fmt.Errorf("%s failed: %w", kind, err)
		}
		return complete(kind, len(buffer), result{code: code}, limit)
	}
	cancelled := !operation.Wait(milliseconds(limit))
	if cancelled {
		operation.Cancel()
	}
	count, code := operation.Reap()
	// The system wrote into the buffer until the reap, so it must stay reachable until here.
	runtime.KeepAlive(buffer)
	return complete(kind, len(buffer), result{cancelled: cancelled, code: code, count: count}, limit)
}
