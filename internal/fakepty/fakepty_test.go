//go:build linux

package fakepty

import (
	"bytes"
	"os"
	"testing"
)

func TestBytesPassUnchangedInBothDirections(test *testing.T) {
	master, slave, _ := Open(test)
	test.Cleanup(func() { _ = master.Close(); _ = slave.Close() })
	data := []byte{0x00, 0x0A, 0x0D, 0x03, 0x7F, 0xFF, 0x0A, 0x0D}
	for _, direction := range []struct {
		name           string
		writer, reader *os.File
	}{
		{"master to slave", master, slave},
		{"slave to master", slave, master},
	} {
		if _, err := direction.writer.Write(data); err != nil {
			test.Fatalf("%s: %v", direction.name, err)
		}
		got := make([]byte, len(data))
		for filled := 0; filled < len(got); {
			count, err := direction.reader.Read(got[filled:])
			if err != nil {
				test.Fatalf("%s: %v", direction.name, err)
			}
			filled += count
		}
		if !bytes.Equal(got, data) {
			test.Fatalf("%s: got % x, want % x", direction.name, got, data)
		}
	}
}
