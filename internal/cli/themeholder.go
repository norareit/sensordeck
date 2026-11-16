package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// themeHolder uploads the theme of the config at the start and again when a reload gives a compiled theme of other
// bytes than the last upload, which it keeps in memory.
type themeHolder struct {
	theme  *configTheme
	last   []byte // the compiled file of the last upload that completed, nil when there is none
	pause  func(time.Duration)
	now    func() time.Time
	say    func(line string)
	stderr io.Writer
	due    bool          // the theme is to be uploaded
	next   time.Time     // no attempt before this moment
	wait   time.Duration // the wait after the next failure; firstRetry, doubled up to longestRetry
}

// newThemeHolder makes the holder of the theme of the config, which is due at once when there is one.
func newThemeHolder(theme *configTheme, pause func(time.Duration), now func() time.Time,
	say func(line string), stderr io.Writer) *themeHolder {
	return &themeHolder{theme: theme, pause: pause, now: now, say: say, stderr: stderr, due: theme != nil, wait: firstRetry}
}

// ensure tells whether values may be sent, and whether an upload completed in this call. It uploads the theme when it
// is due and the wait has passed. failure is the error of an upload that failed, after which the screen is lost.
func (holder *themeHolder) ensure(stop context.Context, opened transport) (ready, uploaded bool, failure error) {
	if holder.theme == nil || !holder.due {
		return true, false, nil
	}
	if stop.Err() != nil || holder.now().Before(holder.next) {
		return false, false, nil
	}
	if err := holder.upload(opened); err != nil {
		return false, false, err
	}
	return true, true, nil
}

// onScreen tells whether values may be sent now without an upload, which only a round makes.
func (holder *themeHolder) onScreen() bool {
	return holder.theme == nil || !holder.due
}

func (holder *themeHolder) upload(opened transport) error {
	// An upload cut short may leave the flash with neither theme, so only its completion makes the screen's theme known.
	holder.last = nil
	holder.say("uploading theme " + holder.theme.value)
	// The progress writes nothing: an upload of run is one line before and one after.
	err := smartmonitor.Upload(opened, holder.theme.file, holder.pause, func(smartmonitor.Stage, int, int) {})
	if err != nil {
		_, _ = fmt.Fprintf(holder.stderr, "theme upload failed: %v\n", err)
		holder.next = holder.now().Add(holder.wait)
		holder.wait = min(2*holder.wait, longestRetry)
		return err
	}
	holder.say(fmt.Sprintf("theme uploaded: %d bytes", len(holder.theme.file)))
	holder.last = holder.theme.file
	holder.due, holder.next, holder.wait = false, time.Time{}, firstRetry
	return nil
}

// setTheme replaces the theme of the config. A theme that compiles to other bytes than the last upload, or a first
// theme, makes an upload due. No theme clears what is due and forgets the last upload, since the screen may be given
// another theme by other means while the config has tags.
func (holder *themeHolder) setTheme(theme *configTheme) {
	holder.theme = theme
	if theme == nil {
		holder.last, holder.due = nil, false
		return
	}
	holder.due = !bytes.Equal(theme.file, holder.last)
}

// restartRecovered makes an upload that is due be made in the round that opened the screen, without the wait that the
// failure of the attempt before set: a restart that made a hung screen leave the bus is new evidence that the next
// attempt can succeed. The wait after a failure of that upload is the one that doubling had reached.
func (holder *themeHolder) restartRecovered() {
	if holder.due {
		holder.next = time.Time{}
	}
}
