package winhid_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/fakehid"
	"github.com/norareit/sensordeck/internal/winhid"
)

func TestListOrdersByPathWithoutRegardToCase(test *testing.T) {
	fake := fakehid.New(screen(`\\?\hid#B`, ""), screen(`\\?\hid#a`, ""), screen(`\\?\hid#C`, ""))
	collections, err := winhid.List(fake)
	if err != nil {
		test.Fatal(err)
	}
	var paths []string
	for _, collection := range collections {
		paths = append(paths, collection.Path)
	}
	if want := []string{`\\?\hid#a`, `\\?\hid#B`, `\\?\hid#C`}; !reflect.DeepEqual(paths, want) {
		test.Errorf("got %v, want %v", paths, want)
	}
}

func TestListLeavesOutWhatCannotBeRead(test *testing.T) {
	broken := errors.New("broken")
	failedOpen := screen("open", "")
	failedOpen.QueryError = broken
	failedAttributes := screen("attributes", "")
	failedAttributes.AttributesError = broken
	failedCaps := screen("caps", "")
	failedCaps.CapsError = broken
	fake := fakehid.New(failedOpen, failedAttributes, failedCaps, screen("good", ""))
	collections, err := winhid.List(fake)
	if err != nil || len(collections) != 1 || collections[0].Path != "good" {
		test.Fatalf("got %v, %v", collections, err)
	}
	if !fake.AllClosed() {
		test.Error("a handle was left open")
	}
}

func TestListKeepsACollectionWhoseStringsFail(test *testing.T) {
	strings := screen("p", "serial")
	strings.ProductError, strings.SerialError = errors.New("no"), errors.New("no")
	fake := fakehid.New(strings)
	collections, err := winhid.List(fake)
	if err != nil || len(collections) != 1 || collections[0].Name != "" || collections[0].Serial != "" {
		test.Fatalf("got %v, %v", collections, err)
	}
}

func TestListOpensEveryPathWithNoAccessOnly(test *testing.T) {
	fake := fakehid.New(screen("a", ""), screen("b", ""))
	if _, err := winhid.List(fake); err != nil {
		test.Fatal(err)
	}
	for _, opening := range fake.Openings() {
		if opening.ReadWrite {
			test.Errorf("%s was opened for reading and writing", opening.Path)
		}
	}
	if len(fake.Openings()) != 2 || !fake.AllClosed() {
		test.Errorf("openings %v, closed %v", fake.Openings(), fake.AllClosed())
	}
}

func TestScreensKeepsTheScreenLengthsAndIgnoresOtherDevices(test *testing.T) {
	short := screen("short", "")
	short.Caps = winhid.Caps{InputLength: 64, OutputLength: 65}
	other := fakehid.Collection{Path: "keyboard", Vendor: 0x046d, Product: 0x0065, Caps: winhid.Caps{InputLength: 8}}
	collections, _ := winhid.List(fakehid.New(short, other, screen("one", ""), screen("two", "")))
	screens, nearMisses := winhid.Screens(collections)
	if len(screens) != 2 || screens[0].Path != "one" || screens[1].Path != "two" {
		test.Errorf("screens %v", screens)
	}
	if nearMisses != nil {
		test.Errorf("a near miss beside a screen is no news: %v", nearMisses)
	}
}

func TestScreensGivesTheNearMissesWhenThereIsNoScreen(test *testing.T) {
	first := screen("a", "")
	first.Caps = winhid.Caps{InputLength: 64, OutputLength: 65}
	second := screen("b", "")
	second.Caps = winhid.Caps{InputLength: 65, OutputLength: 32}
	other := fakehid.Collection{Path: "c", Vendor: 1, Product: 2}
	collections, _ := winhid.List(fakehid.New(second, first, other))
	screens, nearMisses := winhid.Screens(collections)
	if len(screens) != 0 || len(nearMisses) != 2 {
		test.Fatalf("screens %v near misses %v", screens, nearMisses)
	}
	want := []string{
		"found a with reports of 64 and 65 bytes, those of the screen are 65",
		"found b with reports of 65 and 32 bytes, those of the screen are 65",
	}
	if got := winhid.NearMissLines(nearMisses); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v, want %v", got, want)
	}
}

func TestACollectionIsAScreenLineOfTheDevicesCommand(test *testing.T) {
	collections, _ := winhid.List(fakehid.New(screen("p", "")))
	if got, want := collections[0].Line(), "p\t0483:0065\tScreen\t-"; got != want {
		test.Errorf("got %q, want %q", got, want)
	}
}
