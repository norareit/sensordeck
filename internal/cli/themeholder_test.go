package cli

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/uploadstate"
)

type holderFixture struct {
	holder *themeHolder
	state  stateFile
	fake   *fakeTransport
	clock  *fakeClock
	said   []string
	stderr *bytes.Buffer
}

func newHolderFixture(test *testing.T, theme *configTheme) *holderFixture {
	test.Helper()
	fixture := &holderFixture{stderr: &bytes.Buffer{}}
	fixture.clock = &fakeClock{current: time.Date(2026, 11, 14, 15, 4, 5, 0, time.UTC)}
	fixture.fake = &fakeTransport{clock: fixture.clock, ackWhenEmpty: true, serial: "S1"}
	fixture.state = stateFile{path: uploadstate.Path(test.TempDir()), stderr: fixture.stderr}
	fixture.holder = newThemeHolder(theme, fixture.state, func(time.Duration) {}, fixture.clock.now,
		func(line string) { fixture.said = append(fixture.said, line) }, fixture.stderr)
	return fixture
}

func aTheme(content string) *configTheme {
	return &configTheme{value: "mine", file: []byte(content), hash: uploadstate.Hash([]byte(content))}
}

func (fixture *holderFixture) ensure() (ready, uploaded, failed bool) {
	return fixture.holder.ensure(context.Background(), fixture.fake)
}

func TestHolderUploadsAtTheFirstCheckWithNoStateFile(test *testing.T) {
	theme := aTheme("theme one")
	fixture := newHolderFixture(test, theme)
	ready, uploaded, failed := fixture.ensure()
	if !ready || !uploaded || failed {
		test.Fatalf("ready %v uploaded %v failed %v", ready, uploaded, failed)
	}
	if want := []string{"uploading theme mine", "theme uploaded: 9 bytes"}; !reflect.DeepEqual(fixture.said, want) {
		test.Errorf("said %v", fixture.said)
	}
	if !fixture.state.holds("S1", theme.hash) {
		test.Error("the hash was not recorded")
	}
}

func TestHolderUploadsNothingWhenTheHashIsStoredForTheSerialAndSaysSoOnce(test *testing.T) {
	theme := aTheme("theme one")
	fixture := newHolderFixture(test, theme)
	writeEntries(test, fixture.state, map[string]string{"S1": theme.hash})
	for range 2 {
		if ready, uploaded, failed := fixture.ensure(); !ready || uploaded || failed {
			test.Fatalf("ready %v uploaded %v failed %v", ready, uploaded, failed)
		}
	}
	fixture.holder.screenBack()
	fixture.ensure()
	if want := []string{"theme mine is on the screen"}; !reflect.DeepEqual(fixture.said, want) || len(fixture.fake.writes) != 0 {
		test.Errorf("said %v, %d reports", fixture.said, len(fixture.fake.writes))
	}
}

func TestHolderUploadsWhenTheStoredHashDiffersOrIsOfAnotherSerialOrTheSerialIsEmpty(test *testing.T) {
	theme := aTheme("theme one")
	cases := []struct {
		name    string
		entries map[string]string
		serial  string
	}{
		{"another hash", map[string]string{"S1": "other"}, "S1"},
		{"another serial", map[string]string{"S2": theme.hash}, "S1"},
		{"empty serial", map[string]string{"S1": theme.hash}, ""},
	}
	for _, each := range cases {
		fixture := newHolderFixture(test, theme)
		fixture.fake.serial = each.serial
		writeEntries(test, fixture.state, each.entries)
		if _, uploaded, _ := fixture.ensure(); !uploaded {
			test.Errorf("%s: no upload", each.name)
		}
	}
	fixture := newHolderFixture(test, theme)
	fixture.fake.serial = ""
	fixture.ensure()
	fixture.holder.screenBack()
	if _, uploaded, _ := fixture.ensure(); uploaded {
		test.Error("a second upload for the empty serial")
	}
}

func TestHolderWithNoThemeIsAlwaysReady(test *testing.T) {
	fixture := newHolderFixture(test, nil)
	if ready, uploaded, failed := fixture.ensure(); !ready || uploaded || failed || len(fixture.fake.writes) != 0 {
		test.Errorf("ready %v uploaded %v failed %v", ready, uploaded, failed)
	}
}

func TestHolderRetriesAFailedUploadAfterTenSecondsDoublingToFiveMinutes(test *testing.T) {
	fixture := newHolderFixture(test, aTheme("theme one"))
	fixture.fake.ackWhenEmpty = false
	wait := 10 * time.Second
	for attempt := range 6 {
		// every read is a time out, which is no answer to the ymodem command
		fixture.fake.queue = []readResult{{report: reportOf(0x43)}}
		ready, uploaded, failed := fixture.ensure()
		if ready || uploaded || !failed {
			test.Fatalf("attempt %d: ready %v uploaded %v failed %v", attempt, ready, uploaded, failed)
		}
		writes := len(fixture.fake.writes)
		fixture.clock.current = fixture.clock.current.Add(wait - time.Millisecond)
		if ready, _, failed := fixture.ensure(); ready || failed || len(fixture.fake.writes) != writes {
			test.Fatalf("attempt %d: tried again before %v", attempt, wait)
		}
		fixture.clock.current = fixture.clock.current.Add(time.Millisecond)
		wait = min(2*wait, 5*time.Minute)
	}
	if want := 5 * time.Minute; wait != want {
		test.Errorf("the last wait was %v", wait)
	}
	if got := fixture.stderr.String(); !bytes.Contains([]byte(got), []byte("theme upload failed: upload failed at the ymodem command: unexpected answer 0x43\n")) {
		test.Errorf("stderr %q", got)
	}
	if fixture.state.holds("S1", uploadstate.Hash([]byte("theme one"))) {
		test.Error("a failed upload recorded its hash")
	}
	// an upload that completes sets the wait back to ten seconds
	fixture.fake.ackWhenEmpty = true
	fixture.fake.queue = nil
	if _, uploaded, _ := fixture.ensure(); !uploaded || fixture.holder.wait != 10*time.Second {
		test.Errorf("uploaded %v, wait %v", uploaded, fixture.holder.wait)
	}
}

func TestHolderChecksAgainAfterTheScreenCameBackAndAfterAThemeOfAnotherHash(test *testing.T) {
	theme := aTheme("theme one")
	fixture := newHolderFixture(test, theme)
	fixture.ensure()
	fixture.fake.serial = "S2"
	fixture.holder.screenBack()
	if _, uploaded, _ := fixture.ensure(); !uploaded {
		test.Error("no upload for another screen that came back")
	}
	fixture.holder.setTheme(&configTheme{value: "again", file: theme.file, hash: theme.hash})
	if _, uploaded, _ := fixture.ensure(); uploaded || fixture.holder.checkDue {
		test.Error("a theme with the same hash made a check due")
	}
	fixture.holder.setTheme(aTheme("theme two"))
	if _, uploaded, _ := fixture.ensure(); !uploaded {
		test.Error("no upload for a theme with another hash")
	}
}

func TestHolderClearedByNoThemeIsReadyAndUploadsNothing(test *testing.T) {
	fixture := newHolderFixture(test, aTheme("theme one"))
	fixture.fake.ackWhenEmpty = false
	fixture.fake.queue = []readResult{{report: reportOf(0x43)}}
	fixture.ensure()
	fixture.holder.setTheme(nil)
	writes := len(fixture.fake.writes)
	if ready, uploaded, failed := fixture.ensure(); !ready || uploaded || failed || len(fixture.fake.writes) != writes {
		test.Errorf("ready %v uploaded %v failed %v", ready, uploaded, failed)
	}
}

func TestHolderStartsNoUploadOnceStopIsDone(test *testing.T) {
	fixture := newHolderFixture(test, aTheme("theme one"))
	stop, cancel := context.WithCancel(context.Background())
	cancel()
	if ready, uploaded, failed := fixture.holder.ensure(stop, fixture.fake); ready || uploaded || failed || len(fixture.fake.writes) != 0 {
		test.Errorf("ready %v uploaded %v failed %v, %d reports", ready, uploaded, failed, len(fixture.fake.writes))
	}
}

func TestHolderOnScreenTellsWhetherValuesMayBeSentWithoutACheckOrAnUpload(test *testing.T) {
	if !newHolderFixture(test, nil).holder.onScreen() {
		test.Error("a holder with no theme is not on screen")
	}
	theme := aTheme("theme one")
	fixture := newHolderFixture(test, theme)
	if fixture.holder.onScreen() {
		test.Error("on screen before the first check")
	}
	writeEntries(test, fixture.state, map[string]string{"S1": theme.hash})
	fixture.ensure()
	if !fixture.holder.onScreen() {
		test.Error("not on screen after a check that found the hash")
	}
	fixture.holder.screenBack()
	if fixture.holder.onScreen() {
		test.Error("on screen after the screen came back")
	}
	fixture.ensure()
	fixture.holder.setTheme(aTheme("theme two"))
	if fixture.holder.onScreen() {
		test.Error("on screen after a theme of another hash")
	}
	fixture.ensure()
	if !fixture.holder.onScreen() {
		test.Error("not on screen after an upload that completed")
	}
	failing := newHolderFixture(test, aTheme("theme three"))
	failing.fake.ackWhenEmpty = false
	failing.fake.queue = []readResult{{report: reportOf(0x43)}}
	failing.ensure()
	if failing.holder.onScreen() {
		test.Error("on screen after an upload that failed")
	}
}
