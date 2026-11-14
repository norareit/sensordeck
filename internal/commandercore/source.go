//go:build linux

package commandercore

import (
	"context"
	"os"
	"time"

	"github.com/norareit/sensordeck/internal/hidraw"
	"github.com/norareit/sensordeck/internal/sensors"
)

// ReplyTimeLimit is how long a reading waits for each answer of the controller. It is about 15 times the slowest
// answer seen, and a failed sequence ends after at most two time limits, the failed step and the return to hardware
// mode.
const ReplyTimeLimit = time.Second

// Source reads the probes of the controller, opening its node anew for every reading, so that nothing is carried from
// one reading to the next and a replug under another hidraw number needs nothing.
type Source struct {
	SysfsRoot, DevRoot string
	TimeLimit          time.Duration
}

// Read gives the wanted probes that the controller reports. A missing controller, a node that cannot be opened and
// every failure of the sequence make them unavailable, which is no error.
func (source Source) Read(stop context.Context, wanted func(name string) bool) []sensors.Reading {
	if !wantsAProbe(wanted) || stop.Err() != nil {
		return nil
	}
	node, found := findNode(source.SysfsRoot, source.DevRoot)
	if !found {
		return nil
	}
	file, err := hidraw.OpenNode(node)
	if err != nil {
		return nil
	}
	defer file.Close()
	return readingsOf(runSequence(stop, file, source.TimeLimit), wanted)
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
func runSequence(stop context.Context, file *os.File, timeLimit time.Duration) []probe {
	probes := runSteps(stop, file, timeLimit)
	exchange(file, steps[returnToHardwareMode], timeLimit)
	return probes
}

// runSteps runs steps 1 to 7 and gives the probes of a valid step 6, or none. It ends at the first failure, and when
// stop is done before a step after the first. A read in progress is not ended by stop, it ends at the time limit.
func runSteps(stop context.Context, file *os.File, timeLimit time.Duration) []probe {
	var probes []probe
	for position, command := range steps[:returnToHardwareMode] {
		if position != enterSoftwareMode && stop.Err() != nil {
			return probes
		}
		answer, ok := exchange(file, command, timeLimit)
		if !ok {
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

// exchange writes the command as one write and reads one answer within the time limit. It gives false when the write
// failed or was short, no answer came, or the answer is not valid.
func exchange(file *os.File, command []byte, timeLimit time.Duration) ([]byte, bool) {
	written, err := file.Write(encode(command))
	if err != nil || written < writeSize {
		return nil, false
	}
	if err := file.SetReadDeadline(time.Now().Add(timeLimit)); err != nil {
		return nil, false
	}
	answer := make([]byte, answerSize+1)
	count, err := file.Read(answer)
	if err != nil {
		return nil, false
	}
	answer = answer[:count]
	return answer, valid(command, answer)
}
