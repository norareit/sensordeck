// Package windowssensors reads the metrics that Windows gives a program with nothing to install: the memory, the times
// of the CPU and the counters of the network interfaces, and with them the GPU. The calls into Windows are behind a
// seam, so that everything but the calls is tested on any system. Only the files of the seam are limited to Windows.
package windowssensors

// Interface is one row of the table of network interfaces, with the fields that the sources use.
type Interface struct {
	LUID      uint64 // unique while the interface exists, which its index is not
	Flags     uint8  // InterfaceAndOperStatusFlags
	InOctets  uint64 // bytes received
	OutOctets uint64 // bytes sent
}

// Seam is the calls into Windows that the sources make, and nothing else. Each gives plain values, or a failure.
type Seam interface {
	// Memory is GlobalMemoryStatusEx: the physical memory the system can use, and the part that is available, in bytes.
	Memory() (total, available uint64, err error)
	// SystemTimes is GetSystemTimes: the idle, kernel and user time of all processors since boot, in units of 100
	// nanoseconds. The kernel time includes the idle time.
	SystemTimes() (idle, kernel, user uint64, err error)
	// Interfaces is GetIfTable2Ex at the level MibIfTableNormal, with the table freed again.
	Interfaces() ([]Interface, error)
}
