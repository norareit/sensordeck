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

// reportOf is the answer of answerOf as a full input report of 384 bytes, with 0xFF after the first 96 bytes.
func reportOf(head ...byte) []byte {
	return append(answerOf(head...), bytes.Repeat([]byte{0xFF}, reportSize-answerSize)...)
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
		{"an answer of 384 bytes", reportOf(0x00, 0x0D, 0x00), true},
		{"an answer of 97 bytes", reportOf(0x00, 0x0D, 0x00)[:97], true},
		{"an answer of 96 bytes", answerOf(0x00, 0x0D, 0x00), true},
		{"byte 0 is not checked", answerOf(0x07, 0x0D, 0x00), true},
		{"an answer of 95 bytes", answerOf(0x00, 0x0D, 0x00)[:95], false},
		{"a wrong echo", answerOf(0x00, 0x08, 0x00), false},
		{"byte 2 not zero", answerOf(0x00, 0x0D, 0x01), false},
	} {
		if got := invalidity(command, testCase.answer) == ""; got != testCase.want {
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

func TestTemperaturesOfTheTraceInAFullReportIgnoreTheBytesAfterTheAnswer(test *testing.T) {
	answer := reportOf(0x00, 0x08, 0x00, 0x10, 0x00, 0x02, 0x00, 0x75, 0x01, 0x00, 0xE3, 0x00)
	if invalidity(steps[readTemperatures], answer) != "" {
		test.Fatal("the answer is not valid")
	}
	probes, ok := temperatures(answer)
	if !ok || len(probes) != 2 {
		test.Fatalf("got %v, %v", probes, ok)
	}
	if probes[0] != (probe{available: true, celsius: 37.3}) || probes[1] != (probe{available: true, celsius: 22.7}) {
		test.Fatalf("got %+v", probes)
	}
}

func TestTemperaturesTheCountIsBoundedByTheAnswerInAFullReport(test *testing.T) {
	if _, ok := temperatures(reportOf(0x00, 0x08, 0x00, 0x10, 0x00, 31)); ok {
		test.Error("the count 31 is accepted")
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

func TestInvalidityNamesWhatIsWrong(test *testing.T) {
	command := []byte{0x08, 0x08, 0x01}
	good := make([]byte, answerSize)
	good[1] = 0x08
	cases := map[string][]byte{
		"answer of 95 bytes": good[:95],
		"echo 07":            append([]byte{0, 0x07}, good[2:]...),
		"status 03":          append([]byte{0, 0x08, 0x03}, good[3:]...),
		"":                   good,
	}
	for want, answer := range cases {
		if got := invalidity(command, answer); got != want {
			test.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestTheCommandsOfAnEndpointOnAChannel(test *testing.T) {
	cases := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"open 0x20 on channel 1", openCommand(channelOne, 0x20), []byte{0x08, 0x0D, 0x01, 0x20}},
		{"read 0x20 on channel 1", readCommand(channelOne, 0x20), []byte{0x08, 0x08, 0x01, 0x20}},
		{"close 0x20 on channel 1", closeCommand(channelOne, 0x20), []byte{0x08, 0x05, 0x01, 0x01, 0x20}},
		{"open 0x22 on channel 0", openCommand(channelZero, ledEndpoint), []byte{0x08, 0x0D, 0x00, 0x22}},
		{"close 0x22 on channel 0", closeCommand(channelZero, ledEndpoint), []byte{0x08, 0x05, 0x01, 0x00, 0x22}},
	}
	for _, each := range cases {
		if !bytes.Equal(each.got, each.want) {
			test.Errorf("%s: got % x, want % x", each.name, each.got, each.want)
		}
	}
}

func TestStepNameIsMadeFromTheCommand(test *testing.T) {
	cases := map[string][]byte{
		"close of endpoint 0x20 on channel 1": closeCommand(channelOne, 0x20),
		"open of endpoint 0x22 on channel 0":  openCommand(channelZero, ledEndpoint),
		"read of endpoint 0x1a on channel 1":  readCommand(channelOne, 0x1A),
		"enter software mode":                 enterSoftware,
		"return to hardware mode":             returnHardware,
		"colour write":                        colourCommand(1, 0, 0, 0),
	}
	for want, command := range cases {
		if got := stepName(command); got != want {
			test.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestColourCommand(test *testing.T) {
	sixteen := colourCommand(16, 0xFF, 0, 0)
	want := append([]byte{0x08, 0x06, 0x00, 0x32, 0x00, 0x00, 0x00, 0x12, 0x00}, bytes.Repeat([]byte{0xFF, 0x00, 0x00}, 16)...)
	want = append(want, make([]byte, 96-len(want))...)
	if !bytes.Equal(sixteen, want) {
		test.Errorf("16 LEDs: got % x", sixteen)
	}
	twentyFour := colourCommand(24, 1, 2, 3)
	if len(twentyFour) != 96 || !bytes.Equal(twentyFour[:9], []byte{0x08, 0x06, 0x00, 0x4A, 0x00, 0x00, 0x00, 0x12, 0x00}) ||
		!bytes.Equal(twentyFour[9:81], bytes.Repeat([]byte{1, 2, 3}, 24)) || !bytes.Equal(twentyFour[81:], make([]byte, 15)) {
		test.Errorf("24 LEDs: got % x", twentyFour)
	}
}

// portsAnswer makes an answer of the endpoint 0x20 with the entries as mode and LED count pairs.
func portsAnswer(entries ...[2]uint16) []byte {
	answer := answerOf(0x00, 0x08, 0x00, 0x0F, 0x00, byte(len(entries)))
	for position, entry := range entries {
		offset := 6 + 4*position
		if offset+4 > len(answer) {
			break
		}
		copy(answer[offset:], []byte{byte(entry[0]), byte(entry[0] >> 8), byte(entry[1]), byte(entry[1] >> 8)})
	}
	return answer
}

func TestLedCount(test *testing.T) {
	host := portsAnswer([2]uint16{0, 0}, [2]uint16{3, 0}, [2]uint16{3, 0}, [2]uint16{2, 4}, [2]uint16{2, 4}, [2]uint16{2, 4}, [2]uint16{2, 4})
	if count, err := ledCount(host); err != nil || count != 16 {
		test.Errorf("host: %d, %v", count, err)
	}
	disconnected := portsAnswer([2]uint16{2, 4}, [2]uint16{3, 8}, [2]uint16{0, 8})
	if count, err := ledCount(disconnected); err != nil || count != 4 {
		test.Errorf("only mode 2 counts: %d, %v", count, err)
	}
	if count, err := ledCount(portsAnswer([2]uint16{2, 29})); err != nil || count != 29 {
		test.Errorf("29: %d, %v", count, err)
	}
	refusals := []struct {
		name   string
		answer []byte
		want   string
	}{
		{"data type", append([]byte{0x00, 0x08, 0x00, 0x06, 0x00, 0x00}, make([]byte, 90)...), "data type 06 00"},
		{"23 entries", portsAnswer(make([][2]uint16, 23)...), "count 23 leaves no room for its entries"},
		{"no LEDs", portsAnswer([2]uint16{2, 0}), "LED count 0 is outside 1 to 29"},
		{"30 LEDs", portsAnswer([2]uint16{2, 15}, [2]uint16{2, 15}), "LED count 30 is outside 1 to 29"},
	}
	for _, each := range refusals {
		if _, err := ledCount(each.answer); err == nil || err.Error() != each.want {
			test.Errorf("%s: %v, want %q", each.name, err, each.want)
		}
	}
}

func hostPorts() []byte {
	return portsAnswer([2]uint16{0, 0}, [2]uint16{3, 0}, [2]uint16{3, 0}, [2]uint16{2, 4}, [2]uint16{2, 4}, [2]uint16{2, 4}, [2]uint16{2, 4})
}

func TestWantedConfigurationAndTheConfigurationWrite(test *testing.T) {
	for _, testCase := range []struct {
		name   string
		answer []byte
		want   []byte
	}{
		{"the host", hostPorts(), []byte{0x0D, 0x00, 0x07, 0x00, 0x00, 0x00, 0x01, 0x03, 0x01, 0x03, 0x01, 0x03, 0x01, 0x03}},
		{
			"LEDs on EXT and the ports 1 and 2",
			portsAnswer([2]uint16{2, 4}, [2]uint16{2, 4}, [2]uint16{2, 4}, [2]uint16{0, 0}, [2]uint16{0, 0}, [2]uint16{0, 0}, [2]uint16{0, 0}),
			[]byte{0x0D, 0x00, 0x07, 0x01, 0x03, 0x01, 0x03, 0x01, 0x03, 0x00, 0x00, 0x00, 0x00},
		},
		{"mode 2 with 0 LEDs and mode 3 with 4 LEDs", portsAnswer([2]uint16{2, 0}, [2]uint16{3, 4}), []byte{0x0D, 0x00, 0x02, 0x00, 0x00}},
	} {
		test.Run(testCase.name, func(test *testing.T) {
			got := wantedConfiguration(testCase.answer)
			if !bytes.Equal(got, testCase.want) {
				test.Fatalf("wanted: % x", got)
			}
			command := configurationCommand(got)
			head := append([]byte{0x08, 0x06, 0x01, byte(len(got)), 0x00, 0x00, 0x00}, got...)
			if len(command) != 96 || !bytes.Equal(command[:len(head)], head) || !bytes.Equal(command[len(head):], make([]byte, 96-len(head))) {
				test.Fatalf("write: % x", command)
			}
		})
	}
	host := configurationCommand(wantedConfiguration(hostPorts()))
	trace := []byte{0x08, 0x06, 0x01, 0x0E, 0x00, 0x00, 0x00, 0x0D, 0x00, 0x07, 0x00, 0x00, 0x00, 0x01, 0x03, 0x01, 0x03, 0x01, 0x03, 0x01, 0x03}
	if !bytes.HasPrefix(host, trace) {
		test.Errorf("the write of the host: % x", host)
	}
}

func TestPortTypeProblemNamesTheFirstEntryWithLEDsOtherThanFour(test *testing.T) {
	if name, leds := portTypeProblem(hostPorts()); name != "" || leds != 0 {
		test.Errorf("host: %q, %d", name, leds)
	}
	eight := portsAnswer([2]uint16{0, 0}, [2]uint16{2, 4}, [2]uint16{2, 8}, [2]uint16{2, 6})
	if name, leds := portTypeProblem(eight); name != "port 2" || leds != 8 {
		test.Errorf("8 LEDs: %q, %d", name, leds)
	}
	if name, leds := portTypeProblem(portsAnswer([2]uint16{2, 29})); name != "EXT" || leds != 29 {
		test.Errorf("EXT: %q, %d", name, leds)
	}
	if name, _ := portTypeProblem(portsAnswer([2]uint16{3, 8})); name != "" {
		test.Errorf("a port of mode 3 has no LEDs: %q", name)
	}
}

func TestStepNameOfTheEndpointOfTheConfiguration(test *testing.T) {
	cases := map[string][]byte{
		"write of the LED port configuration": configurationCommand(wantedConfiguration(hostPorts())),
		"command 08 15 01":                    afterPortConfiguration,
		"read of endpoint 0x1e on channel 1":  readCommand(channelOne, portConfigurationEndpoint),
		"open of endpoint 0x1e on channel 1":  openCommand(channelOne, portConfigurationEndpoint),
		"close of endpoint 0x1e on channel 1": closeCommand(channelOne, portConfigurationEndpoint),
	}
	for want, command := range cases {
		if got := stepName(command); got != want {
			test.Errorf("got %q, want %q", got, want)
		}
	}
}
