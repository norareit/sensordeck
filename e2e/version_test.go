//go:build linux

package e2e

import (
	"regexp"
	"testing"

	"github.com/norareit/sensordeck/internal/fakescreen"
)

func TestVersionPrintsOneLineAndSucceeds(test *testing.T) {
	got := runWith(test, fakescreen.New(test), nil, "--version")
	if !regexp.MustCompile(`^sensordeck version \S+\n$`).MatchString(got.stdout) || got.stderr != "" || got.status != 0 {
		test.Errorf("status %d, stdout %q, stderr %q", got.status, got.stdout, got.stderr)
	}
}
