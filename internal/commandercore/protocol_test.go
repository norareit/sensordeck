//go:build linux

package commandercore

import (
	"bytes"
	"testing"
)

func answerOf(head ...byte) []byte {
	answer := make([]byte, answerSize)
	copy(answer, head)
	return answer
}

func TestEncodeGivesTheReportIDTheCommandAndZeros(test *testing.T) {
	for position, command := range steps {
		written := encode(command)
		if len(written) != 97 || written[0] != 0x00 || !bytes.Equal(written[1:1+len(command)], command) {
			test.Fatalf("step %d: % x", position+1, written)
		}
		if !bytes.Equal(written[1+len(command):], make([]byte, 96-len(command))) {
			test.Fatalf("step %d: the bytes after the command are not zero", position+1)
		}
	}
}

func TestTheStepsAreTheEightCommandsOfTheTrace(test *testing.T) {
	want := [][]byte{
		{0x08, 0x01, 0x03, 0x00, 0x02}, {0x08, 0x0D, 0x00, 0x17}, {0x08, 0x08, 0x00}, {0x08, 0x05, 0x01},
		{0x08, 0x0D, 0x00, 0x21}, {0x08, 0x08, 0x00}, {0x08, 0x05, 0x01}, {0x08, 0x01, 0x03, 0x00, 0x01},
	}
	for position := range want {
		if !bytes.Equal(steps[position], want[position]) {
			test.Errorf("step %d: % x", position+1, steps[position])
		}
	}
}

func TestValid(test *testing.T) {
	command := []byte{0x08, 0x0D, 0x00, 0x17}
	for _, testCase := range []struct {
		name   string
		answer []byte
		want   bool
	}{
		{"a good answer", answerOf(0x00, 0x0D, 0x00), true},
		{"byte 0 is not checked", answerOf(0x07, 0x0D, 0x00), true},
		{"a short answer", answerOf(0x00, 0x0D, 0x00)[:95], false},
		{"a long answer", append(answerOf(0x00, 0x0D, 0x00), 0), false},
		{"a wrong echo", answerOf(0x00, 0x08, 0x00), false},
		{"byte 2 not zero", answerOf(0x00, 0x0D, 0x01), false},
	} {
		if got := valid(command, testCase.answer); got != testCase.want {
			test.Errorf("%s: got %v", testCase.name, got)
		}
	}
}

func TestTemperaturesOfTheTrace(test *testing.T) {
	answer := answerOf(0x00, 0x08, 0x00, 0x10, 0x00, 0x02, 0x00, 0x75, 0x01, 0x00, 0xE3, 0x00)
	probes, ok := temperatures(answer)
	if !ok || len(probes) != 2 {
		test.Fatalf("got %v, %v", probes, ok)
	}
	if probes[0] != (probe{available: true, celsius: 37.3}) || probes[1] != (probe{available: true, celsius: 22.7}) {
		test.Fatalf("got %+v", probes)
	}
}

func TestTemperaturesRefuseAWrongDataType(test *testing.T) {
	if _, ok := temperatures(answerOf(0x00, 0x08, 0x00, 0x06, 0x00, 0x02)); ok {
		test.Fatal("accepted")
	}
}

func TestTemperaturesTheCountMustLeaveRoomForItsEntries(test *testing.T) {
	if _, ok := temperatures(answerOf(0x00, 0x08, 0x00, 0x10, 0x00, 31)); ok {
		test.Error("the count 31 is accepted")
	}
	probes, ok := temperatures(answerOf(0x00, 0x08, 0x00, 0x10, 0x00, 30))
	if !ok || len(probes) != 30 {
		test.Errorf("the count 30: got %d, %v", len(probes), ok)
	}
}

func TestTemperaturesAProbeWithAStatusIsUnavailableAndKeepsItsNumber(test *testing.T) {
	answer := answerOf(0x00, 0x08, 0x00, 0x10, 0x00, 0x02, 0x01, 0x75, 0x01, 0x00, 0xE3, 0x00)
	probes, ok := temperatures(answer)
	if !ok || len(probes) != 2 || probes[0].available || probes[1] != (probe{available: true, celsius: 22.7}) {
		test.Fatalf("got %+v, %v", probes, ok)
	}
}

func TestTemperaturesAreSigned(test *testing.T) {
	answer := answerOf(0x00, 0x08, 0x00, 0x10, 0x00, 0x01, 0x00, 0x9C, 0xFF)
	probes, ok := temperatures(answer)
	if !ok || len(probes) != 1 || probes[0] != (probe{available: true, celsius: -10}) {
		test.Fatalf("got %+v, %v", probes, ok)
	}
}
