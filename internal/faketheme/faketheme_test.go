package faketheme

import (
	"bytes"
	"testing"
)

func TestAnEmptyBuilderGivesTheZeroTable(test *testing.T) {
	if got := New().Bytes(); !bytes.Equal(got, make([]byte, 4096)) {
		test.Errorf("%d bytes, not all zero", len(got))
	}
}

func TestHeaderAndSlotWriteTheirBytesAtTheirPlace(test *testing.T) {
	builder := New()
	var header, slot [64]byte
	for position := range header {
		header[position], slot[position] = byte(position+1), byte(200-position)
	}
	builder.Header(header)
	builder.Slot(3, slot)
	file := builder.Bytes()
	if !bytes.Equal(file[:64], header[:]) || !bytes.Equal(file[192:256], slot[:]) {
		test.Error("the header or the slot is not where it belongs")
	}
	if !bytes.Equal(file[64:192], make([]byte, 128)) || !bytes.Equal(file[256:], make([]byte, 4096-256)) {
		test.Error("other bytes of the table changed")
	}
}

func TestResourcesFollowTheTableOneAfterTheOther(test *testing.T) {
	builder := New()
	first := builder.Resource([]byte{1, 2, 3})
	second := builder.Resource([]byte{4})
	if first != 0x1000 || second != 0x1003 {
		test.Errorf("offsets %#x and %#x", first, second)
	}
	if got := builder.Bytes()[4096:]; !bytes.Equal(got, []byte{1, 2, 3, 4}) {
		test.Errorf("resources % x", got)
	}
}

func TestBytesGivesANewSliceEachTime(test *testing.T) {
	builder := New()
	builder.Bytes()[0] = 9
	if builder.Bytes()[0] != 0 {
		test.Error("the builder shares its storage")
	}
}

func TestHeadAndPutWriteBigEndian(test *testing.T) {
	slot := Head(0x84, 7, 9, 0x0102, 0x0304, 0x0506, 0x0708)
	want := []byte{0x84, 7, 9, 1, 2, 3, 4, 5, 6, 7, 8, 0}
	if !bytes.Equal(slot[:12], want) {
		test.Errorf("head % x", slot[:12])
	}
	Put16(&slot, 20, 0xabcd)
	Put32(&slot, 30, 0x01020304)
	if !bytes.Equal(slot[20:22], []byte{0xab, 0xcd}) || !bytes.Equal(slot[30:34], []byte{1, 2, 3, 4}) {
		test.Error("Put16 or Put32 wrote the wrong bytes")
	}
}

func TestSampleHasTheSlotsOfItsDescription(test *testing.T) {
	file := Sample()
	if file[0] != 0x96 {
		test.Errorf("header byte %#x", file[0])
	}
	types := map[int]byte{1: 0x81, 2: 0x94, 3: 0x84, 4: 0x93, 5: 0x92, 6: 0x8e, 7: 0x8b, 9: 0x77}
	for slot := 1; slot < 64; slot++ {
		want := types[slot]
		if got := file[slot*64]; got != want {
			test.Errorf("slot %d type %#x, want %#x", slot, got, want)
		}
	}
	// The last resource is the datetime glyphs: 258 bytes at the offset of byte 15 of slot 6.
	offset := int(file[6*64+15])<<24 | int(file[6*64+16])<<16 | int(file[6*64+17])<<8 | int(file[6*64+18])
	if offset+258 != len(file) {
		test.Errorf("the last resource ends at %d, the file at %d", offset+258, len(file))
	}
	if x := int(file[3*64+3])<<8 | int(file[3*64+4]); x != 10 {
		test.Errorf("image x %d", x)
	}
}
