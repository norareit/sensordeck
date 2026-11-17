package devicelock

import "testing"

func TestHolderText(test *testing.T) {
	cases := []struct {
		name  string
		pid   int
		line  string
		known bool
		want  string
	}{
		{"with command line", 42, "sensordeck run --config c.yaml", true, " (process 42: sensordeck run --config c.yaml)"},
		{"unknown", 42, "", false, " (process 42)"},
		{"empty", 42, "", true, " (process 42)"},
	}
	for _, each := range cases {
		if got := HolderText(each.pid, each.line, each.known); got != each.want {
			test.Errorf("%s: got %q, want %q", each.name, got, each.want)
		}
	}
}

func TestInUseErrorText(test *testing.T) {
	if got := (&InUseError{Holder: " (process 42)"}).Error(); got != "in use by another sensordeck process (process 42)" {
		test.Errorf("got %q", got)
	}
	if got := (&InUseError{}).Error(); got != "in use by another sensordeck process" {
		test.Errorf("got %q", got)
	}
}
