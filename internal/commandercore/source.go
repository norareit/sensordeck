package commandercore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/norareit/sensordeck/internal/devicelock"
	"github.com/norareit/sensordeck/internal/sensors"
)

// ReplyTimeLimit is how long a reading waits for each answer of the controller. It is about 15 times the slowest
// answer seen, and a failed sequence ends after at most two time limits, the failed step and the return to hardware
// mode.
const ReplyTimeLimit = time.Second

// Source reads the probes of the controller, opening its node anew for every reading, so that nothing is carried from
// one reading to the next and a replug under another hidraw number needs nothing.
type Source struct {
	Finder    Finder
	TimeLimit time.Duration
	// Refused, when it is not nil, is called with the error of an opening that another sensordeck process holds.
	Refused func(err error)
}

// Read gives the wanted probes that the controller reports. A missing controller, a node that cannot be opened and
// every failure of the sequence make them unavailable, which is no error.
func (source Source) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wantsAProbe(wanted) || stop.Err() != nil {
		return nil
	}
	path, err := source.Finder.find()
	if err != nil {
		return nil
	}
	opened, err := source.Finder.open(path)
	if err != nil {
		source.report(err)
		return nil
	}
	defer opened.close()
	return readingsOf(runSequence(stop, opened, source.TimeLimit), wanted)
}

func (source Source) report(err error) {
	var inUse *devicelock.InUseError
	if source.Refused != nil && errors.As(err, &inUse) {
		source.Refused(err)
	}
}

// wantsAProbe asks for every name that an answer can hold. A name beyond maxProbes can never be given, so none that is
// wanted is missed.
func wantsAProbe(wanted func(name string) bool) bool {
	for number := 1; number <= maxProbes; number++ {
		if wanted(sensors.CommanderTemperature(number)) {
			return true
		}
	}
	return false
}

func readingsOf(probes []probe, wanted func(name string) bool) []sensors.Reading {
	var readings []sensors.Reading
	for position, reading := range probes {
		name := sensors.CommanderTemperature(position + 1)
		if reading.available && wanted(name) {
			readings = append(readings, sensors.Reading{Name: name, Value: reading.celsius})
		}
	}
	return readings
}

// runSequence runs the steps and then returns the controller to hardware mode. Once step 1 is written step 8 is owed
// whatever happens, because what the controller does in software mode with nobody sending it settings is not known.
// A failure of step 8 changes nothing of the probes the steps gave.
func runSequence(stop context.Context, opened node, timeLimit time.Duration) []probe {
	// Discovery and opening come before this point, and a stop can arrive during them. Nothing is written then, because
	// step 8 is owed only once step 1 has been attempted.
	if stop.Err() != nil {
		return nil
	}
	probes := runSteps(stop, opened, timeLimit)
	_, _ = exchange(opened, steps[returnToHardwareMode], timeLimit)
	return probes
}

// runSteps runs steps 1 to 7 and gives the probes of a valid step 6, or none. It ends at the first failure, and when
// stop is done before a step after the first. A read in progress is not ended by stop, it ends at the time limit.
func runSteps(stop context.Context, opened node, timeLimit time.Duration) []probe {
	var probes []probe
	for position, command := range steps[:returnToHardwareMode] {
		if position != enterSoftwareMode && stop.Err() != nil {
			return probes
		}
		answer, err := exchange(opened, command, timeLimit)
		if err != nil {
			return probes
		}
		if position == readTemperatures {
			parsed, ok := temperatures(answer)
			if !ok {
				return nil
			}
			probes = parsed
		}
	}
	return probes
}

// exchangeError is a failed exchange. answered tells whether an answer came, which decides whether a teardown closes
// channel 0.
type exchangeError struct {
	answered bool
	problem  string
}

func (failure *exchangeError) Error() string { return failure.problem }

// exchange writes the command as one write and reads one answer within the time limit. It gives the first 96 bytes of
// a valid answer, and an exchangeError that says what failed when the write failed or was short, no answer came, or
// the answer is not valid.
func exchange(opened node, command []byte, timeLimit time.Duration) ([]byte, error) {
	if err := opened.send(command); err != nil {
		return nil, &exchangeError{problem: err.Error()}
	}
	answer, err := opened.receive(timeLimit)
	if errors.Is(err, errNoAnswer) {
		return nil, &exchangeError{problem: fmt.Sprintf("no answer within %s", timeLimit)}
	}
	if err != nil {
		return nil, &exchangeError{problem: err.Error()}
	}
	if problem := invalidity(command, answer); problem != "" {
		return nil, &exchangeError{answered: true, problem: problem}
	}
	return answer[:answerSize], nil
}
