package themefile_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/faketheme"
	"github.com/norareit/sensordeck/internal/themefile"
)

func TestParseReadsTheSample(test *testing.T) {
	file := faketheme.Sample()
	theme, warnings, err := themefile.Parse(file)
	if err != nil {
		test.Fatal(err)
	}
	if theme.FileSize != len(file) || theme.Header[0] != 0x96 || !reflect.DeepEqual(theme.Header[1:], make([]byte, 63)) {
		test.Errorf("header %v size %d", theme.Header, theme.FileSize)
	}
	wantSlots := []int{1, 2, 3, 4, 5, 6, 7, 9}
	wantTypes := []byte{0x81, 0x94, 0x84, 0x93, 0x92, 0x8e, 0x8b, 0x77}
	if len(theme.Entries) != len(wantSlots) {
		test.Fatalf("%d entries", len(theme.Entries))
	}
	for position, entry := range theme.Entries {
		encoded := entry.Record.Encode()
		if entry.Slot != wantSlots[position] || encoded[0] != wantTypes[position] {
			test.Errorf("entry %d is slot %d type %#x", position, entry.Slot, encoded[0])
		}
		if string(encoded[:]) != string(file[entry.Slot*64:entry.Slot*64+64]) {
			test.Errorf("slot %d does not encode to its bytes", entry.Slot)
		}
	}
	if want := []string{"slot 9: record type 0x77 is not known, kept as raw bytes"}; !reflect.DeepEqual(warnings, want) {
		test.Errorf("warnings %q", warnings)
	}
}

func TestAnEmptyTableGivesNoEntriesAndNoWarning(test *testing.T) {
	theme, warnings, err := themefile.Parse(make([]byte, 4096))
	if err != nil || len(theme.Entries) != 0 || len(warnings) != 0 {
		test.Errorf("%v %q %v", theme.Entries, warnings, err)
	}
}

func TestEntriesFollowTheOrderOfTheSlots(test *testing.T) {
	builder := faketheme.New()
	builder.Slot(5, faketheme.Head(0x8b, 1, 0, 0, 0, 1, 1))
	builder.Slot(2, faketheme.Head(0x8b, 9, 0, 0, 0, 1, 1))
	theme, _, err := themefile.Parse(builder.Bytes())
	if err != nil {
		test.Fatal(err)
	}
	if theme.Entries[0].Slot != 2 || theme.Entries[1].Slot != 5 {
		test.Errorf("slots %d and %d", theme.Entries[0].Slot, theme.Entries[1].Slot)
	}
}

func TestATypeByteOfZeroOrOfNoKnownTypeIsAnUnknownRecord(test *testing.T) {
	builder := faketheme.New()
	var zero, other [64]byte
	zero[10] = 5
	other[0], other[1] = 0xaa, 1
	builder.Slot(3, zero)
	builder.Slot(4, other)
	theme, warnings, err := themefile.Parse(builder.Bytes())
	if err != nil {
		test.Fatal(err)
	}
	want := []string{"slot 3: record type 0x00 is not known, kept as raw bytes", "slot 4: record type 0xaa is not known, kept as raw bytes"}
	if !reflect.DeepEqual(warnings, want) {
		test.Errorf("warnings %q", warnings)
	}
	if first := theme.Entries[0].Record.(themefile.Unknown); first.Type != 0 || first.Bytes1To63[9] != 5 {
		test.Errorf("first %+v", first)
	}
}

func datetimeWithFormat(format []byte) []byte {
	builder := faketheme.New()
	slot := faketheme.Head(0x8e, 1, 0, 0, 0, 1, 1)
	copy(slot[45:], format)
	faketheme.Put32(&slot, 15, builder.Resource(nil))
	builder.Slot(1, slot)
	return builder.Bytes()
}

func TestAnUnknownCharacterOfADatetimeFormatIsAWarningEachTime(test *testing.T) {
	_, warnings, err := themefile.Parse(datetimeWithFormat([]byte{'1', 'x', '2', 0x7f, 0, 'y'}))
	if err != nil {
		test.Fatal(err)
	}
	want := []string{"slot 1 (datetime): format character 'x' is not known", "slot 1 (datetime): format character '\\x7f' is not known"}
	if !reflect.DeepEqual(warnings, want) {
		test.Errorf("warnings %q", warnings)
	}
}

func TestAFormatOfNineteenBytesIsReadToTheEndOfTheSlot(test *testing.T) {
	format := make([]byte, 19)
	for position := range format {
		format[position] = '1'
	}
	format[18] = 'z'
	_, warnings, err := themefile.Parse(datetimeWithFormat(format))
	if err != nil || len(warnings) != 1 || warnings[0] != "slot 1 (datetime): format character 'z' is not known" {
		test.Errorf("%q %v", warnings, err)
	}
	theme, _, _ := themefile.Parse(datetimeWithFormat([]byte("1-2-3 4:5:6.,")))
	if got := string(theme.Entries[0].Record.(themefile.Datetime).FormatText()); got != "1-2-3 4:5:6.," {
		test.Errorf("format %q", got)
	}
}

func TestAShortFileIsAnError(test *testing.T) {
	for _, length := range []int{0, 4095} {
		_, _, err := themefile.Parse(make([]byte, length))
		want := fmt.Sprintf("the file is %d bytes, shorter than the record table of 4096 bytes", length)
		if err == nil || err.Error() != want {
			test.Errorf("length %d: %v", length, err)
		}
	}
}

func TestAResourceThatRunsPastTheEndIsAnErrorThatNamesIt(test *testing.T) {
	cases := []struct {
		name, typeName string
		build          func(slot *[64]byte, offset uint32)
		recordType     byte
		size           uint64
	}{
		{"background", "background", func(slot *[64]byte, offset uint32) { faketheme.Put32(slot, 14, offset) }, 0x81, 8},
		{"startup", "startup", func(slot *[64]byte, offset uint32) { faketheme.Put32(slot, 11, offset); slot[15] = 2 }, 0x94, 16},
		{"image", "image", func(slot *[64]byte, offset uint32) { faketheme.Put32(slot, 11, offset); slot[15] = 2 }, 0x84, 24},
		{"text", "text", func(slot *[64]byte, offset uint32) { faketheme.Put32(slot, 11, offset) }, 0x93, 4},
		{"number", "number", func(slot *[64]byte, offset uint32) {
			faketheme.Put32(slot, 16, offset)
			faketheme.Put16(slot, 20, 2)
			faketheme.Put16(slot, 22, 3)
		}, 0x92, 6},
		{"datetime", "datetime", func(slot *[64]byte, offset uint32) {
			faketheme.Put32(slot, 15, offset)
			faketheme.Put16(slot, 19, 2)
			faketheme.Put16(slot, 21, 1)
		}, 0x8e, 20},
	}
	for _, each := range cases {
		slot := faketheme.Head(each.recordType, 1, 0, 0, 0, 2, 2)
		builder := faketheme.New()
		builder.Resource(make([]byte, 100))
		offset := uint32(0x1000+100) - uint32(each.size) + 1
		each.build(&slot, offset)
		builder.Slot(1, slot)
		file := builder.Bytes()
		_, _, err := themefile.Parse(file)
		var resourceError *themefile.ResourceError
		if !errors.As(err, &resourceError) {
			test.Errorf("%s: %v", each.name, err)
			continue
		}
		want := fmt.Sprintf("slot 1 (%s): offset 0x%x with %d bytes runs past the end of the file of %d bytes", each.typeName, offset, each.size, len(file))
		if resourceError.Slot != 1 || resourceError.TypeName != each.typeName || resourceError.Field != "offset" || resourceError.Offset != offset ||
			resourceError.Size != each.size || resourceError.FileSize != len(file) || err.Error() != want {
			test.Errorf("%s: %+v, text %q, want %q", each.name, *resourceError, err.Error(), want)
		}
	}
}

func TestTheSizeOfAResourceIsComputedIn64Bits(test *testing.T) {
	builder := faketheme.New()
	slot := faketheme.Head(0x84, 1, 0, 0, 0, 65535, 65535)
	faketheme.Put32(&slot, 11, 0xfffffff0)
	slot[15] = 255
	builder.Slot(1, slot)
	_, _, err := themefile.Parse(builder.Bytes())
	var resourceError *themefile.ResourceError
	if !errors.As(err, &resourceError) {
		test.Fatal(err)
	}
	wantSize := uint64(255) * 3 * 65535 * 65535
	if resourceError.Size != wantSize || resourceError.Offset != 0xfffffff0 {
		test.Errorf("%+v", *resourceError)
	}
	want := fmt.Sprintf("slot 1 (image): offset 0xfffffff0 with %d bytes runs past the end of the file of 4096 bytes", wantSize)
	if err.Error() != want {
		test.Errorf("%q", err.Error())
	}
}

func TestTheErrorIsThatOfTheFirstMalformedSlot(test *testing.T) {
	builder := faketheme.New()
	for _, number := range []int{4, 2} {
		slot := faketheme.Head(0x93, 1, 0, 0, 0, 1, 1)
		faketheme.Put32(&slot, 11, 0x5000)
		builder.Slot(number, slot)
	}
	_, _, err := themefile.Parse(builder.Bytes())
	var resourceError *themefile.ResourceError
	if !errors.As(err, &resourceError) || resourceError.Slot != 2 {
		test.Errorf("%v", err)
	}
}

func TestWhatTheSpecLeavesUnchecked(test *testing.T) {
	builder := faketheme.New()
	var header [64]byte
	header[0], header[63] = 0x01, 0xff
	builder.Header(header)
	// A resource inside the record table, a box off the screen, a tag and an index that no sample has.
	text := faketheme.Head(0x93, 200, 99, 60000, 60000, 2, 1)
	faketheme.Put32(&text, 11, 0)
	builder.Slot(1, text)
	// A gap between two resources.
	builder.Resource([]byte{1, 2})
	gap := builder.Resource([]byte{3, 4})
	second := faketheme.Head(0x93, 1, 0, 0, 0, 2, 1)
	faketheme.Put32(&second, 11, gap+100)
	builder.Slot(2, second)
	file := append(builder.Bytes(), make([]byte, 200)...)
	_, warnings, err := themefile.Parse(file)
	if err != nil || len(warnings) != 0 {
		test.Errorf("%q %v", warnings, err)
	}
}

func TestNoPrefixOfAFileMakesTheReaderPanic(test *testing.T) {
	file := faketheme.Sample()
	for length := 0; length <= len(file); length++ {
		theme, _, err := themefile.Parse(file[:length])
		if err != nil && len(theme.Entries) != 0 {
			test.Errorf("length %d gave an error and a theme", length)
		}
		if err == nil && length < 4096 {
			test.Errorf("length %d gave a theme", length)
		}
		if length == len(file) && err != nil {
			test.Errorf("the whole file gave %v", err)
		}
	}
}
