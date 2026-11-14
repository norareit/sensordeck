//go:build linux

// Package fakescreen is test support: a directory tree that looks like the sysfs and the device nodes of a screen on
// Linux, with pseudo terminals behind the nodes. It is shared by the tests of the transport and the end-to-end tests.
package fakescreen

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakepty"
	"github.com/norareit/sensordeck/internal/smartmonitor"
)

const (
	// writtenReportSize is a report as the transport writes it: the report ID byte and the 64 bytes.
	writtenReportSize = 1 + len(smartmonitor.Report{})
	// quietPeriod is how long no byte may arrive before Received gives what it holds.
	quietPeriod  = 200 * time.Millisecond
	pollInterval = 20 * time.Millisecond
)

// Fake holds the roots that the discovery reads.
type Fake struct {
	SysfsRoot string // <temp>/sys
	DevRoot   string // <temp>/dev

	test testing.TB
}

// New makes the two roots in a temporary directory of the test and closes every terminal when the test ends.
func New(test testing.TB) *Fake {
	test.Helper()
	root := test.TempDir()
	fake := &Fake{SysfsRoot: filepath.Join(root, "sys"), DevRoot: filepath.Join(root, "dev"), test: test}
	for _, directory := range []string{filepath.Join(fake.SysfsRoot, "class", "hidraw"), fake.DevRoot} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			test.Fatalf("fake screen: %v", err)
		}
	}
	return fake
}

// answerRule sends a report once the count-th written report has arrived.
type answerRule struct {
	count  int
	report smartmonitor.Report
	fired  bool
}

// moveRule moves the screen once the count-th written report has arrived.
type moveRule struct {
	count  int
	number int
	fired  bool
}

// Screen is one fake screen. It can be taken away and brought back under another hidraw number.
type Screen struct {
	fake         *Fake
	name, serial string

	mutex       sync.Mutex
	number      int
	master      *os.File
	slave       *os.File // held open so that the raw settings survive a close of the node by the transport
	received    []byte
	lastArrival time.Time
	answers     []*answerRule
	moves       []*moveRule
}

// Add makes a screen with the given hidraw number, name and serial.
func (fake *Fake) Add(number int, name, serial string) *Screen {
	fake.test.Helper()
	screen := &Screen{fake: fake, name: name, serial: serial, lastArrival: time.Now()}
	screen.mutex.Lock()
	defer screen.mutex.Unlock()
	screen.appear(number)
	fake.test.Cleanup(screen.Remove)
	return screen
}

// Node is the path of the device node of the screen now.
func (screen *Screen) Node() string {
	screen.mutex.Lock()
	defer screen.mutex.Unlock()
	return screen.node()
}

func (screen *Screen) node() string {
	return filepath.Join(screen.fake.DevRoot, fmt.Sprintf("hidraw%d", screen.number))
}

// Send writes the 64 bytes of the report to the screen's end of the terminal now.
func (screen *Screen) Send(report smartmonitor.Report) {
	screen.fake.test.Helper()
	screen.SendBytes(report[:])
}

// SendBytes writes the bytes as given, which is how a short report is sent.
func (screen *Screen) SendBytes(data []byte) {
	screen.fake.test.Helper()
	screen.mutex.Lock()
	master := screen.master
	screen.mutex.Unlock()
	if master == nil {
		screen.fake.test.Errorf("fake screen: send on a screen that is away")
		return
	}
	if _, err := master.Write(data); err != nil {
		screen.fake.test.Errorf("fake screen: send: %v", err)
	}
}

// AnswerAfter sends the report once the count-th written report has arrived.
func (screen *Screen) AnswerAfter(count int, report smartmonitor.Report) {
	screen.mutex.Lock()
	defer screen.mutex.Unlock()
	screen.answers = append(screen.answers, &answerRule{count: count, report: report})
}

// MoveAfter moves the screen to another hidraw number once the count-th written report has arrived.
func (screen *Screen) MoveAfter(count int, number int) {
	screen.mutex.Lock()
	defer screen.mutex.Unlock()
	screen.moves = append(screen.moves, &moveRule{count: count, number: number})
}

// Move takes the screen away and brings it back as hidraw<number>.
func (screen *Screen) Move(number int) {
	screen.fake.test.Helper()
	screen.mutex.Lock()
	defer screen.mutex.Unlock()
	screen.disappear()
	screen.appear(number)
}

// Remove takes the screen away for good.
func (screen *Screen) Remove() {
	screen.mutex.Lock()
	defer screen.mutex.Unlock()
	screen.disappear()
}

// Received gives every byte written to the node, over all moves. It waits until nothing has arrived for a while,
// which covers a transport in the same process and an executable that has ended.
func (screen *Screen) Received() []byte {
	for {
		screen.mutex.Lock()
		quiet := time.Since(screen.lastArrival) >= quietPeriod
		received := append([]byte(nil), screen.received...)
		screen.mutex.Unlock()
		if quiet {
			return received
		}
		time.Sleep(pollInterval)
	}
}

// Reports gives the complete reports that arrived so far, over all moves, each without the report ID byte. A last
// report that is cut short is left out.
func (screen *Screen) Reports() []smartmonitor.Report {
	screen.mutex.Lock()
	defer screen.mutex.Unlock()
	var reports []smartmonitor.Report
	for start := 0; start+writtenReportSize <= len(screen.received); start += writtenReportSize {
		var report smartmonitor.Report
		copy(report[:], screen.received[start+1:start+writtenReportSize])
		reports = append(reports, report)
	}
	return reports
}

// WaitReports waits until at least count reports have arrived or the limit has passed, polling as Received does, and
// gives the reports and whether the count was reached.
func (screen *Screen) WaitReports(count int, limit time.Duration) ([]smartmonitor.Report, bool) {
	deadline := time.Now().Add(limit)
	for {
		reports := screen.Reports()
		if len(reports) >= count {
			return reports, true
		}
		if !time.Now().Before(deadline) {
			return reports, false
		}
		time.Sleep(pollInterval)
	}
}

// appear makes the terminal, the sysfs entry and the link, and starts the reader. The caller holds the mutex.
func (screen *Screen) appear(number int) {
	test := screen.fake.test
	test.Helper()
	master, slave, slavePath := fakepty.Open(test)
	screen.number, screen.master, screen.slave = number, master, slave

	entry := filepath.Join(screen.fake.SysfsRoot, "class", "hidraw", fmt.Sprintf("hidraw%d", number), "device")
	if err := os.MkdirAll(entry, 0o755); err != nil {
		test.Fatalf("fake screen: %v", err)
	}
	uevent := "DRIVER=hid-generic\n" +
		"HID_ID=0003:00000483:00000065\n" +
		"HID_NAME=" + screen.name + "\n" +
		"HID_PHYS=usb-0000:00:14.0-3/input0\n" +
		"HID_UNIQ=" + screen.serial + "\n" +
		"MODALIAS=hid:b0003g0001v00000483p00000065\n"
	if err := os.WriteFile(filepath.Join(entry, "uevent"), []byte(uevent), 0o644); err != nil {
		test.Fatalf("fake screen: %v", err)
	}
	if err := os.Symlink(slavePath, screen.node()); err != nil {
		test.Fatalf("fake screen: %v", err)
	}
	go screen.receive(master)
}

// disappear removes the entry and the link and closes the terminal. The caller holds the mutex.
func (screen *Screen) disappear() {
	if screen.master == nil {
		return
	}
	_ = os.RemoveAll(filepath.Join(screen.fake.SysfsRoot, "class", "hidraw", fmt.Sprintf("hidraw%d", screen.number)))
	_ = os.Remove(screen.node())
	_ = screen.master.Close()
	_ = screen.slave.Close()
	screen.master, screen.slave = nil, nil
}

// receive reads the master until it is closed, and carries out the rules whose count has been reached.
func (screen *Screen) receive(master *os.File) {
	buffer := make([]byte, 4096)
	for {
		count, err := master.Read(buffer)
		if count > 0 {
			screen.mutex.Lock()
			screen.received = append(screen.received, buffer[:count]...)
			screen.lastArrival = time.Now()
			answers, moves := screen.dueRules(len(screen.received) / writtenReportSize)
			screen.mutex.Unlock()
			for _, report := range answers {
				screen.Send(report)
			}
			for _, number := range moves {
				screen.Move(number)
			}
		}
		if err != nil {
			return
		}
	}
}

// dueRules marks the rules whose count has been reached as fired and returns what they do. The caller holds the mutex.
func (screen *Screen) dueRules(reports int) (answers []smartmonitor.Report, moves []int) {
	for _, rule := range screen.answers {
		if !rule.fired && reports >= rule.count {
			rule.fired = true
			answers = append(answers, rule.report)
		}
	}
	for _, rule := range screen.moves {
		if !rule.fired && reports >= rule.count {
			rule.fired = true
			moves = append(moves, rule.number)
		}
	}
	return answers, moves
}
