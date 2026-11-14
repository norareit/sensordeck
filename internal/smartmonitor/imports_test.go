package smartmonitor

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestImportsAreStandardLibraryWithoutSideEffects(test *testing.T) {
	forbidden := map[string]bool{"os": true, "os/exec": true, "os/signal": true, "net": true, "syscall": true, "unsafe": true}
	files, err := filepath.Glob("*.go")
	if err != nil {
		test.Fatal(err)
	}
	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			test.Fatal(err)
		}
		for _, spec := range parsed.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				test.Fatal(err)
			}
			if forbidden[path] {
				test.Errorf("%s imports %s", file, path)
			}
			if strings.Contains(strings.SplitN(path, "/", 2)[0], ".") {
				test.Errorf("%s imports %s, which is not the standard library", file, path)
			}
		}
	}
}
