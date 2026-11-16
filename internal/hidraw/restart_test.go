//go:build linux

package hidraw

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakeusb"
)

// restartHarness holds a screen of fakeusb and the fakes of the seams of Restart, with a clock that only the fakes move.
type restartHarness struct {
	test      *testing.T
	device    *fakeusb.Device
	sysfsRoot string
	devRoot   string
	clock     time.Time

	opened      []string
	openErr     error
	requests    []ControlRequest
	answer      func(index int) error
	perRequest  time.Duration // how far each request moves the clock
	usbfsClosed int

	screenCloses   int
	reopenedNodes  []string
	reopenAttempts int

	waits  []time.Duration
	onWait func(count int)
	stop   context.Context
	cancel context.CancelFunc
}

func newRestartHarness(test *testing.T) *restartHarness {
	test.Helper()
	root := test.TempDir()
	harness := &restartHarness{
		test:      test,
		sysfsRoot: filepath.Join(root, "sys"),
		devRoot:   filepath.Join(root, "dev"),
		clock:     time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC),
		answer:    func(int) error { return syscall.EPIPE },
	}
	harness.device = fakeusb.New(test, harness.sysfsRoot, harness.devRoot, 3)
	harness.stop, harness.cancel = context.WithCancel(context.Background())
	test.Cleanup(harness.cancel)
	return harness
}

func (harness *restartHarness) restart(node string) Restart {
	return Restart{
		SysfsRoot: harness.sysfsRoot,
		DevRoot:   harness.devRoot,
		Node:      node,
		Usbfs:     fakeUsbfs{harness},
		Screen:    fakeScreen{harness},
		Now:       func() time.Time { return harness.clock },
		Wait: func(stop context.Context, duration time.Duration) bool {
			harness.waits = append(harness.waits, duration)
			harness.clock = harness.clock.Add(duration)
			if harness.onWait != nil {
				harness.onWait(len(harness.waits))
			}
			return stop.Err() == nil
		},
	}
}

func (harness *restartHarness) run() RestartResult {
	return harness.restart(harness.device.Node).Run(harness.stop)
}

func (harness *restartHarness) sentIndexes() []int {
	indexes := make([]int, len(harness.requests))
	for position, request := range harness.requests {
		indexes[position] = int(request.Value & 0xff)
	}
	return indexes
}

type fakeUsbfs struct{ harness *restartHarness }

func (usbfs fakeUsbfs) Open(path string) (UsbfsNode, error) {
	usbfs.harness.opened = append(usbfs.harness.opened, path)
	if usbfs.harness.openErr != nil {
		return nil, usbfs.harness.openErr
	}
	return fakeUsbfsNode{usbfs.harness}, nil
}

type fakeUsbfsNode struct{ harness *restartHarness }

func (node fakeUsbfsNode) Control(request ControlRequest) error {
	harness := node.harness
	harness.requests = append(harness.requests, request)
	harness.clock = harness.clock.Add(harness.perRequest)
	return harness.answer(int(request.Value & 0xff))
}

func (node fakeUsbfsNode) Close() error {
	node.harness.usbfsClosed++
	return nil
}

type fakeScreen struct{ harness *restartHarness }

func (screen fakeScreen) Close() error {
	screen.harness.screenCloses++
	return nil
}

func (screen fakeScreen) Reopen() error {
	harness := screen.harness
	harness.reopenAttempts++
	screens, err := Discover(harness.sysfsRoot, harness.devRoot)
	if err != nil {
		return err
	}
	if len(screens) != 1 {
		return errors.New("no screen")
	}
	harness.reopenedNodes = append(harness.reopenedNodes, screens[0].Node)
	return nil
}

func restarted(result RestartResult) bool {
	return result.Outcome == RestartedOpen || result.Outcome == RestartedNotOpen
}

func TestRestartOpensTheUsbfsNodeOfADiscoveredNode(test *testing.T) {
	harness := newRestartHarness(test)
	harness.run()
	want := filepath.Join(harness.devRoot, "bus", "usb", "001", "005")
	if len(harness.opened) != 1 || harness.opened[0] != want {
		test.Fatalf("opened %v, want %s", harness.opened, want)
	}
}

func TestRestartOpensTheUsbfsNodeOfANodeGivenByALink(test *testing.T) {
	harness := newRestartHarness(test)
	link := filepath.Join(test.TempDir(), "screen")
	if err := os.Symlink(harness.device.Node, link); err != nil {
		test.Fatal(err)
	}
	harness.restart(link).Run(harness.stop)
	want := filepath.Join(harness.devRoot, "bus", "usb", "001", "005")
	if len(harness.opened) != 1 || harness.opened[0] != want {
		test.Fatalf("opened %v, want %s", harness.opened, want)
	}
}

func TestRestartRequestsStringDescriptorsFromIndexFourInOrder(test *testing.T) {
	harness := newRestartHarness(test)
	harness.answer = func(index int) error {
		if index == 6 {
			return syscall.ENODEV
		}
		return syscall.EPIPE
	}
	harness.run()
	for position, request := range harness.requests {
		want := ControlRequest{RequestType: 0x80, Request: 6, Value: 0x0300 | uint16(4+position), Index: 0x0409, Length: 255, Timeout: time.Second}
		if request != want {
			test.Errorf("request %d is %+v, want %+v", position, request, want)
		}
	}
	if got := harness.sentIndexes(); len(got) != 3 || got[0] != 4 || got[1] != 5 || got[2] != 6 {
		test.Fatalf("indexes %v", got)
	}
}

func TestRestartEndsWhereTheScreenLeavesTheBus(test *testing.T) {
	cases := []struct {
		name      string
		leaveAt   int
		answer    func(harness *restartHarness, index int) error
		wantIndex int
	}{
		{"ENODEV", 33, func(_ *restartHarness, index int) error {
			if index == 33 {
				return syscall.ENODEV
			}
			return syscall.EPIPE
		}, 33},
		{"directory removed", 20, func(harness *restartHarness, index int) error {
			if index == 20 {
				harness.device.Unplug()
			}
			return syscall.EPIPE
		}, 20},
		{"devnum changed", 9, func(harness *restartHarness, index int) error {
			if index == 9 {
				harness.device.SetDevnum("6\n")
			}
			return syscall.EPIPE
		}, 9},
	}
	for _, testCase := range cases {
		test.Run(testCase.name, func(test *testing.T) {
			harness := newRestartHarness(test)
			harness.answer = func(index int) error { return testCase.answer(harness, index) }
			harness.onWait = func(int) { harness.cancel() }
			result := harness.run()
			if !restarted(result) || result.Index != testCase.wantIndex {
				test.Fatalf("got %+v", result)
			}
			sent := harness.sentIndexes()
			if sent[len(sent)-1] != testCase.wantIndex {
				test.Fatalf("a request was sent after the screen left: %v", sent)
			}
			if harness.screenCloses != 1 || harness.usbfsClosed != 1 {
				test.Fatalf("closes: screen %d, usbfs %d", harness.screenCloses, harness.usbfsClosed)
			}
		})
	}
}

func TestRestartClosesTheScreenBeforeTheFirstRequest(test *testing.T) {
	harness := newRestartHarness(test)
	closedAtFirstRequest := -1
	harness.answer = func(int) error {
		if closedAtFirstRequest < 0 {
			closedAtFirstRequest = harness.screenCloses
		}
		return syscall.ENODEV
	}
	harness.onWait = func(int) { harness.cancel() }
	harness.run()
	if closedAtFirstRequest != 1 {
		test.Fatalf("the screen had been closed %d times at the first request", closedAtFirstRequest)
	}
}

func TestRestartGoesOnAfterAnAnsweredRequestAndAnotherError(test *testing.T) {
	harness := newRestartHarness(test)
	harness.answer = func(index int) error {
		switch index {
		case 4:
			return nil
		case 5:
			return syscall.EIO
		case 6:
			return syscall.ENODEV
		}
		return syscall.EPIPE
	}
	harness.onWait = func(int) { harness.cancel() }
	result := harness.run()
	if !restarted(result) || result.Index != 6 {
		test.Fatalf("got %+v", result)
	}
}

func TestRestartEndsAtTheLastIndexAsNotRestarted(test *testing.T) {
	harness := newRestartHarness(test)
	result := harness.run()
	if result.Outcome != NotRestarted {
		test.Fatalf("got %+v", result)
	}
	sent := harness.sentIndexes()
	if len(sent) != 252 || sent[0] != 4 || sent[251] != 255 {
		test.Fatalf("sent %d requests, from %d to %d", len(sent), sent[0], sent[len(sent)-1])
	}
	if harness.reopenAttempts != 0 || harness.usbfsClosed != 1 {
		test.Fatalf("reopenings %d, usbfs closes %d", harness.reopenAttempts, harness.usbfsClosed)
	}
}

func TestRestartEndsAtTheTimeLimitAsNotRestarted(test *testing.T) {
	harness := newRestartHarness(test)
	harness.perRequest = time.Second
	result := harness.run()
	if result.Outcome != NotRestarted {
		test.Fatalf("got %+v", result)
	}
	sent := harness.sentIndexes()
	if len(sent) != 10 || sent[0] != 4 || sent[9] != 13 {
		test.Fatalf("sent %v", sent)
	}
}

func TestRestartIsNotPossibleWhenTheUsbfsNodeDoesNotOpen(test *testing.T) {
	harness := newRestartHarness(test)
	harness.openErr = &os.PathError{Op: "open", Path: "/dev/bus/usb/001/005", Err: syscall.EACCES}
	result := harness.run()
	if result.Outcome != RestartNotPossible || !errors.Is(result.Err, syscall.EACCES) {
		test.Fatalf("got %+v", result)
	}
	if len(harness.requests) != 0 || harness.screenCloses != 0 {
		test.Fatalf("requests %d, screen closes %d", len(harness.requests), harness.screenCloses)
	}
}

func TestRestartIsNotPossibleForANodeWithNoEntry(test *testing.T) {
	harness := newRestartHarness(test)
	other := filepath.Join(harness.devRoot, "hidraw9")
	if err := os.WriteFile(other, nil, 0o644); err != nil {
		test.Fatal(err)
	}
	result := harness.restart(other).Run(harness.stop)
	if result.Outcome != RestartNotPossible || result.Err == nil {
		test.Fatalf("got %+v", result)
	}
	if len(harness.opened) != 0 || len(harness.requests) != 0 || harness.screenCloses != 0 {
		test.Fatalf("opened %v, requests %d, screen closes %d", harness.opened, len(harness.requests), harness.screenCloses)
	}
}

func TestRestartIsNotPossibleWhenDevnumIsNotANumber(test *testing.T) {
	harness := newRestartHarness(test)
	harness.device.SetDevnum("soon\n")
	result := harness.run()
	if result.Outcome != RestartNotPossible || result.Err == nil || len(harness.opened) != 0 {
		test.Fatalf("got %+v, opened %v", result, harness.opened)
	}
}

func TestRestartOpensTheScreenThatEnumeratedWithAnotherNumber(test *testing.T) {
	harness := newRestartHarness(test)
	harness.answer = func(index int) error {
		if index == 33 {
			harness.device.Unplug()
			return syscall.ENODEV
		}
		return syscall.EPIPE
	}
	harness.onWait = func(count int) {
		if count == 3 {
			fakeusb.New(test, harness.sysfsRoot, harness.devRoot, 7)
		}
	}
	result := harness.run()
	if result.Outcome != RestartedOpen || result.Index != 33 {
		test.Fatalf("got %+v", result)
	}
	if len(harness.reopenedNodes) != 1 || harness.reopenedNodes[0] != filepath.Join(harness.devRoot, "hidraw7") {
		test.Fatalf("reopened %v", harness.reopenedNodes)
	}
	if len(harness.waits) != 3 {
		test.Fatalf("waited %v", harness.waits)
	}
	for _, wait := range harness.waits {
		if wait != 100*time.Millisecond {
			test.Fatalf("waited %v", harness.waits)
		}
	}
}

func TestRestartGivesUpWaitingForTheScreenAfterTenSeconds(test *testing.T) {
	harness := newRestartHarness(test)
	harness.answer = func(index int) error {
		harness.device.Unplug()
		return syscall.ENODEV
	}
	result := harness.run()
	if result.Outcome != RestartedNotOpen || result.Index != 4 {
		test.Fatalf("got %+v", result)
	}
	var total time.Duration
	for _, wait := range harness.waits {
		total += wait
	}
	if total != 10*time.Second {
		test.Fatalf("waited %v in all", total)
	}
}

func TestRestartStopsDuringTheRequests(test *testing.T) {
	harness := newRestartHarness(test)
	harness.answer = func(index int) error {
		if index == 6 {
			harness.cancel()
		}
		return syscall.EPIPE
	}
	result := harness.run()
	if result.Outcome != NotRestarted {
		test.Fatalf("got %+v", result)
	}
	if got := harness.sentIndexes(); got[len(got)-1] != 6 {
		test.Fatalf("sent %v", got)
	}
	if harness.usbfsClosed != 1 {
		test.Fatalf("usbfs closes %d", harness.usbfsClosed)
	}
}

func TestRestartStopsDuringTheWait(test *testing.T) {
	harness := newRestartHarness(test)
	harness.answer = func(int) error {
		harness.device.Unplug()
		return syscall.ENODEV
	}
	harness.onWait = func(int) { harness.cancel() }
	result := harness.run()
	if result.Outcome != RestartedNotOpen {
		test.Fatalf("got %+v", result)
	}
	if len(harness.waits) != 1 || harness.reopenAttempts != 1 {
		test.Fatalf("waits %d, reopenings %d", len(harness.waits), harness.reopenAttempts)
	}
}
