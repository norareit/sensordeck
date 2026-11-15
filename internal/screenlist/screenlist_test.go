package screenlist

import "testing"

func TestLineIsTheFormatOfTheDevicesCommand(t *testing.T) {
	got := Line("/dev/hidraw3", 0x0483, 0x0065, "Smart Monitor", "ABC123")
	want := "/dev/hidraw3\t0483:0065\tSmart Monitor\tABC123"
	if got != want {
		t.Errorf("Line = %q, want %q", got, want)
	}
}

func TestLineWritesADashForAnEmptyNameAndAnEmptySerial(t *testing.T) {
	got := Line("p", 0x0a0b, 0x00cd, "", "")
	want := "p\t0a0b:00cd\t-\t-"
	if got != want {
		t.Errorf("Line = %q, want %q", got, want)
	}
}

func TestNoneFoundIsTheTextOfTheMissingScreen(t *testing.T) {
	if NoneFound != "no supported screen found" {
		t.Errorf("NoneFound = %q", NoneFound)
	}
}

func TestSeveralListsTheLines(t *testing.T) {
	got := Several([]string{"one", "two"}).Error()
	want := "more than one supported screen is connected, name one with --device:\none\ntwo"
	if got != want {
		t.Errorf("Several = %q, want %q", got, want)
	}
}
