// Package commandercore reads the temperature probes of a Corsair iCUE Commander Core XT over its hidraw node. What
// the controller expects on the wire is described in docs/commander-core-xt.md.
package commandercore

import (
	"bytes"
	"fmt"
)

const (
	vendor  = 0x1B1C
	product = 0x0C2A
)

const (
	// answerSize is what the driver interprets of an answer, and reportSize the input report of the report descriptor
	// of interface 0, which a read takes whole.
	answerSize = 96
	reportSize = 384
	// maxProbes is the largest count whose entries fit into an answer: 6 + 3·30 = 96.
	maxProbes = (answerSize - 6) / 3

	probeOffset = 6
	probeSize   = 3
)

// steps are the eight commands of a reading, after the report ID byte, in their order: enter software mode, open the
// endpoint of the fan speeds, read it, close it, open the endpoint of the temperatures, read it, close it, and return
// to hardware mode.
var steps = [8][]byte{
	enterSoftware,
	{0x08, 0x0D, 0x00, 0x17},
	{0x08, 0x08, 0x00},
	{0x08, 0x05, 0x01},
	{0x08, 0x0D, 0x00, 0x21},
	{0x08, 0x08, 0x00},
	{0x08, 0x05, 0x01},
	returnHardware,
}

const (
	channelZero, channelOne byte = 0x00, 0x01

	ledEndpoint   byte = 0x22
	portsEndpoint byte = 0x20

	// maxLEDs is the count of triples that fit into a write of 97 bytes: 9 + 3·29 = 96.
	maxLEDs = (answerSize - colourHeaderSize) / 3
	// colourHeaderSize is the bytes of a colour write before its first triple.
	colourHeaderSize = 9
	// portsHeaderSize and portSize are the bytes of an answer of the endpoint 0x20 before its entries, and one entry.
	portsHeaderSize = 6
	portSize        = 4
	// connectedMode is the mode of an entry of the endpoint 0x20 that has a connected port.
	connectedMode = 2
)

var (
	enterSoftware  = []byte{0x08, 0x01, 0x03, 0x00, 0x02}
	returnHardware = []byte{0x08, 0x01, 0x03, 0x00, 0x01}
	portsDataType  = []byte{0x0F, 0x00}
	colourDataType = []byte{0x12, 0x00}
)

// openCommand opens the endpoint on the channel.
func openCommand(channel, endpoint byte) []byte { return []byte{0x08, 0x0D, channel, endpoint} }

// readCommand reads the endpoint that is open on the channel.
func readCommand(channel, endpoint byte) []byte { return []byte{0x08, 0x08, channel, endpoint} }

// closeCommand closes the endpoint on the channel. The endpoint byte is the form of the trace, and whether it matters
// is not known.
func closeCommand(channel, endpoint byte) []byte { return []byte{0x08, 0x05, 0x01, channel, endpoint} }

// ledCount gives the LED count of an answer of the endpoint 0x20 that is valid: the sum of the LED counts of the
// entries whose mode is connected, which must be from 1 to the 29 that a write holds.
func ledCount(answer []byte) (int, error) {
	if !bytes.Equal(answer[3:5], portsDataType) {
		return 0, fmt.Errorf("data type %02x %02x", answer[3], answer[4])
	}
	entries := int(answer[5])
	if portsHeaderSize+portSize*entries > answerSize {
		return 0, fmt.Errorf("count %d leaves no room for its entries", entries)
	}
	total := 0
	for position := range entries {
		entry := answer[portsHeaderSize+portSize*position:]
		if mode := int(entry[0]) | int(entry[1])<<8; mode == connectedMode {
			total += int(entry[2]) | int(entry[3])<<8
		}
	}
	if total < 1 || total > maxLEDs {
		return 0, fmt.Errorf("LED count %d is outside 1 to %d", total, maxLEDs)
	}
	return total, nil
}

// colourCommand gives the 96 data bytes of the colour write that gives every one of count LEDs the colour.
func colourCommand(count int, red, green, blue byte) []byte {
	command := make([]byte, answerSize)
	length := len(colourDataType) + 3*count
	copy(command, []byte{0x08, 0x06, 0x00, byte(length), byte(length >> 8), 0x00, 0x00})
	copy(command[7:], colourDataType)
	for position := range count {
		copy(command[colourHeaderSize+3*position:], []byte{red, green, blue})
	}
	return command
}

// stepName names the exchange of the command for an error that says what failed.
func stepName(command []byte) string {
	switch {
	case bytes.Equal(command, enterSoftware):
		return "enter software mode"
	case bytes.Equal(command, returnHardware):
		return "return to hardware mode"
	case bytes.HasPrefix(command, []byte{0x08, 0x06, 0x00}):
		return "colour write"
	case bytes.HasPrefix(command, []byte{0x08, 0x05, 0x01}):
		return fmt.Sprintf("close of endpoint 0x%02x on channel %d", command[4], command[3])
	case bytes.HasPrefix(command, []byte{0x08, 0x0D}):
		return fmt.Sprintf("open of endpoint 0x%02x on channel %d", command[3], command[2])
	case bytes.HasPrefix(command, []byte{0x08, 0x08}):
		return fmt.Sprintf("read of endpoint 0x%02x on channel %d", command[3], command[2])
	}
	return fmt.Sprintf("command % x", command)
}

// Positions in steps of the commands that a reading treats on their own.
const (
	enterSoftwareMode    = 0
	readTemperatures     = 5
	returnToHardwareMode = 7
)

// temperatureDataType is what bytes 3 and 4 of the answer of the temperature endpoint hold.
var temperatureDataType = []byte{0x10, 0x00}

// probe is one temperature probe of the answer of the temperature endpoint.
type probe struct {
	available bool
	celsius   float64
}

// invalidity names what is wrong with an answer of the command, or gives the empty text for one that is valid: at
// least 96 bytes, the second byte of the command echoed in byte 1, and byte 2, taken as a status, zero.
func invalidity(command, answer []byte) string {
	switch {
	case len(answer) < answerSize:
		return fmt.Sprintf("answer of %d bytes", len(answer))
	case answer[1] != command[1]:
		return fmt.Sprintf("echo %02x", answer[1])
	case answer[2] != 0x00:
		return fmt.Sprintf("status %02x", answer[2])
	}
	return ""
}

// temperatures reads the probes of an answer of the temperature endpoint that is valid. It gives false when the data
// type is not the temperatures or the count leaves no room for its entries. A probe whose status is not 0x00 is
// unavailable and keeps its place.
func temperatures(answer []byte) ([]probe, bool) {
	if !bytes.Equal(answer[3:5], temperatureDataType) {
		return nil, false
	}
	count := int(answer[5])
	if count > maxProbes {
		return nil, false
	}
	probes := make([]probe, count)
	for position := range probes {
		entry := answer[probeOffset+probeSize*position:]
		if entry[0] != 0x00 {
			continue
		}
		tenths := int16(uint16(entry[1]) | uint16(entry[2])<<8)
		probes[position] = probe{available: true, celsius: float64(tenths) / 10}
	}
	return probes, true
}
