package cli

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type holderFixture struct {
	holder *themeHolder
	fake   *fakeTransport
	clock  *fakeClock
	said   []string
	stderr *bytes.Buffer
}

func newHolderFixture(test *testing.T, theme *configTheme) *holderFixture {
	test.Helper()
	fixture := &holderFixture{stderr: &bytes.Buffer{}}
	fixture.clock = &fakeClock{current: time.Date(2026, 11, 14, 15, 4, 5, 0, time.UTC)}
	fixture.fake = &fakeTransport{clock: fixture.clock, ackWhenEmpty: true}
	fixture.holder = newThemeHolder(theme, func(time.Duration) {}, fixture.clock.now,
		func(line string) { fixture.said = append(fixture.said, line) }, fixture.stderr)
	return fixture
}

func aTheme(content string) *configTheme {
	return &configTheme{value: "mine", file: []byte(content)}
}

func (fixture *holderFixture) ensure() (ready, uploaded bool, failure error) {
	return fixture.holder.ensure(context.Background(), fixture.fake)
}

func TestHolderUploadsAtTheFirstEnsureAndSaysTwoLines(test *testing.T) {
	fixture := newHolderFixture(test, aTheme("theme one"))
	ready, uploaded, failure := fixture.ensure()
	if !ready || !uploaded || failure != nil {
		test.Fatalf("ready %v uploaded %v failure %v", ready, uploaded, failure)
	}
	if want := []string{"uploading theme mine", "theme uploaded: 9 bytes"}; !reflect.DeepEqual(fixture.said, want) {
		test.Errorf("said %v", fixture.said)
	}
	if ready, uploaded, _ := fixture.ensure(); !ready || uploaded {
		test.Error("a second upload with nothing due")
	}
}

func TestHolderOfAnotherStartUploadsAgainAndNeverSaysTheThemeIsOnTheScreen(test *testing.T) {
	theme := aTheme("theme one")
	for range 2 {
		fixture := newHolderFixture(test, theme)
		if _, uploaded, _ := fixture.ensure(); !uploaded {
			test.Fatal("no upload at a start")
		}
		for _, line := range fixture.said {
			if strings.Contains(line, "is on the screen") {
				test.Errorf("said %q", line)
			}
		}
	}
}

func TestHolderUploadsAReloadedThemeOfOtherBytesAndNotOneOfTheSameBytes(test *testing.T) {
	theme := aTheme("theme one")
	fixture := newHolderFixture(test, theme)
	fixture.ensure()
	fixture.holder.setTheme(&configTheme{value: "again", file: theme.file})
	if _, uploaded, _ := fixture.ensure(); uploaded || !fixture.holder.onScreen() {
		test.Error("a theme with the same bytes made an upload due")
	}
	fixture.holder.setTheme(aTheme("theme two"))
	if fixture.holder.onScreen() {
		test.Error("on screen after a theme of other bytes")
	}
	if _, uploaded, _ := fixture.ensure(); !uploaded {
		test.Error("no upload for a theme of other bytes")
	}
}

func TestHolderUploadsATagsRunsFirstThemeAndTheSameThemeAfterNoTheme(test *testing.T) {
	fixture := newHolderFixture(test, nil)
	fixture.holder.setTheme(aTheme("theme one"))
	if _, uploaded, _ := fixture.ensure(); !uploaded {
		test.Error("no upload for the first theme")
	}
	fixture.holder.setTheme(nil)
	fixture.holder.setTheme(aTheme("theme one"))
	if _, uploaded, _ := fixture.ensure(); !uploaded {
		test.Error("no upload for the same theme after no theme")
	}
}

func TestHolderUploadsTheThemeOfTheLastUploadAgainAfterAFailedUploadOfAnother(test *testing.T) {
	fixture := newHolderFixture(test, aTheme("theme one"))
	fixture.ensure()
	fixture.holder.setTheme(aTheme("theme two"))
	fixture.fake.failWrites, fixture.fake.failErr = []bool{true}, errFake
	if _, uploaded, failure := fixture.ensure(); uploaded || failure == nil {
		test.Fatal("the upload of theme two did not fail")
	}
	fixture.holder.setTheme(aTheme("theme one"))
	fixture.clock.current = fixture.clock.current.Add(firstRetry)
	if _, uploaded, _ := fixture.ensure(); !uploaded {
		test.Error("no upload of theme one, which the failed upload may have broken")
	}
}

func TestHolderWithNoThemeIsAlwaysReady(test *testing.T) {
	fixture := newHolderFixture(test, nil)
	if ready, uploaded, failure := fixture.ensure(); !ready || uploaded || failure != nil || len(fixture.fake.writes) != 0 {
		test.Errorf("ready %v uploaded %v failure %v", ready, uploaded, failure)
	}
}

func TestHolderRetriesAFailedUploadAfterTenSecondsDoublingToFiveMinutes(test *testing.T) {
	fixture := newHolderFixture(test, aTheme("theme one"))
	fixture.fake.ackWhenEmpty = false
	wait := 10 * time.Second
	for attempt := range 6 {
		// every read is a time out, which is no answer to the ymodem command
		fixture.fake.queue = []readResult{{report: reportOf(0x43)}}
		ready, uploaded, failure := fixture.ensure()
		if ready || uploaded || failure == nil {
			test.Fatalf("attempt %d: ready %v uploaded %v failed %v", attempt, ready, uploaded, failure)
		}
		writes := len(fixture.fake.writes)
		fixture.clock.current = fixture.clock.current.Add(wait - time.Millisecond)
		if ready, _, failure := fixture.ensure(); ready || failure != nil || len(fixture.fake.writes) != writes {
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
	// an upload that completes sets the wait back to ten seconds
	fixture.fake.ackWhenEmpty = true
	fixture.fake.queue = nil
	if _, uploaded, _ := fixture.ensure(); !uploaded || fixture.holder.wait != 10*time.Second {
		test.Errorf("uploaded %v, wait %v", uploaded, fixture.holder.wait)
	}
}

func TestHolderClearedByNoThemeIsReadyAndUploadsNothing(test *testing.T) {
	fixture := newHolderFixture(test, aTheme("theme one"))
	fixture.fake.ackWhenEmpty = false
	fixture.fake.queue = []readResult{{report: reportOf(0x43)}}
	fixture.ensure()
	fixture.holder.setTheme(nil)
	writes := len(fixture.fake.writes)
	if ready, uploaded, failure := fixture.ensure(); !ready || uploaded || failure != nil || len(fixture.fake.writes) != writes {
		test.Errorf("ready %v uploaded %v failure %v", ready, uploaded, failure)
	}
}

func TestHolderStartsNoUploadOnceStopIsDone(test *testing.T) {
	fixture := newHolderFixture(test, aTheme("theme one"))
	stop, cancel := context.WithCancel(context.Background())
	cancel()
	if ready, uploaded, failure := fixture.holder.ensure(stop, fixture.fake); ready || uploaded || failure != nil || len(fixture.fake.writes) != 0 {
		test.Errorf("ready %v uploaded %v failure %v, %d reports", ready, uploaded, failure, len(fixture.fake.writes))
	}
}

func TestHolderOnScreenTellsWhetherValuesMayBeSentWithoutAnUpload(test *testing.T) {
	if !newHolderFixture(test, nil).holder.onScreen() {
		test.Error("a holder with no theme is not on screen")
	}
	fixture := newHolderFixture(test, aTheme("theme one"))
	if fixture.holder.onScreen() {
		test.Error("on screen before the first upload")
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

func TestHolderGivesTheErrorOfAFailedUploadWhichWrapsTheErrorOfTheWrite(test *testing.T) {
	fixture := newHolderFixture(test, aTheme("theme one"))
	fixture.fake.failWrites, fixture.fake.failErr = []bool{true}, errFake
	ready, uploaded, failure := fixture.ensure()
	if ready || uploaded || !errors.Is(failure, errFake) {
		test.Errorf("ready %v uploaded %v failure %v", ready, uploaded, failure)
	}
}
