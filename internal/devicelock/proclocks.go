package devicelock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// holderOfNode names the process that holds a FLOCK lock on the device and inode, from <procRoot>/locks. A waiter, a
// line whose second field is "->", is skipped. It gives the empty string when no holder is found.
func holderOfNode(procRoot string, major, minor uint32, inode uint64) string {
	content, err := os.ReadFile(filepath.Join(procRoot, "locks"))
	if err != nil {
		return ""
	}
	wanted := fmt.Sprintf("%02x:%02x:%d", major, minor, inode)
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[1] != "FLOCK" || fields[5] != wanted {
			continue
		}
		pid, err := strconv.Atoi(fields[4])
		if err != nil || pid <= 0 {
			return ""
		}
		commandLine, known := commandLineOf(procRoot, pid)
		return HolderText(pid, commandLine, known)
	}
	return ""
}

func commandLineOf(procRoot string, pid int) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "cmdline"))
	if err != nil || len(raw) == 0 {
		return "", false
	}
	arguments := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
	return strings.Join(arguments, " "), true
}
