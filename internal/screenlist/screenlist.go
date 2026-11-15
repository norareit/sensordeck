// Package screenlist holds the texts that every system's discovery of the screen shares: the line of a screen, and
// the two errors of a discovery that does not find exactly one. It does no I/O.
package screenlist

import (
	"errors"
	"fmt"
	"strings"
)

// NoneFound is the text of the error of a discovery that finds no supported screen.
const NoneFound = "no supported screen found"

// Line is a screen as the command devices writes it, and as the error of several screens lists it.
func Line(path string, vendor, product uint16, name, serial string) string {
	return fmt.Sprintf("%s\t%04x:%04x\t%s\t%s", path, vendor, product, orDash(name), orDash(serial))
}

// Several is the error of a discovery that finds more than one screen, which lists their lines.
func Several(lines []string) error {
	return errors.New("more than one supported screen is connected, name one with --device:\n" +
		strings.Join(lines, "\n"))
}

func orDash(text string) string {
	if text == "" {
		return "-"
	}
	return text
}
