package winhid

import "strings"

const interfaceZeroField = "mi_00"

// InterfaceZero tells whether the path is a collection of USB interface 0 of a composite device: the part between its
// first and its second # holds, among its fields separated by &, one that is mi_00 in either case. Whole fields are
// compared, so that mi_000 and xmi_00 do not match.
func InterfaceZero(path string) bool {
	_, rest, found := strings.Cut(path, "#")
	if !found {
		return false
	}
	instance, _, found := strings.Cut(rest, "#")
	if !found {
		return false
	}
	for _, field := range strings.Split(instance, "&") {
		if strings.EqualFold(field, interfaceZeroField) {
			return true
		}
	}
	return false
}
