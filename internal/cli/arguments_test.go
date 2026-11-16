package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/norareit/sensordeck/internal/smartmonitor"
)

func TestParseValues(test *testing.T) {
	cases := []struct {
		args []string
		want []smartmonitor.Value
	}{
		{[]string{"1=0x1234", "255=1"}, []smartmonitor.Value{{Tag: 1, Value: 0x1234}, {Tag: 255, Value: 1}}},
		{[]string{"010=1"}, []smartmonitor.Value{{Tag: 10, Value: 1}}},
		{[]string{"0x1F=0XFFFF"}, []smartmonitor.Value{{Tag: 31, Value: -1}}},
		{[]string{"1=-1"}, []smartmonitor.Value{{Tag: 1, Value: -1}}},
		{[]string{"1=-32768"}, []smartmonitor.Value{{Tag: 1, Value: -32768}}},
		{[]string{"1=-0x8000"}, []smartmonitor.Value{{Tag: 1, Value: -32768}}},
		{[]string{"1=32768"}, []smartmonitor.Value{{Tag: 1, Value: -32768}}},
		{[]string{"1=0x8000"}, []smartmonitor.Value{{Tag: 1, Value: -32768}}},
		{[]string{"1=65535"}, []smartmonitor.Value{{Tag: 1, Value: -1}}},
		{[]string{"1=-0"}, []smartmonitor.Value{{Tag: 1, Value: 0}}},
		{[]string{"1=32767"}, []smartmonitor.Value{{Tag: 1, Value: 32767}}},
		{[]string{"0=0"}, []smartmonitor.Value{{Tag: 0, Value: 0}}},
	}
	for _, scenario := range cases {
		got, err := parseValues(scenario.args)
		if err != nil || !reflect.DeepEqual(got, scenario.want) {
			test.Errorf("%v: got %v, %v", scenario.args, got, err)
		}
	}
}

func TestParseValuesEncodeAsTwosComplement(test *testing.T) {
	encode := func(args ...string) smartmonitor.Report {
		values, err := parseValues(args)
		if err != nil {
			test.Fatal(err)
		}
		return smartmonitor.EncodeValues(values)[0]
	}
	want := reportOf(0x02, 0x02, 0x01, 0xFF, 0xFF, 0x02, 0x7F, 0xFF)
	if got := encode("1=-1", "2=32767"); got != want {
		test.Errorf("got % X", got[:])
	}
	if encode("1=65535") != encode("1=-1") {
		test.Error("65535 and -1 differ")
	}
}

func TestParseValuesErrors(test *testing.T) {
	form := func(argument string) string { return `invalid argument "` + argument + `": want <tag>=<value>` }
	cases := []struct {
		args []string
		want string
	}{
		{nil, "values needs at least one <tag>=<value>"},
		{[]string{"1"}, form("1")},
		{[]string{"1=2=3"}, form("1=2=3")},
		{[]string{"=1"}, form("=1")},
		{[]string{"1="}, form("1=")},
		{[]string{"-1=1"}, form("-1=1")},
		{[]string{"+1=1"}, form("+1=1")},
		{[]string{" 1=1"}, form(" 1=1")},
		{[]string{"0x=1"}, form("0x=1")},
		{[]string{"0b1=1"}, form("0b1=1")},
		{[]string{"1=1.0"}, form("1=1.0")},
		{[]string{"18446744073709551616=1"}, `invalid argument "18446744073709551616=1": the tag is outside 0 to 255`},
		{[]string{"1=18446744073709551616"}, `invalid argument "1=18446744073709551616": the value is outside -32768 to 65535`},
		{[]string{"0x10000000000000000=1"}, `invalid argument "0x10000000000000000=1": the tag is outside 0 to 255`},
		{[]string{"1=0x10000000000000000"}, `invalid argument "1=0x10000000000000000": the value is outside -32768 to 65535`},
		{[]string{"1=" + strings.Repeat("9", 30)}, `invalid argument "1=` + strings.Repeat("9", 30) + `": the value is outside -32768 to 65535`},
		{[]string{"256=1"}, `invalid argument "256=1": the tag is outside 0 to 255`},
		{[]string{"1=65536"}, `invalid argument "1=65536": the value is outside -32768 to 65535`},
		{[]string{"1=1", "2"}, form("2")},
		{[]string{"1=-32769"}, `invalid argument "1=-32769": the value is outside -32768 to 65535`},
		{[]string{"1=-0x8001"}, `invalid argument "1=-0x8001": the value is outside -32768 to 65535`},
		{[]string{"1=-" + strings.Repeat("9", 30)}, `invalid argument "1=-` + strings.Repeat("9", 30) + `": the value is outside -32768 to 65535`},
		{[]string{"1=-18446744073709551616"}, `invalid argument "1=-18446744073709551616": the value is outside -32768 to 65535`},
		{[]string{"1=-"}, form("1=-")},
		{[]string{"1=--1"}, form("1=--1")},
		{[]string{"1=+1"}, form("1=+1")},
		{[]string{"1=- 1"}, form("1=- 1")},
		{[]string{"1=-0x"}, form("1=-0x")},
	}
	for _, scenario := range cases {
		_, err := parseValues(scenario.args)
		if err == nil || err.Error() != scenario.want {
			test.Errorf("%q: got %v, want %q", scenario.args, err, scenario.want)
		}
	}
}

func reportOf(data ...byte) smartmonitor.Report {
	var report smartmonitor.Report
	copy(report[:], data)
	return report
}

func TestParseRawBytes(test *testing.T) {
	want := reportOf(0x03, 0x01, 0x15)
	for _, args := range [][]string{{"03", "01", "15"}, {"030115"}, {"0301", "15"}} {
		got, err := parseRawBytes(args)
		if err != nil || got != want {
			test.Errorf("%v: got %v, %v", args, got, err)
		}
	}
	for _, text := range []string{"ab", "AB"} {
		got, err := parseRawBytes([]string{text})
		if err != nil || got != reportOf(0xAB) {
			test.Errorf("%s: got %v, %v", text, got, err)
		}
	}
	got, err := parseRawBytes([]string{strings.Repeat("11", 31), strings.Repeat("22", 31), "3333"})
	if err != nil || got[0] != 0x11 || got[63] != 0x33 {
		test.Errorf("64 bytes: got %v, %v", got, err)
	}
}

func TestParseRawBytesErrors(test *testing.T) {
	pairs := func(argument string) string {
		return `invalid argument "` + argument + `": want pairs of hexadecimal digits`
	}
	cases := []struct {
		args []string
		want string
	}{
		{nil, "raw needs at least one byte"},
		{[]string{"0x03"}, pairs("0x03")},
		{[]string{"3"}, pairs("3")},
		{[]string{""}, pairs("")},
		{[]string{"0g"}, pairs("0g")},
		{[]string{"01", "abc"}, pairs("abc")},
		{[]string{strings.Repeat("11", 65)}, "raw takes at most 64 bytes, got 65"},
		{[]string{strings.Repeat("11", 40), strings.Repeat("22", 25)}, "raw takes at most 64 bytes, got 65"},
	}
	for _, scenario := range cases {
		_, err := parseRawBytes(scenario.args)
		if err == nil || err.Error() != scenario.want {
			test.Errorf("%q: got %v, want %q", scenario.args, err, scenario.want)
		}
	}
}

func TestParseSeconds(test *testing.T) {
	for text, want := range map[string]int{"1": 1, "5": 5, "05": 5} {
		got, err := parseSeconds(text)
		if err != nil || got != want {
			test.Errorf("%q: got %d, %v", text, got, err)
		}
	}
	for _, text := range []string{"0", "-1", "+1", "1.5", "x", "", strings.Repeat("9", 30)} {
		_, err := parseSeconds(text)
		want := `invalid --seconds "` + text + `": want a whole number of 1 or more`
		if err == nil || err.Error() != want {
			test.Errorf("%q: got %v", text, err)
		}
	}
}
