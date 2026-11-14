//go:build linux

package fakecommander

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/hidraw"
)

const limit = 100 * time.Millisecond

// commands are the eight commands of the sequence, after the report ID byte.
var commands = [8][]byte{
	{0x08, 0x01, 0x03, 0x00, 0x02}, {0x08, 0x0D, 0x00, 0x17}, {0x08, 0x08, 0x00}, {0x08, 0x05, 0x01},
	{0x08, 0x0D, 0x00, 0x21}, {0x08, 0x08, 0x00}, {0x08, 0x05, 0x01}, {0x08, 0x01, 0x03, 0x00, 0x01},
}

type harness struct {
	test       *testing.T
	fake       *Fake
	controller *Controller
	file       *os.File
}

func newHarness(test *testing.T) *harness {
	test.Helper()
	root := test.TempDir()
	fake := New(test, filepath.Join(root, "sys"), filepath.Join(root, "dev"))
	controller := fake.Add(9, 10)
	file, err := hidraw.OpenNode(filepath.Join(fake.devRoot, "hidraw9"))
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { _ = file.Close() })
	return &harness{test: test, fake: fake, controller: controller, file: file}
}

// send writes the command as the driver would, as 97 bytes in one write, and gives the answer, or nil when none
// arrived within the limit.
func (harness *harness) send(command []byte) []byte {
	harness.test.Helper()
	return exchange(harness.test, harness.file, command)
}

func exchange(test *testing.T, file *os.File, command []byte) []byte {
	test.Helper()
	written := make([]byte, writeSize)
	copy(written[1:], command)
	if _, err := file.Write(written); err != nil {
		test.Fatal(err)
	}
	if err := file.SetReadDeadline(time.Now().Add(limit)); err != nil {
		test.Fatal(err)
	}
	answer := make([]byte, answerSize)
	count, err := file.Read(answer)
	if err != nil || count != answerSize {
		return nil
	}
	return answer
}

// padded is the head of an answer followed by zeros up to 96 bytes.
func padded(head ...byte) []byte {
	answer := make([]byte, answerSize)
	copy(answer, head)
	return answer
}

func TestTheEightCommandsGetTheAnswersOfTheTrace(test *testing.T) {
	harness := newHarness(test)
	want := [8][]byte{
		padded(0x00, 0x01, 0x00), padded(0x00, 0x0D, 0x00),
		padded(0x00, 0x08, 0x00, 0x06, 0x00, 0x06), padded(0x00, 0x05, 0x00),
		padded(0x00, 0x0D, 0x00),
		padded(0x00, 0x08, 0x00, 0x10, 0x00, 0x02, 0x00, 0x75, 0x01, 0x00, 0xE3, 0x00),
		padded(0x00, 0x05, 0x00), padded(0x00, 0x01, 0x00),
	}
	for step, command := range commands {
		if got := harness.send(command); !bytes.Equal(got, want[step]) {
			test.Fatalf("step %d: got % x, want % x", step+1, got, want[step])
		}
		switch step {
		case 0:
			if harness.controller.Mode() != SoftwareMode {
				test.Fatal("the mode is not software after step 1")
			}
		case 7:
			if harness.controller.Mode() != HardwareMode {
				test.Fatal("the mode is not hardware after step 8")
			}
		}
	}
	if writes := harness.controller.Writes(); len(writes) != 8 || writes[7][0] != 0x00 || writes[7][5] != 0x01 {
		test.Fatalf("got %d writes: % x", len(writes), writes)
	}
}

func TestModeIsNoneBeforeAnyCommand(test *testing.T) {
	if mode := newHarness(test).controller.Mode(); mode != NoMode {
		test.Fatalf("mode %d", mode)
	}
}

func TestSetProbesChangesTheAnswerOfStepSix(test *testing.T) {
	harness := newHarness(test)
	harness.controller.SetProbes(Probe{Status: 1, Tenths: -100}, Probe{Tenths: 5})
	for step := 0; step < 4; step++ {
		harness.send(commands[step])
	}
	answer := harness.send(commands[4])
	if answer == nil {
		test.Fatal("no answer to step 5")
	}
	answer = harness.send(commands[5])
	if !bytes.Equal(answer, padded(0x00, 0x08, 0x00, 0x10, 0x00, 0x02, 0x01, 0x9C, 0xFF, 0x00, 0x05, 0x00)) {
		test.Fatalf("got % x", answer)
	}
}

func TestEachFailureChangesTheAnswerOfItsStep(test *testing.T) {
	for _, testCase := range []struct {
		name    string
		failure Failure
		check   func(answer []byte) bool
	}{
		{"wrong echo", WrongEcho, func(answer []byte) bool { return answer[1] != 0x08 }},
		{"bad status", BadStatus, func(answer []byte) bool { return answer[2] == 0x01 }},
		{"wrong data type", WrongDataType, func(answer []byte) bool { return answer[3] == 0x06 && answer[4] == 0x00 }},
		{"count too large", CountTooLarge, func(answer []byte) bool { return answer[5] == 31 }},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			harness := newHarness(test)
			harness.controller.Fail(6, testCase.failure)
			for step := 0; step < 5; step++ {
				harness.send(commands[step])
			}
			answer := harness.send(commands[5])
			if answer == nil || !testCase.check(answer) {
				test.Fatalf("got % x", answer)
			}
		})
	}
}

func TestNoAnswerSendsNothingButStillSetsTheMode(test *testing.T) {
	harness := newHarness(test)
	harness.controller.Fail(1, NoAnswer)
	if answer := harness.send(commands[0]); answer != nil {
		test.Fatalf("got % x", answer)
	}
	if harness.controller.Mode() != SoftwareMode {
		test.Fatal("the mode command did not set the mode")
	}
}

func TestOnStepRunsAfterTheWriteAndBeforeTheAnswer(test *testing.T) {
	harness := newHarness(test)
	arrived := make(chan int, 1)
	harness.controller.OnStep(1, func() {
		writes, _ := harness.controller.WaitWrites(1, limit)
		arrived <- len(writes)
	})
	answer := harness.send(commands[0])
	if answer == nil || <-arrived != 1 {
		test.Fatal("the action did not run once the write had arrived, before the answer")
	}
}

func TestAWriteOnInterfaceOneIsKeptAndGetsNoAnswer(test *testing.T) {
	harness := newHarness(test)
	file, err := hidraw.OpenNode(filepath.Join(harness.fake.devRoot, "hidraw10"))
	if err != nil {
		test.Fatal(err)
	}
	defer file.Close()
	if answer := exchange(test, file, commands[0]); answer != nil {
		test.Fatalf("got % x", answer)
	}
	if writes := harness.controller.WritesOnInterfaceOne(); len(writes) != 1 || len(writes[0]) != writeSize {
		test.Fatalf("got %d writes", len(writes))
	}
	if writes := harness.controller.Writes(); len(writes) != 0 {
		test.Fatalf("interface 0 got %d writes", len(writes))
	}
	if harness.controller.Mode() != NoMode {
		test.Fatal("a command on interface 1 set the mode")
	}
}

func TestBothSysfsEntriesHoldTheirInterface(test *testing.T) {
	harness := newHarness(test)
	for number, suffix := range map[string]string{"hidraw9": "/input0", "hidraw10": "/input1"} {
		content, err := os.ReadFile(filepath.Join(harness.fake.sysfsRoot, "class", "hidraw", number, "device", "uevent"))
		if err != nil {
			test.Fatal(err)
		}
		lines := strings.Split(string(content), "\n")
		var gotID, gotName, gotPhys bool
		for _, line := range lines {
			gotID = gotID || line == "HID_ID=0003:00001B1C:00000C2A"
			gotName = gotName || line == "HID_NAME=Corsair CORSAIR iCUE COMMANDER CORE XT"
			gotPhys = gotPhys || (strings.HasPrefix(line, "HID_PHYS=") && strings.HasSuffix(line, suffix))
		}
		if !gotID || !gotName || !gotPhys {
			test.Fatalf("%s: %q", number, content)
		}
	}
}

func TestRemoveNodeLeavesTheEntries(test *testing.T) {
	harness := newHarness(test)
	harness.controller.RemoveNode()
	if _, err := os.Lstat(filepath.Join(harness.fake.devRoot, "hidraw9")); !os.IsNotExist(err) {
		test.Fatalf("the node is still there: %v", err)
	}
	entry := filepath.Join(harness.fake.sysfsRoot, "class", "hidraw", "hidraw9", "device", "uevent")
	if _, err := os.Stat(entry); err != nil {
		test.Fatal(err)
	}
}

func TestAddInterfaceOneMakesOnlyThatEntry(test *testing.T) {
	root := test.TempDir()
	fake := New(test, filepath.Join(root, "sys"), filepath.Join(root, "dev"))
	controller := fake.AddInterfaceOne(4)
	entries, err := hidraw.List(fake.sysfsRoot, fake.devRoot)
	if err != nil || len(entries) != 1 || entries[0].Number != 4 || !strings.HasSuffix(entries[0].Phys, "/input1") {
		test.Fatalf("got %+v, %v", entries, err)
	}
	if writes := controller.Writes(); len(writes) != 0 {
		test.Fatalf("got %d writes", len(writes))
	}
}
