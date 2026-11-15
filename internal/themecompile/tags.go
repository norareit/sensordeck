package themecompile

import (
	"slices"
	"strconv"
	"strings"

	"github.com/norareit/sensordeck/internal/config"
)

// clockTag is the tag that the device fills with the clock, so that no metric gets it.
const clockTag = 21

// assignTags gives the distinct metric names, sorted by their bytes, the tags 1, 2, 3 and up, skipping the tag of the
// clock. Sorting makes the tags independent of the order of the widgets.
func assignTags(metrics []string) []config.Mapping {
	names := slices.Clone(metrics)
	slices.Sort(names)
	names = slices.Compact(names)
	mappings := make([]config.Mapping, 0, len(names))
	tag := uint8(1)
	for _, name := range names {
		if tag == clockTag {
			tag++
		}
		mappings = append(mappings, config.Mapping{Name: name, Tag: tag})
		tag++
	}
	return mappings
}

// TagsFile gives the content of the tags file: a block in the shape of the tags section of the config file of run.
func TagsFile(tags []config.Mapping) []byte {
	var text strings.Builder
	text.WriteString("# Paste this block into the config file of sensordeck run.\n")
	if len(tags) == 0 {
		text.WriteString("tags: {}\n")
		return []byte(text.String())
	}
	text.WriteString("tags:\n")
	for _, mapping := range tags {
		text.WriteString("  " + mapping.Name + ": " + strconv.Itoa(int(mapping.Tag)) + "\n")
	}
	return []byte(text.String())
}
