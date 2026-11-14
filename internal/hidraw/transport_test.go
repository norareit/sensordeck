//go:build linux

package hidraw

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakescreen"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

const shortLimit = 50 * time.Millisecond

func reportOf(data ...byte) smartmonitor.Report {
	var report smartmonitor.Report
	copy(report[:], data)
	return report
}

func openFake(test *testing.T, fake *fakescreen.Fake) *Transport {
	test.Helper()
	transport, err := OpenDiscovered(fake.SysfsRoot, fake.DevRoot)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { _ = transport.Close() })
	return transport
}

func TestWriteArrivesAs65Bytes(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	transport := openFake(test, fake)
	report := reportOf(0x01, 'r', 'e', 's', 'e', 't')
	if err := transport.Write(report); err != nil {
		test.Fatal(err)
	}
	want := append([]byte{0x00}, report[:]...)
	if got := screen.Received(); !bytes.Equal(got, want) {
		test.Fatalf("got %d bytes % X", len(got), got)
	}
}

func TestReadGivesAReport(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	transport := openFake(test, fake)
	screen.Send(reportOf(0x06))
	got, err := transport.Read(time.Second)
	if err != nil || got != reportOf(0x06) {
		test.Fatalf("got %v, %v", got, err)
	}
}

func TestTwoReportsAreReadAsTwo(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	transport := openFake(test, fake)
	screen.Send(reportOf(0x01, 0xAA))
	screen.Send(reportOf(0x02, 0xBB))
	for _, want := range []smartmonitor.Report{reportOf(0x01, 0xAA), reportOf(0x02, 0xBB)} {
		got, err := transport.Read(time.Second)
		if err != nil || got != want {
			test.Fatalf("got %v, %v", got, err)
		}
	}
}

func TestShortReadIsFilledUp(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	transport := openFake(test, fake)
	screen.SendBytes([]byte{0x06})
	got, err := transport.Read(time.Second)
	if err != nil || got != reportOf(0x06) {
		test.Fatalf("got %v, %v", got, err)
	}
}

func TestReadTimeLimitPasses(test *testing.T) {
	fake := fakescreen.New(test)
	fake.Add(3, "name", "")
	transport := openFake(test, fake)
	started := time.Now()
	_, err := transport.Read(shortLimit)
	if !errors.Is(err, smartmonitor.ErrReadTimeout) {
		test.Fatalf("got %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		test.Errorf("the read took %v", elapsed)
	}
}

func TestReadAfterTheScreenWentIsAnotherError(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	transport := openFake(test, fake)
	screen.Remove()
	_, err := transport.Read(time.Second)
	if err == nil || errors.Is(err, smartmonitor.ErrReadTimeout) {
		test.Fatalf("got %v", err)
	}
}

func TestReopenFindsTheScreenUnderAnotherNumber(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	transport := openFake(test, fake)
	screen.Move(7)
	if err := transport.Reopen(); err != nil {
		test.Fatal(err)
	}
	if err := transport.Write(reportOf(0x02)); err != nil {
		test.Fatal(err)
	}
	if got := screen.Received(); len(got) != 65 || got[1] != 0x02 {
		test.Fatalf("got %d bytes", len(got))
	}
}

func TestReopenWithNoScreenFailsAndASecondOneSucceeds(test *testing.T) {
	fake := fakescreen.New(test)
	screen := fake.Add(3, "name", "")
	transport := openFake(test, fake)
	screen.Remove()
	if err := transport.Reopen(); err == nil || err.Error() != "no supported screen found" {
		test.Fatalf("got %v", err)
	}
	if err := transport.Write(reportOf(1)); err == nil {
		test.Error("a write succeeded with no node open")
	}
	if _, err := transport.Read(shortLimit); err == nil || errors.Is(err, smartmonitor.ErrReadTimeout) {
		test.Errorf("a read gave %v with no node open", err)
	}
	replacement := fake.Add(5, "name", "")
	if err := transport.Reopen(); err != nil {
		test.Fatal(err)
	}
	if err := transport.Write(reportOf(0x09)); err != nil {
		test.Fatal(err)
	}
	if got := replacement.Received(); len(got) != 65 || got[1] != 0x09 {
		test.Fatalf("got %d bytes", len(got))
	}
}

func TestOpenDiscoveredErrors(test *testing.T) {
	fake := fakescreen.New(test)
	if _, err := OpenDiscovered(fake.SysfsRoot, fake.DevRoot); err == nil || err.Error() != "no supported screen found" {
		test.Fatalf("got %v", err)
	}
	fake.Add(2, "llhmi.com llTechCo.,Ltd", "")
	fake.Add(10, "other", "S1")
	_, err := OpenDiscovered(fake.SysfsRoot, fake.DevRoot)
	want := "more than one supported screen is connected, name one with --device:\n" +
		fake.DevRoot + "/hidraw2\t0483:0065\tllhmi.com llTechCo.,Ltd\t-\n" +
		fake.DevRoot + "/hidraw10\t0483:0065\tother\tS1"
	if err == nil || err.Error() != want {
		test.Fatalf("got %v, want %q", err, want)
	}
}

func TestOpenPathOpensTheNodeGiven(test *testing.T) {
	fake := fakescreen.New(test)
	first := fake.Add(2, "name", "")
	second := fake.Add(4, "name", "")
	transport, err := OpenPath(second.Node())
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { _ = transport.Close() })
	if err := transport.Write(reportOf(0x05)); err != nil {
		test.Fatal(err)
	}
	if got := second.Received(); len(got) != 65 || got[1] != 0x05 {
		test.Fatalf("got %d bytes", len(got))
	}
	if got := first.Received(); len(got) != 0 {
		test.Fatalf("the other screen got %d bytes", len(got))
	}
	if err := transport.Reopen(); err != nil {
		test.Fatalf("reopening on the given path: %v", err)
	}
}

func TestOpenPathOnANodeWithoutATimeLimit(test *testing.T) {
	path := filepath.Join(test.TempDir(), "regular")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	_, err := OpenPath(path)
	if err == nil || !strings.Contains(err.Error(), "does not support a time limit for reading") {
		test.Fatalf("got %v", err)
	}
}

func TestOpenErrorForDeniedPermission(test *testing.T) {
	err := openError("/dev/hidraw3", &fs.PathError{Op: "open", Path: "/dev/hidraw3", Err: syscall.EACCES})
	want := `cannot open /dev/hidraw3: permission denied. Install the udev rule and replug the screen, see "Linux: access to the screen" in https://github.com/norareit/sensordeck#linux-access-to-the-screen`
	if err.Error() != want {
		test.Errorf("got %q", err)
	}
	if !errors.Is(err, fs.ErrPermission) {
		test.Error("the system error is not reachable")
	}
}

func TestOpenErrorForOtherFailures(test *testing.T) {
	err := openError("/dev/hidraw3", &fs.PathError{Op: "open", Path: "/dev/hidraw3", Err: syscall.ENOENT})
	if want := "cannot open /dev/hidraw3: no such file or directory"; err.Error() != want {
		test.Errorf("got %q", err)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		test.Error("the system error is not reachable")
	}
	_, err = OpenPath(filepath.Join(test.TempDir(), "missing"))
	if err == nil || !strings.HasSuffix(err.Error(), ": no such file or directory") || !strings.HasPrefix(err.Error(), "cannot open ") {
		test.Errorf("got %v", err)
	}
}
