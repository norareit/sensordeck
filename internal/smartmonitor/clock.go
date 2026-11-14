package smartmonitor

import (
	"fmt"
	"time"
)

// EncodeClock gives the report that sets the clock, from the fields of moment in the location it carries.
func EncodeClock(moment time.Time) (Report, error) {
	year := moment.Year()
	if year < 2000 || year > 2255 {
		return Report{}, fmt.Errorf("clock year %d is outside 2000 to 2255", year)
	}
	weekday := int(moment.Weekday())
	if moment.Weekday() == time.Sunday {
		weekday = 7
	}
	var report Report
	copy(report[:], []byte{
		0x03, 0x01, 0x15,
		byte(year - 2000), byte(moment.Month()), byte(moment.Day()),
		byte(moment.Hour()), byte(moment.Minute()), byte(moment.Second()),
		byte(weekday), 0x64,
	})
	return report, nil
}
