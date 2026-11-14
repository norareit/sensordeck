package smartmonitor

import "testing"

func sequentialBytes(count int) []byte {
	data := make([]byte, count)
	for index := range data {
		data[index] = byte(index%250 + 1)
	}
	return data
}

func TestToReportsCounts(test *testing.T) {
	cases := map[int]int{1: 1, 63: 1, 64: 1, 65: 2, 133: 3, 1029: 17}
	for size, want := range cases {
		if got := len(toReports(sequentialBytes(size))); got != want {
			test.Errorf("%d bytes gave %d reports, want %d", size, got, want)
		}
	}
}

func TestToReportsLastReportIsFilledWithZeros(test *testing.T) {
	data := sequentialBytes(133)
	reports := toReports(data)
	last := reports[2]
	for index := 0; index < 5; index++ {
		if last[index] != data[128+index] {
			test.Errorf("last report byte %d = %#x, want %#x", index, last[index], data[128+index])
		}
	}
	for index := 5; index < 64; index++ {
		if last[index] != 0 {
			test.Errorf("last report byte %d = %#x, want 0", index, last[index])
		}
	}
}

func TestToReportsKeepsOrder(test *testing.T) {
	data := sequentialBytes(200)
	for position, report := range toReports(data) {
		for index := 0; index < 64 && position*64+index < len(data); index++ {
			if report[index] != data[position*64+index] {
				test.Fatalf("report %d byte %d is out of order", position, index)
			}
		}
	}
}

func TestToReportsOneByte(test *testing.T) {
	reports := toReports([]byte{0x04})
	if len(reports) != 1 || reports[0][0] != 0x04 {
		test.Fatalf("got %v", reports)
	}
	for index := 1; index < 64; index++ {
		if reports[0][index] != 0 {
			test.Fatalf("byte %d is %#x, want 0", index, reports[0][index])
		}
	}
}
