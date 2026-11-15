// Package faketheme builds synthetic theme files in code from the layout of the spec of unit
// 0010-theme-file-reader, for the tests of the theme packages. It does not use the reader, so that a test of the reader
// does not check the reader against itself.
package faketheme

import "encoding/binary"

const tableSize = 4096

// Builder makes a theme file: the record table of 4096 bytes and the resources appended after it.
type Builder struct {
	table     [tableSize]byte
	resources []byte
}

func New() *Builder { return &Builder{} }

// Header sets the 64 bytes of slot 0.
func (builder *Builder) Header(header [64]byte) { builder.Slot(0, header) }

// Slot sets the 64 bytes of the slot with the number.
func (builder *Builder) Slot(number int, slot [64]byte) {
	copy(builder.table[number*64:], slot[:])
}

// Resource appends the bytes after what was appended before and gives their absolute offset.
func (builder *Builder) Resource(data []byte) uint32 {
	offset := uint32(tableSize + len(builder.resources))
	builder.resources = append(builder.resources, data...)
	return offset
}

// Bytes gives a new slice: the table followed by the resources.
func (builder *Builder) Bytes() []byte {
	file := make([]byte, 0, tableSize+len(builder.resources))
	file = append(file, builder.table[:]...)
	return append(file, builder.resources...)
}

// Head gives a slot with the common head of the spec written in bytes 0 to 10, big-endian, and zeros after it.
func Head(recordType, index, tag byte, x, y, width, height uint16) [64]byte {
	var slot [64]byte
	slot[0], slot[1], slot[2] = recordType, index, tag
	Put16(&slot, 3, x)
	Put16(&slot, 5, y)
	Put16(&slot, 7, width)
	Put16(&slot, 9, height)
	return slot
}

// Put16 writes the value big-endian at the position.
func Put16(slot *[64]byte, at int, value uint16) { binary.BigEndian.PutUint16(slot[at:], value) }

// Put32 writes the value big-endian at the position.
func Put32(slot *[64]byte, at int, value uint32) { binary.BigEndian.PutUint32(slot[at:], value) }

// Sample is the small valid file of the tests: header byte 0 0x96, a record of each of the seven types in slots 1 to
// 7, slot 8 empty and a record of the unknown type 0x77 in slot 9. The resources are appended in the order of the
// slots, each with distinct documented bytes:
//
//   - slot 1, background 4 x 2 at 0,0: eight pixels, pixel p stored as the two bytes 0x10+p and p, so its colour,
//     read little-endian, is p x 256 + 0x10 + p, for p = 0 to 7
//   - slot 2, startup 2 x 2: frame 1 has the bytes 0x20 to 0x27 and frame 2 the bytes 0x28 to 0x2f
//   - slot 3, image 3 x 2, 2 frames: frame 1 alpha 0x31 to 0x36 and colour bytes 0x40 to 0x4b, frame 2 alpha 0x51 to
//     0x56 and colour bytes 0x60 to 0x6b
//   - slot 4, text 5 x 3: the mask bytes 0x70 to 0x7e
//   - slot 5, number, glyph height 3, widths 2 (ten digits), 1 and 2: the mask bytes 0x80 + n for n = 0 up to the 69
//     bytes of the glyphs
//   - slot 6, datetime, glyph height 3, digit width 2, other widths 1 to 11: the mask bytes 0x90 + n for n = 0 up to
//     the 3 x (20 + 66) = 258 bytes of the glyphs, which wrap modulo 256 above 0xff
//   - slot 7 is a bar and has no resource
func Sample() []byte {
	builder := New()
	var header [64]byte
	header[0] = 0x96
	builder.Header(header)

	slot := Head(0x81, 1, 0, 0, 0, 4, 2)
	slot[11] = 1
	Put16(&slot, 12, 0xf79e)
	pixels := make([]byte, 16)
	for pixel := range 8 {
		pixels[2*pixel], pixels[2*pixel+1] = byte(0x10+pixel), byte(pixel)
	}
	Put32(&slot, 14, builder.Resource(pixels))
	copy(slot[18:], []byte{1, 0, 0, 1})
	builder.Slot(1, slot)

	slot = Head(0x94, 2, 0, 0, 0, 2, 2)
	Put32(&slot, 11, builder.Resource(sequence(0x20, 16)))
	slot[15] = 2
	Put16(&slot, 16, 1500)
	Put16(&slot, 18, 150)
	builder.Slot(2, slot)

	slot = Head(0x84, 3, 0, 10, 10, 3, 2)
	frames := append(sequence(0x31, 6), sequence(0x40, 12)...)
	frames = append(frames, sequence(0x51, 6)...)
	frames = append(frames, sequence(0x60, 12)...)
	Put32(&slot, 11, builder.Resource(frames))
	slot[15], slot[16] = 2, 1
	Put16(&slot, 17, 200)
	builder.Slot(3, slot)

	slot = Head(0x93, 4, 0, 20, 20, 5, 3)
	Put32(&slot, 11, builder.Resource(sequence(0x70, 15)))
	Put16(&slot, 15, 0xffff)
	slot[17] = 0xff
	builder.Slot(4, slot)

	slot = Head(0x92, 5, 1, 30, 30, 20, 3)
	slot[11] = 1
	Put16(&slot, 12, 0x07e0)
	slot[15] = 0xff
	numberWidths := []uint16{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 1, 2}
	total := 0
	for position, width := range numberWidths {
		Put16(&slot, 22+2*position, width)
		total += int(width)
	}
	Put16(&slot, 20, 3)
	Put32(&slot, 16, builder.Resource(sequence(0x80, 3*total)))
	builder.Slot(5, slot)

	slot = Head(0x8e, 6, 21, 40, 40, 60, 3)
	slot[11] = 1
	Put16(&slot, 12, 0x001f)
	slot[14] = 0xff
	Put16(&slot, 19, 3)
	Put16(&slot, 21, 2)
	total = 20
	for position := range 11 {
		Put16(&slot, 23+2*position, uint16(position+1))
		total += position + 1
	}
	copy(slot[45:], "1-2-3 4:5:6")
	Put32(&slot, 15, builder.Resource(sequence(0x90, 3*total)))
	builder.Slot(6, slot)

	slot = Head(0x8b, 7, 0, 50, 50, 6, 4)
	copy(slot[11:], []byte{0x01, 0x00, 0x00, 0xa8, 0x00, 0xa8, 0x00, 0x76, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x76, 0xb0})
	builder.Slot(7, slot)

	var unknown [64]byte
	for position := range unknown {
		unknown[position] = byte(position)
	}
	unknown[0] = 0x77
	builder.Slot(9, unknown)
	return builder.Bytes()
}

// sequence gives count bytes from first upward, wrapping modulo 256.
func sequence(first byte, count int) []byte {
	data := make([]byte, count)
	for position := range data {
		data[position] = first + byte(position)
	}
	return data
}
