package winhid_test

import (
	"errors"
	"io"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakehid"
	"github.com/norareit/sensordeck/internal/smartmonitor"
	"github.com/norareit/sensordeck/internal/winhid"
)

func openFake(test *testing.T, collection fakehid.Collection) (*fakehid.Fake, *winhid.Device) {
	test.Helper()
	fake := fakehid.New(collection)
	device, err := winhid.Open(fake, collection.Path)
	if err != nil {
		test.Fatal(err)
	}
	return fake, device
}

func TestOpenOpensOnlyThePathForReadingAndWritingAndReadsCaps(test *testing.T) {
	fake := fakehid.New(screen("a", ""), screen("b", "S1"))
	device, err := winhid.Open(fake, "b")
	if err != nil {
		test.Fatal(err)
	}
	if openings := fake.Openings(); len(openings) != 1 || openings[0].Path != "b" || !openings[0].ReadWrite {
		test.Errorf("openings %v", openings)
	}
	if device.Caps() != screenCaps {
		test.Errorf("caps %v", device.Caps())
	}
}

func TestOpenWordsAFailedOpeningAndClosesWhenTheCapsFail(test *testing.T) {
	denied := screen("a", "")
	denied.OpenError = syscall.Errno(5)
	if _, err := winhid.Open(fakehid.New(denied), "a"); err == nil || err.Error() !=
		"cannot open a: "+syscall.Errno(5).Error()+". Another program may hold the device, such as Corsair iCUE or the software of the screen's vendor; close it and try again" {
		test.Errorf("got %v", err)
	}
	noCaps := screen("b", "")
	noCaps.CapsError = errors.New("no caps")
	fake := fakehid.New(noCaps)
	if _, err := winhid.Open(fake, "b"); err == nil || !fake.AllClosed() {
		test.Errorf("err %v, closed %v", err, fake.AllClosed())
	}
}

func TestWriteReportWritesABufferOfTheOutputLength(test *testing.T) {
	fake, device := openFake(test, screen("a", ""))
	if err := device.WriteReport([]byte{1, 2, 3}, winhid.WriteTimeLimit); err != nil {
		test.Fatal(err)
	}
	written := fake.Writes()[0]
	if len(written) != 65 || written[0] != 0 || written[1] != 1 || written[3] != 3 || written[4] != 0 {
		test.Errorf("written % x", written)
	}
	if waits := fake.Waits(); len(waits) != 1 || waits[0] != 5000 {
		test.Errorf("waits %v", waits)
	}
	if fake.Pending() {
		test.Error("an operation is pending")
	}
}

func TestWriteReportTimesOutAfterTheLimitAndReaps(test *testing.T) {
	fake, device := openFake(test, screen("a", ""))
	fake.SetEndings(fakehid.Ending{WaitsOut: true, Code: 995, Count: fakehid.Bytes(0)})
	err := device.WriteReport(nil, winhid.WriteTimeLimit)
	if err == nil || err.Error() != "write timed out after 5s" || errors.Is(err, smartmonitor.ErrReadTimeout) {
		test.Fatalf("got %v", err)
	}
	if fake.Pending() || fake.Cancels() != 1 || fake.ReapedWithoutCancel() != 0 {
		test.Errorf("pending %v cancels %d", fake.Pending(), fake.Cancels())
	}
}

func TestAWriteThatCompletedAfterTheWaitIsASuccess(test *testing.T) {
	fake, device := openFake(test, screen("a", ""))
	fake.SetEndings(fakehid.Ending{WaitsOut: true})
	if err := device.WriteReport(nil, winhid.WriteTimeLimit); err != nil {
		test.Fatal(err)
	}
	if fake.Pending() {
		test.Error("an operation is pending")
	}
}

func TestAShortWriteFails(test *testing.T) {
	fake, device := openFake(test, screen("a", ""))
	fake.SetEndings(fakehid.Ending{Count: fakehid.Bytes(40)})
	if err := device.WriteReport(nil, winhid.WriteTimeLimit); err == nil || err.Error() != "short write: 40 of 65 bytes" {
		test.Fatalf("got %v", err)
	}
}

func TestReadReportDropsTheReportID(test *testing.T) {
	collection := screen("a", "")
	collection.Input = [][]byte{append([]byte{0}, 9, 8, 7)}
	fake, device := openFake(test, collection)
	got, err := device.ReadReport(time.Second)
	if err != nil || len(got) != 64 || got[0] != 9 || got[2] != 7 {
		test.Fatalf("got %v, %v", got, err)
	}
	if waits := fake.Waits(); len(waits) != 1 || waits[0] != 1000 {
		test.Errorf("waits %v", waits)
	}
}

func TestReadReportOfNoBytesIsEOF(test *testing.T) {
	fake, device := openFake(test, screen("a", ""))
	fake.SetEndings(fakehid.Ending{Count: fakehid.Bytes(0)})
	if _, err := device.ReadReport(time.Second); !errors.Is(err, io.EOF) {
		test.Fatalf("got %v", err)
	}
}

func TestReadReportOfAShortReportGivesItsBytes(test *testing.T) {
	collection := screen("a", "")
	collection.Input = [][]byte{{0, 5, 6}}
	fake, device := openFake(test, collection)
	fake.SetEndings(fakehid.Ending{Count: fakehid.Bytes(3)})
	got, err := device.ReadReport(time.Second)
	if err != nil || len(got) != 2 || got[0] != 5 || got[1] != 6 {
		test.Fatalf("got %v, %v", got, err)
	}
}

func TestReadReportCancelsAndReapsWhenTheWaitEndsWithoutCompletion(test *testing.T) {
	fake, device := openFake(test, screen("a", ""))
	fake.SetEndings(fakehid.Ending{WaitsOut: true, Code: 995, Count: fakehid.Bytes(0)})
	_, err := device.ReadReport(1500 * time.Microsecond)
	if !errors.Is(err, smartmonitor.ErrReadTimeout) {
		test.Fatalf("got %v", err)
	}
	if waits := fake.Waits(); waits[0] != 2 || fake.Cancels() != 1 || fake.Pending() {
		test.Errorf("waits %v cancels %d pending %v", waits, fake.Cancels(), fake.Pending())
	}
}

func TestAReadThatCompletedAfterTheWaitEndedGivesItsReport(test *testing.T) {
	collection := screen("a", "")
	collection.Input = [][]byte{{0, 42}}
	fake, device := openFake(test, collection)
	fake.SetEndings(fakehid.Ending{WaitsOut: true, Count: fakehid.Bytes(2)})
	got, err := device.ReadReport(time.Second)
	if err != nil || len(got) != 1 || got[0] != 42 || fake.Pending() {
		test.Fatalf("got %v, %v", got, err)
	}
}

func TestAFailedStartStartsNothingMoreAndLeavesNothingPending(test *testing.T) {
	fake, device := openFake(test, screen("a", ""))
	fake.SetEndings(fakehid.Ending{StartError: syscall.Errno(31)}, fakehid.Ending{StartError: syscall.Errno(31)})
	for _, err := range []error{device.WriteReport(nil, time.Second), func() error { _, err := device.ReadReport(time.Second); return err }()} {
		var code syscall.Errno
		if !errors.As(err, &code) || code != 31 || errors.Is(err, smartmonitor.ErrReadTimeout) {
			test.Errorf("got %v", err)
		}
	}
	if fake.Pending() || len(fake.Waits()) != 0 {
		test.Error("something was waited for")
	}
}

func TestACompletionWithACodeFailsWithoutTheTimeout(test *testing.T) {
	fake, device := openFake(test, screen("a", ""))
	fake.SetEndings(fakehid.Ending{Code: 31, Count: fakehid.Bytes(0)})
	if _, err := device.ReadReport(time.Second); err == nil || errors.Is(err, smartmonitor.ErrReadTimeout) ||
		err.Error() != "read failed: "+syscall.Errno(31).Error() {
		test.Errorf("got %v", err)
	}
}

func TestCloseIsIdempotentAndAClosedDeviceFails(test *testing.T) {
	fake, device := openFake(test, screen("a", ""))
	_ = device.Close()
	_ = device.Close()
	if !fake.AllClosed() {
		test.Error("the handle is open")
	}
	if device.WriteReport(nil, time.Second) == nil {
		test.Error("a write on a closed device succeeded")
	}
	if _, err := device.ReadReport(time.Second); err == nil || errors.Is(err, smartmonitor.ErrReadTimeout) {
		test.Errorf("got %v", err)
	}
}
