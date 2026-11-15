package windowssensors

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// errorGeneralFailure stands for the code of a failure that the system gave as something other than a code.
const errorGeneralFailure syscall.Errno = 31

// kernelLibrary is loaded from the system directory only, and lazily, so that a procedure that cannot be found is an
// error at the first call that needs it and not a panic.
var kernelLibrary = windows.NewLazySystemDLL("kernel32.dll")

var (
	procGlobalMemoryStatusEx = kernelLibrary.NewProc("GlobalMemoryStatusEx")
	procGetSystemTimes       = kernelLibrary.NewProc("GetSystemTimes")
)

// memoryStatus is MEMORYSTATUSEX, which is 64 bytes.
type memoryStatus struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type windowsSeam struct{}

// System is the seam of the system, which calls kernel32.dll and iphlpapi.dll.
func System() Seam { return windowsSeam{} }

// call calls the procedure after finding it, so that a procedure that cannot be found is an error and not the panic of
// LazyProc.Call. It gives an error when the call returned zero, which both procedures use for a failure. It is repeated
// from the seam of the HID layer and not shared, because only the files of the seams may import the Windows package.
//
// The go:uintptrescapes directive makes the conversion of a pointer to uintptr at a call of this function a
// recognised system call boundary: the object stays alive and is not moved until the call returned.
//
//go:uintptrescapes
func call(procedure *windows.LazyProc, arguments ...uintptr) error {
	if err := procedure.Find(); err != nil {
		return err
	}
	result, _, callError := procedure.Call(arguments...)
	if result != 0 {
		return nil
	}
	var code syscall.Errno
	if errors.As(callError, &code) && code != 0 {
		return code
	}
	return errorGeneralFailure
}

func (windowsSeam) Memory() (total, available uint64, err error) {
	status := memoryStatus{Length: uint32(unsafe.Sizeof(memoryStatus{}))}
	if err := call(procGlobalMemoryStatusEx, uintptr(unsafe.Pointer(&status))); err != nil {
		return 0, 0, err
	}
	return status.TotalPhys, status.AvailPhys, nil
}

func (windowsSeam) SystemTimes() (idle, kernel, user uint64, err error) {
	var idleTime, kernelTime, userTime windows.Filetime
	if err := call(procGetSystemTimes,
		uintptr(unsafe.Pointer(&idleTime)), uintptr(unsafe.Pointer(&kernelTime)), uintptr(unsafe.Pointer(&userTime))); err != nil {
		return 0, 0, 0, err
	}
	return count(idleTime), count(kernelTime), count(userTime), nil
}

// count is the 100 nanosecond units of a FILETIME, from its two halves.
func count(time windows.Filetime) uint64 {
	return uint64(time.HighDateTime)<<32 | uint64(time.LowDateTime)
}

// Interfaces reads the table of interfaces and frees it. The Windows package binds GetIfTable2Ex so that it panics
// with the error of the library when the procedure cannot be found, which is recovered here as the failure of the call
// and nothing else is.
func (windowsSeam) Interfaces() (rows []Interface, err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		var notFound *windows.DLLError
		if failure, isError := recovered.(error); isError && errors.As(failure, &notFound) {
			rows, err = nil, notFound
			return
		}
		panic(recovered)
	}()
	var table *windows.MibIfTable2
	if err := windows.GetIfTable2Ex(windows.MibIfTableNormal, &table); err != nil {
		return nil, err
	}
	defer windows.FreeMibTable(unsafe.Pointer(table))
	if table.NumEntries == 0 {
		return nil, nil
	}
	for _, row := range unsafe.Slice(&table.Table[0], table.NumEntries) {
		rows = append(rows, Interface{
			LUID: row.InterfaceLuid, Flags: row.InterfaceAndOperStatusFlags, InOctets: row.InOctets, OutOctets: row.OutOctets,
		})
	}
	return rows, nil
}
