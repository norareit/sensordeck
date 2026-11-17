//go:build windows

package devicelock

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcquireIn(test *testing.T) {
	directory := test.TempDir()
	release, err := acquireIn(directory, "screen.lock")
	if err != nil {
		test.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(directory, "screen.lock"))
	if err != nil || !strings.HasPrefix(holderOfContent(content), " (process ") {
		test.Fatalf("content %q, error %v", content, err)
	}
	var inUse *InUseError
	if _, err := acquireIn(directory, "screen.lock"); !errors.As(err, &inUse) || inUse.Holder == "" {
		test.Fatalf("second acquire: got %v", err)
	}
	release()
	release()
	again, err := acquireIn(directory, "screen.lock")
	if err != nil {
		test.Fatalf("after release: %v", err)
	}
	again()
}

func TestAcquireInNamesThePathOfAFailure(test *testing.T) {
	file := filepath.Join(test.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	if _, err := acquireIn(file, "screen.lock"); err == nil || !strings.Contains(err.Error(), file) {
		test.Fatalf("got %v", err)
	}
}
