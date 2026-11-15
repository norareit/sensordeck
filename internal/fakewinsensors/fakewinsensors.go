// Package fakewinsensors is a fake of the seam of the sources of Windows. It gives what the test sets and counts the
// calls of each, so that the sources are tested on any system. It is safe for use from several goroutines.
package fakewinsensors

import (
	"sync"

	"github.com/norareit/sensordeck/internal/windowssensors"
)

// Times is what GetSystemTimes gives.
type Times struct{ Idle, Kernel, User uint64 }

// Fake is the seam. The lists of times and of tables are taken one for each call, and the last one is given again once
// the list is used up. A list that is empty gives zeros and no rows.
type Fake struct {
	// Total and Available are what the call of the memory gives, and MemoryError makes it fail.
	Total, Available uint64
	MemoryError      error
	// TimesList is the times the calls give, and TimesError makes them fail.
	TimesList  []Times
	TimesError error
	// Tables is the tables of interfaces the calls give, and InterfacesError makes them fail.
	Tables          [][]windowssensors.Interface
	InterfacesError error

	mutex                                   sync.Mutex
	memoryCalls, timesCalls, interfaceCalls int
}

var _ windowssensors.Seam = (*Fake)(nil)

func (fake *Fake) Memory() (total, available uint64, err error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.memoryCalls++
	return fake.Total, fake.Available, fake.MemoryError
}

func (fake *Fake) SystemTimes() (idle, kernel, user uint64, err error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.timesCalls++
	if fake.TimesError != nil || len(fake.TimesList) == 0 {
		return 0, 0, 0, fake.TimesError
	}
	taken := fake.TimesList[min(fake.timesCalls, len(fake.TimesList))-1]
	return taken.Idle, taken.Kernel, taken.User, nil
}

func (fake *Fake) Interfaces() ([]windowssensors.Interface, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.interfaceCalls++
	if fake.InterfacesError != nil || len(fake.Tables) == 0 {
		return nil, fake.InterfacesError
	}
	return append([]windowssensors.Interface(nil), fake.Tables[min(fake.interfaceCalls, len(fake.Tables))-1]...), nil
}

// MemoryCalls is how many times the memory was asked.
func (fake *Fake) MemoryCalls() int { return fake.count(&fake.memoryCalls) }

// TimesCalls is how many times the times of the CPU were asked.
func (fake *Fake) TimesCalls() int { return fake.count(&fake.timesCalls) }

// InterfaceCalls is how many times the table of interfaces was asked.
func (fake *Fake) InterfaceCalls() int { return fake.count(&fake.interfaceCalls) }

func (fake *Fake) count(calls *int) int {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return *calls
}
