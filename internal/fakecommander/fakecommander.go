//go:build linux

// Package fakecommander is test support: a directory tree that looks like the sysfs and the device nodes of a Corsair
// Commander Core XT on Linux, with pseudo terminals behind the nodes. The node of interface 0 answers the commands
// of the controller as the controller did in the traces and the device tests that docs/commander-core-xt.md describes:
// the eight commands of unit 0005, the endpoints on two channels, the LED endpoint with its colour write, the modes and
// a software-mode timeout. It is shared by the tests of the driver and the end-to-end tests.
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
	writeSize = 97
	// answerSize is what an answer of the trace holds, and reportSize the input report of the report descriptor that
	// a reply fills, with 0xFF past the answer.
	answerSize = 96
	reportSize = 384
	// maxProbes is the count that a valid answer can hold: 6 + 3·30 = 96.
	maxProbes = (answerSize - 6) / 3
	// quietPeriod is how long no byte may arrive before Writes gives what it holds.
	quietPeriod  = 200 * time.Millisecond
	pollInterval = 20 * time.Millisecond

	hidName = "Corsair CORSAIR iCUE COMMANDER CORE XT"

	fanEndpoint         = 0x17
	temperatureEndpoint = 0x21
	portsEndpoint       = 0x20
	unknownEndpoint     = 0x1A
	ledEndpoint         = 0x22

	statusUnavailable = 0x03
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

// Port is one entry of the answer of the endpoint 0x20: the mode of a port of the controller and its LED count.
type Port struct{ Mode, LEDs uint16 }

// Lighting is what the LEDs of the controller show.
type Lighting int

const (
	StoredLighting Lighting = iota // in hardware mode, and before any mode command
	Dark                           // in software mode before a colour write
	SentColour                     // in software mode after a colour write
)

// commandHook is a failure or an action chosen by the bytes a command starts with.
type commandHook struct {
	prefix  []byte
	failure Failure
	action  func()
}

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
		ports:    []Port{{0, 0}, {3, 0}, {3, 0}, {2, 4}, {2, 4}, {2, 4}, {2, 4}},
		failures: map[int]Failure{},
		actions:  map[int]func(){},
	}
	fake.test.Cleanup(controller.close)
	return controller
}

// Controller is one fake controller. It keeps the mode, the endpoint open on each channel, the colour writes and what
// its LEDs show, and every command on interface 0 gets the answer that handle works out.
type Controller struct {
	fake  *Fake
	nodes [2]*node // by interface; nil where the controller has none

	mutex          sync.Mutex
	probes         []Probe
	failures       map[int]Failure
	actions        map[int]func()
	open           [2]byte // the endpoint open on each channel, 0 when none is
	mode           Mode
	ports          []Port
	lighting       Lighting
	colour         [3]byte
	writtenColours [][]byte
	hooks          []commandHook
	timeout        time.Duration
	timer          *time.Timer
	idle           int // counts the commands in software mode, so that a timer that fired late can tell it is stale
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

// SetPorts sets the entries of the answer of the endpoint 0x20. The default is those of the trace of the host: mode 0
// with 0 LEDs, mode 3 with 0 LEDs twice, and mode 2 with 4 LEDs four times.
func (controller *Controller) SetPorts(ports ...Port) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	controller.ports = ports
}

// SetOpenEndpoint makes the endpoint open on the channel, as a program that was killed leaves it.
func (controller *Controller) SetOpenEndpoint(channel, endpoint byte) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	controller.open[channel] = endpoint
}

// OpenEndpoint gives the endpoint open on the channel, or 0 when none is.
func (controller *Controller) OpenEndpoint(channel byte) byte {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	return controller.open[channel]
}

// SetSoftwareModeTimeout makes the controller go to hardware mode when no command arrives for that long in software
// mode, as the device is thought to do, closing every channel and forgetting the colour. Zero, the default, is off.
func (controller *Controller) SetSoftwareModeTimeout(timeout time.Duration) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	controller.timeout = timeout
}

// FailCommand makes the answer of every later command that starts with the prefix wrong as the failure says. It
// applies after the failure that Fail sets for a step.
func (controller *Controller) FailCommand(prefix []byte, failure Failure) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	controller.hooks = append(controller.hooks, commandHook{prefix: bytes.Clone(prefix), failure: failure})
}

// OnCommand runs the action for every command that starts with the prefix, before its answer is sent.
func (controller *Controller) OnCommand(prefix []byte, action func()) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	controller.hooks = append(controller.hooks, commandHook{prefix: bytes.Clone(prefix), action: action})
}

// ColourWrites gives the 96 data bytes of every colour write that was kept, in order.
func (controller *Controller) ColourWrites() [][]byte {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	writes := make([][]byte, len(controller.writtenColours))
	for position, write := range controller.writtenColours {
		writes[position] = bytes.Clone(write)
	}
	return writes
}

// Lighting tells what the LEDs show, and for SentColour the red, green and blue of the last colour write.
func (controller *Controller) Lighting() (Lighting, [3]byte) {
	controller.mutex.Lock()
	defer controller.mutex.Unlock()
	return controller.lighting, controller.colour
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

// answer works out what the command is, runs its actions and sends its answer.
func (controller *Controller) answer(current *node, command []byte) {
	controller.mutex.Lock()
	controller.keepAwake()
	step, status, body, recognised := controller.handle(command)
	if !recognised {
		controller.mutex.Unlock()
		return
	}
	failures := []Failure{controller.failures[step]}
	actions := []func(){controller.actions[step]}
	for _, hook := range controller.hooks {
		if bytes.HasPrefix(command, hook.prefix) {
			failures = append(failures, hook.failure)
			actions = append(actions, hook.action)
		}
	}
	reply := controller.report(command, status, body)
	controller.mutex.Unlock()

	for _, action := range actions {
		if action != nil {
			action()
		}
	}
	for _, failure := range failures {
		if failure == NoAnswer {
			return
		}
	}
	for _, failure := range failures {
		applyFailure(reply, failure)
	}
	_, _ = current.master.Write(reply)
}

// keepAwake restarts the software-mode timeout. The caller holds the mutex.
func (controller *Controller) keepAwake() {
	if controller.timer != nil {
		controller.timer.Stop()
		controller.timer = nil
	}
	if controller.mode != SoftwareMode || controller.timeout == 0 {
		return
	}
	controller.idle++
	idle := controller.idle
	controller.timer = time.AfterFunc(controller.timeout, func() {
		controller.mutex.Lock()
		defer controller.mutex.Unlock()
		if controller.idle != idle || controller.mode != SoftwareMode {
			return
		}
		controller.mode = HardwareMode
		controller.open = [2]byte{}
		controller.lighting = StoredLighting
	})
}

// handle changes the state as the command does and gives the step of unit 0005 that it is, 0 for none, the status of
// the answer, the bytes of the answer from byte 3, and whether the command gets an answer at all. The caller holds the
// mutex. The endpoint steps are those of channel 0 and of the endpoints 0x17 and 0x21 alone.
func (controller *Controller) handle(command []byte) (step int, status byte, body []byte, answered bool) {
	switch {
	case bytes.HasPrefix(command, []byte{0x08, 0x01, 0x03, 0x00, 0x02}):
		controller.mode = SoftwareMode
		controller.lighting = Dark
		controller.keepAwake()
		return 1, 0, nil, true
	case bytes.HasPrefix(command, []byte{0x08, 0x01, 0x03, 0x00, 0x01}):
		controller.mode = HardwareMode
		controller.lighting = StoredLighting
		controller.keepAwake()
		return 8, 0, nil, true
	}
	step = controller.stepOnChannelZero(command)
	if controller.mode != SoftwareMode {
		return step, statusUnavailable, nil, true
	}
	switch {
	case bytes.HasPrefix(command, []byte{0x08, 0x0D}) && len(command) > 3:
		channel, endpoint := command[2], command[3]
		if channel > 1 {
			return 0, 0, nil, false
		}
		if controller.open[channel] != 0 {
			return step, statusUnavailable, nil, true
		}
		controller.open[channel] = endpoint
		return step, 0, nil, true
	case bytes.HasPrefix(command, []byte{0x08, 0x08}) && len(command) > 2:
		channel := command[2]
		if channel > 1 || controller.open[channel] == 0 {
			return 0, 0, nil, false
		}
		return step, 0, controller.answerOf(controller.open[channel]), true
	case bytes.HasPrefix(command, []byte{0x08, 0x05, 0x01}):
		if len(command) > 3 && command[3] <= 1 {
			controller.open[command[3]] = 0
		} else {
			controller.open[0] = 0
		}
		return step, 0, nil, true
	case bytes.HasPrefix(command, []byte{0x08, 0x06, 0x00}):
		if controller.open[0] != ledEndpoint {
			return 0, statusUnavailable, nil, true
		}
		controller.writtenColours = append(controller.writtenColours, bytes.Clone(command))
		controller.lighting = SentColour
		copy(controller.colour[:], command[9:12])
		return 0, 0, nil, true
	}
	return 0, 0, nil, false
}

// stepOnChannelZero gives the step of unit 0005 of the command, from the endpoint open on channel 0 before the command
// changes it. The caller holds the mutex.
func (controller *Controller) stepOnChannelZero(command []byte) int {
	endpoint := controller.open[0]
	switch {
	case bytes.HasPrefix(command, []byte{0x08, 0x0D, 0x00, fanEndpoint}):
		return 2
	case bytes.HasPrefix(command, []byte{0x08, 0x0D, 0x00, temperatureEndpoint}):
		return 5
	case bytes.HasPrefix(command, []byte{0x08, 0x08, 0x00}) && endpoint == fanEndpoint:
		return 3
	case bytes.HasPrefix(command, []byte{0x08, 0x08, 0x00}) && endpoint == temperatureEndpoint:
		return 6
	case bytes.HasPrefix(command, []byte{0x08, 0x05, 0x01}) && endpoint == fanEndpoint:
		return 4
	case bytes.HasPrefix(command, []byte{0x08, 0x05, 0x01}) && endpoint == temperatureEndpoint:
		return 7
	}
	return 0
}

// answerOf gives the bytes from byte 3 of the answer to a read of the endpoint. The caller holds the mutex.
func (controller *Controller) answerOf(endpoint byte) []byte {
	switch endpoint {
	case fanEndpoint:
		return []byte{0x06, 0x00, 0x06}
	case unknownEndpoint:
		return []byte{0x09, 0x00, 0x06, 1, 1, 1, 1, 1, 1}
	case portsEndpoint:
		body := []byte{0x0F, 0x00, byte(len(controller.ports))}
		for _, port := range controller.ports {
			if len(body)+4 > answerSize-3 {
				break
			}
			body = append(body, byte(port.Mode), byte(port.Mode>>8), byte(port.LEDs), byte(port.LEDs>>8))
		}
		return body
	case temperatureEndpoint:
		count := min(len(controller.probes), maxProbes)
		body := []byte{0x10, 0x00, byte(count)}
		for _, probe := range controller.probes[:count] {
			body = append(body, probe.Status, byte(uint16(probe.Tenths)), byte(uint16(probe.Tenths)>>8))
		}
		return body
	}
	return nil
}

// report makes the 384 bytes of the report of an answer: the 96 bytes of the answer, as the trace showed them, and
// 0xFF after them, which is not zero so that a test can show that the driver does not interpret it.
func (controller *Controller) report(command []byte, status byte, body []byte) []byte {
	report := bytes.Repeat([]byte{0xFF}, reportSize)
	answer := report[:answerSize]
	clear(answer)
	answer[1] = command[1]
	answer[2] = status
	copy(answer[3:], body)
	return report
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
