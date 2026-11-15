package themecompile

import (
	"fmt"
	"reflect"
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
