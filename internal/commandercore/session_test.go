//go:build linux

package commandercore

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/fakecommander"
)

type sessionHarness struct {
	test       *testing.T
	fake       *fakecommander.Fake
	controller *fakecommander.Controller
	lighting   Lighting
	stop       context.Context
	cancel     context.CancelFunc
}

func newSessionHarness(test *testing.T) *sessionHarness {
	test.Helper()
	root := test.TempDir()
	sysfsRoot, devRoot := filepath.Join(root, "sys"), filepath.Join(root, "dev")
	stop, cancel := context.WithCancel(context.Background())
	test.Cleanup(cancel)
	fake := fakecommander.New(test, sysfsRoot, devRoot)
	return &sessionHarness{
		test: test, fake: fake, controller: fake.Add(9, 10), stop: stop, cancel: cancel,
		lighting: Lighting{Finder: Hidraw(sysfsRoot, devRoot), TimeLimit: testTimeLimit, Red: 0xFF},
	}
}

// writes gives what the controller received as the commands, without the report ID byte and the zeros after the
// command, for every write.
func (harness *sessionHarness) writes() [][]byte {
	written := harness.controller.Writes()
	commands := make([][]byte, len(written))
	for position, write := range written {
		commands[position] = write[1:]
	}
	return commands
}

// commandsOf gives each command as the 96 data bytes of a write.
func commandsOf(commands ...[]byte) [][]byte {
	padded := make([][]byte, len(commands))
	for position, command := range commands {
		padded[position] = append(bytes.Clone(command), make([]byte, answerSize-len(command))...)
	}
	return padded
}

func readOf(channel, endpoint byte) [][]byte {
	return [][]byte{
		closeCommand(channel, endpoint), openCommand(channel, endpoint), readCommand(channel, endpoint),
		closeCommand(channel, endpoint),
	}
}

func join(parts ...[][]byte) [][]byte {
	var all [][]byte
	for _, part := range parts {
		all = append(all, part...)
	}
	return all
}

// setupWrites are the exchanges of a setup up to and including the one at the position of the list, 1 to 7.
func setupWrites(count int, through int) [][]byte {
	all := join(
		[][]byte{enterSoftware}, readOf(channelOne, portsEndpoint),
		[][]byte{openCommand(channelZero, ledEndpoint), colourCommand(count, 0xFF, 0, 0)},
	)
	return commandsOf(all[:through]...)
}

var teardownWithClose = commandsOf(closeCommand(channelZero, ledEndpoint), returnHardware)
var teardownWithoutClose = commandsOf(returnHardware)

func readingWrites() [][]byte {
	return commandsOf(join(readOf(channelOne, 0x1A), readOf(channelOne, 0x17), readOf(channelOne, 0x21))...)
}

func TestASessionMakesTheWritesOfTheSetupTheReadingAndTheTeardown(test *testing.T) {
	harness := newSessionHarness(test)
	session, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	if want := setupWrites(16, 7); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("setup: got %d writes, want %d", len(harness.writes()), len(want))
	}
	if lighting, colour := harness.controller.Lighting(); lighting != fakecommander.SentColour || colour != [3]byte{0xFF, 0, 0} {
		test.Fatalf("lighting %d, colour %v", lighting, colour)
	}
	readings, err := session.Read(harness.stop)
	if err != nil || !reflect.DeepEqual(readings, bothProbes) {
		test.Fatalf("got %v, %v", readings, err)
	}
	session.End()
	want := join(setupWrites(16, 7), readingWrites(), teardownWithClose)
	if got := harness.writes(); !reflect.DeepEqual(got, want) || len(got) != 7+12+2 {
		test.Fatalf("got %d writes, want %d", len(got), len(want))
	}
	if harness.controller.Mode() != fakecommander.HardwareMode || harness.controller.OpenEndpoint(0) != 0 {
		test.Fatal("the controller is not in hardware mode with channel 0 closed")
	}
	if got := harness.controller.WritesOnInterfaceOne(); len(got) != 0 {
		test.Fatalf("interface 1 got %d writes", len(got))
	}
}

func TestThreePortsOfEightLEDsGiveTwentyFourTriples(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.SetPorts(fakecommander.Port{Mode: 2, LEDs: 8}, fakecommander.Port{Mode: 2, LEDs: 8}, fakecommander.Port{Mode: 2, LEDs: 8})
	session, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	session.End()
	writes := harness.controller.ColourWrites()
	if len(writes) != 1 || !bytes.Equal(writes[0], colourCommand(24, 0xFF, 0, 0)) {
		test.Fatalf("got % x", writes)
	}
}

func TestAnLedCountOfZeroOrThirtyFailsTheSetupWithATeardown(test *testing.T) {
	for _, testCase := range []struct {
		name string
		leds uint16
		want string
	}{
		{"zero", 0, "read of endpoint 0x20 on channel 1: LED count 0 is outside 1 to 29"},
		{"thirty", 30, "read of endpoint 0x20 on channel 1: LED count 30 is outside 1 to 29"},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			harness := newSessionHarness(test)
			harness.controller.SetPorts(fakecommander.Port{Mode: 2, LEDs: testCase.leds})
			session, err := harness.lighting.Start(harness.stop)
			if session != nil || err == nil || err.Error() != testCase.want {
				test.Fatalf("got %v", err)
			}
			if want := join(setupWrites(0, 4), teardownWithClose); !reflect.DeepEqual(harness.writes(), want) {
				test.Fatalf("got %d writes", len(harness.writes()))
			}
		})
	}
}

func TestEachStepOfTheSetupThatFailsEndsItWithTheTeardownAndTheClose(test *testing.T) {
	for _, testCase := range []struct {
		name    string
		prefix  []byte
		through int
		want    string
	}{
		{"enter", enterSoftware, 1, "enter software mode: status 01"},
		{"read of 0x20", readCommand(channelOne, portsEndpoint), 4, "read of endpoint 0x20 on channel 1: status 01"},
		{"open of 0x22", openCommand(channelZero, ledEndpoint), 6, "open of endpoint 0x22 on channel 0: status 01"},
		{"colour write", []byte{0x08, 0x06, 0x00}, 7, "colour write: status 01"},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			harness := newSessionHarness(test)
			harness.controller.FailCommand(testCase.prefix, fakecommander.BadStatus)
			session, err := harness.lighting.Start(harness.stop)
			if session != nil || err == nil || err.Error() != testCase.want {
				test.Fatalf("got %v", err)
			}
			if want := join(setupWrites(16, testCase.through), teardownWithClose); !reflect.DeepEqual(harness.writes(), want) {
				test.Fatalf("got %d writes, want %d", len(harness.writes()), len(want))
			}
		})
	}
}

func TestAFailedReadingEndsWithTheTeardownAndTheClose(test *testing.T) {
	harness := newSessionHarness(test)
	session, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	harness.controller.FailCommand(readCommand(channelOne, 0x21), fakecommander.BadStatus)
	readings, err := session.Read(harness.stop)
	if readings != nil || err == nil || err.Error() != "read of endpoint 0x21 on channel 1: status 01" {
		test.Fatalf("got %v, %v", readings, err)
	}
	want := join(setupWrites(16, 7), commandsOf(join(readOf(channelOne, 0x1A), readOf(channelOne, 0x17),
		readOf(channelOne, 0x21)[:3])...), teardownWithClose)
	if got := harness.writes(); !reflect.DeepEqual(got, want) {
		test.Fatalf("got %d writes, want %d", len(got), len(want))
	}
}

func TestAnAnswerThatIsNotTheTemperaturesFailsTheReading(test *testing.T) {
	harness := newSessionHarness(test)
	session, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	harness.controller.Fail(6, fakecommander.WrongDataType)
	harness.controller.FailCommand(readCommand(channelOne, 0x21), fakecommander.WrongDataType)
	_, err = session.Read(harness.stop)
	if err == nil || err.Error() != "read of endpoint 0x21 on channel 1: not an answer of the temperatures" {
		test.Fatalf("got %v", err)
	}
	want := join(setupWrites(16, 7), commandsOf(join(readOf(channelOne, 0x1A), readOf(channelOne, 0x17),
		readOf(channelOne, 0x21)[:3])...), teardownWithClose)
	if got := harness.writes(); !reflect.DeepEqual(got, want) {
		test.Fatalf("got %d writes, want %d", len(got), len(want))
	}
}

func TestAWrongDataTypeOfThePortsFailsTheSetupBeforeTheLastClose(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.FailCommand(readCommand(channelOne, portsEndpoint), fakecommander.WrongDataType)
	_, err := harness.lighting.Start(harness.stop)
	if err == nil || err.Error() != "read of endpoint 0x20 on channel 1: data type 06 00" {
		test.Fatalf("got %v", err)
	}
	if want := join(setupWrites(0, 4), teardownWithClose); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("got %d writes", len(harness.writes()))
	}
}

func TestNoAnswerEndsWithATeardownWithoutTheClose(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.FailCommand(openCommand(channelZero, ledEndpoint), fakecommander.NoAnswer)
	if _, err := harness.lighting.Start(harness.stop); err == nil || err.Error() != "open of endpoint 0x22 on channel 0: no answer within 50ms" {
		test.Fatalf("got %v", err)
	}
	if want := join(setupWrites(16, 6), teardownWithoutClose); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("setup: got %d writes", len(harness.writes()))
	}

	harness = newSessionHarness(test)
	session, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	harness.controller.FailCommand(readCommand(channelOne, 0x21), fakecommander.NoAnswer)
	if _, err := session.Read(harness.stop); err == nil {
		test.Fatal("the reading succeeded")
	}
	got := harness.writes()
	if tail := got[len(got)-1:]; !reflect.DeepEqual(tail, teardownWithoutClose) || reflect.DeepEqual(got[len(got)-2], teardownWithClose[0]) {
		test.Fatalf("got %d writes", len(got))
	}
}

func TestALedEndpointLeftOpenFailsOneSetupAndTheNextSucceeds(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.SetOpenEndpoint(channelZero, 0x21)
	_, err := harness.lighting.Start(harness.stop)
	if err == nil || err.Error() != "open of endpoint 0x22 on channel 0: status 03" {
		test.Fatalf("got %v", err)
	}
	if harness.controller.OpenEndpoint(channelZero) != 0 {
		test.Fatal("the teardown did not close channel 0")
	}
	session, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	session.End()
}

func TestAStopDuringASetupEndsItWithTheTeardownAndNoColourWrite(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.OnCommand(openCommand(channelZero, ledEndpoint), harness.cancel)
	if _, err := harness.lighting.Start(harness.stop); !errors.Is(err, errStopped) {
		test.Fatalf("got %v", err)
	}
	if want := join(setupWrites(16, 6), teardownWithClose); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("got %d writes", len(harness.writes()))
	}
	if len(harness.controller.ColourWrites()) != 0 {
		test.Fatal("a colour was written")
	}
}

func TestAStopDuringAReadingEndsItWithTheTeardown(test *testing.T) {
	harness := newSessionHarness(test)
	session, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	harness.controller.OnCommand(readCommand(channelOne, 0x17), harness.cancel)
	if _, err := session.Read(harness.stop); !errors.Is(err, errStopped) {
		test.Fatalf("got %v", err)
	}
	want := join(setupWrites(16, 7), commandsOf(join(readOf(channelOne, 0x1A), readOf(channelOne, 0x17)[:3])...), teardownWithClose)
	if got := harness.writes(); !reflect.DeepEqual(got, want) {
		test.Fatalf("got %d writes, want %d", len(got), len(want))
	}
}

func TestAStopBeforeTheSetupWritesNothing(test *testing.T) {
	harness := newSessionHarness(test)
	harness.cancel()
	if _, err := harness.lighting.Start(harness.stop); !errors.Is(err, errStopped) {
		test.Fatalf("got %v", err)
	}
	if got := harness.controller.Writes(); len(got) != 0 {
		test.Fatalf("got %d writes", len(got))
	}
}

func TestNoControllerGivesErrNoControllerAndWritesNothing(test *testing.T) {
	root := test.TempDir()
	lighting := Lighting{Finder: Hidraw(filepath.Join(root, "sys"), filepath.Join(root, "dev")), TimeLimit: testTimeLimit}
	if _, err := lighting.Start(context.Background()); !errors.Is(err, ErrNoController) {
		test.Fatalf("got %v", err)
	}
}

func TestNothingIsWrittenToInterfaceOne(test *testing.T) {
	harness := newSessionHarness(test)
	session, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	if _, err := session.Read(harness.stop); err != nil {
		test.Fatal(err)
	}
	session.End()
	if got := harness.controller.WritesOnInterfaceOne(); len(got) != 0 {
		test.Fatalf("interface 1 got %d writes", len(got))
	}
}
