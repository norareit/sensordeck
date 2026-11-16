package custommetrics

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(test *testing.T, path, content string) string {
	test.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		test.Fatal(err)
	}
	return path
}

func readFileText(path string) (string, bool) {
	content, err := os.ReadFile(path)
	return string(content), err == nil
}
