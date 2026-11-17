package devicelock

import "testing"

func TestHolderOfContent(test *testing.T) {
	if got := holderOfContent(encodeContent(42, "sensordeck run")); got != " (process 42: sensordeck run)" {
		test.Errorf("got %q", got)
	}
	for _, content := range []string{"", "x\nsensordeck\n", "0\nsensordeck\n", "42\n", "42\na\nb\n"} {
		if got := holderOfContent([]byte(content)); got != "" {
			test.Errorf("content %q: got %q, want no holder", content, got)
		}
	}
}
