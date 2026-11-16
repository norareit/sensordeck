//go:build linux

package hidraw

import (
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// controlTransfer is struct usbdevfs_ctrltransfer of the kernel.
type controlTransfer struct {
	requestType uint8
	request     uint8
	value       uint16
	index       uint16
	length      uint16
	timeout     uint32 // milliseconds
	data        unsafe.Pointer
}

// usbdevfsControl is USBDEVFS_CONTROL, _IOWR('U', 0, struct usbdevfs_ctrltransfer): direction read and write in the top
// two bits, the size of the struct, the type 'U' and the number 0.
const usbdevfsControl = uintptr(3<<30 | unsafe.Sizeof(controlTransfer{})<<16 | 'U'<<8)

// SystemUsbfs opens the usbfs nodes of the system.
type SystemUsbfs struct{}

// Open opens the node for reading and writing.
func (SystemUsbfs) Open(path string) (UsbfsNode, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return systemUsbfsNode{file: file}, nil
}

type systemUsbfsNode struct{ file *os.File }

// Control makes the request, whose answer is read into a buffer that is dropped. A failure gives the error of the
// system, so that errors.Is finds it.
func (node systemUsbfsNode) Control(request ControlRequest) error {
	buffer := make([]byte, request.Length)
	transfer := controlTransfer{
		requestType: request.RequestType,
		request:     request.Request,
		value:       request.Value,
		index:       request.Index,
		length:      request.Length,
		timeout:     uint32(request.Timeout.Milliseconds()),
		data:        unsafe.Pointer(&buffer[0]),
	}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, node.file.Fd(), usbdevfsControl, uintptr(unsafe.Pointer(&transfer)))
	runtime.KeepAlive(buffer)
	if errno != 0 {
		return os.NewSyscallError("ioctl USBDEVFS_CONTROL", errno)
	}
	return nil
}

func (node systemUsbfsNode) Close() error { return node.file.Close() }
