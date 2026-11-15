//go:build linux

package commandercore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/norareit/sensordeck/internal/hidraw"
	"github.com/norareit/sensordeck/internal/sensors"
)

// ErrNoController is the setup that found no node of the controller.
var ErrNoController = errors.New("no controller found")

// errStopped is a setup or a reading that ended because the stop was done before its next exchange.
var errStopped = errors.New("stopped")

// Lighting makes the sessions that light the LEDs of the controller in one colour.
type Lighting struct {
	SysfsRoot, DevRoot string
	TimeLimit          time.Duration
	Red, Green, Blue   byte
}

// Session is a held session: the node open, the controller in software mode, the LEDs lit.
type Session struct {
	file      *os.File
	timeLimit time.Duration
}

// Start makes a setup and gives the session it starts. On every error the setup has ended, with its teardown once the
// first command was written, and the node is closed.
func (lighting Lighting) Start(stop context.Context) (*Session, error) {
	node, found := findNode(lighting.SysfsRoot, lighting.DevRoot)
	if !found {
		return nil, ErrNoController
	}
	file, err := hidraw.OpenNode(node)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", node, err)
	}
	if stop.Err() != nil {
		_ = file.Close()
		return nil, errStopped
	}
	session := &Session{file: file, timeLimit: lighting.TimeLimit}
	if err := session.setUp(stop, lighting); err != nil {
		session.teardown(closesChannelZero(err))
		return nil, err
	}
	return session, nil
}

// setUp makes the exchanges of a setup in order and stops at the first that does not succeed.
func (session *Session) setUp(stop context.Context, lighting Lighting) error {
	// The stop was checked before this, and a stop after it must still write the teardown, since the first command
	// is written whatever it says.
	if _, err := session.send(enterSoftware); err != nil {
		return err
	}
	var count int
	err := session.readEndpoint(stop, channelOne, portsEndpoint, func(answer []byte) (err error) {
		count, err = ledCount(answer)
		return err
	})
	if err != nil {
		return err
	}
	if _, err := session.exchange(stop, openCommand(channelZero, ledEndpoint)); err != nil {
		return err
	}
	_, err = session.exchange(stop, colourCommand(count, lighting.Red, lighting.Green, lighting.Blue))
	return err
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
	answer, err := exchange(session.file, command, session.timeLimit)
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
		_, _ = exchange(session.file, closeCommand(channelZero, ledEndpoint), session.timeLimit)
	}
	_, _ = exchange(session.file, returnHardware, session.timeLimit)
	_ = session.file.Close()
}
