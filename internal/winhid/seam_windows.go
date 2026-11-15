package winhid

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// errorGeneralFailure stands for the code of a failure that the system gave as something other than a code.
const errorGeneralFailure syscall.Errno = 31

// hidLibrary is loaded from the system directory only, and lazily, so that a machine without it gives an error at the
// first call that needs it and not a panic.
var hidLibrary = windows.NewLazySystemDLL("hid.dll")

var (
	procGetHidGUID        = hidLibrary.NewProc("HidD_GetHidGuid")
	procGetAttributes     = hidLibrary.NewProc("HidD_GetAttributes")
	procGetProductString  = hidLibrary.NewProc("HidD_GetProductString")
	procGetSerialString   = hidLibrary.NewProc("HidD_GetSerialNumberString")
	procGetPreparsedData  = hidLibrary.NewProc("HidD_GetPreparsedData")
	procFreePreparsedData = hidLibrary.NewProc("HidD_FreePreparsedData")
	procGetCapabilities   = hidLibrary.NewProc("HidP_GetCaps")
	proceduresOfTheSeam   = []*windows.LazyProc{procGetHidGUID, procGetAttributes, procGetProductString,
		procGetSerialString, procGetPreparsedData, procFreePreparsedData, procGetCapabilities}
)

const (
	// hidpStatusSuccess is HIDP_STATUS_SUCCESS.
	hidpStatusSuccess = 0x00110000
	// stringUnits is the size of the buffer of a string, in UTF-16 units.
	stringUnits = 256
	shareMode   = windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE
)

// hidAttributes is HIDD_ATTRIBUTES.
type hidAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

// hidCaps is HIDP_CAPS.
type hidCaps struct {
	Usage, UsagePage                                uint16
	InputReportByteLength, OutputReportByteLength   uint16
	FeatureReportByteLength                         uint16
	Reserved                                        [17]uint16
	NumberLinkCollectionNodes                       uint16
	NumberInputButtonCaps, NumberInputValueCaps     uint16
	NumberInputDataIndices                          uint16
	NumberOutputButtonCaps, NumberOutputValueCaps   uint16
	NumberOutputDataIndices                         uint16
	NumberFeatureButtonCaps, NumberFeatureValueCaps uint16
	NumberFeatureDataIndices                        uint16
}

type windowsSeam struct{}

// Windows is the seam of the system, which calls hid.dll and the file functions of Windows.
func Windows() Seam { return windowsSeam{} }

// call calls the procedure after finding it, so that a procedure that cannot be found is an error that names it and
// not the panic of LazyProc.Call.
// It gives the result, the last error of the thread, which means something only when the result says the call
// failed, and the error of finding.
func call(procedure *windows.LazyProc, arguments ...uintptr) (result uintptr, lastError syscall.Errno, err error) {
	if err := procedure.Find(); err != nil {
		return 0, 0, err
	}
	result, _, callError := procedure.Call(arguments...)
	errors.As(callError, &lastError)
	return result, lastError, nil
}

func (windowsSeam) Paths() ([]string, error) {
	var guid windows.GUID
	if _, _, err := call(procGetHidGUID, uintptr(unsafe.Pointer(&guid))); err != nil {
		return nil, err
	}
	paths, err := windows.CM_Get_Device_Interface_List("", &guid, windows.CM_GET_DEVICE_INTERFACE_LIST_PRESENT)
	if errors.Is(err, windows.ERROR_NO_SUCH_DEVICE_INTERFACE) {
		return nil, nil
	}
	return paths, err
}

func (windowsSeam) OpenQuery(path string) (Handle, error) { return openPath(path, 0, 0) }

func (windowsSeam) OpenReadWrite(path string) (Handle, error) {
	return openPath(path, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_FLAG_OVERLAPPED)
}

func openPath(path string, access, flags uint32) (Handle, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	handle, err := windows.CreateFile(name, access, shareMode, nil, windows.OPEN_EXISTING, flags, 0)
	if err != nil {
		return 0, err
	}
	return Handle(handle), nil
}

func (windowsSeam) Attributes(handle Handle) (uint16, uint16, error) {
	attributes := hidAttributes{Size: uint32(unsafe.Sizeof(hidAttributes{}))}
	if err := succeeded(call(procGetAttributes, uintptr(handle), uintptr(unsafe.Pointer(&attributes)))); err != nil {
		return 0, 0, err
	}
	return attributes.VendorID, attributes.ProductID, nil
}

func (windowsSeam) Product(handle Handle) (string, error) {
	return readString(procGetProductString, handle)
}
func (windowsSeam) Serial(handle Handle) (string, error) {
	return readString(procGetSerialString, handle)
}

func readString(procedure *windows.LazyProc, handle Handle) (string, error) {
	var buffer [stringUnits]uint16
	size := uintptr(len(buffer) * 2)
	if err := succeeded(call(procedure, uintptr(handle), uintptr(unsafe.Pointer(&buffer[0])), size)); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buffer[:]), nil
}

func (windowsSeam) Caps(handle Handle) (Caps, error) {
	var preparsed uintptr
	if err := succeeded(call(procGetPreparsedData, uintptr(handle), uintptr(unsafe.Pointer(&preparsed)))); err != nil {
		return Caps{}, err
	}
	// The preparsed data is freed on every path after it was got.
	defer func() { _, _, _ = call(procFreePreparsedData, preparsed) }()
	var caps hidCaps
	status, _, err := call(procGetCapabilities, preparsed, uintptr(unsafe.Pointer(&caps)))
	if err != nil {
		return Caps{}, err
	}
	if status != hidpStatusSuccess {
		return Caps{}, errors.New("HidP_GetCaps gave the status " + windows.NTStatus(status).Error())
	}
	return Caps{UsagePage: caps.UsagePage, Usage: caps.Usage,
		InputLength: caps.InputReportByteLength, OutputLength: caps.OutputReportByteLength}, nil
}

// succeeded gives the error of a call of a procedure that returns a BOOLEAN, which is zero for a failure.
func succeeded(result uintptr, lastError syscall.Errno, err error) error {
	switch {
	case err != nil:
		return err
	case result != 0:
		return nil
	case lastError == 0:
		return errorGeneralFailure
	}
	return lastError
}

func (windowsSeam) Close(handle Handle) error { return windows.CloseHandle(windows.Handle(handle)) }

func (windowsSeam) StartRead(handle Handle, buffer []byte) (Operation, error) {
	return start(handle, buffer, windows.ReadFile)
}

func (windowsSeam) StartWrite(handle Handle, buffer []byte) (Operation, error) {
	return start(handle, buffer, windows.WriteFile)
}

// operationOfWindows is an overlapped operation. The buffer and the overlapped structure stay reachable through it
// until it was reaped, since the system uses both until then.
type operationOfWindows struct {
	handle     windows.Handle
	event      windows.Handle
	overlapped *windows.Overlapped
	buffer     []byte
}

func start(handle Handle, buffer []byte, transfer func(windows.Handle, []byte, *uint32, *windows.Overlapped) error) (Operation, error) {
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, err
	}
	operation := &operationOfWindows{handle: windows.Handle(handle), event: event, buffer: buffer,
		overlapped: &windows.Overlapped{HEvent: event}}
	var done uint32
	if err := transfer(operation.handle, buffer, &done, operation.overlapped); err != nil &&
		!errors.Is(err, windows.ERROR_IO_PENDING) {
		_ = windows.CloseHandle(event)
		return nil, err
	}
	return operation, nil
}

func (operation *operationOfWindows) Wait(milliseconds uint32) bool {
	signalled, err := windows.WaitForSingleObject(operation.event, milliseconds)
	return err == nil && signalled == windows.WAIT_OBJECT_0
}

func (operation *operationOfWindows) Cancel() {
	_ = windows.CancelIoEx(operation.handle, operation.overlapped)
}

func (operation *operationOfWindows) Reap() (uint32, syscall.Errno) {
	var count uint32
	err := windows.GetOverlappedResult(operation.handle, operation.overlapped, &count, true)
	_ = windows.CloseHandle(operation.event)
	runtime.KeepAlive(operation.buffer)
	runtime.KeepAlive(operation.overlapped)
	if err == nil {
		return count, 0
	}
	var code syscall.Errno
	if !errors.As(err, &code) {
		code = errorGeneralFailure
	}
	return count, code
}
