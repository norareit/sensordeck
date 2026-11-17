//go:build linux

package devicelock

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// LockNode takes an exclusive lock on the opened node, once and without waiting. The lock ends with the close of
// the node, which a process that ends does too. It goes through SyscallConn, never Fd, which would put the node into
// blocking mode and break its read deadlines.
func LockNode(file *os.File, procRoot string) error {
	raw, err := file.SyscallConn()
	if err != nil {
		return lockResult(err, file.Name(), nil)
	}
	var lockErr error
	if err := raw.Control(func(descriptor uintptr) {
		lockErr = unix.Flock(int(descriptor), unix.LOCK_EX|unix.LOCK_NB)
	}); err != nil {
		return lockResult(err, file.Name(), nil)
	}
	return lockResult(lockErr, file.Name(), func() string { return holderOfFile(file, procRoot) })
}

// lockResult words the result of the lock. A lock held by another process gives an InUseError with the holder.
func lockResult(err error, path string, holder func() string) error {
	switch err {
	case nil:
		return nil
	case unix.EWOULDBLOCK:
		inUse := &InUseError{}
		if holder != nil {
			inUse.Holder = holder()
		}
		return inUse
	}
	return fmt.Errorf("cannot lock %s: %w", path, err)
}

// holderOfFile reads the device and inode through Stat, which does not touch the mode of the descriptor.
func holderOfFile(file *os.File, procRoot string) string {
	info, err := file.Stat()
	if err != nil {
		return ""
	}
	status, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ""
	}
	return holderOfNode(procRoot, unix.Major(uint64(status.Dev)), unix.Minor(uint64(status.Dev)), status.Ino)
}
