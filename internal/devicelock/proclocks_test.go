package devicelock

import (
	"os"
	"path/filepath"
	"testing"
)

const fakeLocks = `1: POSIX  ADVISORY  WRITE 100 00:85:1 0 EOF
30: FLOCK  ADVISORY  WRITE 496 00:85:627 0 EOF
30: -> FLOCK  ADVISORY  WRITE 497 00:85:627 0 EOF
`

func fakeProc(test *testing.T, locks string, cmdline *string) string {
	root := test.TempDir()
	if locks != "" {
		if err := os.WriteFile(filepath.Join(root, "locks"), []byte(locks), 0o644); err != nil {
			test.Fatal(err)
		}
	}
	if cmdline != nil {
		if err := os.MkdirAll(filepath.Join(root, "496"), 0o755); err != nil {
			test.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "496", "cmdline"), []byte(*cmdline), 0o644); err != nil {
			test.Fatal(err)
		}
	}
	return root
}

func TestHolderOfNode(test *testing.T) {
	full := "sensordeck\x00run\x00--config\x00c.yaml\x00"
	empty := ""
	cases := []struct {
		name    string
		locks   string
		cmdline *string
		minor   uint32
		inode   uint64
		want    string
	}{
		{"full", fakeLocks, &full, 0x85, 627, " (process 496: sensordeck run --config c.yaml)"},
		{"no cmdline", fakeLocks, nil, 0x85, 627, " (process 496)"},
		{"empty cmdline", fakeLocks, &empty, 0x85, 627, " (process 496)"},
		{"other inode", fakeLocks, &full, 0x85, 628, ""},
		{"other device", fakeLocks, &full, 0x86, 627, ""},
		{"no locks", "", &full, 0x85, 627, ""},
		{"bad pid", "30: FLOCK  ADVISORY  WRITE -1 00:85:627 0 EOF\n", &full, 0x85, 627, ""},
	}
	for _, each := range cases {
		root := fakeProc(test, each.locks, each.cmdline)
		if got := holderOfNode(root, 0, each.minor, each.inode); got != each.want {
			test.Errorf("%s: got %q, want %q", each.name, got, each.want)
		}
	}
}
