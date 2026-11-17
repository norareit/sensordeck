package themedump

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/png"
	"reflect"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/faketheme"
	"github.com/norareit/sensordeck/internal/themefile"
)

// sampleJSON is the theme.json of faketheme.Sample, written out in full to fix every name, the order of the keys, the
// indent, the final newline and the UTF-8 of the glyph order.
const sampleJSON = `{
  "file_size": 4506,
  "header": {
    "byte_0_unknown": 150,
    "raw": "96000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
  },
  "records": [
    {
      "slot": 1,
      "type": 129,
      "type_name": "background",
      "index": 1,
      "tag": 0,
      "x": 0,
      "y": 0,
      "width": 4,
      "height": 2,
      "byte_11_unknown": 1,
      "fill_colour_guess": "f79e",
      "offset": 4096,
      "bytes_18_63_unknown": "01000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "raw": "810100000000000004000201f79e0000100001000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "files": [
        "slot01-background.png"
      ]
    },
    {
      "slot": 2,
      "type": 148,
      "type_name": "startup",
      "index": 2,
      "tag": 0,
      "x": 0,
      "y": 0,
      "width": 2,
      "height": 2,
      "offset": 4112,
      "frame_count": 2,
      "duration_ms_guess": 1500,
      "frame_period_ms_guess": 150,
      "bytes_20_63_unknown": "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "raw": "9402000000000000020002000010100205dc00960000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "files": [
        "slot02-startup-frame1.png",
        "slot02-startup-frame2.png"
      ]
    },
    {
      "slot": 3,
      "type": 132,
      "type_name": "image",
      "index": 3,
      "tag": 0,
      "x": 10,
      "y": 10,
      "width": 3,
      "height": 2,
      "offset": 4128,
      "frame_count": 2,
      "byte_16_unknown": 1,
      "frame_period_ms_guess": 200,
      "bytes_19_63_unknown": "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "raw": "840300000a000a0003000200001020020100c8000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "files": [
        "slot03-image-frame1.png",
        "slot03-image-frame2.png"
      ]
    },
    {
      "slot": 4,
      "type": 147,
      "type_name": "text",
      "index": 4,
      "tag": 0,
      "x": 20,
      "y": 20,
      "width": 5,
      "height": 3,
      "offset": 4164,
      "colour": "ffff",
      "opacity_guess": 255,
      "bytes_18_63_unknown": "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "raw": "930400001400140005000300001044ffffff00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
      "files": [
        "slot04-text.png"
      ]
    },
    {
      "slot": 5,
      "type": 146,
      "type_name": "number",
      "index": 5,
      "tag": 1,
      "x": 30,
      "y": 30,
      "width": 20,
      "height": 3,
      "alignment": 1,
      "colour": "07e0",
      "thousands_mode": 0,
      "opacity": 255,
      "offset": 4179,
      "glyph_height": 3,
      "glyph_widths": [
        2,
        2,
        2,
        2,
        2,
        2,
        2,
        2,
        2,
        2,
        1,
        2
      ],
      "glyph_order": "0123456789.-",
      "bytes_46_63_unknown": "000000000000000000000000000000000000",
      "raw": "920501001e001e001400030107e000ff000010530003000200020002000200020002000200020002000200010002000000000000000000000000000000000000",
      "files": [
        "slot05-number-glyphs.png"
      ]
    },
    {
      "slot": 6,
      "type": 142,
      "type_name": "datetime",
      "index": 6,
      "tag": 21,
      "x": 40,
      "y": 40,
      "width": 60,
      "height": 3,
      "byte_11_unknown": 1,
      "colour": "001f",
      "opacity_guess": 255,
      "offset": 4248,
      "glyph_height": 3,
      "digit_width": 2,
      "other_glyph_widths": [
        1,
        2,
        3,
        4,
        5,
        6,
        7,
        8,
        9,
        10,
        11
      ],
      "other_glyph_order": ".-/ :年月日时分秒",
      "format": "1-2-3 4:5:6",
      "raw": "8e061500280028003c000301001fff0000109800030002000100020003000400050006000700080009000a000b312d322d3320343a353a360000000000000000",
      "files": [
        "slot06-datetime-glyphs.png"
      ]
    },
    {
      "slot": 7,
      "type": 139,
      "type_name": "bar",
      "index": 7,
      "tag": 0,
      "x": 50,
      "y": 50,
      "width": 6,
      "height": 4,
      "byte_11_unknown": 1,
      "empty_colour": "0000",
      "fill_or_frame_colour_14": "a800",
      "fill_or_frame_colour_16": "a800",
      "bytes_18_63_unknown": "76b000000000000076b0000000000000000000000000000000000000000000000000000000000000000000000000",
      "raw": "8b07000032003200060004010000a800a80076b000000000000076b0000000000000000000000000000000000000000000000000000000000000000000000000",
      "files": []
    },
    {
      "slot": 9,
      "type": 119,
      "type_name": "unknown",
      "raw": "770102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f",
      "files": []
    }
  ],
  "warnings": [
    "slot 9: record type 0x77 is not known, kept as raw bytes"
  ]
}
`

func names(files []File) []string {
	var got []string
	for _, file := range files {
		got = append(got, file.Name)
	}
	return got
}

func TestDumpGivesTheFilesInOrderAndTheWarnings(test *testing.T) {
	files, warnings, err := Dump(faketheme.Sample())
	if err != nil {
		test.Fatal(err)
	}
	want := []string{"theme.json", "slot01-background.png", "slot02-startup-frame1.png", "slot02-startup-frame2.png",
		"slot03-image-frame1.png", "slot03-image-frame2.png", "slot04-text.png", "slot05-number-glyphs.png",
		"slot06-datetime-glyphs.png", "preview.png"}
	if got := names(files); !reflect.DeepEqual(got, want) {
		test.Errorf("files %q", got)
	}
	if want := []string{"slot 9: record type 0x77 is not known, kept as raw bytes"}; !reflect.DeepEqual(warnings, want) {
		test.Errorf("warnings %q", warnings)
	}
}

func TestThemeJSONIsWrittenOutInFull(test *testing.T) {
	files, _, err := Dump(faketheme.Sample())
	if err != nil {
		test.Fatal(err)
	}
	if got := string(files[0].Content); got != sampleJSON {
		test.Errorf("theme.json:\n%s", got)
	}
}

func TestEachPNGHasTheBoundsAndPixelsOfItsImage(test *testing.T) {
	files, _, _ := Dump(faketheme.Sample())
	checks := map[string][4]int{
		// name: width, height, column, row of a pixel, with the expected alpha or red below
		"slot01-background.png":      {4, 2, 3, 1},
		"slot03-image-frame1.png":    {3, 2, 0, 0},
		"slot05-number-glyphs.png":   {23, 3, 21, 0},
		"slot06-datetime-glyphs.png": {86, 3, 75, 0},
		"preview.png":                {480, 320, 0, 0},
	}
	for _, file := range files {
		check, wanted := checks[file.Name]
		if !wanted {
			continue
		}
		decoded, err := png.Decode(bytes.NewReader(file.Content))
		if err != nil {
			test.Fatalf("%s: %v", file.Name, err)
		}
		if decoded.Bounds().Dx() != check[0] || decoded.Bounds().Dy() != check[1] {
			test.Errorf("%s: bounds %v", file.Name, decoded.Bounds())
		}
		_, _, _, alpha := decoded.At(check[2], check[3]).RGBA()
		switch file.Name {
		case "slot03-image-frame1.png":
			if alpha>>8 != 0x31 {
				test.Errorf("%s: alpha %#x", file.Name, alpha>>8)
			}
		case "slot05-number-glyphs.png":
			if alpha>>8 != 0x80+63 {
				test.Errorf("%s: alpha %#x", file.Name, alpha>>8)
			}
		case "slot06-datetime-glyphs.png":
			if alpha>>8 != 0x71 {
				test.Errorf("%s: alpha %#x", file.Name, alpha>>8)
			}
		}
	}
}

func datetimeFile(format []byte) []byte {
	builder := faketheme.New()
	slot := faketheme.Head(0x8e, 1, 0, 0, 0, 1, 1)
	copy(slot[45:], format)
	faketheme.Put32(&slot, 15, builder.Resource(nil))
	builder.Slot(1, slot)
	return builder.Bytes()
}

func recordsOf(test *testing.T, file File) []map[string]json.RawMessage {
	test.Helper()
	var decoded struct {
		Records []map[string]json.RawMessage `json:"records"`
	}
	if err := json.Unmarshal(file.Content, &decoded); err != nil {
		test.Fatal(err)
	}
	return decoded.Records
}

func TestTheFormatIsLeftOutWhenItIsNotPrintableAndEmptyWhenItIsEmpty(test *testing.T) {
	files, _, err := Dump(datetimeFile([]byte{'1', 0x80}))
	if err != nil {
		test.Fatal(err)
	}
	if _, present := recordsOf(test, files[0])[0]["format"]; present {
		test.Error("a format with the byte 0x80 is written")
	}
	files, _, _ = Dump(datetimeFile(nil))
	if got := string(recordsOf(test, files[0])[0]["format"]); got != `""` {
		test.Errorf("empty format %s", got)
	}
}

func TestAnImageWithNoPixelsGivesNoFile(test *testing.T) {
	builder := faketheme.New()
	zeroWidth := faketheme.Head(0x84, 1, 0, 0, 0, 0, 5)
	zeroWidth[15] = 2
	faketheme.Put32(&zeroWidth, 11, 0x1000)
	noFrames := faketheme.Head(0x84, 2, 0, 0, 0, 5, 5)
	faketheme.Put32(&noFrames, 11, 0x1000)
	builder.Slot(1, zeroWidth)
	builder.Slot(2, noFrames)
	files, _, err := Dump(builder.Bytes())
	if err != nil {
		test.Fatal(err)
	}
	if got := names(files); !reflect.DeepEqual(got, []string{"theme.json", "preview.png"}) {
		test.Errorf("files %q", got)
	}
	for _, record := range recordsOf(test, files[0]) {
		if string(record["files"]) != "[]" {
			test.Errorf("files %s", record["files"])
		}
	}
}

func TestASharedResourceIsWrittenOnceForEachSlot(test *testing.T) {
	builder := faketheme.New()
	offset := builder.Resource(make([]byte, 6))
	for _, number := range []int{3, 8} {
		slot := faketheme.Head(0x84, byte(number), 0, 0, 0, 1, 1)
		faketheme.Put32(&slot, 11, offset)
		slot[15] = 2
		builder.Slot(number, slot)
	}
	files, _, err := Dump(builder.Bytes())
	if err != nil {
		test.Fatal(err)
	}
	want := []string{"theme.json", "slot03-image-frame1.png", "slot03-image-frame2.png", "slot08-image-frame1.png", "slot08-image-frame2.png", "preview.png"}
	if got := names(files); !reflect.DeepEqual(got, want) {
		test.Errorf("files %q", got)
	}
}

func TestTheSameInputGivesTheSameBytes(test *testing.T) {
	first, _, _ := Dump(faketheme.Sample())
	second, _, _ := Dump(faketheme.Sample())
	if !reflect.DeepEqual(first, second) {
		test.Error("two dumps of the same file differ")
	}
}

func TestAShortFileGivesTheErrorOfTheReaderAndNoFiles(test *testing.T) {
	files, warnings, err := Dump(make([]byte, 100))
	if err == nil || !strings.Contains(err.Error(), "the file is 100 bytes") || files != nil || warnings != nil {
		test.Errorf("%v %v %v", files, warnings, err)
	}
	var resourceError *themefile.ResourceError
	builder := faketheme.New()
	slot := faketheme.Head(0x93, 1, 0, 0, 0, 5, 5)
	faketheme.Put32(&slot, 11, 0x9000)
	builder.Slot(1, slot)
	_, _, err = Dump(builder.Bytes())
	if !errors.As(err, &resourceError) {
		test.Errorf("%v", err)
	}
}
