package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// logLimit is the size that run.log is not to pass, after which it becomes run.log.1.
	logLimit    = 1 << 20
	logTimeForm = "2006-01-02T15:04:05-07:00"
)

// runLog is the log of a background run: run.log in a directory, with run.log.1 beside it.
type runLog struct {
	mutex   sync.Mutex
	path    string
	file    io.WriteCloser // nil after a failed reopening, so that the next line tries again
	size    int64
	pending []byte // what was written after the last line feed
	now     func() time.Time
	open    func(path string, truncate bool) (io.WriteCloser, int64, error) // os.OpenFile with append, or with truncate
	rename  func(from, to string) error                                     // os.Rename
}

// logPath gives sensordeck/run.log in the cache directory.
func logPath(cacheDirectory string) string {
	return filepath.Join(cacheDirectory, "sensordeck", "run.log")
}

// openRunLogFile opens the file for appending, or emptied, and gives its size.
func openRunLogFile(path string, truncate bool) (io.WriteCloser, int64, error) {
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if truncate {
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, err
	}
	return file, info.Size(), nil
}

// openRunLog creates the directory, opens the log for appending, and rotates it first when it is larger than 1 MiB.
func openRunLog(path string, now func() time.Time) (*runLog, error) {
	log := &runLog{path: path, now: now, open: openRunLogFile, rename: os.Rename}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("cannot open the log %s: %w", path, innerError(err))
	}
	file, size, err := log.open(path, false)
	if err != nil {
		return nil, fmt.Errorf("cannot open the log %s: %w", path, innerError(err))
	}
	log.file, log.size = file, size
	if size > logLimit {
		log.rotate()
		if log.file == nil {
			return nil, fmt.Errorf("cannot open the log %s: it could not be rotated or emptied", path)
		}
	}
	return log, nil
}

// Write keeps what follows the last line feed for later, and writes each whole line with its time in front in one
// write. It reports the whole of data as written, because run has nowhere else to put a line.
func (log *runLog) Write(data []byte) (int, error) {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	log.pending = append(log.pending, data...)
	for {
		end := bytes.IndexByte(log.pending, '\n')
		if end < 0 {
			return len(data), nil
		}
		log.writeLine(log.pending[:end+1])
		log.pending = log.pending[end+1:]
	}
}

// Close writes what is pending with a line feed and closes the file.
func (log *runLog) Close() error {
	log.mutex.Lock()
	defer log.mutex.Unlock()
	if len(log.pending) > 0 {
		log.writeLine(append(log.pending, '\n'))
		log.pending = nil
	}
	if log.file == nil {
		return nil
	}
	err := log.file.Close()
	log.file = nil
	return err
}

// writeLine writes the line, which ends in a line feed, rotating first when it would make run.log pass the limit. A
// line larger than the limit goes to an empty run.log, so that the rotation cannot repeat itself.
func (log *runLog) writeLine(line []byte) {
	entry := append([]byte(log.now().Format(logTimeForm)+" "), line...)
	if log.file == nil || (log.size > 0 && log.size+int64(len(entry)) > logLimit) {
		log.rotate()
	}
	if log.file == nil {
		return
	}
	written, err := log.file.Write(entry)
	log.size += int64(written)
	if err != nil {
		_ = log.file.Close()
		log.file = nil
	}
}

// rotate closes the file, renames run.log to run.log.1 and opens a new run.log. When the renaming fails it opens
// run.log emptied, and when that fails too the file stays nil, so that the next line tries again.
func (log *runLog) rotate() {
	if log.file != nil {
		_ = log.file.Close()
		log.file = nil
	}
	truncate := log.rename(log.path, log.path+".1") != nil
	file, size, err := log.open(log.path, truncate)
	if err != nil {
		return
	}
	log.file, log.size = file, size
}
