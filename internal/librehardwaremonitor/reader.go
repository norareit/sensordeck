package librehardwaremonitor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

// TimeLimit bounds a whole reading. The monitor runs on the same machine and builds its document from values it holds,
// so it answers in a few milliseconds, and a hung server delays a round by at most this much.
const TimeLimit = 500 * time.Millisecond

// Reader reads the document of the monitor with one GET, and keeps its connection from one reading to the next.
type Reader struct {
	address   *url.URL
	timeLimit time.Duration
	client    *http.Client
}

// NewReader makes a reader of the URL, with the user information of the URL sent as HTTP Basic authentication. The
// time limit counts from the connection to the end of the body.
func NewReader(address *url.URL, timeLimit time.Duration) *Reader {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// The monitor is on the machine or on its network, and a proxy of the environment would not reach it.
	transport.Proxy = nil
	return &Reader{address: address, timeLimit: timeLimit, client: &http.Client{Transport: transport}}
}

// Read makes one request and gives the readings of the document. It ends when the time limit passes or stop is done.
func (reader *Reader) Read(stop context.Context) ([]sensors.Reading, error) {
	limited, cancel := context.WithTimeout(stop, reader.timeLimit)
	defer cancel()
	request, err := http.NewRequestWithContext(limited, http.MethodGet, reader.address.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := reader.client.Do(request)
	if err != nil {
		return nil, withoutRequest(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return Map(body)
}

// withoutRequest drops the method and the URL that the error of the client names, because the line that reports it
// names the URL without its user information.
func withoutRequest(err error) error {
	var failed *url.Error
	if errors.As(err, &failed) {
		return failed.Err
	}
	return err
}

// Wants tells whether any monitor metric is wanted, by asking for the names a machine can have, up to MaxNumber.
func Wants(wanted func(name string) bool) bool {
	if wanted(sensors.CPUTemperature) || wanted(sensors.CPUClock) {
		return true
	}
	for number := 1; number <= MaxNumber; number++ {
		if wanted(sensors.CCDTemperature(number)) || wanted(sensors.RAMTemperature(number)) || wanted(sensors.NVMeTemperature(number)) {
			return true
		}
	}
	return false
}

// Source reads the monitor metrics, with one request for all of them. Report is told the outcome of each request: every
// reading of the document, or the error, so that the commands can say what the user must do.
type Source struct {
	Reader *Reader
	Report func(found []sensors.Reading, err error)
}

func (source Source) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !Wants(wanted) {
		return nil
	}
	found, err := source.Reader.Read(stop)
	source.Report(found, err)
	var readings []sensors.Reading
	for _, reading := range found {
		if wanted(reading.Name) {
			readings = append(readings, reading)
		}
	}
	return readings
}
