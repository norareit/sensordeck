package smartmonitor

// ResetCommand gives the report that reboots the device.
func ResetCommand() Report {
	var report Report
	copy(report[:], "\x01reset")
	return report
}

// YmodemCommand gives the report that makes the device ready to receive a theme.
func YmodemCommand() Report {
	var report Report
	copy(report[:], "ymodem")
	return report
}
