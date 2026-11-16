package winhid_test

import (
	"errors"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakehid"
	"github.com/norareit/sensordeck/internal/smartmonitor"
	"github.com/norareit/sensordeck/internal/winhid"
)

const shortCapsText = "cannot open p: its reports are 64 and 65 bytes, those of the screen are 65"

func TestOpenDiscoveredWithNoScreen(test *testing.T) {
	_, err := winhid.OpenDiscovered(fakehid.New(fakehid.Collection{Path: "k", Vendor: 1, Product: 1}))
	if err == nil || err.Error() != "no supported screen found" {
		test.Fatalf("got %v", err)
	}
}

func TestOpenDiscoveredWithNearMissesNamesThem(test *testing.T) {
	first, second := screen("a", ""), screen("b", "")
	first.Caps = winhid.Caps{InputLength: 64, OutputLength: 64}
	second.Caps = winhid.Caps{InputLength: 64, OutputLength: 65}
	_, err := winhid.OpenDiscovered(fakehid.New(first, second))
	want := "no supported screen found\n" +
		"found a with reports of 64 and 64 bytes, those of the screen are 65\n" +
		"found b with reports of 64 and 65 bytes, those of the screen are 65"
	if err == nil || err.Error() != want {
		test.Fatalf("got %v", err)
	}
}

func TestOpenDiscoveredBesideANearMissOpensTheScreenAndNamesNothing(test *testing.T) {
	near := screen("a", "")
	near.Caps = winhid.Caps{InputLength: 64, OutputLength: 64}
	fake := fakehid.New(near, screen("b", "S"))
	if _, err := winhid.OpenDiscovered(fake); err != nil {
		test.Fatal(err)
	}
	for _, opening := range fake.Openings() {
		if opening.ReadWrite && opening.Path != "b" {
			test.Errorf("%s was opened for reading and writing", opening.Path)
		}
	}
}

func TestOpenDiscoveredWithTwoScreensListsThem(test *testing.T) {
	_, err := winhid.OpenDiscovered(fakehid.New(screen(screenPath2, "B"), screen(screenPath, "")))
	want := "more than one supported screen is connected, name one with --device:\n" +
		screenPath + "\t0483:0065\tScreen\t-\n" + screenPath2 + "\t0483:0065\tScreen\tB"
	if err == nil || err.Error() != want {
		test.Fatalf("got %v", err)
	}
}

func TestOpenPathOpensWhateverTheIDsAreWhenTheLengthsAreRight(test *testing.T) {
	other := fakehid.Collection{Path: "p", Vendor: 9, Product: 9, Caps: screenCaps, Serial: "X"}
	transport, err := winhid.OpenPath(fakehid.New(other), "p")
	if err != nil || transport == nil {
		test.Fatalf("got %v, %v", err, transport)
	}
}

func TestOpenPathWithWrongLengthsFailsAndClosesTheHandle(test *testing.T) {
	wrong := screen("p", "")
	wrong.Caps = winhid.Caps{InputLength: 64, OutputLength: 65}
	fake := fakehid.New(wrong)
	_, err := winhid.OpenPath(fake, "p")
	if err == nil || err.Error() != shortCapsText || !fake.AllClosed() {
		test.Fatalf("got %v, closed %v", err, fake.AllClosed())
	}
}

func TestOpenPathOfAMissingPathFails(test *testing.T) {
	if _, err := winhid.OpenPath(fakehid.New(), "p"); err == nil {
		test.Fatal("no error")
	}
}

func TestWriteAndReadFrameTheReports(test *testing.T) {
	collection := screen("p", "")
	collection.Input = [][]byte{{0, 1, 2}, report(5, 63)}
	fake := fakehid.New(collection)
	transport, err := winhid.OpenPath(fake, "p")
	if err != nil {
		test.Fatal(err)
	}
	var out smartmonitor.Report
	out[0], out[63] = 0xAA, 0xBB
	if err := transport.Write(out); err != nil {
		test.Fatal(err)
	}
	written := fake.Writes()[0]
	if len(written) != 65 || written[0] != 0 || written[1] != 0xAA || written[64] != 0xBB {
		test.Errorf("written % x", written)
	}
	fake.SetEndings(fakehid.Ending{Count: fakehid.Bytes(3)})
	short, err := transport.Read(time.Second)
	if err != nil || short[0] != 1 || short[1] != 2 || short[2] != 0 {
		test.Fatalf("short read %v, %v", short, err)
	}
	full, err := transport.Read(time.Second)
	if err != nil || full[0] != 5 {
		test.Fatalf("full read %v, %v", full, err)
	}
	if fake.Pending() {
		test.Error("an operation is pending")
	}
}

func TestReadTimesOutWithTheTimeoutOfTheProtocolPackage(test *testing.T) {
	fake := fakehid.New(screen("p", ""))
	transport, _ := winhid.OpenPath(fake, "p")
	fake.SetEndings(fakehid.Ending{WaitsOut: true, Code: 995, Count: fakehid.Bytes(0)})
	if _, err := transport.Read(10 * time.Millisecond); !errors.Is(err, smartmonitor.ErrReadTimeout) {
		test.Fatalf("got %v", err)
	}
	if fake.Pending() {
		test.Error("an operation is pending")
	}
}

func TestReopenDoesTheDiscoveryAnew(test *testing.T) {
	fake := fakehid.New(screen("a", "A"))
	transport, _ := winhid.OpenDiscovered(fake)
	fake.SetCollections(screen("b", "B"))
	if err := transport.Reopen(); err != nil {
		test.Fatal(err)
	}
	if err := transport.Write(smartmonitor.Report{}); err != nil {
		test.Error(err)
	}
	if !errors.Is(transport.Close(), nil) || transport.Close() != nil || !fake.AllClosed() {
		test.Error("close")
	}
}

func TestReopenOpensTheSamePathAndAFailedReopenLeavesNoHandle(test *testing.T) {
	fake := fakehid.New(screen("a", "A"))
	transport, _ := winhid.OpenPath(fake, "a")
	fake.SetCollections()
	if err := transport.Reopen(); err == nil {
		test.Fatal("the reopen of a missing device succeeded")
	}
	if !fake.AllClosed() {
		test.Error("a handle was left open")
	}
	if err := transport.Write(smartmonitor.Report{}); err == nil {
		test.Error("a write without a handle succeeded")
	}
	if _, err := transport.Read(time.Second); err == nil || errors.Is(err, smartmonitor.ErrReadTimeout) {
		test.Errorf("got %v", err)
	}
	if err := transport.Reopen(); err == nil {
		test.Error("the second reopen succeeded")
	}
	fake.SetCollections(screen("a", "A"))
	if err := transport.Reopen(); err != nil {
		test.Fatal(err)
	}
	if err := transport.Write(smartmonitor.Report{}); err != nil {
		test.Error(err)
	}
}

func TestCloseTwiceDoesNothing(test *testing.T) {
	fake := fakehid.New(screen("a", ""))
	transport, _ := winhid.OpenPath(fake, "a")
	if err := transport.Close(); err != nil {
		test.Fatal(err)
	}
	if err := transport.Close(); err != nil {
		test.Fatal(err)
	}
}
