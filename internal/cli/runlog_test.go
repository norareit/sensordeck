package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var logMoment = time.Date(2026, 11, 14, 15, 4, 5, 0, time.FixedZone("", 3600))

const logStamp = "2026-11-14T15:04:05+01:00 "

func fixedMoment() time.Time { return logMoment }

func TestLogPathIsInTheSensordeckDirectoryOfTheCacheDirectory(test *testing.T) {
	if got, want := logPath("/cache"), filepath.Join("/cache", "sensordeck", "run.log"); got != want {
		test.Errorf("log path %q, want %q", got, want)
	}
}

func openTestLog(test *testing.T) (*runLog, string) {
	test.Helper()
	path := logPath(test.TempDir())
	log, err := openRunLog(path, fixedMoment)
	if err != nil {
		test.Fatal(err)
	}
	return log, path
}

func TestEachLineGetsTheTimeInFrontAndTheDirectoryIsCreated(test *testing.T) {
	log, path := openTestLog(test)
	if _, err := io.WriteString(log, "one\ntwo\n"); err != nil {
		test.Fatal(err)
	}
	if err := log.Close(); err != nil {
		test.Fatal(err)
	}
	if got, want := readTestFile(test, path), logStamp+"one\n"+logStamp+"two\n"; got != want {
		test.Errorf("log %q, want %q", got, want)
	}
}

func TestALineGivenInPiecesIsWrittenWholeInOneWrite(test *testing.T) {
	log, _ := openTestLog(test)
	var writes []string
	log.open = func(path string, truncate bool) (io.WriteCloser, int64, error) {
		return &recordingFile{writes: &writes}, 0, nil
	}
	log.file = &recordingFile{writes: &writes}
	for _, piece := range []string{"par", "tial ", "line\nnext\nrest"} {
		if _, err := io.WriteString(log, piece); err != nil {
			test.Fatal(err)
		}
	}
	if want := []string{logStamp + "partial line\n", logStamp + "next\n"}; !equalStrings(writes, want) {
		test.Errorf("writes %q, want %q", writes, want)
	}
	if err := log.Close(); err != nil {
		test.Fatal(err)
	}
	if want := logStamp + "rest\n"; writes[len(writes)-1] != want {
		test.Errorf("the pending text was written as %q, want %q", writes[len(writes)-1], want)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

type recordingFile struct{ writes *[]string }

func (file *recordingFile) Write(data []byte) (int, error) {
	*file.writes = append(*file.writes, string(data))
	return len(data), nil
}
func (file *recordingFile) Close() error { return nil }

func TestALineThatWouldPassOneMiBRotatesAndReplacesTheOlderFile(test *testing.T) {
	log, path := openTestLog(test)
	writeTestFile(test, path+".1", "an older log\n")
	filler := strings.Repeat("x", 1<<20-len(logStamp)-1) + "\n"
	if _, err := io.WriteString(log, filler); err != nil {
		test.Fatal(err)
	}
	if size := len(readTestFile(test, path)); size != 1<<20 {
		test.Fatalf("the log holds %d bytes before the rotation", size)
	}
	if _, err := io.WriteString(log, "after\n"); err != nil {
		test.Fatal(err)
	}
	if err := log.Close(); err != nil {
		test.Fatal(err)
	}
	if got := readTestFile(test, path); got != logStamp+"after\n" {
		test.Errorf("log %q", got)
	}
	if got := readTestFile(test, path+".1"); got != logStamp+filler {
		test.Errorf("older log of %d bytes", len(got))
	}
}

func TestALogLargerThanOneMiBIsRotatedAtItsOpening(test *testing.T) {
	path := logPath(test.TempDir())
	writeTestFile(test, path, strings.Repeat("y", 1<<20+1))
	log, err := openRunLog(path, fixedMoment)
	if err != nil {
		test.Fatal(err)
	}
	if _, err := io.WriteString(log, "fresh\n"); err != nil {
		test.Fatal(err)
	}
	_ = log.Close()
	if got := readTestFile(test, path); got != logStamp+"fresh\n" {
		test.Errorf("log %q", got)
	}
	if got := len(readTestFile(test, path+".1")); got != 1<<20+1 {
		test.Errorf("older log of %d bytes", got)
	}
}

func TestALogOfExactlyOneMiBIsKeptAtItsOpening(test *testing.T) {
	path := logPath(test.TempDir())
	writeTestFile(test, path, strings.Repeat("y", 1<<20))
	log, err := openRunLog(path, fixedMoment)
	if err != nil {
		test.Fatal(err)
	}
	_ = log.Close()
	if _, err := os.Stat(path + ".1"); !errors.Is(err, os.ErrNotExist) {
		test.Errorf("rotated: %v", err)
	}
}

func TestALineLargerThanOneMiBInAnEmptyLogIsWrittenWithNoRotation(test *testing.T) {
	log, path := openTestLog(test)
	big := strings.Repeat("z", 2<<20) + "\n"
	if _, err := io.WriteString(log, big); err != nil {
		test.Fatal(err)
	}
	_ = log.Close()
	if got := readTestFile(test, path); got != logStamp+big {
		test.Errorf("log of %d bytes", len(got))
	}
	if _, err := os.Stat(path + ".1"); !errors.Is(err, os.ErrNotExist) {
		test.Errorf("rotated: %v", err)
	}
}

func TestAFailedRenamingEmptiesTheLog(test *testing.T) {
	log, path := openTestLog(test)
	// A directory that is not empty cannot be replaced by the renaming of a file.
	writeTestFile(test, filepath.Join(path+".1", "inside"), "x")
	if _, err := io.WriteString(log, strings.Repeat("x", 1<<20-len(logStamp)-1)+"\n"); err != nil {
		test.Fatal(err)
	}
	if _, err := io.WriteString(log, "after\n"); err != nil {
		test.Fatal(err)
	}
	_ = log.Close()
	if got := readTestFile(test, path); got != logStamp+"after\n" {
		test.Errorf("log of %d bytes: %.60q", len(got), got)
	}
}

func TestALineIsDroppedWhenTheLogCannotBeOpenedAgainAndTheNextOneIsWritten(test *testing.T) {
	log, path := openTestLog(test)
	failing := true
	real := log.open
	log.open = func(path string, truncate bool) (io.WriteCloser, int64, error) {
		if failing {
			return nil, 0, errors.New("disk full")
		}
		return real(path, truncate)
	}
	log.rename = func(string, string) error { return errors.New("no renaming") }
	if _, err := io.WriteString(log, strings.Repeat("x", 1<<20-len(logStamp)-1)+"\n"); err != nil {
		test.Fatal(err)
	}
	if _, err := io.WriteString(log, "dropped\n"); err != nil {
		test.Fatal(err)
	}
	failing = false
	if _, err := io.WriteString(log, "written\n"); err != nil {
		test.Fatal(err)
	}
	_ = log.Close()
	if got := readTestFile(test, path); got != logStamp+"written\n" {
		test.Errorf("log %.80q", got)
	}
}

func TestALogThatCannotBeOpenedGivesTheErrorOfTheOpening(test *testing.T) {
	directory := test.TempDir()
	writeTestFile(test, filepath.Join(directory, "sensordeck"), "a file\n")
	path := logPath(directory)
	_, err := openRunLog(path, fixedMoment)
	if err == nil || !strings.HasPrefix(err.Error(), "cannot open the log "+path+": ") || strings.Count(err.Error(), path) != 1 {
		test.Errorf("error %v", err)
	}
}

func TestTheLinesAreAppendedToALogThatExists(test *testing.T) {
	path := logPath(test.TempDir())
	writeTestFile(test, path, "old\n")
	log, err := openRunLog(path, fixedMoment)
	if err != nil {
		test.Fatal(err)
	}
	_, _ = io.WriteString(log, "new\n")
	_ = log.Close()
	if got := readTestFile(test, path); got != "old\n"+logStamp+"new\n" {
		test.Errorf("log %q", got)
	}
}

func TestWriteReportsTheWholeOfTheData(test *testing.T) {
	log, _ := openTestLog(test)
	defer log.Close()
	var buffer bytes.Buffer
	buffer.WriteString("a line\npartial")
	written, err := log.Write(buffer.Bytes())
	if written != buffer.Len() || err != nil {
		test.Errorf("written %d, %v", written, err)
	}
}
