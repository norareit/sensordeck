package winhid_test

import (
	"github.com/norareit/sensordeck/internal/fakehid"
	"github.com/norareit/sensordeck/internal/winhid"
)

const (
	screenPath  = `\\?\hid#vid_0483&pid_0065#7&1&0&0000#{guid}`
	screenPath2 = `\\?\hid#vid_0483&pid_0065#7&2&0&0000#{guid}`
)

var screenCaps = winhid.Caps{InputLength: 65, OutputLength: 65}

func screen(path, serial string) fakehid.Collection {
	return fakehid.Collection{Path: path, Vendor: 0x0483, Product: 0x0065, Name: "Screen", Serial: serial, Caps: screenCaps}
}

func report(first byte, rest int) []byte {
	buffer := make([]byte, 1+rest)
	buffer[1] = first
	return buffer
}
