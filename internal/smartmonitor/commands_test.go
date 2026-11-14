package smartmonitor

import "testing"

func TestCommands(test *testing.T) {
	var reset, ymodem Report
	copy(reset[:], []byte{0x01, 0x72, 0x65, 0x73, 0x65, 0x74})
	copy(ymodem[:], []byte{0x79, 0x6D, 0x6F, 0x64, 0x65, 0x6D})
	if ResetCommand() != reset {
		test.Errorf("reset is % X", ResetCommand())
	}
	if YmodemCommand() != ymodem {
		test.Errorf("ymodem is % X", YmodemCommand())
	}
}
