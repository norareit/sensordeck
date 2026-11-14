//go:build linux

// Package commandercore reads the temperature probes of a Corsair iCUE Commander Core XT over its hidraw node. What
// the controller expects on the wire is described in docs/commander-core-xt.md.
package commandercore

import "bytes"

const (
	answerSize = 96
	writeSize  = 1 + answerSize
	// maxProbes is the largest count whose entries fit into an answer: 6 + 3·30 = 96.
	maxProbes = (answerSize - 6) / 3

	probeOffset = 6
	probeSize   = 3
)

// steps are the eight commands of a reading, after the report ID byte, in their order: enter software mode, open the
// endpoint of the fan speeds, read it, close it, open the endpoint of the temperatures, read it, close it, and return
// to hardware mode.
var steps = [8][]byte{
	{0x08, 0x01, 0x03, 0x00, 0x02},
	{0x08, 0x0D, 0x00, 0x17},
	{0x08, 0x08, 0x00},
	{0x08, 0x05, 0x01},
	{0x08, 0x0D, 0x00, 0x21},
	{0x08, 0x08, 0x00},
	{0x08, 0x05, 0x01},
	{0x08, 0x01, 0x03, 0x00, 0x01},
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

// encode gives the 97 bytes of a write: the report ID byte 0x00, the command, and zeros.
func encode(command []byte) []byte {
	written := make([]byte, writeSize)
	copy(written[1:], command)
	return written
}

// valid tells whether the answer is one of the command: 96 bytes, the second byte of the command echoed in byte 1,
// and byte 2, taken as a status, zero.
func valid(command, answer []byte) bool {
	return len(answer) == answerSize && answer[1] == command[1] && answer[2] == 0x00
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
