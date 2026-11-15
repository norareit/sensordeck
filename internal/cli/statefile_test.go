package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
	"github.com/norareit/sensordeck/internal/uploadstate"
)

func newState(test *testing.T) (stateFile, *bytes.Buffer) {
	test.Helper()
	var stderr bytes.Buffer
	return stateFile{path: uploadstate.Path(test.TempDir()), stderr: &stderr}, &stderr
}

func writeEntries(test *testing.T, state stateFile, entries map[string]string) {
	test.Helper()
	if err := uploadstate.Write(state.path, entries); err != nil {
		test.Fatal(err)
	}
}

func readEntries(test *testing.T, state stateFile) map[string]string {
	test.Helper()
	entries, err := uploadstate.Read(state.path)
	if err != nil {
		test.Fatal(err)
	}
	return entries
}

func TestStateFileHolds(test *testing.T) {
	state, stderr := newState(test)
	if state.holds("S1", "h1") {
		test.Error("a state file that does not exist holds a hash")
	}
	writeEntries(test, state, map[string]string{"S1": "h1", "": "h0"})
	cases := []struct {
		serial, hash string
		want         bool
	}{{"S1", "h1", true}, {"S1", "h2", false}, {"S2", "h1", false}, {"", "h0", true}, {"", "h1", false}}
	for _, each := range cases {
		if got := state.holds(each.serial, each.hash); got != each.want {
			test.Errorf("holds(%q, %q) = %v", each.serial, each.hash, got)
		}
	}
	if stderr.Len() != 0 {
		test.Errorf("stderr %q", stderr.String())
	}
}

func TestStateFileThatIsNotJSONHoldsNothingWithOneLine(test *testing.T) {
	state, stderr := newState(test)
	if err := os.MkdirAll(filepath.Dir(state.path), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(state.path, []byte("garbage"), 0o644); err != nil {
		test.Fatal(err)
	}
	if state.holds("S1", "h1") {
		test.Error("a damaged state file holds a hash")
	}
	want := "state file " + state.path + ": not a JSON object of serials and hashes: "
	if got := stderr.String(); !strings.HasPrefix(got, want) || !strings.HasSuffix(got, ", taken as holding no entry\n") || strings.Count(got, "\n") != 1 {
		test.Errorf("stderr %q", got)
	}
}

func TestStateFileUploadRemovesTheEntryBeforeTheFirstReportAndRecordsTheHash(test *testing.T) {
	state, stderr := newState(test)
	writeEntries(test, state, map[string]string{"S1": "old", "S2": "other"})
	fake := &fakeTransport{clock: &fakeClock{}, ackWhenEmpty: true}
	var atFirstReport map[string]string
	fake.afterWrite = func(count int) {
		if count == 1 {
			atFirstReport = readEntries(test, state)
		}
	}
	theme := []byte("a theme")
	if err := state.upload(fake, "S1", theme, func(time.Duration) {}, func(smartmonitor.Stage, int, int) {}); err != nil {
		test.Fatal(err)
	}
	if want := map[string]string{"S2": "other"}; !reflect.DeepEqual(atFirstReport, want) {
		test.Errorf("at the first report the file held %v, want %v", atFirstReport, want)
	}
	if want := map[string]string{"S1": uploadstate.Hash(theme), "S2": "other"}; !reflect.DeepEqual(readEntries(test, state), want) {
		test.Errorf("after the upload the file holds %v", readEntries(test, state))
	}
	if stderr.Len() != 0 {
		test.Errorf("stderr %q", stderr.String())
	}
}

func TestStateFileUploadRecordsTheEmptySerial(test *testing.T) {
	state, _ := newState(test)
	fake := &fakeTransport{clock: &fakeClock{}, ackWhenEmpty: true}
	if err := state.upload(fake, "", []byte("t"), func(time.Duration) {}, func(smartmonitor.Stage, int, int) {}); err != nil {
		test.Fatal(err)
	}
	if got := readEntries(test, state); got[""] != uploadstate.Hash([]byte("t")) {
		test.Errorf("the file holds %v", got)
	}
}

func TestStateFileUploadThatFailedRecordsNothing(test *testing.T) {
	state, _ := newState(test)
	writeEntries(test, state, map[string]string{"S1": "old"})
	fake := &fakeTransport{clock: &fakeClock{}, queue: []readResult{{report: reportOf(0x43)}}}
	err := state.upload(fake, "S1", []byte("t"), func(time.Duration) {}, func(smartmonitor.Stage, int, int) {})
	if err == nil {
		test.Fatal("the upload gave no error")
	}
	if got := readEntries(test, state); len(got) != 0 {
		test.Errorf("the file holds %v, want the entry removed and nothing recorded", got)
	}
}

func TestStateFileUploadWritesNoReportWhenTheRemovalCannotBeWritten(test *testing.T) {
	directory := test.TempDir()
	blocker := filepath.Join(directory, "cache")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	var stderr bytes.Buffer
	state := stateFile{path: uploadstate.Path(blocker), stderr: &stderr}
	fake := &fakeTransport{clock: &fakeClock{}, ackWhenEmpty: true}
	err := state.upload(fake, "S1", []byte("t"), func(time.Duration) {}, func(smartmonitor.Stage, int, int) {})
	if err == nil || !strings.HasPrefix(err.Error(), "cannot write the state file "+state.path+": ") {
		test.Errorf("error %v", err)
	}
	if len(fake.writes) != 0 {
		test.Errorf("%d reports were written", len(fake.writes))
	}
}

func TestStateFileUploadThatCannotRecordWritesALineAndIsNoError(test *testing.T) {
	state, stderr := newState(test)
	fake := &fakeTransport{clock: &fakeClock{}, ackWhenEmpty: true}
	fake.afterWrite = func(count int) {
		if count == 1 {
			// A directory in place of the file makes the rename of the record fail.
			_ = os.Remove(state.path)
			_ = os.Mkdir(state.path, 0o755)
		}
	}
	if err := state.upload(fake, "S1", []byte("t"), func(time.Duration) {}, func(smartmonitor.Stage, int, int) {}); err != nil {
		test.Fatalf("error %v", err)
	}
	if got := stderr.String(); !strings.Contains(got, "cannot record the theme in the state file "+state.path+": ") {
		test.Errorf("stderr %q", got)
	}
}
