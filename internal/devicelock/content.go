package devicelock

import (
	"fmt"
	"strconv"
	"strings"
)

// encodeContent is what a holder writes into its lock file: its process ID and command line, one on each line.
func encodeContent(pid int, commandLine string) []byte {
	return []byte(fmt.Sprintf("%d\n%s\n", pid, commandLine))
}

// holderOfContent gives the holder text of a lock file's content, or the empty string when the content is not a
// process ID and a command line.
func holderOfContent(content []byte) string {
	lines := strings.Split(string(content), "\n")
	if len(lines) != 3 || lines[2] != "" {
		return ""
	}
	pid, err := strconv.Atoi(lines[0])
	if err != nil || pid <= 0 {
		return ""
	}
	return HolderText(pid, lines[1], true)
}
