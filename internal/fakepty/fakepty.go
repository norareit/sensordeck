//go:build linux

// Package fakepty is test support: pseudo terminals that stand in for the hidraw nodes of fake devices.
package fakepty

import (
	"fmt"
	"os"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// Open makes a pseudo terminal in raw mode. The master is what the fake device reads and writes, and the slave is
// held open by the caller so that the raw settings survive a close of the node by the code under test. The slave is
// reached at slavePath.
func Open(test testing.TB) (master, slave *os.File, slavePath string) {
	test.Helper()
	master, slavePath = makeTerminal(test)
	slave, err := os.OpenFile(slavePath, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		test.Fatalf("fake terminal: opening the slave: %v", err)
	}
	makeRaw(test, slave)
	return master, slave, slavePath
}

// makeTerminal opens a new pseudo terminal and returns its master and the path of its slave. The descriptor is
// reached only through SyscallConn, because Fd would put the file into blocking mode, and a blocked read could then
// not be ended by Close.
func makeTerminal(test testing.TB) (*os.File, string) {
	test.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		test.Fatalf("fake terminal: cannot make a pseudo terminal: %v", err)
	}
	var number int
	control(test, master, func(descriptor int) error {
		if err := unix.IoctlSetPointerInt(descriptor, unix.TIOCSPTLCK, 0); err != nil {
			return err
		}
		number, err = unix.IoctlGetInt(descriptor, unix.TIOCGPTN)
		return err
	})
	return master, fmt.Sprintf("/dev/pts/%d", number)
}

// makeRaw sets the terminal to pass every byte unchanged in both directions, without echo.
func makeRaw(test testing.TB, slave *os.File) {
	test.Helper()
	control(test, slave, func(descriptor int) error {
		settings, err := unix.IoctlGetTermios(descriptor, unix.TCGETS)
		if err != nil {
			return err
		}
		settings.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
		settings.Oflag &^= unix.OPOST
		settings.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
		settings.Cflag &^= unix.CSIZE | unix.PARENB
		settings.Cflag |= unix.CS8
		settings.Cc[unix.VMIN] = 1
		settings.Cc[unix.VTIME] = 0
		return unix.IoctlSetTermios(descriptor, unix.TCSETS, settings)
	})
}

func control(test testing.TB, file *os.File, action func(descriptor int) error) {
	test.Helper()
	connection, err := file.SyscallConn()
	if err != nil {
		test.Fatalf("fake terminal: %v", err)
	}
	var actionErr error
	if err := connection.Control(func(descriptor uintptr) { actionErr = action(int(descriptor)) }); err != nil {
		test.Fatalf("fake terminal: %v", err)
	}
	if actionErr != nil {
		test.Fatalf("fake terminal: %v", actionErr)
	}
}
