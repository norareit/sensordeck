//go:build linux

package commandercore

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/norareit/sensordeck/internal/devicelock"
	"github.com/norareit/sensordeck/internal/fakecommander"
	"github.com/norareit/sensordeck/internal/hidraw"
)

type sessionHarness struct {
	test       *testing.T
	fake       *fakecommander.Fake
	controller *fakecommander.Controller
	lighting   Lighting
	stop       context.Context
	cancel     context.CancelFunc
	devRoot    string
}

func newSessionHarness(test *testing.T) *sessionHarness {
	test.Helper()
	root := test.TempDir()
	sysfsRoot, devRoot := filepath.Join(root, "sys"), filepath.Join(root, "dev")
	stop, cancel := context.WithCancel(context.Background())
	test.Cleanup(cancel)
	fake := fakecommander.New(test, sysfsRoot, devRoot)
	return &sessionHarness{
		test: test, fake: fake, controller: fake.Add(9, 10), stop: stop, cancel: cancel, devRoot: devRoot,
		lighting: Lighting{Finder: Hidraw(sysfsRoot, devRoot, "/nonexistent"), TimeLimit: testTimeLimit, Red: 0xFF},
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

// setupWrites are the exchanges of a setup with the configuration already stored, up to and including the one at the
// position of the list, 1 to 11: enter, the read of 0x20 (2 to 5), the read of 0x1E (6 to 9), the open of 0x22 and the
// colour write.
func setupWrites(count int, through int) [][]byte {
	all := join(
		[][]byte{enterSoftware}, readOf(channelOne, portsEndpoint), readOf(channelOne, portConfigurationEndpoint),
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
	session, _, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	if want := setupWrites(16, 11); !reflect.DeepEqual(harness.writes(), want) {
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
	want := join(setupWrites(16, 11), readingWrites(), teardownWithClose)
	if got := harness.writes(); !reflect.DeepEqual(got, want) || len(got) != 11+12+2 {
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
	session, _, err := harness.lighting.Start(harness.stop)
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
			session, _, err := harness.lighting.Start(harness.stop)
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
		{"open of 0x22", openCommand(channelZero, ledEndpoint), 10, "open of endpoint 0x22 on channel 0: status 01"},
		{"colour write", []byte{0x08, 0x06, 0x00}, 11, "colour write: status 01"},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			harness := newSessionHarness(test)
			harness.controller.FailCommand(testCase.prefix, fakecommander.BadStatus)
			session, _, err := harness.lighting.Start(harness.stop)
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
	session, _, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	harness.controller.FailCommand(readCommand(channelOne, 0x21), fakecommander.BadStatus)
	readings, err := session.Read(harness.stop)
	if readings != nil || err == nil || err.Error() != "read of endpoint 0x21 on channel 1: status 01" {
		test.Fatalf("got %v, %v", readings, err)
	}
	want := join(setupWrites(16, 11), commandsOf(join(readOf(channelOne, 0x1A), readOf(channelOne, 0x17),
		readOf(channelOne, 0x21)[:3])...), teardownWithClose)
	if got := harness.writes(); !reflect.DeepEqual(got, want) {
		test.Fatalf("got %d writes, want %d", len(got), len(want))
	}
}

func TestAnAnswerThatIsNotTheTemperaturesFailsTheReading(test *testing.T) {
	harness := newSessionHarness(test)
	session, _, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	harness.controller.Fail(6, fakecommander.WrongDataType)
	harness.controller.FailCommand(readCommand(channelOne, 0x21), fakecommander.WrongDataType)
	_, err = session.Read(harness.stop)
	if err == nil || err.Error() != "read of endpoint 0x21 on channel 1: not an answer of the temperatures" {
		test.Fatalf("got %v", err)
	}
	want := join(setupWrites(16, 11), commandsOf(join(readOf(channelOne, 0x1A), readOf(channelOne, 0x17),
		readOf(channelOne, 0x21)[:3])...), teardownWithClose)
	if got := harness.writes(); !reflect.DeepEqual(got, want) {
		test.Fatalf("got %d writes, want %d", len(got), len(want))
	}
}

func TestAWrongDataTypeOfThePortsFailsTheSetupBeforeTheLastClose(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.FailCommand(readCommand(channelOne, portsEndpoint), fakecommander.WrongDataType)
	_, _, err := harness.lighting.Start(harness.stop)
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
	if _, _, err := harness.lighting.Start(harness.stop); err == nil || err.Error() != "open of endpoint 0x22 on channel 0: no answer within 50ms" {
		test.Fatalf("got %v", err)
	}
	if want := join(setupWrites(16, 10), teardownWithoutClose); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("setup: got %d writes", len(harness.writes()))
	}

	harness = newSessionHarness(test)
	session, _, err := harness.lighting.Start(harness.stop)
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
	_, _, err := harness.lighting.Start(harness.stop)
	if err == nil || err.Error() != "open of endpoint 0x22 on channel 0: status 03" {
		test.Fatalf("got %v", err)
	}
	if harness.controller.OpenEndpoint(channelZero) != 0 {
		test.Fatal("the teardown did not close channel 0")
	}
	session, _, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	session.End()
}

func TestAStopDuringASetupEndsItWithTheTeardownAndNoColourWrite(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.OnCommand(openCommand(channelZero, ledEndpoint), harness.cancel)
	if _, _, err := harness.lighting.Start(harness.stop); !errors.Is(err, errStopped) {
		test.Fatalf("got %v", err)
	}
	if want := join(setupWrites(16, 10), teardownWithClose); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("got %d writes", len(harness.writes()))
	}
	if len(harness.controller.ColourWrites()) != 0 {
		test.Fatal("a colour was written")
	}
}

func TestAStopDuringAReadingEndsItWithTheTeardown(test *testing.T) {
	harness := newSessionHarness(test)
	session, _, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	harness.controller.OnCommand(readCommand(channelOne, 0x17), harness.cancel)
	if _, err := session.Read(harness.stop); !errors.Is(err, errStopped) {
		test.Fatalf("got %v", err)
	}
	want := join(setupWrites(16, 11), commandsOf(join(readOf(channelOne, 0x1A), readOf(channelOne, 0x17)[:3])...), teardownWithClose)
	if got := harness.writes(); !reflect.DeepEqual(got, want) {
		test.Fatalf("got %d writes, want %d", len(got), len(want))
	}
}

func TestAStopBeforeTheSetupWritesNothing(test *testing.T) {
	harness := newSessionHarness(test)
	harness.cancel()
	if _, _, err := harness.lighting.Start(harness.stop); !errors.Is(err, errStopped) {
		test.Fatalf("got %v", err)
	}
	if got := harness.controller.Writes(); len(got) != 0 {
		test.Fatalf("got %d writes", len(got))
	}
}

func TestNoControllerGivesErrNoControllerAndWritesNothing(test *testing.T) {
	root := test.TempDir()
	lighting := Lighting{Finder: Hidraw(filepath.Join(root, "sys"), filepath.Join(root, "dev"), "/nonexistent"), TimeLimit: testTimeLimit}
	if _, _, err := lighting.Start(context.Background()); !errors.Is(err, ErrNoController) {
		test.Fatalf("got %v", err)
	}
}

func TestNothingIsWrittenToInterfaceOne(test *testing.T) {
	harness := newSessionHarness(test)
	session, _, err := harness.lighting.Start(harness.stop)
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

var (
	hostConfiguration = []byte{0x0D, 0x00, 0x07, 0x00, 0x00, 0x00, 0x01, 0x03, 0x01, 0x03, 0x01, 0x03, 0x01, 0x03}
	icueConfiguration = []byte{0x0D, 0x00, 0x07, 0x06, 0x06, 0x06, 0x06, 0x06, 0x06, 0x06, 0x00, 0x00, 0x01, 0x03, 0x01, 0x03,
		0x01, 0x03, 0x01, 0x03}
)

// configurationWrites are the five exchanges that write the wanted configuration of the host: the close, the open, the
// write and the close of 0x1E on channel 1, and 08 15 01.
func configurationWrites() [][]byte {
	return commandsOf(
		closeCommand(channelOne, portConfigurationEndpoint), openCommand(channelOne, portConfigurationEndpoint),
		configurationCommand(hostConfiguration), closeCommand(channelOne, portConfigurationEndpoint), afterPortConfiguration,
	)
}

// repairedWrites are the writes of a setup that finds the configuration of iCUE stored, up to the colour write.
func repairedWrites() [][]byte {
	return join(setupWrites(16, 9), configurationWrites(), setupWrites(16, 11)[9:])
}

func TestASetupThatFindsTheWantedConfigurationStoredWritesNothingToIt(test *testing.T) {
	harness := newSessionHarness(test)
	session, configuration, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	session.End()
	if want := join(setupWrites(16, 11), teardownWithClose); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("got %d writes, want %d", len(harness.writes()), len(want))
	}
	if configuration != (PortConfiguration{}) {
		test.Fatalf("got %+v", configuration)
	}
}

func TestASetupThatFindsTheConfigurationOfICUEWritesTheWantedOneBeforeTheColour(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.SetPortConfiguration(icueConfiguration)
	session, configuration, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	if lighting, colour := harness.controller.Lighting(); lighting != fakecommander.SentColour || colour != [3]byte{0xFF, 0, 0} {
		test.Fatalf("lighting %d, colour %v", lighting, colour)
	}
	session.End()
	if want := join(repairedWrites(), teardownWithClose); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("got %d writes, want %d", len(harness.writes()), len(want))
	}
	if got := harness.controller.PortConfiguration(); !bytes.Equal(got, hostConfiguration) {
		test.Fatalf("stored % x", got)
	}
	if configuration != (PortConfiguration{Written: true}) {
		test.Fatalf("got %+v", configuration)
	}
}

func TestAConfigurationThatDiffersOnlyInItsLastByteIsWritten(test *testing.T) {
	harness := newSessionHarness(test)
	last := bytes.Clone(hostConfiguration)
	last[len(last)-1] = 0x02
	harness.controller.SetPortConfiguration(last)
	session, configuration, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	session.End()
	if want := join(repairedWrites(), teardownWithClose); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("got %d writes, want %d", len(harness.writes()), len(want))
	}
	if !configuration.Written {
		test.Fatalf("got %+v", configuration)
	}
}

func TestAPortOfEightLEDsLeavesTheConfigurationAsStored(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.SetPortConfiguration(icueConfiguration)
	harness.controller.SetPorts(fakecommander.Port{}, fakecommander.Port{Mode: 3}, fakecommander.Port{Mode: 3},
		fakecommander.Port{Mode: 2, LEDs: 8}, fakecommander.Port{Mode: 2, LEDs: 4}, fakecommander.Port{Mode: 2, LEDs: 4},
		fakecommander.Port{Mode: 2, LEDs: 4})
	session, configuration, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	session.End()
	want := join(
		[][]byte{enterSoftware}, readOf(channelOne, portsEndpoint),
		[][]byte{openCommand(channelZero, ledEndpoint), colourCommand(20, 0xFF, 0, 0)},
	)
	if want = join(commandsOf(want...), teardownWithClose); !reflect.DeepEqual(harness.writes(), want) {
		test.Fatalf("got %d writes, want %d", len(harness.writes()), len(want))
	}
	if got := harness.controller.PortConfiguration(); !bytes.Equal(got, icueConfiguration) {
		test.Fatalf("stored % x", got)
	}
	if configuration != (PortConfiguration{Unknown: "port 3", LEDs: 8}) {
		test.Fatalf("got %+v", configuration)
	}
}

func TestAFailedExchangeOfTheConfigurationEndsTheSetup(test *testing.T) {
	for _, testCase := range []struct {
		name    string
		prefix  []byte
		through [][]byte
		written bool
		failure string
	}{
		{"read of 0x1E", readCommand(channelOne, portConfigurationEndpoint), setupWrites(16, 8), false,
			"read of endpoint 0x1e on channel 1"},
		{"configuration write", []byte{0x08, 0x06, 0x01}, join(setupWrites(16, 9), configurationWrites()[:3]), false,
			"write of the LED port configuration"},
		{"08 15 01", afterPortConfiguration, join(setupWrites(16, 9), configurationWrites()), true, "command 08 15 01"},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			for _, failure := range []struct {
				name     string
				kind     fakecommander.Failure
				teardown [][]byte
				ending   string
			}{
				{"status 03", fakecommander.UnavailableStatus, teardownWithClose, ": status 03"},
				{"no answer", fakecommander.NoAnswer, teardownWithoutClose, ": no answer within 50ms"},
			} {
				test.Run(failure.name, func(test *testing.T) {
					harness := newSessionHarness(test)
					harness.controller.SetPortConfiguration(icueConfiguration)
					harness.controller.FailCommand(testCase.prefix, failure.kind)
					session, configuration, err := harness.lighting.Start(harness.stop)
					if session != nil || err == nil || err.Error() != testCase.failure+failure.ending {
						test.Fatalf("got %v", err)
					}
					if want := join(testCase.through, failure.teardown); !reflect.DeepEqual(harness.writes(), want) {
						test.Fatalf("got %d writes, want %d", len(harness.writes()), len(want))
					}
					if configuration.Written != testCase.written {
						test.Fatalf("got %+v", configuration)
					}
				})
			}
		})
	}
}

func TestEverySetupOfASessionThatIsSetUpAgainReadsTheConfigurationAndReadingsWriteNothingToIt(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.SetPortConfiguration(icueConfiguration)
	first, configuration, err := harness.lighting.Start(harness.stop)
	if err != nil || !configuration.Written {
		test.Fatalf("first: %+v, %v", configuration, err)
	}
	if _, err := first.Read(harness.stop); err != nil {
		test.Fatal(err)
	}
	first.End()
	second, configuration, err := harness.lighting.Start(harness.stop)
	if err != nil || configuration.Written {
		test.Fatalf("second: %+v, %v", configuration, err)
	}
	second.End()
	want := join(repairedWrites(), readingWrites(), teardownWithClose, setupWrites(16, 11), teardownWithClose)
	if got := harness.writes(); !reflect.DeepEqual(got, want) {
		test.Fatalf("got %d writes, want %d", len(got), len(want))
	}
}

func TestASetupOfAControllerHeldByAnotherProcessFailsWithNoWrite(test *testing.T) {
	harness := newSessionHarness(test)
	file := holdNode(test, harness.devRoot)
	_, _, err := harness.lighting.Start(harness.stop)
	var inUse *devicelock.InUseError
	if !errors.As(err, &inUse) || err.Error() != "the controller is in use by another sensordeck process" {
		test.Fatalf("got %v", err)
	}
	if writes := harness.controller.Writes(); len(writes) != 0 {
		test.Fatalf("got %d writes", len(writes))
	}
	_ = file.Close()
	session, _, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	other, err := hidraw.OpenNode(filepath.Join(harness.devRoot, "hidraw9"))
	if err != nil {
		test.Fatal(err)
	}
	defer other.Close()
	if err := devicelock.LockNode(other, "/nonexistent"); !errors.As(err, &inUse) {
		test.Fatalf("while the session is held: got %v", err)
	}
	session.End()
	if err := devicelock.LockNode(other, "/nonexistent"); err != nil {
		test.Fatalf("after End: %v", err)
	}
}

// lockIsFree tells whether another sensordeck process could take the lock of the controller now.
func lockIsFree(test *testing.T, devRoot string) bool {
	test.Helper()
	other, err := hidraw.OpenNode(filepath.Join(devRoot, "hidraw9"))
	if err != nil {
		test.Fatal(err)
	}
	defer other.Close()
	return devicelock.LockNode(other, "/nonexistent") == nil
}

func TestTheLockEndsWithTheTeardownOfAFailedReadingAndOfAFailedSetup(test *testing.T) {
	harness := newSessionHarness(test)
	harness.controller.FailCommand(openCommand(channelZero, ledEndpoint), fakecommander.BadStatus)
	if _, _, err := harness.lighting.Start(harness.stop); err == nil {
		test.Fatal("the setup did not fail")
	}
	if !lockIsFree(test, harness.devRoot) {
		test.Fatal("the lock is held after a setup that failed")
	}

	harness = newSessionHarness(test)
	session, _, err := harness.lighting.Start(harness.stop)
	if err != nil {
		test.Fatal(err)
	}
	if lockIsFree(test, harness.devRoot) {
		test.Fatal("the lock is free while the session is held")
	}
	harness.controller.FailCommand(readCommand(channelOne, 0x21), fakecommander.BadStatus)
	if _, err := session.Read(harness.stop); err == nil {
		test.Fatal("the reading did not fail")
	}
	if !lockIsFree(test, harness.devRoot) {
		test.Fatal("the lock is held after a keepalive that failed")
	}
}
