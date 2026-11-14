package smartmonitor

import "strconv"

const (
	startOfHeader   = 0x01
	startOfText     = 0x02
	endOfTransmit   = 0x04
	paddingByte     = 0x1A
	headerDataSize  = 128
	dataBlockSize   = 1024
	themeFileName   = "img.dat"
	crcPolynomial   = 0x1021
	blockNumberBits = 0xFF
)

// crc16 is CRC-16/XMODEM: polynomial 0x1021, initial value 0, no reflection and no final XOR.
func crc16(data []byte) uint16 {
	var crc uint16
	for _, value := range data {
		crc ^= uint16(value) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ crcPolynomial
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// frameBlock lays out a YMODEM block: the start byte, the block number, its one's complement, the data and the CRC of
// the data with the high byte first.
func frameBlock(start byte, number byte, data []byte) []byte {
	block := make([]byte, 0, 3+len(data)+2)
	block = append(block, start, number, blockNumberBits-number)
	block = append(block, data...)
	checksum := crc16(data)
	return append(block, byte(checksum>>8), byte(checksum))
}

// headerBlock is block 0, which names the file and gives the size of the theme.
func headerBlock(size int) []byte {
	data := make([]byte, headerDataSize)
	used := copy(data, themeFileName)
	used++
	copy(data[used:], strconv.Itoa(size))
	return frameBlock(startOfHeader, 0, data)
}

// dataBlocks cuts the theme into blocks of 1024 data bytes, numbered from 1. It copies the bytes it needs.
func dataBlocks(theme []byte) [][]byte {
	var blocks [][]byte
	for start := 0; start < len(theme); start += dataBlockSize {
		data := make([]byte, dataBlockSize)
		filled := copy(data, theme[start:])
		for index := filled; index < dataBlockSize; index++ {
			data[index] = paddingByte
		}
		sequence := start/dataBlockSize + 1
		blocks = append(blocks, frameBlock(startOfText, byte(sequence), data))
	}
	return blocks
}

func endOfTransmission() []byte {
	return []byte{endOfTransmit}
}

// closingBlock is the empty block 0 that ends the session.
func closingBlock() []byte {
	return frameBlock(startOfHeader, 0, make([]byte, headerDataSize))
}
