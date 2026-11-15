package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// themeHolder decides whether the theme of the config is on the screen, and uploads it when it is not.
type themeHolder struct {
	theme    *configTheme
	state    stateFile
	pause    func(time.Duration)
	now      func() time.Time
	say      func(line string)
	stderr   io.Writer
	checkDue bool          // the state file is to be read at the next call of ensure
	starting bool          // the next check is the one at the start, which writes its line
	due      bool          // the theme is not on the screen
	next     time.Time     // no attempt before this moment
	wait     time.Duration // the wait after the next failure; firstRetry, doubled up to longestRetry
}

// newThemeHolder makes the holder of the theme of the config, whose first check is the one at the start.
func newThemeHolder(theme *configTheme, state stateFile, pause func(time.Duration), now func() time.Time,
	say func(line string), stderr io.Writer) *themeHolder {
	return &themeHolder{
		theme: theme, state: state, pause: pause, now: now, say: say, stderr: stderr,
		checkDue: theme != nil, starting: true, wait: firstRetry,
	}
}

// ensure tells whether values may be sent, and whether an upload completed in this call. It reads the state file
// when a check is due and uploads the theme when it is not on the screen and the wait has passed. failed is an upload
// that returned an error, after which the screen is lost.
func (holder *themeHolder) ensure(stop context.Context, opened transport) (ready, uploaded, failed bool) {
	if holder.theme == nil {
		return true, false, false
	}
	if holder.checkDue {
		holder.check(opened.Serial())
	}
	if !holder.due {
		return true, false, false
	}
	if stop.Err() != nil || holder.now().Before(holder.next) {
		return false, false, false
	}
	if err := holder.upload(opened); err != nil {
		return false, false, true
	}
	return true, true, false
}

func (holder *themeHolder) check(serial string) {
	holder.checkDue = false
	holder.due = !holder.state.holds(serial, holder.theme.hash)
	if !holder.due && holder.starting {
		holder.say(fmt.Sprintf("theme %s is on the screen", holder.theme.value))
	}
	holder.starting = false
}

func (holder *themeHolder) upload(opened transport) error {
	holder.say("uploading theme " + holder.theme.value)
	// The progress writes nothing: an upload of run is one line before and one after.
	err := holder.state.upload(opened, opened.Serial(), holder.theme.file, holder.pause, func(smartmonitor.Stage, int, int) {})
	if err != nil {
		_, _ = fmt.Fprintf(holder.stderr, "theme upload failed: %v\n", err)
		holder.next = holder.now().Add(holder.wait)
		holder.wait = min(2*holder.wait, longestRetry)
		return err
	}
	holder.say(fmt.Sprintf("theme uploaded: %d bytes", len(holder.theme.file)))
	holder.due, holder.next, holder.wait = false, time.Time{}, firstRetry
	return nil
}

// setTheme replaces the theme of the config. A theme whose hash differs from the one before, or a first theme, makes a
// check due. No theme clears what is due.
func (holder *themeHolder) setTheme(theme *configTheme) {
	if theme == nil {
		holder.theme, holder.checkDue, holder.due = nil, false, false
		return
	}
	if holder.theme == nil || holder.theme.hash != theme.hash {
		holder.checkDue = true
	}
	holder.theme = theme
}

// screenBack makes a check due.
func (holder *themeHolder) screenBack() { holder.checkDue = holder.theme != nil }
