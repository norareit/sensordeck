package smartmonitor

import (
	"reflect"
	"testing"
)

func TestEncodeValuesExample(test *testing.T) {
	reports := EncodeValues([]Value{{Tag: 0x01, Value: 0x1234}, {Tag: 0xFF, Value: 0x0001}})
	if len(reports) != 1 {
		test.Fatalf("got %d reports", len(reports))
	}
	var want Report
	copy(want[:], []byte{0x02, 0x02, 0x01, 0x12, 0x34, 0xFF, 0x00, 0x01})
	if reports[0] != want {
		test.Fatalf("got % X", reports[0][:])
	}
}

func TestEncodeValuesSigned(test *testing.T) {
	reports := EncodeValues([]Value{{Tag: 1, Value: -1}, {Tag: 2, Value: -5}, {Tag: 3, Value: -32768}, {Tag: 4, Value: 32767}, {Tag: 5, Value: 0}})
	var want Report
	copy(want[:], []byte{0x02, 0x05, 0x01, 0xFF, 0xFF, 0x02, 0xFF, 0xFB, 0x03, 0x80, 0x00, 0x04, 0x7F, 0xFF, 0x05, 0x00, 0x00})
	if len(reports) != 1 || reports[0] != want {
		test.Fatalf("got % X", reports[0][:])
	}
}

func pairs(count int) []Value {
	values := make([]Value, count)
	for index := range values {
		values[index] = Value{Tag: uint8(index), Value: int16(index * 3)}
	}
	return values
}

func TestEncodeValuesSplitsInOrder(test *testing.T) {
	cases := map[int][]int{20: {20}, 21: {20, 1}, 40: {20, 20}, 41: {20, 20, 1}}
	for count, wantCounts := range cases {
		reports := EncodeValues(pairs(count))
		var gotCounts []int
		tag := 0
		for _, report := range reports {
			gotCounts = append(gotCounts, int(report[1]))
			for pair := 0; pair < int(report[1]); pair++ {
				if int(report[2+pair*3]) != tag {
					test.Fatalf("%d pairs: tag %d out of order", count, tag)
				}
				tag++
			}
		}
		if !reflect.DeepEqual(gotCounts, wantCounts) {
			test.Errorf("%d pairs gave counts %v, want %v", count, gotCounts, wantCounts)
		}
	}
}

func TestEncodeValuesEmptyGivesNoReport(test *testing.T) {
	if got := EncodeValues(nil); len(got) != 0 {
		test.Errorf("nil gave %d reports", len(got))
	}
	if got := EncodeValues([]Value{}); len(got) != 0 {
		test.Errorf("empty gave %d reports", len(got))
	}
}

func TestEncodeValuesDoesNotChangeInput(test *testing.T) {
	input := pairs(45)
	before := append([]Value(nil), input...)
	EncodeValues(input)
	if !reflect.DeepEqual(before, input) {
		test.Error("the input was changed")
	}
}

func TestEncodeValuesEncodesDuplicateTagTwice(test *testing.T) {
	reports := EncodeValues([]Value{{Tag: 7, Value: 1}, {Tag: 7, Value: 2}})
	if reports[0][1] != 2 || reports[0][2] != 7 || reports[0][5] != 7 {
		test.Fatalf("got % X", reports[0][:8])
	}
}
