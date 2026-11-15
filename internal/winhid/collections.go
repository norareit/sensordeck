package winhid

import (
	"fmt"
	"slices"
	"strings"

	"github.com/norareit/sensordeck/internal/screenlist"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

// screenReportLength is the length of both reports of the screen in the caps: the report ID byte and the report of
// the protocol.
const screenReportLength uint16 = 1 + uint16(len(smartmonitor.Report{}))

// Collection is one device interface of the HID class, as read with no access.
type Collection struct {
	Path            string
	Vendor, Product uint16
	Name, Serial    string
	Caps            Caps
}

// Line is the collection as the command devices writes a screen.
func (collection Collection) Line() string {
	return screenlist.Line(collection.Path, collection.Vendor, collection.Product, collection.Name, collection.Serial)
}

// List gives the present collections ordered by path without regard to case. It opens each path with no access, which
// sends nothing to any device, and leaves out a path that cannot be opened or whose attributes or caps cannot be
// read. A string that cannot be read is empty.
func List(seam Seam) ([]Collection, error) {
	paths, err := seam.Paths()
	if err != nil {
		return nil, fmt.Errorf("cannot list the HID devices: %w", err)
	}
	var collections []Collection
	for _, path := range paths {
		if collection, readable := query(seam, path); readable {
			collections = append(collections, collection)
		}
	}
	slices.SortStableFunc(collections, func(left, right Collection) int {
		return strings.Compare(strings.ToLower(left.Path), strings.ToLower(right.Path))
	})
	return collections, nil
}

func query(seam Seam, path string) (Collection, bool) {
	handle, err := seam.OpenQuery(path)
	if err != nil {
		return Collection{}, false
	}
	defer func() { _ = seam.Close(handle) }()
	vendor, product, err := seam.Attributes(handle)
	if err != nil {
		return Collection{}, false
	}
	caps, err := seam.Caps(handle)
	if err != nil {
		return Collection{}, false
	}
	name, err := seam.Product(handle)
	if err != nil {
		name = ""
	}
	serial, err := seam.Serial(handle)
	if err != nil {
		serial = ""
	}
	return Collection{Path: path, Vendor: vendor, Product: product, Name: name, Serial: serial, Caps: caps}, true
}

// Screens gives the supported screens among the collections, which are those of the screen's IDs with reports of the
// length of the screen. When there is none it gives the near misses as well, the collections of the IDs with other
// lengths, so that the human sees what was found. Beside a screen they are no news, since a second collection of one
// device is normal on Windows.
func Screens(collections []Collection) (screens, nearMisses []Collection) {
	for _, collection := range collections {
		if collection.Vendor != smartmonitor.VendorID || collection.Product != smartmonitor.ProductID {
			continue
		}
		if collection.Caps.InputLength == screenReportLength && collection.Caps.OutputLength == screenReportLength {
			screens = append(screens, collection)
		} else {
			nearMisses = append(nearMisses, collection)
		}
	}
	if len(screens) > 0 {
		nearMisses = nil
	}
	return screens, nearMisses
}

// NearMissLines gives one line for each near miss, in the order given.
func NearMissLines(nearMisses []Collection) []string {
	lines := make([]string, len(nearMisses))
	for position, collection := range nearMisses {
		lines[position] = fmt.Sprintf("found %s with reports of %d and %d bytes, those of the screen are %d",
			collection.Path, collection.Caps.InputLength, collection.Caps.OutputLength, screenReportLength)
	}
	return lines
}
