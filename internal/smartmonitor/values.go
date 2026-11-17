package smartmonitor

const maxPairsPerReport = 20

// Value sets the numeric tag of a widget of the theme.
type Value struct {
	Tag uint8
	// Value is sent as a 16-bit two's complement number, which is how the screen reads it (docs/protocol.md).
	Value int16
}

// EncodeValues gives the value update reports for the values, at most 20 pairs in each, in the order of the list.
func EncodeValues(values []Value) []Report {
	units := make([][]Value, len(values))
	for position := range values {
		units[position] = values[position : position+1]
	}
	return EncodeUnits(units)
}

// EncodeUnits gives the value update reports for the units, in their order. The values of a unit are always in one
// report, so that the screen never shows half of them: a unit goes into the current report while it has room for all of
// it, and otherwise starts a new report, and the room left in the one before stays empty.
func EncodeUnits(units [][]Value) []Report {
	var reports []Report
	var current []Value
	for _, unit := range units {
		if len(current) > 0 && len(current)+len(unit) > maxPairsPerReport {
			reports = append(reports, encodeValueReport(current))
			current = nil
		}
		current = append(current, unit...)
	}
	if len(current) > 0 {
		reports = append(reports, encodeValueReport(current))
	}
	return reports
}

func encodeValueReport(values []Value) Report {
	var report Report
	report[0] = 0x02
	report[1] = byte(len(values))
	for position, value := range values {
		offset := 2 + position*3
		report[offset] = value.Tag
		bits := uint16(value.Value)
		report[offset+1] = byte(bits >> 8)
		report[offset+2] = byte(bits)
	}
	return report
}
