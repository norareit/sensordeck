package commandercore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakehid"
	"github.com/norareit/sensordeck/internal/sensors"
	"github.com/norareit/sensordeck/internal/winhid"
)

const (
	zeroPath    = `\\?\hid#vid_1b1c&pid_0c2a&mi_00#7&1&0&0000#{guid}`
	zeroPathTwo = `\\?\hid#vid_1b1c&pid_0c2a&mi_00&col02#7&1&0&0000#{guid}`
	onePath     = `\\?\hid#vid_1b1c&pid_0c2a&mi_01#7&1&0&0000#{guid}`
)

var interfaceZeroCaps = winhid.Caps{UsagePage: 0xFF42, Usage: 0x0001, InputLength: 385, OutputLength: 385}

func controller(path string, caps winhid.Caps) fakehid.Collection {
	return fakehid.Collection{Path: path, Vendor: 0x1B1C, Product: 0x0C2A, Caps: caps}
}

// answering gives the collection the answers of the trace: the echo of the command, a status of zero, one available
// and one unavailable probe at the temperature endpoint, and one connected port of eight LEDs at the ports endpoint.
func answering(collection fakehid.Collection) fakehid.Collection {
	var endpoint byte
	collection.Respond = func(written []byte) []byte {
		command := written[1:]
		report := make([]byte, 385)
		report[2], report[3] = command[1], 0
		switch {
		case command[1] == 0x0D:
			endpoint = command[3]
		case command[1] == 0x08 && endpoint == 0x21:
			copy(report[4:], []byte{0x10, 0x00, 2, 0x00, 0xF4, 0x01, 0x01, 0x00, 0x00})
		case command[1] == 0x08 && endpoint == 0x20:
			copy(report[4:], []byte{0x0F, 0x00, 1, 2, 0, 8, 0})
		}
		return report
	}
	return collection
}

func findPath(test *testing.T, collections ...fakehid.Collection) (string, error) {
	test.Helper()
	return HID(fakehid.New(collections...)).find()
}

func TestFindGivesTheFirstCollectionOfInterfaceZeroInTheOrderOfThePaths(test *testing.T) {
	other := controller(`\\?\hid#vid_1b1c&pid_0c2b&mi_00#a#{guid}`, interfaceZeroCaps)
	other.Product = 0x0C2B
	fake := fakehid.New(controller(zeroPathTwo, interfaceZeroCaps), other, controller(onePath, interfaceZeroCaps),
		controller(zeroPath, interfaceZeroCaps))
	path, err := HID(fake).find()
	if err != nil || path != zeroPath {
		test.Fatalf("got %q, %v", path, err)
	}
	for _, opening := range fake.Openings() {
		if opening.ReadWrite {
			test.Errorf("%s was opened for reading and writing", opening.Path)
		}
	}
}

func TestFindGivesErrNoControllerForNoCollectionAndForOthers(test *testing.T) {
	for name, collections := range map[string][]fakehid.Collection{
		"none":               nil,
		"interface one":      {controller(onePath, interfaceZeroCaps)},
		"interface one near": {controller(onePath, winhid.Caps{UsagePage: 1, Usage: 1, InputLength: 9, OutputLength: 9})},
		"another device":     {{Path: zeroPath, Vendor: 1, Product: 2, Caps: interfaceZeroCaps}},
	} {
		if _, err := findPath(test, collections...); !errors.Is(err, ErrNoController) {
			test.Errorf("%s: got %v", name, err)
		}
	}
}

func TestANearMissIsNamedWithWhatItGave(test *testing.T) {
	near := controller(zeroPath, winhid.Caps{UsagePage: 0xFF00, Usage: 0x0002, InputLength: 65, OutputLength: 64})
	_, err := findPath(test, near)
	want := `no collection of interface 0 matches: ` + zeroPath +
		` has usage page 0xFF00, usage 0x0002 and reports of 65 and 64 bytes, ` +
		`interface 0 has usage page 0xFF42, usage 0x0001 and reports of 385 and 385 bytes`
	if err == nil || err.Error() != want || errors.Is(err, ErrNoController) {
		test.Fatalf("got %v", err)
	}
}

func TestSeveralNearMissesAreJoinedInTheOrderOfThePaths(test *testing.T) {
	second := controller(zeroPathTwo, winhid.Caps{UsagePage: 0x0001, Usage: 0x000A, InputLength: 385, OutputLength: 385})
	first := controller(zeroPath, winhid.Caps{UsagePage: 0xFF42, Usage: 0x0001, InputLength: 384, OutputLength: 385})
	_, err := findPath(test, second, first)
	if err == nil || !strings.Contains(err.Error(), zeroPath+" has usage page 0xFF42, usage 0x0001 and reports of 384 and 385 bytes; "+
		zeroPathTwo+" has usage page 0x0001, usage 0x000A and reports of 385 and 385 bytes, interface 0 has") {
		test.Fatalf("got %v", err)
	}
}

func TestANearMissBesideAMatchGivesNothing(test *testing.T) {
	near := controller(zeroPath, winhid.Caps{UsagePage: 0xFF00, Usage: 2, InputLength: 65, OutputLength: 64})
	path, err := findPath(test, near, controller(zeroPathTwo, interfaceZeroCaps))
	if err != nil || path != zeroPathTwo {
		test.Fatalf("got %q, %v", path, err)
	}
	if HID(fakehid.New(near, controller(zeroPathTwo, interfaceZeroCaps))).Problem() != nil {
		test.Error("a match gives no problem")
	}
}

func TestProblemIsTheErrorOfANearMissAndNilOtherwise(test *testing.T) {
	near := controller(zeroPath, winhid.Caps{UsagePage: 1, Usage: 1, InputLength: 385, OutputLength: 385})
	if err := HID(fakehid.New(near)).Problem(); err == nil || !strings.HasPrefix(err.Error(), "no collection of interface 0 matches: ") {
		test.Errorf("got %v", err)
	}
	if err := HID(fakehid.New()).Problem(); err != nil {
		test.Errorf("got %v", err)
	}
}

func openNodeOf(test *testing.T, fake *fakehid.Fake) node {
	test.Helper()
	finder := HID(fake)
	path, err := finder.find()
	if err != nil {
		test.Fatal(err)
	}
	opened, err := finder.open(path)
	if err != nil {
		test.Fatal(err)
	}
	return opened
}

func TestTheNodeWritesTheFullOutputReport(test *testing.T) {
	fake := fakehid.New(answering(controller(zeroPath, interfaceZeroCaps)))
	opened := openNodeOf(test, fake)
	command := []byte{0x08, 0x01, 0x03, 0x00, 0x02}
	if err := opened.send(command); err != nil {
		test.Fatal(err)
	}
	want := make([]byte, 385)
	copy(want[1:], command)
	if got := fake.Writes()[0]; !reflect.DeepEqual(got, want) {
		test.Errorf("got % x", got)
	}
	if waits := fake.Waits(); waits[0] != uint32(winhid.WriteTimeLimit/time.Millisecond) {
		test.Errorf("waits %v", waits)
	}
}

func TestTheNodeReadsTheFullInputReportAndDropsTheReportID(test *testing.T) {
	collection := controller(zeroPath, interfaceZeroCaps)
	collection.Input = [][]byte{append([]byte{0, 0x77}, make([]byte, 383)...)}
	fake := fakehid.New(collection)
	answer, err := openNodeOf(test, fake).receive(300 * time.Millisecond)
	if err != nil || len(answer) != 384 || answer[0] != 0x77 {
		test.Fatalf("got %d bytes, %v", len(answer), err)
	}
	if waits := fake.Waits(); waits[0] != 300 {
		test.Errorf("waits %v", waits)
	}
}

func TestTheNodeMapsTheTimeoutToNoAnswerAndOtherFailuresToTheirText(test *testing.T) {
	fake := fakehid.New(controller(zeroPath, interfaceZeroCaps))
	opened := openNodeOf(test, fake)
	fake.SetEndings(fakehid.Ending{WaitsOut: true, Code: 995, Count: fakehid.Bytes(0)},
		fakehid.Ending{Count: fakehid.Bytes(0)}, fakehid.Ending{Code: 31, Count: fakehid.Bytes(0)})
	if _, err := opened.receive(time.Millisecond); !errors.Is(err, errNoAnswer) {
		test.Errorf("got %v", err)
	}
	if _, err := opened.receive(time.Millisecond); err == nil || err.Error() != "read failed: EOF" {
		test.Errorf("got %v", err)
	}
	if _, err := opened.receive(time.Millisecond); err == nil || err.Error() != "read failed: "+syscall.Errno(31).Error() {
		test.Errorf("got %v", err)
	}
	if fake.Pending() {
		test.Error("an operation is pending")
	}
}

func hidSource(fake *fakehid.Fake) Source {
	return Source{Finder: HID(fake), TimeLimit: 200 * time.Millisecond}
}

func wantAll(string) bool { return true }

func TestASourceReadsTheProbesAndLeavesNothingPendingOrOpen(test *testing.T) {
	fake := fakehid.New(answering(controller(zeroPath, interfaceZeroCaps)))
	readings := hidSource(fake).Read(context.Background(), wantAll)
	want := []sensors.Reading{{Name: sensors.CommanderTemperature(1), Value: 50}}
	if !reflect.DeepEqual(readings, want) {
		test.Fatalf("got %v, want %v", readings, want)
	}
	if len(fake.Writes()) != 8 || fake.Pending() || !fake.AllClosed() {
		test.Errorf("writes %d, pending %v, closed %v", len(fake.Writes()), fake.Pending(), fake.AllClosed())
	}
}

func TestASourceWithNoCollectionOrAFailedOpeningGivesNoProbes(test *testing.T) {
	if got := hidSource(fakehid.New()).Read(context.Background(), wantAll); got != nil {
		test.Errorf("got %v", got)
	}
	denied := answering(controller(zeroPath, interfaceZeroCaps))
	denied.OpenError = syscall.Errno(5)
	if got := hidSource(fakehid.New(denied)).Read(context.Background(), wantAll); got != nil {
		test.Errorf("got %v", got)
	}
}

func TestASourceWaitsTheTimeLimitOfTheSourceForAnAnswer(test *testing.T) {
	fake := fakehid.New(controller(zeroPath, interfaceZeroCaps))
	fake.SetEndings(fakehid.Ending{}, fakehid.Ending{WaitsOut: true, Code: 995, Count: fakehid.Bytes(0)})
	hidSource(fake).Read(context.Background(), wantAll)
	if waits := fake.Waits(); len(waits) < 2 || waits[1] != 200 || fake.Pending() || !fake.AllClosed() {
		test.Errorf("waits %v, pending %v, closed %v", waits, fake.Pending(), fake.AllClosed())
	}
}

func lighting(fake *fakehid.Fake) Lighting {
	return Lighting{Finder: HID(fake), TimeLimit: 200 * time.Millisecond, Red: 1, Green: 2, Blue: 3}
}

func TestLightingStartGivesErrNoControllerWithNoCollection(test *testing.T) {
	if _, _, err := lighting(fakehid.New()).Start(context.Background()); !errors.Is(err, ErrNoController) {
		test.Fatalf("got %v", err)
	}
}

func TestLightingStartGivesTheErrorOfANearMiss(test *testing.T) {
	near := controller(zeroPath, winhid.Caps{UsagePage: 1, Usage: 1, InputLength: 385, OutputLength: 385})
	_, _, err := lighting(fakehid.New(near)).Start(context.Background())
	if err == nil || errors.Is(err, ErrNoController) || !strings.HasPrefix(err.Error(), "no collection of interface 0 matches: ") {
		test.Fatalf("got %v", err)
	}
}

func TestLightingStartLightsTheLEDsAndEndReturnsToHardwareMode(test *testing.T) {
	fake := fakehid.New(answering(controller(zeroPath, interfaceZeroCaps)))
	session, _, err := lighting(fake).Start(context.Background())
	if err != nil {
		test.Fatal(err)
	}
	if fake.AllClosed() {
		test.Error("the session holds the collection open")
	}
	session.End()
	writes := fake.Writes()
	if colour := writes[len(writes)-3][1:]; colour[1] != 0x06 || colour[9] != 1 || colour[10] != 2 || colour[11] != 3 {
		test.Errorf("colour write % x", colour)
	}
	if last := writes[len(writes)-1][1:6]; !reflect.DeepEqual(last, returnHardware) {
		test.Errorf("last write % x", last)
	}
	if fake.Pending() || !fake.AllClosed() {
		test.Errorf("pending %v, closed %v", fake.Pending(), fake.AllClosed())
	}
}
