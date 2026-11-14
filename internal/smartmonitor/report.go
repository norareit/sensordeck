package smartmonitor

// Report is one HID report of the device, without the report ID byte that a transport adds.
type Report [64]byte

// toReports is the only place that decides how a block, the end of transmission and the closing block are carried
// in reports: as consecutive pieces of 64 bytes, the last one filled up with zeros.
func toReports(data []byte) []Report {
	count := (len(data) + len(Report{}) - 1) / len(Report{})
	reports := make([]Report, count)
	for position := range reports {
		copy(reports[position][:], data[position*len(Report{}):])
	}
	return reports
}
