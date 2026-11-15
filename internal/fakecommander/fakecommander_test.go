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

// send writes the command as the driver would, as 97 bytes in one write, and gives the report of 384 bytes that
// answers it in one read, or nil when none arrived within the limit or the read gave fewer bytes.
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
	report := make([]byte, reportSize)
	count, err := file.Read(report)
	if err != nil || count != reportSize {
		return nil
	}
	return report
}

// padded is the report of 384 bytes that holds the head of an answer, followed by zeros up to 96 bytes and by 0xFF
// after them.
func padded(head ...byte) []byte {
	report := bytes.Repeat([]byte{0xFF}, reportSize)
	clear(report[:answerSize])
	copy(report, head)
	return report
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

var (
	enterSoftware  = []byte{0x08, 0x01, 0x03, 0x00, 0x02}
	returnHardware = []byte{0x08, 0x01, 0x03, 0x00, 0x01}
)

// openReadClose makes the four exchanges of a reading of the endpoint on channel 1 and gives the answer of the read.
func (harness *harness) openReadClose(endpoint byte) []byte {
	harness.test.Helper()
	harness.send([]byte{0x08, 0x05, 0x01, 0x01, endpoint})
	harness.send([]byte{0x08, 0x0D, 0x01, endpoint})
	answer := harness.send([]byte{0x08, 0x08, 0x01, endpoint})
	harness.send([]byte{0x08, 0x05, 0x01, 0x01, endpoint})
	return answer
}

func TestEndpointsOnChannelOneGiveTheirAnswers(test *testing.T) {
	harness := newHarness(test)
	harness.send(enterSoftware)
	host := padded(0x00, 0x08, 0x00, 0x0F, 0x00, 0x07,
		0, 0, 0, 0, 3, 0, 0, 0, 3, 0, 0, 0, 2, 0, 4, 0, 2, 0, 4, 0, 2, 0, 4, 0, 2, 0, 4, 0)
	if got := harness.openReadClose(0x20); !bytes.Equal(got, host) {
		test.Errorf("0x20: got % x", got)
	}
	harness.controller.SetPorts(Port{2, 8}, Port{2, 8}, Port{2, 8})
	three := padded(0x00, 0x08, 0x00, 0x0F, 0x00, 0x03, 2, 0, 8, 0, 2, 0, 8, 0, 2, 0, 8, 0)
	if got := harness.openReadClose(0x20); !bytes.Equal(got, three) {
		test.Errorf("0x20 with three ports: got % x", got)
	}
	if got, want := harness.openReadClose(0x1A), padded(0x00, 0x08, 0x00, 0x09, 0x00, 0x06, 1, 1, 1, 1, 1, 1); !bytes.Equal(got, want) {
		test.Errorf("0x1A: got % x", got)
	}
	if got, want := harness.openReadClose(0x17), padded(0x00, 0x08, 0x00, 0x06, 0x00, 0x06); !bytes.Equal(got, want) {
		test.Errorf("0x17: got % x", got)
	}
	want := padded(0x00, 0x08, 0x00, 0x10, 0x00, 0x02, 0x00, 0x75, 0x01, 0x00, 0xE3, 0x00)
	if got := harness.openReadClose(0x21); !bytes.Equal(got, want) {
		test.Errorf("0x21: got % x", got)
	}
}

func TestAnOpenOnAChannelThatHasAnEndpointOpenAnswersStatusThree(test *testing.T) {
	harness := newHarness(test)
	harness.send(enterSoftware)
	harness.send([]byte{0x08, 0x0D, 0x00, 0x22})
	for _, endpoint := range []byte{0x21, 0x22} {
		if got := harness.send([]byte{0x08, 0x0D, 0x00, endpoint}); got == nil || got[2] != 0x03 {
			test.Fatalf("open of %#x: got % x", endpoint, got)
		}
		if harness.controller.OpenEndpoint(0) != 0x22 {
			test.Fatal("the open endpoint changed")
		}
	}
	harness.send([]byte{0x08, 0x05, 0x01, 0x00, 0x22})
	if got := harness.send([]byte{0x08, 0x0D, 0x00, 0x21}); got == nil || got[2] != 0x00 || harness.controller.OpenEndpoint(0) != 0x21 {
		test.Fatalf("after the close: got % x", got)
	}
}

func TestSetOpenEndpointBlocksTheLedEndpoint(test *testing.T) {
	harness := newHarness(test)
	harness.send(enterSoftware)
	harness.controller.SetOpenEndpoint(0, 0x21)
	if got := harness.send([]byte{0x08, 0x0D, 0x00, 0x22}); got == nil || got[2] != 0x03 {
		test.Fatalf("got % x", got)
	}
}

func TestOutsideSoftwareModeOnlyTheModeCommandsAnswerZero(test *testing.T) {
	commandsOutside := [][]byte{{0x08, 0x0D, 0x00, 0x22}, {0x08, 0x08, 0x00}, {0x08, 0x05, 0x01, 0x00, 0x22}, {0x08, 0x06, 0x00}}
	check := func(when string, harness *harness) {
		for _, command := range commandsOutside {
			if got := harness.send(command); got == nil || got[2] != 0x03 {
				test.Errorf("%s, % x: got % x", when, command, got)
			}
		}
	}
	harness := newHarness(test)
	check("before any mode command", harness)
	harness.send(enterSoftware)
	if got := harness.send(returnHardware); got == nil || got[2] != 0x00 {
		test.Fatalf("return to hardware mode: got % x", got)
	}
	check("after the return", harness)
}

func TestLightingFollowsTheModeAndTheColourWrites(test *testing.T) {
	harness := newHarness(test)
	controller := harness.controller
	lighting := func(want Lighting, colour [3]byte) {
		test.Helper()
		if got, gotColour := controller.Lighting(); got != want || (want == SentColour && gotColour != colour) {
			test.Errorf("lighting %d, colour %v", got, gotColour)
		}
	}
	lighting(StoredLighting, [3]byte{})
	harness.send(enterSoftware)
	lighting(Dark, [3]byte{})
	if got := harness.send([]byte{0x08, 0x06, 0x00, 0x32}); got == nil || got[2] != 0x03 {
		test.Fatalf("a colour write with nothing open: got % x", got)
	}
	if len(controller.ColourWrites()) != 0 {
		test.Fatal("a write was kept with nothing open")
	}
	harness.send([]byte{0x08, 0x0D, 0x00, 0x22})
	command := append([]byte{0x08, 0x06, 0x00, 0x32, 0x00, 0x00, 0x00, 0x12, 0x00}, bytes.Repeat([]byte{0xFF, 0x00, 0x00}, 16)...)
	if got := harness.send(command); got == nil || got[2] != 0x00 {
		test.Fatalf("a colour write: got % x", got)
	}
	lighting(SentColour, [3]byte{0xFF, 0, 0})
	writes := controller.ColourWrites()
	if len(writes) != 1 || len(writes[0]) != answerSize || !bytes.Equal(writes[0][:len(command)], command) {
		test.Fatalf("writes % x", writes)
	}
	harness.send(returnHardware)
	lighting(StoredLighting, [3]byte{})
}

func TestTheSoftwareModeTimeout(test *testing.T) {
	open := func(harness *harness) {
		harness.send(enterSoftware)
		harness.send([]byte{0x08, 0x0D, 0x00, 0x22})
		harness.send([]byte{0x08, 0x06, 0x00, 0x03, 0x00, 0x00, 0x00, 0x12, 0x00, 1, 2, 3})
	}
	test.Run("fires", func(test *testing.T) {
		harness := newHarness(test)
		harness.controller.SetSoftwareModeTimeout(100 * time.Millisecond)
		open(harness)
		time.Sleep(300 * time.Millisecond)
		if lighting, _ := harness.controller.Lighting(); harness.controller.Mode() != HardwareMode ||
			harness.controller.OpenEndpoint(0) != 0 || lighting != StoredLighting {
			test.Fatal("the controller did not fall back to hardware mode")
		}
		harness.send(enterSoftware)
		if lighting, _ := harness.controller.Lighting(); lighting != Dark {
			test.Fatalf("lighting %d after a new software mode", lighting)
		}
	})
	test.Run("commands keep it awake", func(test *testing.T) {
		harness := newHarness(test)
		harness.controller.SetSoftwareModeTimeout(100 * time.Millisecond)
		open(harness)
		for range 6 {
			time.Sleep(50 * time.Millisecond)
			harness.send([]byte{0x08, 0x05, 0x01, 0x01, 0x21})
		}
		if harness.controller.Mode() != SoftwareMode {
			test.Fatal("the controller fell back")
		}
	})
	test.Run("off by default", func(test *testing.T) {
		harness := newHarness(test)
		open(harness)
		time.Sleep(300 * time.Millisecond)
		if harness.controller.Mode() != SoftwareMode {
			test.Fatal("the controller fell back")
		}
	})
}

func TestFailCommandChangesOnlyTheCommandsWithThePrefix(test *testing.T) {
	for _, testCase := range []struct {
		name    string
		failure Failure
		check   func(answer []byte) bool
	}{
		{"no answer", NoAnswer, func(answer []byte) bool { return answer == nil }},
		{"wrong echo", WrongEcho, func(answer []byte) bool { return answer[1] != 0x08 }},
		{"bad status", BadStatus, func(answer []byte) bool { return answer[2] == 0x01 }},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			harness := newHarness(test)
			harness.controller.FailCommand([]byte{0x08, 0x08, 0x01, 0x21}, testCase.failure)
			harness.send(enterSoftware)
			if answer := harness.openReadClose(0x17); answer == nil || answer[2] != 0x00 || answer[1] != 0x08 {
				test.Fatalf("another read: got % x", answer)
			}
			answer := harness.openReadClose(0x21)
			if !testCase.check(answer) {
				test.Fatalf("got % x", answer)
			}
		})
	}
}

func TestOnCommandRunsAfterTheWriteAndBeforeTheAnswer(test *testing.T) {
	harness := newHarness(test)
	arrived := make(chan int, 4)
	harness.controller.OnCommand([]byte{0x08, 0x0D, 0x00, 0x22}, func() {
		writes, _ := harness.controller.WaitWrites(2, limit)
		arrived <- len(writes)
	})
	harness.send(enterSoftware)
	answer := harness.send([]byte{0x08, 0x0D, 0x00, 0x22})
	if answer == nil || <-arrived != 2 {
		test.Fatal("the action did not run once the write had arrived, before the answer")
	}
}
