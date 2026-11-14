//go:build linux

// Package fakecommander is test support: a directory tree that looks like the sysfs and the device nodes of a Corsair
// Commander Core XT on Linux, with pseudo terminals behind the nodes. The node of interface 0 answers the commands
// of the controller as the controller did in the trace that docs/commander-core-xt.md describes. It is shared by the
// tests of the driver and the end-to-end tests.
package fakecommander

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/fakepty"
)

const (
	writeSize  = 97
	answerSize = 96
	// maxProbes is the count that a valid answer can hold: 6 + 3·30 = 96.
	maxProbes = (answerSize - 6) / 3
	// quietPeriod is how long no byte may arrive before Writes gives what it holds.
	quietPeriod  = 200 * time.Millisecond
	pollInterval = 20 * time.Millisecond

	hidName = "Corsair CORSAIR iCUE COMMANDER CORE XT"

	fanEndpoint         = 0x17
	temperatureEndpoint = 0x21
)

// Probe is one entry of the answer of step 6.
type Probe struct {
	Status byte
	Tenths int16 // the temperature in tenths of a degree, written low byte first
}

// Failure is a way in which the answer of a step is wrong.
type Failure int

const (
	NoAnswer      Failure = iota + 1
	WrongEcho             // byte 1 of the answer is not the second written byte of the command
	BadStatus             // byte 2 of the answer is 0x01
	WrongDataType         // bytes 3 and 4 are 06 00; meaningful for step 6
	CountTooLarge         // byte 5 is 31, so that 6 + 3·31 > 96; meaningful for step 6
)

// Mode is the mode that the last mode command set.
type Mode int

const (
	NoMode Mode = iota // no mode command has arrived
	SoftwareMode
	HardwareMode
)

// Fake holds the roots that the discovery reads.
type Fake struct {
	test      testing.TB
	sysfsRoot string
	devRoot   string
}

// New makes controllers under the given roots, which the fake screen and the fake sensors may share.
func New(test testing.TB, sysfsRoot, devRoot string) *Fake {
	return &Fake{test: test, sysfsRoot: sysfsRoot, devRoot: devRoot}
}

// Add makes a controller whose node of interface 0 is hidraw<interface0> and whose node of interface 1 is
// hidraw<interface1>, each with a sysfs entry that holds the HID_ID 0003:00001B1C:00000C2A, the HID_NAME of the trace,
// and a HID_PHYS ending in /input0 and /input1. Only the node of interface 0 answers.
func (fake *Fake) Add(interface0, interface1 int) *Controller {
	fake.test.Helper()
	controller := fake.newController()
	controller.nodes[0] = controller.makeNode(0, interface0)
	controller.nodes[1] = controller.makeNode(1, interface1)
	return controller
}

// AddInterfaceOne makes only the entry and the node of interface 1, as hidraw<number>.
func (fake *Fake) AddInterfaceOne(number int) *Controller {
	fake.test.Helper()
	controller := fake.newController()
	controller.nodes[1] = controller.makeNode(1, number)
	return controller
}

func (fake *Fake) newController() *Controller {
	controller := &Controller{
		fake:     fake,
		probes:   []Probe{{Status: 0, Tenths: 373}, {Status: 0, Tenths: 227}},
		failures: map[int]Failure{},
		actions:  map[int]func(){},
	}
	fake.test.Cleanup(controller.close)
	return controller
}

// Controller is one fake controller.
type Controller struct {
	fake  *Fake
	nodes [2]*node // by interface; nil where the controller has none

	mutex    sync.Mutex
	probes   []Probe
	failures map[int]Failure
	actions  map[int]func()
	endpoint byte // the endpoint that the last open command named, 0 once it was closed
	mode     Mode
}

// node is the hidraw node of one interface.
type node struct {
	number    int
	master    *os.File
	slave     *os.File // held open so that the raw settings survive a close of the node by the driver
	link      string
	entry     string
	writes    [][]byte
	pending   []byte
	lastWrite time.Time
}

func (controller *Controller) makeNode(interfaceNumber, number int) *node {
	test := controller.fake.test
	test.Helper()
	master, slave, slavePath := fakepty.Open(test)
	hidraw := fmt.Sprintf("hidraw%d", number)
	entry := filepath.Join(controller.fake.sysfsRoot, "class", "hidraw", hidraw)
	if err := os.MkdirAll(filepath.Join(entry, "device"), 0o755); err != nil {
		test.Fatalf("fake controller: %v", err)
	}
	uevent := "DRIVER=hid-generic\n" +
		"HID_ID=0003:00001B1C:00000C2A\n" +
		"HID_NAME=" + hidName + "\n" +
		fmt.Sprintf("HID_PHYS=usb-0000:00:14.0-5/input%d\n", interfaceNumber) +
		"HID_UNIQ=\n" +
		"MODALIAS=hid:b0003g0001v00001B1Cp00000C2A\n"
	if err := os.WriteFile(filepath.Join(entry, "device", "uevent"), []byte(uevent), 0o644); err != nil {
		test.Fatalf("fake controller: %v", err)
	}
	if err := os.MkdirAll(controller.fake.devRoot, 0o755); err != nil {
		test.Fatalf("fake controller: %v", err)
	}
	link := filepath.Join(controller.fake.devRoot, hidraw)
	if err := os.Symlink(slavePath, link); err != nil {
		test.Fatalf("fake controller: %v", err)
	}
	created := &node{number: number, master: master, slave: slave, link: link, entry: entry, lastWrite: time.Now()}
	go controller.receive(interfaceNumber, created)
	return created
}

// SetProbes sets the probes of the answer of step 6. The default is {0, 373} and {0, 227}.
func (controller *Controller) SetProbes(probes ...Probe) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	controller.probes = probes
}

// Fail makes the answer of the step, 1 to 8, wrong as the failure says.
func (controller *Controller) Fail(step int, failure Failure) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	controller.failures[step] = failure
}

// OnStep runs the action once that step has arrived, before its answer is sent.
func (controller *Controller) OnStep(step int, action func()) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	controller.actions[step] = action
}

// RemoveNode removes the node of interface 0 and leaves the sysfs entries.
func (controller *Controller) RemoveNode() {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	if controller.nodes[0] == nil {
		return
	}
	_ = os.Remove(controller.nodes[0].link)
	controller.nodes[0].closeTerminal()
}

// Mode gives the mode that the last mode command set.
func (controller *Controller) Mode() Mode {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	return controller.mode
}

// Writes gives the writes that arrived on interface 0, 97 bytes each. It waits until nothing has arrived for a
// while, which covers a driver in the same process and an executable that has ended.
func (controller *Controller) Writes() [][]byte { return controller.quietWrites(0) }

// WritesOnInterfaceOne gives the writes that arrived on interface 1, as Writes does for interface 0.
func (controller *Controller) WritesOnInterfaceOne() [][]byte { return controller.quietWrites(1) }

func (controller *Controller) quietWrites(interfaceNumber int) [][]byte {
	for {
		controller.mutex.Lock()
		current := controller.nodes[interfaceNumber]
		if current == nil || time.Since(current.lastWrite) >= quietPeriod {
			writes := current.copyOfWrites()
			controller.mutex.Unlock()
			return writes
		}
		controller.mutex.Unlock()
		time.Sleep(pollInterval)
	}
}

// WaitWrites waits until at least count writes have arrived on interface 0 or the limit has passed, and gives the
// writes and whether the count was reached.
func (controller *Controller) WaitWrites(count int, limit time.Duration) ([][]byte, bool) {
	deadline := time.Now().Add(limit)
	for {
		controller.mutex.Lock()
		writes := controller.nodes[0].copyOfWrites()
		controller.mutex.Unlock()
		if len(writes) >= count {
			return writes, true
		}
		if !time.Now().Before(deadline) {
			return writes, false
		}
		time.Sleep(pollInterval)
	}
}

func (current *node) copyOfWrites() [][]byte {
	if current == nil {
		return nil
	}
	writes := make([][]byte, len(current.writes))
	for position, write := range current.writes {
		writes[position] = bytes.Clone(write)
	}
	return writes
}

func (current *node) closeTerminal() {
	_ = current.master.Close()
	_ = current.slave.Close()
}

func (controller *Controller) close() {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	for _, current := range controller.nodes {
		if current != nil {
			current.closeTerminal()
		}
	}
}

// receive reads the master until it is closed, splits what arrives into writes of 97 bytes, keeps them, and answers
// each one on interface 0.
func (controller *Controller) receive(interfaceNumber int, current *node) {
	buffer := make([]byte, 4096)
	for {
		count, err := current.master.Read(buffer)
		if count > 0 {
			controller.accept(interfaceNumber, current, buffer[:count])
		}
		if err != nil {
			return
		}
	}
}

func (controller *Controller) accept(interfaceNumber int, current *node, data []byte) {
	controller.mutex.Lock()
	current.pending = append(current.pending, data...)
	var complete [][]byte
	for len(current.pending) >= writeSize {
		write := bytes.Clone(current.pending[:writeSize])
		current.pending = current.pending[writeSize:]
		current.writes = append(current.writes, write)
		current.lastWrite = time.Now()
		complete = append(complete, write)
	}
	controller.mutex.Unlock()
	if interfaceNumber != 0 {
		return
	}
	for _, write := range complete {
		controller.answer(current, write[1:])
	}
}

// answer works out which step the command is, runs the action of the step and sends its answer.
func (controller *Controller) answer(current *node, command []byte) {
	controller.mutex.Lock()
	step := controller.identify(command)
	if step == 0 {
		controller.mutex.Unlock()
		return
	}
	switch step {
	case 1:
		controller.mode = SoftwareMode
	case 8:
		controller.mode = HardwareMode
	}
	failure := controller.failures[step]
	action := controller.actions[step]
	reply := controller.reply(step, command)
	controller.mutex.Unlock()

	if action != nil {
		action()
	}
	if failure == NoAnswer {
		return
	}
	applyFailure(reply, failure)
	_, _ = current.master.Write(reply)
}

// identify gives the step of the command, 1 to 8, or 0 for a command that is none of the eight. It follows the
// endpoint that the last open command named. The caller holds the mutex.
func (controller *Controller) identify(command []byte) int {
	switch {
	case bytes.HasPrefix(command, []byte{0x08, 0x01, 0x03, 0x00, 0x02}):
		return 1
	case bytes.HasPrefix(command, []byte{0x08, 0x01, 0x03, 0x00, 0x01}):
		return 8
	case bytes.HasPrefix(command, []byte{0x08, 0x0D, 0x00, fanEndpoint}):
		controller.endpoint = fanEndpoint
		return 2
	case bytes.HasPrefix(command, []byte{0x08, 0x0D, 0x00, temperatureEndpoint}):
		controller.endpoint = temperatureEndpoint
		return 5
	case bytes.HasPrefix(command, []byte{0x08, 0x08, 0x00}):
		return controller.stepOfOpenEndpoint(3, 6)
	case bytes.HasPrefix(command, []byte{0x08, 0x05, 0x01}):
		return controller.stepOfOpenEndpoint(4, 7)
	}
	return 0
}

func (controller *Controller) stepOfOpenEndpoint(fanStep, temperatureStep int) int {
	switch controller.endpoint {
	case fanEndpoint:
		return fanStep
	case temperatureEndpoint:
		return temperatureStep
	}
	return 0
}

// reply makes the 96 bytes of the answer of the step, as the trace showed them. The caller holds the mutex.
func (controller *Controller) reply(step int, command []byte) []byte {
	answer := make([]byte, answerSize)
	answer[1] = command[1]
	switch step {
	case 3:
		copy(answer[3:], []byte{0x06, 0x00, 0x06})
	case 6:
		count := min(len(controller.probes), maxProbes)
		copy(answer[3:], []byte{0x10, 0x00, byte(count)})
		for position, probe := range controller.probes[:count] {
			offset := 6 + 3*position
			answer[offset] = probe.Status
			answer[offset+1] = byte(uint16(probe.Tenths))
			answer[offset+2] = byte(uint16(probe.Tenths) >> 8)
		}
	}
	return answer
}

func applyFailure(answer []byte, failure Failure) {
	switch failure {
	case WrongEcho:
		answer[1] = 0xFF
	case BadStatus:
		answer[2] = 0x01
	case WrongDataType:
		copy(answer[3:], []byte{0x06, 0x00})
	case CountTooLarge:
		answer[5] = maxProbes + 1
	}
}
