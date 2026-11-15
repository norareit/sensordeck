package cli

import (
	"fmt"
	"io"
	"net/url"

	"github.com/norareit/sensordeck/internal/sensors"
)

// withoutUser gives the URL as text without its user information, so that a password does not reach a log.
func withoutUser(address *url.URL) string {
	shown := *address
	shown.User = nil
	return shown.String()
}

// monitorFailureLine says why the temperatures and the clock are missing, and what to check.
func monitorFailureLine(address *url.URL, err error) string {
	return fmt.Sprintf("LibreHardwareMonitor cannot be read at %s: %v. The temperatures and cpu.clock need it running, "+
		"with Options > Remote Web Server > Run checked", withoutUser(address), err)
}

// monitorEmptyLine is for a document that was read and holds no metric that sensordeck knows, which a change of names
// in a later version of LibreHardwareMonitor would cause.
func monitorEmptyLine(address *url.URL) string {
	return fmt.Sprintf("LibreHardwareMonitor at %s gives no temperature and no clock that sensordeck knows", withoutUser(address))
}

func monitorReturnLine(address *url.URL) string {
	return "LibreHardwareMonitor read again at " + withoutUser(address)
}

// monitorOutage writes a line for each outage of the monitor, and not for each reading that fails, so that a service
// whose monitor is not running does not fill its log. A line says when it is back.
type monitorOutage struct {
	stderr  io.Writer
	address *url.URL
	out     bool
}

// observe is told the outcome of each reading of the monitor.
func (outage *monitorOutage) observe(found []sensors.Reading, err error) {
	switch {
	case err != nil && !outage.out:
		outage.out = true
		_, _ = fmt.Fprintln(outage.stderr, monitorFailureLine(outage.address, err))
	case err == nil && outage.out:
		outage.out = false
		_, _ = fmt.Fprintln(outage.stderr, monitorReturnLine(outage.address))
	}
}
