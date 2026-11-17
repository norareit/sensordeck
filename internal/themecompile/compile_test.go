package themecompile

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/image/font/gofont/gomono"

	"github.com/norareit/sensordeck/internal/themefile"
)

func solidPNG(test *testing.T, width, height int) []byte {
	test.Helper()
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	picture.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	return encodePNG(test, picture)
}

// roundTripFolder holds a widget of each type, two numbers of one metric and a third number.
func roundTripFolder(test *testing.T) fstest.MapFS {
	test.Helper()
	theme := `name: round trip
background: back.png
fonts:
  mono: {file: fonts/mono.ttf, size: 18, color: "#ff8000"}
  body: {builtin: go-bold, size: 24, color: "#0080ff"}
widgets:
  - {type: text, font: body, text: Hello, x: 10, y: 10}
  - {type: image, file: pictures/dot.png, x: 100, y: 100}
  - {type: number, font: mono, metric: gpu.temp, x: 10, y: 60}
  - {type: number, font: mono, metric: gpu.temp, x: 100, y: 60, digits: 2}
  - {type: number, font: body, metric: cpu.temp, x: 200, y: 60}
  - {type: number, font: mono, metric: cpu.util, x: 300, y: 60, align: right, opacity: 0.5, thousands: true}
  - {type: bar, metric: cpu.util, x: 10, y: 250, width: 100, height: 8, color: "#00ff00", background: "#ff0000"}
  - {type: datetime, font: body, format: "YYYY-MM-DD HH:mm:ss", x: 10, y: 200}
`
	return fstest.MapFS{
		"theme.yaml":       {Data: []byte(theme)},
		"back.png":         {Data: solidPNG(test, 480, 320)},
		"pictures/dot.png": {Data: solidPNG(test, 4, 4)},
		"fonts/mono.ttf":   {Data: gomono.TTF},
	}
}

func TestRoundTrip(test *testing.T) {
	result, err := Compile(roundTripFolder(test))
	if err != nil {
		test.Fatal(err)
	}
	theme, warnings, err := themefile.Parse(result.File)
	if err != nil || len(warnings) != 0 {
		test.Fatalf("%v, %v", err, warnings)
	}
	if theme.Header[0] != 0x96 || len(theme.Entries) != 9 || len(result.Records) != 9 {
		test.Fatalf("header %x, %d entries, %d records", theme.Header[0], len(theme.Entries), len(result.Records))
	}
	for position, entry := range theme.Entries {
		if entry.Slot != position+1 || !reflect.DeepEqual(entry.Record, result.Records[position]) {
			test.Errorf("slot %d differs from the record built", entry.Slot)
		}
	}
	checkWidgets(test, theme)
}

func checkWidgets(test *testing.T, theme themefile.Theme) {
	test.Helper()
	text := theme.Entries[1].Record.(themefile.Text)
	if text.X != 10 || text.Colour != rgb565(0, 0x80, 0xff) {
		test.Errorf("text %+v", text.Head)
	}
	image := theme.Entries[2].Record.(themefile.Image)
	if image.X != 100 || image.Y != 100 || image.Frames()[0].Alpha[0] != 255 || image.Frames()[0].Colour[0] != byte(rgb565(1, 2, 3)) {
		test.Errorf("image %+v", image.Head)
	}
	first, second := theme.Entries[3].Record.(themefile.Number), theme.Entries[4].Record.(themefile.Number)
	third := theme.Entries[5].Record.(themefile.Number)
	if first.Tag != 3 || second.Tag != 3 || third.Tag != 1 || first.Colour != rgb565(255, 0x80, 0) {
		test.Errorf("tags %d, %d, %d", first.Tag, second.Tag, third.Tag)
	}
	options := theme.Entries[6].Record.(themefile.Number)
	if options.Tag != 2 || options.Alignment != 2 || options.Opacity != 128 || options.ThousandsMode != 1 || first.Alignment != 1 || first.Opacity != 255 || first.ThousandsMode != 0 {
		test.Errorf("number %+v", options)
	}
	bar := theme.Entries[7].Record.(themefile.Bar)
	if bar.X != 10 || bar.Y != 250 || bar.Width != 100 || bar.Height != 8 || bar.Tag != 2 || bar.Byte11 != 1 || bar.EmptyColour != rgb565(255, 0, 0) ||
		bar.FillOrFrameColour14 != rgb565(0, 255, 0) || bar.FillOrFrameColour16 != rgb565(0, 255, 0) ||
		bar.Bytes18To63[0] != 0x76 || bar.Bytes18To63[1] != 0xb0 || bar.Bytes18To63[8] != 0x76 || bar.Bytes18To63[9] != 0xb0 {
		test.Errorf("bar %+v", bar)
	}
	datetime := theme.Entries[8].Record.(themefile.Datetime)
	if datetime.Tag != 21 || string(datetime.FormatText()) != "1-2-3 4:5:6" || datetime.Y != 200 {
		test.Errorf("datetime %+v", datetime.Head)
	}
}

func compileError(test *testing.T, theme string, files map[string][]byte) string {
	test.Helper()
	folder := fstest.MapFS{"theme.yaml": {Data: []byte(theme)}}
	for name, content := range files {
		folder[name] = &fstest.MapFile{Data: content}
	}
	_, err := Compile(folder)
	if err == nil {
		test.Fatal("no error")
	}
	return err.Error()
}

func TestFileFaults(test *testing.T) {
	notAPNG := map[string][]byte{"x.png": []byte("not a png")}
	cases := []struct {
		name, theme string
		files       map[string][]byte
		want        string
	}{
		{"font file missing", "name: t\nbackground: \"#000000\"\nfonts:\n  f: {file: a.ttf, size: 10, color: \"#000000\"}\n", nil, "font f: cannot read a.ttf: file does not exist"},
		{"background missing", "name: t\nbackground: b.png\n", nil, "background: cannot read b.png: file does not exist"},
		{"png does not decode", "name: t\nbackground: x.png\n", notAPNG, "background: cannot decode x.png: png: invalid format: not a PNG file"},
		{"not a font", "name: t\nbackground: \"#000000\"\nfonts:\n  f: {file: x.ttf, size: 10, color: \"#000000\"}\n", map[string][]byte{"x.ttf": []byte("nothing")}, "font f: cannot decode x.ttf:"},
		{"font collection", "name: t\nbackground: \"#000000\"\nfonts:\n  f: {file: x.ttc, size: 10, color: \"#000000\"}\n", map[string][]byte{"x.ttc": []byte("ttcf....")}, "font f: x.ttc holds a font collection, not one font"},
		{"background of 480 x 321", "name: t\nbackground: b.png\n", map[string][]byte{"b.png": solidPNG(test, 480, 321)}, "background: b.png is 480 x 321, want 480 x 320"},
		{"path with ..", "name: t\nbackground: ../x.png\n", nil, `background: "../x.png" is not a path inside the folder`},
		{"absolute path", "name: t\nbackground: /x.png\n", nil, `background: "/x.png" is not a path inside the folder`},
		{"path out by cleaning", "name: t\nbackground: a/../../x.png\n", nil, `background: "a/../../x.png" is not a path inside the folder`},
		{"glyph the font lacks", solidTheme + "widgets:\n  - {type: text, font: body, text: 年, x: 0, y: 0}\n", nil, `widget 1 (text): the font body has no glyph for "年"`},
		{"box past the right edge", solidTheme + "widgets:\n  - {type: image, file: i.png, x: 477, y: 0}\n", map[string][]byte{"i.png": solidPNG(test, 4, 4)}, "widget 1 (image): the box at 477,0 of 4 x 4 is not within the screen of 480 x 320"},
	}
	for _, each := range cases {
		if got := compileError(test, each.theme, each.files); !strings.HasPrefix(got, each.want) {
			test.Errorf("%s: got %q, want %q", each.name, got, each.want)
		}
	}
}

func TestPathsThatStayInsideTheFolderOpen(test *testing.T) {
	files := map[string][]byte{"x.png": solidPNG(test, 480, 320)}
	compileFiles(test, "name: t\nbackground: a/../x.png\n", files)
	theme := solidTheme + "widgets:\n  - {type: image, file: i.png, x: 476, y: 316}\n"
	compileFiles(test, theme, map[string][]byte{"i.png": solidPNG(test, 4, 4)})
}

// hugePNG is a 1 x 1 PNG whose header claims a size of 10000 x 10000, with its CRC made again.
func hugePNG(test *testing.T) []byte {
	test.Helper()
	content := solidPNG(test, 1, 1)
	binary.BigEndian.PutUint32(content[16:], 10000)
	binary.BigEndian.PutUint32(content[20:], 10000)
	binary.BigEndian.PutUint32(content[29:], crc32.ChecksumIEEE(content[12:29]))
	return content
}

func TestLargeHeaderIsRefusedBeforeDecoding(test *testing.T) {
	files := map[string][]byte{"h.png": hugePNG(test)}
	got := compileError(test, solidTheme+"widgets:\n  - {type: image, file: h.png, x: 0, y: 0}\n", files)
	if want := "widget 1 (image): the box at 0,0 of 10000 x 10000 is not within the screen of 480 x 320"; got != want {
		test.Errorf("image: got %q", got)
	}
	got = compileError(test, "name: t\nbackground: h.png\n", files)
	if want := "background: h.png is 10000 x 10000, want 480 x 320"; got != want {
		test.Errorf("background: got %q", got)
	}
	got = compileError(test, solidTheme+"widgets:\n  - {type: text, font: body, text: "+strings.Repeat("W", 80)+", x: 0, y: 0}\n", nil)
	if !strings.HasPrefix(got, "widget 1 (text): the box at 0,0 of ") || !strings.HasSuffix(got, "is not within the screen of 480 x 320") {
		test.Errorf("text: got %q", got)
	}
}

func TestWidgetCountAndFlashLimit(test *testing.T) {
	compileFiles(test, solidTheme+"widgets:\n"+strings.Repeat("  - {type: text, font: body, text: a, x: 0, y: 0}\n", 62), nil)
	theme := solidTheme + "widgets:\n" + strings.Repeat("  - {type: image, file: i.png, x: 0, y: 0}\n", 9)
	got := compileError(test, theme, map[string][]byte{"i.png": solidPNG(test, 480, 320)})
	if want := "widgets: the compiled file is 4458496 bytes, larger than the flash limit of 4194304 bytes"; got != want {
		test.Errorf("got %q", got)
	}
}

func TestSameFolderGivesSameBytes(test *testing.T) {
	first, err := Compile(roundTripFolder(test))
	if err != nil {
		test.Fatal(err)
	}
	second, err := Compile(roundTripFolder(test))
	if err != nil {
		test.Fatal(err)
	}
	if !bytes.Equal(first.File, second.File) || !reflect.DeepEqual(first.Tags, second.Tags) {
		test.Error("two compilations differ")
	}
}

func TestMaximumCoordinatesAreOutsideTheScreen(test *testing.T) {
	const largest = "9223372036854775807"
	files := map[string][]byte{"h.png": hugePNG(test), "i.png": solidPNG(test, 4, 4)}
	for _, widget := range []string{
		"{type: text, font: body, text: a, x: " + largest + ", y: " + largest + "}",
		"{type: text, font: body, text: a, x: " + largest + ", y: 0}",
		"{type: text, font: body, text: a, x: 0, y: " + largest + "}",
		"{type: number, font: body, metric: cpu.temp, x: " + largest + ", y: " + largest + "}",
		"{type: image, file: i.png, x: " + largest + ", y: 0}",
		"{type: image, file: h.png, x: " + largest + ", y: " + largest + "}",
	} {
		got := compileError(test, solidTheme+"widgets:\n  - "+widget+"\n", files)
		if !strings.Contains(got, "is not within the screen of 480 x 320") || strings.Contains(got, "cannot decode") {
			test.Errorf("%s: got %q", widget, got)
		}
	}
}

func TestTheFileOfTheThemeCannotBeReadOrParsed(test *testing.T) {
	_, err := Compile(fstest.MapFS{"other.txt": {Data: []byte("x")}})
	if err == nil || err.Error() != "cannot read the file: file does not exist" {
		test.Errorf("no theme.yaml: got %v", err)
	}
	_, err = Compile(fstest.MapFS{"theme.yaml": {Data: []byte("name: [a\n")}})
	if err == nil || !strings.HasPrefix(err.Error(), "yaml: ") {
		test.Errorf("invalid YAML: got %v", err)
	}
}

func TestATextWithAnAccentedLetterCompiles(test *testing.T) {
	compileFiles(test, solidTheme+"widgets:\n  - {type: text, font: body, text: café, x: 0, y: 0}\n", nil)
}

func TestACustomMetricGetsATag(test *testing.T) {
	folder := roundTripFolder(test)
	theme := strings.Replace(string(folder["theme.yaml"].Data), "metric: cpu.temp", "metric: water.temp", 1)
	folder["theme.yaml"] = &fstest.MapFile{Data: []byte(theme)}
	result, err := Compile(folder)
	if err != nil {
		test.Fatal(err)
	}
	if tagOf(result.Tags, "water.temp") == 0 {
		test.Fatalf("tags = %v", result.Tags)
	}
}
