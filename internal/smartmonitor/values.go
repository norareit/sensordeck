package smartmonitor

const maxPairsPerReport = 20

// Value sets the numeric tag of a widget of the theme.
type Value struct {
	Tag   uint8
	Value uint16
}

// EncodeValues gives the value update reports for the values, at most 20 pairs in each, in the order of the list.
func EncodeValues(values []Value) []Report {
	var reports []Report
	for start := 0; start < len(values); start += maxPairsPerReport {
		end := min(start+maxPairsPerReport, len(values))
		reports = append(reports, encodeValueReport(values[start:end]))
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
		report[offset+1] = byte(value.Value >> 8)
		report[offset+2] = byte(value.Value)
	}
	return report
}
