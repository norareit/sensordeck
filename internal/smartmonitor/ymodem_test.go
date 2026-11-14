package smartmonitor

import (
	"bytes"
	"testing"
)

func TestCRC16(test *testing.T) {
	if got := crc16([]byte("123456789")); got != 0x31C3 {
		test.Errorf("check value %#04x", got)
	}
	if got := crc16(nil); got != 0 {
		test.Errorf("empty %#04x", got)
	}
	if got := crc16(make([]byte, 128)); got != 0 {
		test.Errorf("zeros %#04x", got)
	}
}

func TestHeaderBlock(test *testing.T) {
	block := headerBlock(2500)
	if len(block) != 133 {
		test.Fatalf("length %d", len(block))
	}
	if !bytes.Equal(block[:3], []byte{0x01, 0x00, 0xFF}) {
		test.Errorf("prefix % X", block[:3])
	}
	name := []byte{0x69, 0x6D, 0x67, 0x2E, 0x64, 0x61, 0x74, 0x00, 0x32, 0x35, 0x30, 0x30, 0x00}
	if !bytes.Equal(block[3:3+len(name)], name) {
		test.Errorf("data % X", block[3:3+len(name)])
	}
	if !bytes.Equal(block[3+len(name):131], make([]byte, 131-3-len(name))) {
		test.Error("data is not filled with zeros")
	}
	if !bytes.Equal(block[131:], []byte{0x82, 0xBF}) {
		test.Errorf("crc % X", block[131:])
	}
}

func testTheme(size int) []byte {
	theme := make([]byte, size)
	for index := range theme {
		theme[index] = byte(index)
	}
	return theme
}

func TestDataBlocksOfTheExampleTheme(test *testing.T) {
	theme := testTheme(2500)
	original := append([]byte(nil), theme...)
	blocks := dataBlocks(theme)
	if len(blocks) != 3 {
		test.Fatalf("%d blocks", len(blocks))
	}
	prefixes := [][]byte{{0x02, 0x01, 0xFE}, {0x02, 0x02, 0xFD}, {0x02, 0x03, 0xFC}}
	crcs := [][]byte{{0xC2, 0xE0}, {0xC2, 0xE0}, {0xFC, 0x8B}}
	for index, block := range blocks {
		if len(block) != 1029 {
			test.Errorf("block %d length %d", index+1, len(block))
		}
		if !bytes.Equal(block[:3], prefixes[index]) {
			test.Errorf("block %d prefix % X", index+1, block[:3])
		}
		if !bytes.Equal(block[1027:], crcs[index]) {
			test.Errorf("block %d crc % X", index+1, block[1027:])
		}
	}
	last := blocks[2]
	if !bytes.Equal(last[3:3+452], theme[2048:]) {
		test.Error("last block does not hold the last 452 theme bytes")
	}
	if !bytes.Equal(last[3+452:1027], bytes.Repeat([]byte{0x1A}, 572)) {
		test.Error("last block is not filled with 0x1A")
	}
	if !bytes.Equal(theme, original) {
		test.Error("the theme was changed")
	}
}

func TestDataBlocksOfAMultipleOf1024HaveNoExtraBlock(test *testing.T) {
	if got := len(dataBlocks(testTheme(2048))); got != 2 {
		test.Errorf("%d blocks", got)
	}
}

func TestDataBlockNumbersWrapAround(test *testing.T) {
	blocks := dataBlocks(testTheme(257 * 1024))
	want := map[int][]byte{255: {0xFF, 0x00}, 256: {0x00, 0xFF}, 257: {0x01, 0xFE}}
	for sequence, bytesWanted := range want {
		if got := blocks[sequence-1][1:3]; !bytes.Equal(got, bytesWanted) {
			test.Errorf("block %d has % X", sequence, got)
		}
	}
}

func TestEndOfTransmissionAndClosingBlock(test *testing.T) {
	if !bytes.Equal(endOfTransmission(), []byte{0x04}) {
		test.Error("end of transmission")
	}
	want := append([]byte{0x01, 0x00, 0xFF}, make([]byte, 130)...)
	if !bytes.Equal(closingBlock(), want) {
		test.Errorf("closing block % X", closingBlock())
	}
}
