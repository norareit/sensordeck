package themecompile

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/config"
)

func TestTagsAreSortedAndShared(test *testing.T) {
	got := assignTags([]string{"gpu.temp", "cpu.temp", "cpu.temp"})
	want := []config.Mapping{{Name: "cpu.temp", Tag: 1}, {Name: "gpu.temp", Tag: 2}}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %+v", got)
	}
}

func TestTagsSkipTheClock(test *testing.T) {
	var names []string
	for number := 1; number <= 22; number++ {
		names = append(names, fmt.Sprintf("nvme.%d.temp", number))
	}
	got := assignTags(names)
	// Sorted by bytes: 1, 10 to 19, 2, 20 to 22, 3 to 9.
	if len(got) != 22 {
		test.Fatalf("%d tags", len(got))
	}
	for position, mapping := range got {
		want := uint8(position + 1)
		if position >= 20 {
			want++
		}
		if mapping.Tag != want || mapping.Tag == 0 || mapping.Tag == 21 {
			test.Errorf("%s: tag %d, want %d", mapping.Name, mapping.Tag, want)
		}
	}
	if got[0].Name != "nvme.1.temp" || got[1].Name != "nvme.10.temp" {
		test.Errorf("not in byte order: %s, %s", got[0].Name, got[1].Name)
	}
}

func TestTagsFile(test *testing.T) {
	tags := []config.Mapping{{Name: "cpu.temp", Tag: 1}, {Name: "gpu.temp", Tag: 2}}
	text := TagsFile(tags)
	want := "# Paste this block into the config file of sensordeck run.\ntags:\n  cpu.temp: 1\n  gpu.temp: 2\n"
	if string(text) != want {
		test.Errorf("got %q", text)
	}
	parsed, err := config.Parse(text)
	if err != nil || !reflect.DeepEqual(parsed.Mappings, tags) {
		test.Errorf("config read back %+v, %v", parsed.Mappings, err)
	}
	if got := string(TagsFile(nil)); got != "# Paste this block into the config file of sensordeck run.\ntags: {}\n" {
		test.Errorf("no tags: %q", got)
	}
}

func TestABarGetsTheTagOfItsMetric(test *testing.T) {
	shared := compileFiles(test, solidTheme+"widgets:\n"+
		"  - {type: bar, metric: cpu.util, x: 0, y: 0, width: 5, height: 5, color: \"#00ff00\"}\n"+
		"  - {type: number, font: body, metric: cpu.util, x: 0, y: 10}\n", nil)
	if want := []config.Mapping{{Name: "cpu.util", Tag: 1}}; !reflect.DeepEqual(shared.Tags, want) {
		test.Errorf("tags %+v", shared.Tags)
	}
	if shared.File[2*64+2] != 1 || shared.File[3*64+2] != 1 {
		test.Errorf("tags of the records %d, %d", shared.File[2*64+2], shared.File[3*64+2])
	}
	alone := compileFiles(test, solidTheme+"widgets:\n  - {type: bar, metric: gpu.util, x: 0, y: 0, width: 5, height: 5, color: \"#00ff00\"}\n", nil)
	if want := []config.Mapping{{Name: "gpu.util", Tag: 1}}; !reflect.DeepEqual(alone.Tags, want) || alone.File[2*64+2] != 1 {
		test.Errorf("tags %+v, record tag %d", alone.Tags, alone.File[2*64+2])
	}
	if !strings.Contains(string(TagsFile(alone.Tags)), "  gpu.util: 1\n") {
		test.Errorf("tags file %q", TagsFile(alone.Tags))
	}
}
