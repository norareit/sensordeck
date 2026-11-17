//go:build windows

package devicelock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows"
)

// lockOffsetHigh puts the locked byte at offset 2^62, far beyond the content, so that another process can read the
// content while the lock is held.
const lockOffsetHigh = 0x40000000

// Acquire takes the lock of the file of that name in the directory sensordeck of the cache directory of the user,
// once and without waiting. release ends the lock, and a second call does nothing.
func Acquire(name string) (release func(), err error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find the cache directory for the lock file %s: %w", name, err)
	}
	directory := filepath.Join(cache, "sensordeck")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create the lock file %s: %w", filepath.Join(directory, name), err)
	}
	return acquireIn(directory, name)
}

func acquireIn(directory, name string) (func(), error) {
	path := filepath.Join(directory, name)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("cannot create the lock file %s: %w", path, err)
	}
	overlapped := &windows.Overlapped{OffsetHigh: lockOffsetHigh}
	err = windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, overlapped)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		content, _ := os.ReadFile(path)
		_ = file.Close()
		return nil, &InUseError{Holder: holderOfContent(content)}
	}
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("cannot lock %s: %w", path, err)
	}
	if err := writeContent(file); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("cannot write the lock file %s: %w", path, err)
	}
	var once sync.Once
	return func() { once.Do(func() { _ = file.Close() }) }, nil
}

func writeContent(file *os.File) error {
	if err := file.Truncate(0); err != nil {
		return err
	}
	_, err := file.WriteAt(encodeContent(os.Getpid(), strings.Join(os.Args, " ")), 0)
	return err
}
