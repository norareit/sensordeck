package themecompile

import "github.com/norareit/sensordeck/internal/themefile"

// headerByte is byte 0 of the header slot, which every sample holds.
const headerByte = 0x96

// layout writes the bytes of the file: the record table, whose slot 0 is the header, slot 1 the background and slot 2
// and up the widgets, then the resources back to back in the order of the slots.
func layout(records []built) []byte {
	file := make([]byte, themefile.TableSize)
	file[0] = headerByte
	for position, each := range records {
		slot := each.record.Encode()
		copy(file[(position+1)*themefile.SlotSize:], slot[:])
	}
	for _, each := range records {
		file = append(file, each.resource...)
	}
	return file
}
