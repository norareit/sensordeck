package smartmonitor

import (
	"reflect"
	"testing"
)

func TestEncodeValuesExample(t *testing.T) {
	reports := EncodeValues([]Value{{Tag: 0x01, Value: 0x1234}, {Tag: 0xFF, Value: 0x0001}})
	if len(reports) != 1 {
		t.Fatalf("got %d reports", len(reports))
	}
	var want Report
	copy(want[:], []byte{0x02, 0x02, 0x01, 0x12, 0x34, 0xFF, 0x00, 0x01})
	if reports[0] != want {
		t.Fatalf("got % X", reports[0][:])
	}
}

func pairs(count int) []Value {
	values := make([]Value, count)
	for index := range values {
		values[index] = Value{Tag: uint8(index), Value: uint16(index * 3)}
	}
	return values
}

func TestEncodeValuesSplitsInOrder(t *testing.T) {
	cases := map[int][]int{20: {20}, 21: {20, 1}, 40: {20, 20}, 41: {20, 20, 1}}
	for count, wantCounts := range cases {
		reports := EncodeValues(pairs(count))
		var gotCounts []int
		tag := 0
		for _, report := range reports {
			gotCounts = append(gotCounts, int(report[1]))
			for pair := 0; pair < int(report[1]); pair++ {
				if int(report[2+pair*3]) != tag {
					t.Fatalf("%d pairs: tag %d out of order", count, tag)
				}
				tag++
			}
		}
		if !reflect.DeepEqual(gotCounts, wantCounts) {
			t.Errorf("%d pairs gave counts %v, want %v", count, gotCounts, wantCounts)
		}
	}
}

func TestEncodeValuesEmptyGivesNoReport(t *testing.T) {
	if got := EncodeValues(nil); len(got) != 0 {
		t.Errorf("nil gave %d reports", len(got))
	}
	if got := EncodeValues([]Value{}); len(got) != 0 {
		t.Errorf("empty gave %d reports", len(got))
	}
}

func TestEncodeValuesDoesNotChangeInput(t *testing.T) {
	input := pairs(45)
	before := append([]Value(nil), input...)
	EncodeValues(input)
	if !reflect.DeepEqual(before, input) {
		t.Error("the input was changed")
	}
}

func TestEncodeValuesEncodesDuplicateTagTwice(t *testing.T) {
	reports := EncodeValues([]Value{{Tag: 7, Value: 1}, {Tag: 7, Value: 2}})
	if reports[0][1] != 2 || reports[0][2] != 7 || reports[0][5] != 7 {
		t.Fatalf("got % X", reports[0][:8])
	}
}
