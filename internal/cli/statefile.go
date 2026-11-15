package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
	"github.com/norareit/sensordeck/internal/uploadstate"
)

// stateFile puts theme files on a screen and keeps the state file true for it, for upload and for run.
type stateFile struct {
	path   string
	stderr io.Writer
}

// stateFile gives the state file of the user, or the error of a cache directory that cannot be found.
func (current *session) stateFile() (stateFile, error) {
	directory, err := current.env.userCacheDir()
	if err != nil {
		return stateFile{}, fmt.Errorf("cannot find the cache directory: %w", err)
	}
	return stateFile{path: uploadstate.Path(directory), stderr: current.env.stderr}, nil
}

// entries gives what the file holds. A file that cannot be read or decoded holds nothing, with a line that names it.
func (state stateFile) entries() map[string]string {
	entries, err := uploadstate.Read(state.path)
	if err != nil {
		_, _ = fmt.Fprintf(state.stderr, "state file %s: %v, taken as holding no entry\n", state.path, innerError(err))
	}
	return entries
}

// holds tells whether the state file holds the hash for the serial.
func (state stateFile) holds(serial, hash string) bool {
	held, found := state.entries()[serial]
	return found && held == hash
}

// upload removes the entry of the serial and writes the file, uploads the theme, and records its hash under the
// serial after an upload that completed. Its error is the failed removal, before any report was written, or the
// error of the upload. A failed record writes its line and is no error.
func (state stateFile) upload(opened smartmonitor.Transport, serial string, theme []byte, pause func(time.Duration),
	progress func(stage smartmonitor.Stage, block, blocks int)) error {
	// The entry goes first, so that an upload cut short never leaves the file naming a theme the screen may not hold.
	entries := state.entries()
	delete(entries, serial)
	if err := uploadstate.Write(state.path, entries); err != nil {
		return fmt.Errorf("cannot write the state file %s: %w", state.path, innerError(err))
	}
	if err := smartmonitor.Upload(opened, theme, pause, progress); err != nil {
		return err
	}
	// The upload took long enough for another process to have written the file, so it is read again.
	entries = state.entries()
	entries[serial] = uploadstate.Hash(theme)
	if err := uploadstate.Write(state.path, entries); err != nil {
		_, _ = fmt.Fprintf(state.stderr, "cannot record the theme in the state file %s: %v\n", state.path, innerError(err))
	}
	return nil
}
