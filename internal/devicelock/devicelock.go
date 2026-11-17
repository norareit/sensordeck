// Package devicelock holds the advisory locks that make one sensordeck process at a time use the controller or the
// screen, and the naming of the process that holds one. A lock binds sensordeck processes and no other program.
package devicelock

import (
	"fmt"
	"strconv"
)

// InUseError is a lock that another sensordeck process holds. Holder is the text of HolderText, or empty when the
// holder cannot be found.
type InUseError struct{ Holder string }

func (inUse *InUseError) Error() string {
	return "in use by another sensordeck process" + inUse.Holder
}

// HolderText names a process as " (process <pid>: <command line>)", or " (process <pid>)" when the command line is
// not known or is empty.
func HolderText(pid int, commandLine string, commandLineKnown bool) string {
	if !commandLineKnown || commandLine == "" {
		return " (process " + strconv.Itoa(pid) + ")"
	}
	return fmt.Sprintf(" (process %d: %s)", pid, commandLine)
}
