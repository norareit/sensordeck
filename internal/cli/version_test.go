package cli

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(test *testing.T) {
	info := func(version string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: version}}
	}
	cases := []struct {
		name   string
		linked string
		info   *debug.BuildInfo
		ok     bool
		want   string
	}{
		{"a linked version wins over the build information", "v1.2.3", info("v0.1.0"), true, "v1.2.3"},
		{"a linked version with no build information", "v1.2.3", nil, false, "v1.2.3"},
		{"the build information when nothing is linked", "", info("v0.1.0"), true, "v0.1.0"},
		{"an empty main module version", "", info(""), true, "(devel)"},
		{"no build information", "", nil, false, "(devel)"},
	}
	for _, each := range cases {
		if got := resolveVersion(each.linked, each.info, each.ok); got != each.want {
			test.Errorf("%s: got %q, want %q", each.name, got, each.want)
		}
	}
}
