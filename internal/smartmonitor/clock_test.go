package smartmonitor

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEncodeClockExample(t *testing.T) {
	report, err := EncodeClock(time.Date(2026, 11, 14, 15, 4, 5, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var want Report
	copy(want[:], []byte{0x03, 0x01, 0x15, 0x1A, 0x0B, 0x0E, 0x0F, 0x04, 0x05, 0x06, 0x64})
	if report != want {
		t.Fatalf("got % X", report[:])
	}
}

func TestEncodeClockUsesFieldsOfItsOwnLocation(t *testing.T) {
	zone := time.FixedZone("test", 9*3600)
	report, err := EncodeClock(time.Date(2026, 11, 14, 23, 59, 58, 0, zone))
	if err != nil {
		t.Fatal(err)
	}
	if report[5] != 14 || report[6] != 23 || report[7] != 59 || report[8] != 58 {
		t.Fatalf("got % X", report[:11])
	}
}

func TestEncodeClockWeekdays(t *testing.T) {
	sunday, _ := EncodeClock(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	monday, _ := EncodeClock(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC))
	if sunday[9] != 7 || monday[9] != 1 {
		t.Fatalf("sunday %d, monday %d", sunday[9], monday[9])
	}
}

func TestEncodeClockYearRange(t *testing.T) {
	for _, year := range []int{1999, 2256} {
		report, err := EncodeClock(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))
		if err == nil || !strings.Contains(err.Error(), strconv.Itoa(year)) {
			t.Errorf("year %d: error %v", year, err)
		}
		if report != (Report{}) {
			t.Errorf("year %d: report is not zero", year)
		}
	}
	for _, year := range []int{2000, 2255} {
		if _, err := EncodeClock(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
			t.Errorf("year %d: %v", year, err)
		}
	}
}
