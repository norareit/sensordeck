package smartmonitor

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

type callKind string

const (
	callWrite  callKind = "write"
	callRead   callKind = "read"
	callReopen callKind = "reopen"
	// callProgress is a call of the progress function of Upload, recorded in the same list as the transport calls.
	callProgress callKind = "progress"
)

type call struct {
	kind    callKind
	report  Report
	timeout time.Duration
	stage   Stage
	block   int
	blocks  int
}

// fakeTransport records every call. A read answers ACK unless readResults has an entry for its position, and a
// reopen succeeds unless reopenResults has one. A write fails when writeErrors has an entry for its position.
type fakeTransport struct {
	calls         []call
	readResults   map[int]readResult
	reopenResults []error
	writeErrors   map[int]error
	reads         int
	writes        int
	reopens       int
}

type readResult struct {
	first byte
	err   error
}

func (fake *fakeTransport) Write(report Report) error {
	fake.calls = append(fake.calls, call{kind: callWrite, report: report})
	position := fake.writes
	fake.writes++
	return fake.writeErrors[position]
}

func (fake *fakeTransport) Read(timeout time.Duration) (Report, error) {
	fake.calls = append(fake.calls, call{kind: callRead, timeout: timeout})
	position := fake.reads
	fake.reads++
	result, scripted := fake.readResults[position]
	if !scripted {
		result = readResult{first: answerACK}
	}
	var report Report
	report[0] = result.first
	return report, result.err
}

func (fake *fakeTransport) Reopen() error {
	fake.calls = append(fake.calls, call{kind: callReopen})
	position := fake.reopens
	fake.reopens++
	if position < len(fake.reopenResults) {
		return fake.reopenResults[position]
	}
	return nil
}

// progressInto gives a progress function that appends to the calls of the fake, so that the order of progress calls
// among the transport calls is checked.
func (fake *fakeTransport) progressInto() func(Stage, int, int) {
	return func(stage Stage, block, blocks int) {
		fake.calls = append(fake.calls, call{kind: callProgress, stage: stage, block: block, blocks: blocks})
	}
}

type recordedPauses struct{ durations []time.Duration }

func (pauses *recordedPauses) pause(duration time.Duration) {
	pauses.durations = append(pauses.durations, duration)
}

func uploadTheme() []byte { return testTheme(2500) }

func writtenReports(calls []call) []Report {
	var reports []Report
	for _, recorded := range calls {
		if recorded.kind == callWrite {
			reports = append(reports, recorded.report)
		}
	}
	return reports
}

func expectUploadError(test *testing.T, err error, stage Stage, block int) *UploadError {
	test.Helper()
	var uploadError *UploadError
	if !errors.As(err, &uploadError) {
		test.Fatalf("error %v is not an UploadError", err)
	}
	if uploadError.Stage != stage || uploadError.Block != block {
		test.Fatalf("stage %v block %d, want %v block %d", uploadError.Stage, uploadError.Block, stage, block)
	}
	return uploadError
}

func TestUploadCompleteOrderOfCalls(test *testing.T) {
	theme := uploadTheme()
	original := append([]byte(nil), theme...)
	fake := &fakeTransport{}
	pauses := &recordedPauses{}

	if err := Upload(fake, theme, pauses.pause, fake.progressInto()); err != nil {
		test.Fatal(err)
	}

	var want []call
	write := func(reports []Report) {
		for _, report := range reports {
			want = append(want, call{kind: callWrite, report: report})
		}
	}
	read := func() { want = append(want, call{kind: callRead, timeout: 10 * time.Second}) }
	progress := func(stage Stage, block int) {
		want = append(want, call{kind: callProgress, stage: stage, block: block, blocks: 3})
	}
	progress(StageReset, 0)
	write([]Report{ResetCommand()})
	progress(StageReopenAfterReset, 0)
	want = append(want, call{kind: callReopen})
	progress(StageYmodemCommand, 0)
	write([]Report{YmodemCommand()})
	read()
	progress(StageHeader, 0)
	write(toReports(headerBlock(len(theme))))
	read()
	for position, block := range dataBlocks(theme) {
		progress(StageData, position+1)
		write(toReports(block))
		read()
	}
	progress(StageEndOfTransmission, 0)
	write(toReports(endOfTransmission()))
	read()
	progress(StageClosing, 0)
	write(toReports(closingBlock()))
	progress(StageReopenAfterUpload, 0)
	want = append(want, call{kind: callReopen})

	if !reflect.DeepEqual(fake.calls, want) {
		test.Fatalf("calls differ: got %d, want %d", len(fake.calls), len(want))
	}
	if !reflect.DeepEqual(pauses.durations, []time.Duration{time.Second, time.Second}) {
		test.Errorf("pauses %v", pauses.durations)
	}
	if !bytes.Equal(theme, original) {
		test.Error("the theme was changed")
	}
}

func TestUploadWritesABlockAgainOnNAK(test *testing.T) {
	// reads: ymodem 0, header 1, data block 1 is 2, data block 2 is 3 and its repeat is 4
	fake := &fakeTransport{readResults: map[int]readResult{3: {first: answerNAK}}}
	if err := Upload(fake, uploadTheme(), (&recordedPauses{}).pause, fake.progressInto()); err != nil {
		test.Fatal(err)
	}
	block := toReports(dataBlocks(uploadTheme())[1])
	reports := writtenReports(fake.calls)
	var occurrences int
	for start := 0; start+len(block) <= len(reports); start++ {
		if reflect.DeepEqual(reports[start:start+len(block)], block) {
			occurrences++
		}
	}
	if occurrences != 2 {
		test.Fatalf("block 2 was written %d times", occurrences)
	}
}

func TestUploadWritesAgainOnNAKForEveryKindOfItem(test *testing.T) {
	cases := map[string]int{"ymodem command": 0, "header block": 1, "end of transmission": 5}
	for name, readPosition := range cases {
		fake := &fakeTransport{readResults: map[int]readResult{readPosition: {first: answerNAK}}}
		if err := Upload(fake, uploadTheme(), (&recordedPauses{}).pause, fake.progressInto()); err != nil {
			test.Errorf("%s: %v", name, err)
		}
		if fake.reads != 7 {
			test.Errorf("%s: %d reads, want 7", name, fake.reads)
		}
	}
}

func TestUploadEndsAfterTenNAKs(test *testing.T) {
	results := map[int]readResult{}
	for position := 3; position < 13; position++ {
		results[position] = readResult{first: answerNAK}
	}
	fake := &fakeTransport{readResults: results}
	err := Upload(fake, uploadTheme(), (&recordedPauses{}).pause, fake.progressInto())
	expectUploadError(test, err, StageData, 2)
	if !strings.Contains(err.Error(), "data block 2") {
		test.Errorf("text %q", err)
	}
	if fake.reads != 13 {
		test.Errorf("%d reads", fake.reads)
	}
	if fake.calls[len(fake.calls)-1].kind != callRead {
		test.Error("something was written after the tenth answer")
	}
	// reset, ymodem command, header block, data block 1 and ten writes of data block 2
	if got, want := len(writtenReports(fake.calls)), 1+1+3+17+10*17; got != want {
		test.Errorf("%d reports written, want %d", got, want)
	}
}

func TestUploadEndsWhenAReadFails(test *testing.T) {
	timeLimit := errors.New("time limit passed")
	fake := &fakeTransport{readResults: map[int]readResult{1: {err: timeLimit}}}
	err := Upload(fake, uploadTheme(), (&recordedPauses{}).pause, fake.progressInto())
	expectUploadError(test, err, StageHeader, 0)
	if !errors.Is(err, timeLimit) {
		test.Error("the read error is not reachable")
	}
	if fake.calls[len(fake.calls)-1].kind != callRead {
		test.Error("something was done after the failed read")
	}
}

func TestUploadEndsOnAnUnexpectedAnswer(test *testing.T) {
	fake := &fakeTransport{readResults: map[int]readResult{2: {first: 0x43}}}
	err := Upload(fake, uploadTheme(), (&recordedPauses{}).pause, fake.progressInto())
	expectUploadError(test, err, StageData, 1)
	if !strings.Contains(err.Error(), "0x43") {
		test.Errorf("text %q", err)
	}
	if fake.calls[len(fake.calls)-1].kind != callRead {
		test.Error("something was done after the answer")
	}
}

func TestUploadEndsWhenAWriteFails(test *testing.T) {
	writeFailure := errors.New("write failed")
	closingFirstWrite := 1 + 1 + 3 + 3*17 + 1
	cases := map[string]struct {
		position int
		stage    Stage
	}{
		"reset":   {position: 0, stage: StageReset},
		"closing": {position: closingFirstWrite, stage: StageClosing},
	}
	for name, scenario := range cases {
		fake := &fakeTransport{writeErrors: map[int]error{scenario.position: writeFailure}}
		err := Upload(fake, uploadTheme(), (&recordedPauses{}).pause, fake.progressInto())
		expectUploadError(test, err, scenario.stage, 0)
		if !errors.Is(err, writeFailure) {
			test.Errorf("%s: the write error is not reachable", name)
		}
		if fake.writes != scenario.position+1 || fake.calls[len(fake.calls)-1].kind != callWrite {
			test.Errorf("%s: calls followed the failing write", name)
		}
	}
}

func TestUploadRetriesReopening(test *testing.T) {
	failure := errors.New("i/o error")
	fake := &fakeTransport{reopenResults: []error{failure, failure, failure, nil, failure, failure, failure, nil}}
	pauses := &recordedPauses{}
	if err := Upload(fake, uploadTheme(), pauses.pause, fake.progressInto()); err != nil {
		test.Fatal(err)
	}
	segment := []time.Duration{time.Second, 500 * time.Millisecond, 500 * time.Millisecond, 500 * time.Millisecond}
	want := append(append([]time.Duration(nil), segment...), segment...)
	if !reflect.DeepEqual(pauses.durations, want) {
		test.Errorf("pauses %v", pauses.durations)
	}
}

func TestUploadEndsWhenReopeningFails20Times(test *testing.T) {
	failure := errors.New("i/o error")
	failures := make([]error, 20)
	for position := range failures {
		failures[position] = failure
	}
	stages := map[string]struct {
		results []error
		stage   Stage
		writes  int
	}{
		"after the reset":  {results: failures, stage: StageReopenAfterReset, writes: 1},
		"after the upload": {results: append(make([]error, 1), failures...), stage: StageReopenAfterUpload},
	}
	for name, scenario := range stages {
		fake := &fakeTransport{reopenResults: scenario.results}
		pauses := &recordedPauses{}
		err := Upload(fake, uploadTheme(), pauses.pause, fake.progressInto())
		expectUploadError(test, err, scenario.stage, 0)
		if !errors.Is(err, failure) {
			test.Errorf("%s: the reopen error is not reachable", name)
		}
		if fake.calls[len(fake.calls)-1].kind != callReopen {
			test.Errorf("%s: calls followed the last reopen", name)
		}
		if scenario.writes != 0 && len(writtenReports(fake.calls)) != scenario.writes {
			test.Errorf("%s: %d reports written", name, len(writtenReports(fake.calls)))
		}
		last := pauses.durations[len(pauses.durations)-19:]
		for _, duration := range last {
			if duration != 500*time.Millisecond {
				test.Errorf("%s: pause %v", name, duration)
			}
		}
	}
}

func TestStageNamesAndErrorTexts(test *testing.T) {
	names := map[Stage]string{
		StageReset:             "reset",
		StageReopenAfterReset:  "reopening after the reset",
		StageYmodemCommand:     "ymodem command",
		StageHeader:            "header block",
		StageData:              "data block",
		StageEndOfTransmission: "end of transmission",
		StageClosing:           "closing block",
		StageReopenAfterUpload: "reopening after the upload",
	}
	for stage, name := range names {
		if stage.String() != name {
			test.Errorf("stage %d is %q", int(stage), stage.String())
		}
		if stage == StageData {
			continue
		}
		text := (&UploadError{Stage: stage, Err: errors.New("cause")}).Error()
		if text != "upload failed at the "+name+": cause" {
			test.Errorf("text %q", text)
		}
	}
}

// progressCalls gives the progress entries of the recorded calls, with the position of each in the list.
func progressCalls(calls []call) map[int]call {
	entries := map[int]call{}
	for position, recorded := range calls {
		if recorded.kind == callProgress {
			entries[position] = recorded
		}
	}
	return entries
}

func countProgress(calls []call, stage Stage, block int) (count int, firstPosition int) {
	firstPosition = -1
	for position, recorded := range calls {
		if recorded.kind == callProgress && recorded.stage == stage && recorded.block == block {
			count++
			if firstPosition < 0 {
				firstPosition = position
			}
		}
	}
	return count, firstPosition
}

func TestUploadReportsProgressOnceWhateverTheRetries(test *testing.T) {
	failure := errors.New("i/o error")
	cases := map[string]struct {
		fake  *fakeTransport
		stage Stage
		block int
	}{
		"NAK on data block 2":   {fake: &fakeTransport{readResults: map[int]readResult{3: {first: answerNAK}}}, stage: StageData, block: 2},
		"NAK on ymodem command": {fake: &fakeTransport{readResults: map[int]readResult{0: {first: answerNAK}}}, stage: StageYmodemCommand},
		"reopening tried again": {fake: &fakeTransport{reopenResults: []error{failure, failure, failure, nil}}, stage: StageReopenAfterReset},
	}
	for name, scenario := range cases {
		if err := Upload(scenario.fake, uploadTheme(), (&recordedPauses{}).pause, scenario.fake.progressInto()); err != nil {
			test.Fatalf("%s: %v", name, err)
		}
		count, first := countProgress(scenario.fake.calls, scenario.stage, scenario.block)
		if count != 1 {
			test.Errorf("%s: %d progress calls, want 1", name, count)
			continue
		}
		if next := scenario.fake.calls[first+1]; next.kind != callWrite && next.kind != callReopen {
			test.Errorf("%s: the progress call is followed by %v", name, next.kind)
		}
	}
}

func TestUploadFailingYmodemCommandMakesThreeProgressCalls(test *testing.T) {
	fake := &fakeTransport{readResults: map[int]readResult{0: {first: 0x43}}}
	if err := Upload(fake, uploadTheme(), (&recordedPauses{}).pause, fake.progressInto()); err == nil {
		test.Fatal("no error")
	}
	var stages []Stage
	for _, entry := range progressCalls(fake.calls) {
		stages = append(stages, entry.stage)
	}
	slices.Sort(stages)
	want := []Stage{StageReset, StageReopenAfterReset, StageYmodemCommand}
	if !slices.Equal(stages, want) {
		test.Errorf("stages %v, want %v", stages, want)
	}
}
