package commandercore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

// ErrNoController is the setup that found no node of the controller.
var ErrNoController = errors.New("no controller found")

// errStopped is a setup or a reading that ended because the stop was done before its next exchange.
var errStopped = errors.New("stopped")

// Lighting makes the sessions that light the LEDs of the controller in one colour.
type Lighting struct {
	Finder           Finder
	TimeLimit        time.Duration
	Red, Green, Blue byte
}

// PortConfiguration is what a setup did with the LED port configuration of the endpoint 0x1E.
type PortConfiguration struct {
	Written bool   // the configuration write succeeded, whatever the rest of the setup gave
	Unknown string // the name of the first entry with LEDs whose count is not 4, empty when the configuration is known
	LEDs    int    // the LED count of that entry
}

// Session is a held session: the node open, the controller in software mode, the LEDs lit.
type Session struct {
	opened    node
	timeLimit time.Duration
}

// Start makes a setup and gives the session it starts, and what the setup did with the LED port configuration, on every
// error too. On every error the setup has ended, with its teardown once the first command was written, and the node
// is closed.
func (lighting Lighting) Start(stop context.Context) (*Session, PortConfiguration, error) {
	path, err := lighting.Finder.find()
	if err != nil {
		return nil, PortConfiguration{}, err
	}
	opened, err := lighting.Finder.open(path)
	if err != nil {
		return nil, PortConfiguration{}, err
	}
	if stop.Err() != nil {
		_ = opened.close()
		return nil, PortConfiguration{}, errStopped
	}
	session := &Session{opened: opened, timeLimit: lighting.TimeLimit}
	configuration, err := session.setUp(stop, lighting)
	if err != nil {
		session.teardown(closesChannelZero(err))
		return nil, configuration, err
	}
	return session, configuration, nil
}

// setUp makes the exchanges of a setup in order and stops at the first that does not succeed.
func (session *Session) setUp(stop context.Context, lighting Lighting) (PortConfiguration, error) {
	// The stop was checked before this, and a stop after it must still write the teardown, since the first command
	// is written whatever it says.
	if _, err := session.send(enterSoftware); err != nil {
		return PortConfiguration{}, err
	}
	var count int
	var ports []byte
	err := session.readEndpoint(stop, channelOne, portsEndpoint, func(answer []byte) (err error) {
		ports = answer
		count, err = ledCount(answer)
		return err
	})
	if err != nil {
		return PortConfiguration{}, err
	}
	configuration, err := session.repairConfiguration(stop, ports)
	if err != nil {
		return configuration, err
	}
	if _, err := session.exchange(stop, openCommand(channelZero, ledEndpoint)); err != nil {
		return configuration, err
	}
	_, err = session.exchange(stop, colourCommand(count, lighting.Red, lighting.Green, lighting.Blue))
	return configuration, err
}

// repairConfiguration reads the LED port configuration and writes the wanted one when the stored one differs. It
// leaves the configuration alone when the answer of the ports holds a port with LEDs that the type is not known for.
func (session *Session) repairConfiguration(stop context.Context, ports []byte) (PortConfiguration, error) {
	if name, leds := portTypeProblem(ports); name != "" {
		return PortConfiguration{Unknown: name, LEDs: leds}, nil
	}
	var stored []byte
	err := session.readEndpoint(stop, channelOne, portConfigurationEndpoint, func(answer []byte) error {
		stored = answer
		return nil
	})
	if err != nil {
		return PortConfiguration{}, err
	}
	wanted := wantedConfiguration(ports)
	// The stored bytes are compared where a write would put them, from byte 3, and are not parsed: the content that
	// iCUE leaves does not have the layout of the one that is written.
	if bytes.Equal(stored[3:3+len(wanted)], wanted) {
		return PortConfiguration{}, nil
	}
	written, err := session.writeConfiguration(stop, wanted)
	return PortConfiguration{Written: written}, err
}

// writeConfiguration makes the exchanges that store the configuration, and tells whether the write itself succeeded.
func (session *Session) writeConfiguration(stop context.Context, configuration []byte) (written bool, err error) {
	for _, command := range [][]byte{
		closeCommand(channelOne, portConfigurationEndpoint), openCommand(channelOne, portConfigurationEndpoint),
	} {
		if _, err := session.exchange(stop, command); err != nil {
			return false, err
		}
	}
	if _, err := session.exchange(stop, configurationCommand(configuration)); err != nil {
		return false, err
	}
	for _, command := range [][]byte{closeCommand(channelOne, portConfigurationEndpoint), afterPortConfiguration} {
		if _, err := session.exchange(stop, command); err != nil {
			return true, err
		}
	}
	return true, nil
}

// Read makes one reading and gives every available probe. On every error the session has ended with its teardown and
// the node is closed.
func (session *Session) Read(stop context.Context) ([]sensors.Reading, error) {
	var probes []probe
	for _, endpoint := range []byte{0x1A, 0x17, 0x21} {
		var check func(answer []byte) error
		if endpoint == 0x21 {
			check = func(answer []byte) error {
				var valid bool
				if probes, valid = temperatures(answer); !valid {
					return errors.New("not an answer of the temperatures")
				}
				return nil
			}
		}
		if err := session.readEndpoint(stop, channelOne, endpoint, check); err != nil {
			session.teardown(closesChannelZero(err))
			return nil, err
		}
	}
	return readingsOf(probes, func(string) bool { return true }), nil
}

// End makes the teardown with the close of channel 0 and closes the node.
func (session *Session) End() { session.teardown(true) }

// readEndpoint makes the four exchanges of a read of the endpoint on the channel. The check of the answer of the read,
// when there is one, is part of that exchange, so a failed check ends the read before the last close.
func (session *Session) readEndpoint(stop context.Context, channel, endpoint byte, check func(answer []byte) error) error {
	for _, command := range [][]byte{
		closeCommand(channel, endpoint), openCommand(channel, endpoint), readCommand(channel, endpoint),
		closeCommand(channel, endpoint),
	} {
		answer, err := session.exchange(stop, command)
		if err == nil && check != nil && command[1] == 0x08 {
			err = check(answer)
			if err != nil {
				err = fmt.Errorf("%s: %w", stepName(command), err)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// exchange makes one exchange after a check of the stop, and names the step in the error it gives.
func (session *Session) exchange(stop context.Context, command []byte) ([]byte, error) {
	if stop.Err() != nil {
		return nil, errStopped
	}
	return session.send(command)
}

func (session *Session) send(command []byte) ([]byte, error) {
	answer, err := exchange(session.opened, command, session.timeLimit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", stepName(command), err)
	}
	return answer, nil
}

// closesChannelZero tells whether the teardown after the error closes channel 0. It does not after a write that failed
// or an answer that did not come, so that a controller that stopped answering costs a stop at most two time limits.
func closesChannelZero(err error) bool {
	var failed *exchangeError
	return !errors.As(err, &failed) || failed.answered
}

// teardown closes channel 0 when asked to, returns the controller to hardware mode, which is never left out, and
// closes the node. The results of the two exchanges are not used.
func (session *Session) teardown(closeChannel bool) {
	if closeChannel {
		_, _ = exchange(session.opened, closeCommand(channelZero, ledEndpoint), session.timeLimit)
	}
	_, _ = exchange(session.opened, returnHardware, session.timeLimit)
	_ = session.opened.close()
}
