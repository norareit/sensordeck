package cli

import (
	"bytes"
	"errors"
	"net/url"
	"testing"
)

func monitorAddress(test *testing.T, text string) *url.URL {
	test.Helper()
	address, err := url.Parse(text)
	if err != nil {
		test.Fatal(err)
	}
	return address
}

func TestTheFailureLineNamesTheURLWithoutItsUserAndPassword(test *testing.T) {
	address := monitorAddress(test, "http://monitor:s3cret@localhost:8085/data.json?a=b")
	got := monitorFailureLine(address, errors.New("status 401 Unauthorized"))
	want := "LibreHardwareMonitor cannot be read at http://localhost:8085/data.json?a=b: status 401 Unauthorized. " +
		"The temperatures and cpu.clock need it running, with Options > Remote Web Server > Run checked"
	if got != want {
		test.Errorf("got %q", got)
	}
	if address.User == nil {
		test.Error("the URL of the caller lost its user information")
	}
}

func TestTheLineOfADocumentWithNoMetricAndTheLineOfReturn(test *testing.T) {
	address := monitorAddress(test, "http://u:p@host:9000/x")
	if got, want := monitorEmptyLine(address), "LibreHardwareMonitor at http://host:9000/x gives no temperature and no clock that sensordeck knows"; got != want {
		test.Errorf("got %q", got)
	}
	if got, want := monitorReturnLine(address), "LibreHardwareMonitor read again at http://host:9000/x"; got != want {
		test.Errorf("got %q", got)
	}
}

func TestAnOutageWritesItsLineOnceAndTheLineOfReturnAfterIt(test *testing.T) {
	var stderr bytes.Buffer
	address := monitorAddress(test, "http://localhost:8085/data.json")
	outage := &monitorOutage{stderr: &stderr, address: address}
	outage.observe(nil, nil) // a success with no failure before writes nothing
	if stderr.Len() != 0 {
		test.Fatalf("stderr %q", stderr.String())
	}
	failure := errors.New("status 500 Internal Server Error")
	for range 3 {
		outage.observe(nil, failure)
	}
	want := monitorFailureLine(address, failure) + "\n"
	if stderr.String() != want {
		test.Fatalf("stderr %q", stderr.String())
	}
	outage.observe(nil, nil)
	outage.observe(nil, nil)
	want += monitorReturnLine(address) + "\n"
	if stderr.String() != want {
		test.Errorf("stderr %q", stderr.String())
	}
	outage.observe(nil, failure)
	if want += monitorFailureLine(address, failure) + "\n"; stderr.String() != want {
		test.Errorf("a second outage: stderr %q", stderr.String())
	}
}
