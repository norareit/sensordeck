//go:build linux

package devicelock

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func openTemporary(test *testing.T, path string) *os.File {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { _ = file.Close() })
	return file
}

func TestLockNodeExcludesAnotherOpening(test *testing.T) {
	path := filepath.Join(test.TempDir(), "node")
	first := openTemporary(test, path)
	if err := LockNode(first, "/nonexistent"); err != nil {
		test.Fatalf("first lock: %v", err)
	}
	second := openTemporary(test, path)
	var inUse *InUseError
	if err := LockNode(second, "/nonexistent"); !errors.As(err, &inUse) {
		test.Fatalf("second lock: got %v, want an InUseError", err)
	}
	link := filepath.Join(filepath.Dir(path), "link")
	if err := os.Symlink(path, link); err != nil {
		test.Fatal(err)
	}
	if err := LockNode(openTemporary(test, link), "/nonexistent"); !errors.As(err, &inUse) {
		test.Fatalf("lock through a link: got %v, want an InUseError", err)
	}
	_ = first.Close()
	if err := LockNode(second, "/nonexistent"); err != nil {
		test.Fatalf("after the close: %v", err)
	}
}

func TestLockNodeNamesTheHolder(test *testing.T) {
	if _, err := os.ReadFile("/proc/locks"); err != nil {
		test.Skip("/proc/locks cannot be read")
	}
	path := filepath.Join(test.TempDir(), "node")
	first := openTemporary(test, path)
	if err := LockNode(first, "/proc"); err != nil {
		test.Fatal(err)
	}
	var inUse *InUseError
	err := LockNode(openTemporary(test, path), "/proc")
	if !errors.As(err, &inUse) {
		test.Fatalf("got %v, want an InUseError", err)
	}
	if !strings.HasPrefix(inUse.Holder, " (process "+strconv.Itoa(os.Getpid())) {
		test.Errorf("holder %q does not name this process", inUse.Holder)
	}
}

func TestLockResult(test *testing.T) {
	if err := lockResult(nil, "/dev/x", nil); err != nil {
		test.Errorf("nil: %v", err)
	}
	var inUse *InUseError
	err := lockResult(unix.EWOULDBLOCK, "/dev/x", func() string { return " (process 1)" })
	if !errors.As(err, &inUse) || inUse.Holder != " (process 1)" {
		test.Errorf("EWOULDBLOCK: got %v", err)
	}
	if err := lockResult(unix.EBADF, "/dev/x", nil); err == nil || err.Error() != "cannot lock /dev/x: bad file descriptor" {
		test.Errorf("EBADF: got %v", err)
	}
}
