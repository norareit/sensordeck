package themecompile

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

// textAt is a text of the font body at a place.
func textAt(text string, x, y int) string {
	return fmt.Sprintf("  - {type: text, font: body, text: %s, x: %d, y: %d}\n", text, x, y)
}

func numberAt(x, y int) string {
	return fmt.Sprintf("  - {type: number, font: body, metric: cpu.temp, digits: 4, x: %d, y: %d}\n", x, y)
}

func parsedRecords(test *testing.T, result Result) []themefile.Entry {
	test.Helper()
	theme, _, err := themefile.Parse(result.File)
	if err != nil {
		test.Fatal(err)
	}
	return theme.Entries
}

func isTextRecord(record themefile.Record) bool {
	_, isText := record.(themefile.Text)
	return isText
}

func TestALoneTextIsDrawnIntoTheBackground(test *testing.T) {
	result := compileFiles(test, solidTheme+"widgets:\n"+textAt("Hi", 10, 20), nil)
	if len(result.Records) != 1 || len(result.File) != 0x1000+2*480*320 {
		test.Errorf("%d records, %d bytes", len(result.Records), len(result.File))
	}
}

func TestATextOverAnEarlierRecordStaysARecord(test *testing.T) {
	image := solidPNG(test, 40, 40)
	cases := []struct{ name, before string }{
		{"number", numberAt(10, 20)},
		{"image", "  - {type: image, file: i.png, x: 10, y: 20}\n"},
	}
	for _, each := range cases {
		theme := solidTheme + "widgets:\n" + each.before + textAt("Hi", 14, 24)
		entries := parsedRecords(test, compileFiles(test, theme, map[string][]byte{"i.png": image}))
		last := entries[len(entries)-1]
		if !isTextRecord(last.Record) || last.Slot != len(entries) || int(last.Record.(themefile.Text).Index) != len(entries)-1 {
			test.Errorf("%s: last is slot %d, %T", each.name, last.Slot, last.Record)
		}
	}
}

// TestATextOverAnEarlierTextKeptAsARecordStaysARecord puts the last text where it overlaps only the first text, which
// is a record because it overlaps the number, so that a compiler that forgot the boxes of the texts it keeps fails.
func TestATextOverAnEarlierTextKeptAsARecordStaysARecord(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 20))
	numberRight := 10 + 4*glyphs.digitWidth
	theme := solidTheme + "widgets:\n" + numberAt(10, 20) + textAt("Hi", numberRight-2, 20) + textAt("Hi", numberRight+glyphs.textWidth("H"), 20)
	entries := parsedRecords(test, compileFiles(test, theme, nil))
	if len(entries) != 4 {
		test.Fatalf("%d entries", len(entries))
	}
	last := entries[3]
	if _, isText := last.Record.(themefile.Text); !isText || last.Slot != 4 || last.Record.(themefile.Text).Index != 3 {
		test.Errorf("last is slot %d, %T", last.Slot, last.Record)
	}
	number := entries[1].Record.(themefile.Number)
	first := entries[2].Record.(themefile.Text)
	if int(last.Record.(themefile.Text).Offset) != int(first.Offset)+len(first.Resource) || number.Index != 1 {
		test.Errorf("offset %d, first at %d of %d bytes", last.Record.(themefile.Text).Offset, first.Offset, len(first.Resource))
	}
}

func TestATextOverOnlyATextInTheBackgroundIsDrawnIntoIt(test *testing.T) {
	theme := solidTheme + "widgets:\n" + textAt("Hi", 10, 20) + textAt("Ho", 12, 22)
	if result := compileFiles(test, theme, nil); len(result.Records) != 1 {
		test.Errorf("%d records", len(result.Records))
	}
}

func TestATextThatOnlyTouchesARecordIsDrawnIntoTheBackground(test *testing.T) {
	glyphs := newGlyphFont(regularFace(test, 20))
	right, bottom := 10+4*glyphs.digitWidth, 20+glyphs.height
	for _, place := range [][2]int{{right, 20}, {10, bottom}} {
		theme := solidTheme + "widgets:\n" + numberAt(10, 20) + textAt("Hi", place[0], place[1])
		if result := compileFiles(test, theme, nil); len(result.Records) != 2 {
			test.Errorf("text at %v: %d records", place, len(result.Records))
		}
	}
	theme := solidTheme + "widgets:\n" + numberAt(10, 20) + textAt("Hi", right-1, 20)
	if result := compileFiles(test, theme, nil); len(result.Records) != 3 {
		test.Errorf("text sharing a column: %d records", len(result.Records))
	}
}

func TestTheErrorsOfATextAreTheSameOnBothPaths(test *testing.T) {
	for _, before := range []string{"", numberAt(10, 20)} {
		got := compileError(test, solidTheme+"widgets:\n"+before+textAt("\"漢\"", 10, 20), nil)
		if got != `widget `+fmt.Sprint(1+strings.Count(before, "\n"))+` (text): the font body has no glyph for "漢"` {
			test.Errorf("glyph: got %q", got)
		}
		got = compileError(test, solidTheme+"widgets:\n"+before+textAt("Hi", 470, 20), nil)
		if !strings.Contains(got, "(text): the box at 470,20 of ") || !strings.HasSuffix(got, "is not within the screen of 480 x 320") {
			test.Errorf("box: got %q", got)
		}
	}
}

func TestTheSlotsWithTextsBetweenTheOtherWidgets(test *testing.T) {
	alarm := "  - {type: number, font: body, metric: gpu.temp, digits: 4, x: 1, y: 2, alarm: {above: 80, color: \"#ff3030\"}}\n"
	theme := solidTheme + "widgets:\n" + textAt("a", 300, 100) + alarm + textAt("b", 2, 3) + textAt("c", 300, 200) +
		"  - {type: bar, metric: cpu.util, x: 100, y: 100, width: 10, height: 4, color: \"#00ff00\", background: \"#ff0000\"}\n"
	result := compileFiles(test, theme, nil)
	entries := parsedRecords(test, result)
	if len(entries) != 5 {
		test.Fatalf("%d entries", len(entries))
	}
	glyphs := newGlyphFont(regularFace(test, 20))
	set := glyphs.height * (10*glyphs.digitWidth + glyphs.advance('.') + 4*glyphs.digitWidth)
	text := entries[3].Record.(themefile.Text)
	bar := entries[4].Record.(themefile.Bar)
	if _, ok := entries[1].Record.(themefile.Number); !ok || entries[1].Record.(themefile.Number).Index != 1 ||
		entries[2].Record.(themefile.Number).Index != 2 || text.Index != 3 || bar.Index != 4 {
		test.Errorf("indexes: %v", entries)
	}
	if text.Offset != uint32(widgetOffset+2*set) || len(result.File) != widgetOffset+2*set+len(text.Resource) {
		test.Errorf("text at %d, file of %d bytes", text.Offset, len(result.File))
	}
	if len(result.AlarmSlots) != 1 || result.AlarmSlots[0] != 3 {
		test.Errorf("alarm slots %v", result.AlarmSlots)
	}
}

// expectedPlane draws the texts over a plane as the rule of R2 says, written apart from the preview: each pixel expanded,
// each mask blended over it, the result rounded.
func expectedPlane(plane []byte, texts []themefile.Text) []byte {
	screen := image.NewRGBA(image.Rect(0, 0, 480, 320))
	for index := range 480 * 320 {
		colour := themefile.Colour(uint16(plane[2*index]) | uint16(plane[2*index+1])<<8)
		screen.Set(index%480, index/480, themeimage.Expand(colour))
	}
	for _, text := range texts {
		for _, picture := range themeimage.Resources(text) {
			target := image.Rect(int(text.X), int(text.Y), int(text.X)+picture.Bounds().Dx(), int(text.Y)+picture.Bounds().Dy())
			draw.Draw(screen, target, picture, image.Point{}, draw.Over)
		}
	}
	var drawn []byte
	for y := range 320 {
		for x := range 480 {
			pixel := screen.RGBAAt(x, y)
			drawn = appendPixel(drawn, rgb565(pixel.R, pixel.G, pixel.B))
		}
	}
	return drawn
}

// textFor builds the record of a text as a record, as the compiler would write it, for the expectation.
func textFor(test *testing.T, text string, x, y int, colour themefile.Colour) themefile.Text {
	test.Helper()
	glyphs := newGlyphFont(regularFace(test, 20))
	built, err := buildText(widgetInput{spec: widgetSpec{text: text, x: int64(x), y: int64(y)},
		font: loadedFont{name: "body", glyphs: glyphs, colour: colour}}, "widget 1")
	if err != nil {
		test.Fatal(err)
	}
	return built.record.(themefile.Text)
}

func backgroundOf(test *testing.T, result Result) themefile.Background {
	test.Helper()
	return parsedRecords(test, result)[0].Record.(themefile.Background)
}

func TestTheBackgroundWithATextOnASolidColour(test *testing.T) {
	result := compileFiles(test, "name: t\nbackground: \"#102030\"\n"+goodFonts+"widgets:\n"+textAt("Hi", 10, 20), nil)
	solid, white := rgb565(0x10, 0x20, 0x30), rgb565(255, 255, 255)
	want := expectedPlane(solidPlane(solid), []themefile.Text{textFor(test, "Hi", 10, 20, white)})
	background := backgroundOf(test, result)
	if string(background.Resource) != string(want) {
		test.Error("the plane differs from the rule")
	}
	if background.FillColourGuess != solid || background.Offset != 0x1000 || background.Bytes18To63[0] != 1 {
		test.Errorf("the background record: %+v", background)
	}
	if string(background.Resource) == string(solidPlane(solid)) {
		test.Error("the text left no pixel")
	}
}

func variedPicture() *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, 480, 320))
	for y := range 320 {
		for x := range 480 {
			picture.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: uint8(x + y), A: uint8(100 + (x*y)%156)})
		}
	}
	return picture
}

func TestTheBackgroundWithATextOnAPNG(test *testing.T) {
	picture := variedPicture()
	files := map[string][]byte{"b.png": encodePNG(test, picture)}
	result := compileFiles(test, "name: t\nbackground: b.png\n"+goodFonts+"widgets:\n"+textAt("Hi", 10, 20), files)
	white := rgb565(255, 255, 255)
	want := expectedPlane(backgroundPlane(picture), []themefile.Text{textFor(test, "Hi", 10, 20, white)})
	if string(backgroundOf(test, result).Resource) != string(want) {
		test.Error("the plane differs from the rule")
	}
}

func TestTwoOverlappingTextsAreDrawnInTheOrderOfTheList(test *testing.T) {
	theme := "name: t\nbackground: \"#102030\"\nfonts:\n  body: {builtin: go-regular, size: 20, color: \"#ffffff\"}\n  red: {builtin: go-regular, size: 20, color: \"#ff0000\"}\n"
	for _, order := range [][2]string{{"body", "red"}, {"red", "body"}} {
		widgets := "widgets:\n"
		for position, font := range order {
			widgets += fmt.Sprintf("  - {type: text, font: %s, text: Hi, x: %d, y: 20}\n", font, 10+3*position)
		}
		result := compileFiles(test, theme+widgets, nil)
		colours := map[string]themefile.Colour{"body": rgb565(255, 255, 255), "red": rgb565(255, 0, 0)}
		want := expectedPlane(solidPlane(rgb565(0x10, 0x20, 0x30)), []themefile.Text{
			textFor(test, "Hi", 10, 20, colours[order[0]]), textFor(test, "Hi", 13, 20, colours[order[1]])})
		if len(result.Records) != 1 || string(backgroundOf(test, result).Resource) != string(want) {
			test.Errorf("order %v: %d records or a plane that differs", order, len(result.Records))
		}
	}
}

func TestTheBackgroundOfAThemeWithNoTextIsAsBefore(test *testing.T) {
	result := compileFiles(test, numberTheme(""), nil)
	if string(backgroundOf(test, result).Resource) != string(solidPlane(rgb565(255, 255, 255))) {
		test.Error("the plane is not the solid colour")
	}
	if got := [64]byte(result.File[64:128]); got != solidBackgroundSlot() {
		test.Errorf("slot 1: %x", got)
	}
}

func absDifference(first, second uint8) int {
	if first > second {
		return int(first - second)
	}
	return int(second - first)
}

// TestThePreviewMovesOnlyWhereATextIsAntiAliased is R6: the same theme with its texts drawn into the background and
// with each of them written as a record, drawn by the preview.
func TestThePreviewMovesOnlyWhereATextIsAntiAliased(test *testing.T) {
	picture := variedPicture()
	opaque := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	draw.Draw(opaque, opaque.Bounds(), image.NewUniform(color.NRGBA{R: 9, G: 99, B: 199, A: 255}), image.Point{}, draw.Src)
	files := map[string][]byte{"b.png": encodePNG(test, picture), "i.png": encodePNG(test, opaque)}
	theme := "name: t\nbackground: b.png\n" + goodFonts + "widgets:\n" + textAt("Hi", 10, 20) + textAt("Ho", 13, 22) +
		numberAt(200, 100) + textAt("Hi", 200, 100) + "  - {type: image, file: i.png, x: 12, y: 20}\n"
	result := compileFiles(test, theme, files)
	after := themeimage.Preview(themefile.Theme{Entries: parsedRecords(test, result)})

	white := rgb565(255, 255, 255)
	first, second := textFor(test, "Hi", 10, 20, white), textFor(test, "Ho", 13, 22, white)
	entries := []themefile.Entry{{Slot: 1, Record: backgroundOf(test, compileFiles(test, "name: t\nbackground: b.png\n", files))},
		{Slot: 2, Record: first}, {Slot: 3, Record: second}}
	for _, entry := range parsedRecords(test, result)[1:] {
		entry.Slot += 2
		entries = append(entries, entry)
	}
	before := themeimage.Preview(themefile.Theme{Entries: entries})

	soft := map[image.Point]bool{}
	for _, text := range []themefile.Text{first, second} {
		for position, alpha := range text.Resource {
			if alpha > 0 && alpha < 255 {
				soft[image.Pt(int(text.X)+position%int(text.Width), int(text.Y)+position/int(text.Width))] = true
			}
		}
	}
	moved := 0
	for y := range 320 {
		for x := range 480 {
			was, now := before.RGBAAt(x, y), after.RGBAAt(x, y)
			if was == now {
				continue
			}
			moved++
			if !soft[image.Pt(x, y)] || absDifference(was.R, now.R) > 4 || absDifference(was.G, now.G) > 2 || absDifference(was.B, now.B) > 4 {
				test.Fatalf("pixel %d,%d is %v, was %v", x, y, now, was)
			}
		}
	}
	if moved == 0 {
		test.Error("no pixel moved, so the test saw no anti-aliased pixel")
	}
}

func TestThePreviewOfAThemeWithNoTextIsTheSame(test *testing.T) {
	files := map[string][]byte{"b.png": encodePNG(test, variedPicture())}
	result := compileFiles(test, "name: t\nbackground: b.png\n"+goodFonts+"widgets:\n"+numberAt(200, 100), files)
	entries := parsedRecords(test, result)
	if string(entries[0].Record.(themefile.Background).Resource) != string(backgroundPlane(variedPicture())) {
		test.Error("the background differs from today")
	}
}
