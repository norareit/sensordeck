package uploadstate

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestHashIsTheSHA256InLowerCaseHexadecimal(test *testing.T) {
	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" // SHA-256 of "abc"
	if got := Hash([]byte("abc")); got != want {
		test.Errorf("Hash = %s, want %s", got, want)
	}
}

func TestPathIsUnderTheCacheDirectory(test *testing.T) {
	want := filepath.Join("cache", "sensordeck", "uploaded-themes.json")
	if got := Path("cache"); got != want {
		test.Errorf("Path = %s, want %s", got, want)
	}
}

func TestDecode(test *testing.T) {
	good := map[string]map[string]string{
		`{"2022100600": "abc"}`: {"2022100600": "abc"},
		`{}`:                    {},
		` {"": "abc"} `:         {"": "abc"},
	}
	for content, want := range good {
		got, err := Decode([]byte(content))
		if err != nil || !reflect.DeepEqual(got, want) {
			test.Errorf("Decode(%s) = %v, %v, want %v", content, got, err, want)
		}
	}
	for _, content := range []string{"", "not json", "null", "[]", `{"a": 1}`, `{"a": null}`, `{"a": "b"} x`} {
		if got, err := Decode([]byte(content)); err == nil {
			test.Errorf("Decode(%q) = %v, want an error", content, got)
		}
	}
}

func TestEncodeSortsTheKeysAndEndsInALineFeed(test *testing.T) {
	entries := map[string]string{"b": "2", "": "0", "a": "1"}
	content := Encode(entries)
	if !strings.HasSuffix(string(content), "}\n") {
		test.Errorf("Encode = %q, want a final line feed", content)
	}
	if text := string(content); strings.Index(text, `"":`) > strings.Index(text, `"a":`) || strings.Index(text, `"a":`) > strings.Index(text, `"b":`) {
		test.Errorf("Encode = %q, want sorted keys", content)
	}
	got, err := Decode(content)
	if err != nil || !reflect.DeepEqual(got, entries) {
		test.Errorf("Decode(Encode) = %v, %v", got, err)
	}
}

func TestReadOfAFileThatDoesNotExistHoldsNothing(test *testing.T) {
	entries, err := Read(filepath.Join(test.TempDir(), "missing.json"))
	if err != nil || entries == nil || len(entries) != 0 {
		test.Errorf("Read = %v, %v", entries, err)
	}
}

func TestReadOfAFileThatIsNotJSONGivesAnEmptyMapAndTheError(test *testing.T) {
	path := filepath.Join(test.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("garbage"), 0o644); err != nil {
		test.Fatal(err)
	}
	entries, err := Read(path)
	if err == nil || entries == nil || len(entries) != 0 {
		test.Errorf("Read = %v, %v", entries, err)
	}
}

func TestWriteCreatesTheDirectoryAndLeavesOnlyTheStateFile(test *testing.T) {
	path := Path(test.TempDir())
	entries := map[string]string{"2022100600": "abc"}
	if err := Write(path, entries); err != nil {
		test.Fatal(err)
	}
	got, err := Read(path)
	if err != nil || !reflect.DeepEqual(got, entries) {
		test.Errorf("Read = %v, %v", got, err)
	}
	listing, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(listing) != 1 || listing[0].Name() != "uploaded-themes.json" {
		test.Errorf("the directory holds %v, %v", listing, err)
	}
}

func TestWriteReplacesTheFileByARename(test *testing.T) {
	path := Path(test.TempDir())
	if err := Write(path, map[string]string{"a": "old"}); err != nil {
		test.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(path), "old-link")
	if err := os.Link(path, link); err != nil {
		test.Fatal(err)
	}
	if err := Write(path, map[string]string{"a": "new"}); err != nil {
		test.Fatal(err)
	}
	old, err := Read(link)
	if err != nil || old["a"] != "old" {
		test.Errorf("the old file holds %v, %v, so it was written in place", old, err)
	}
	current, err := Read(path)
	if err != nil || current["a"] != "new" {
		test.Errorf("the state file holds %v, %v", current, err)
	}
}

func TestWriteFailsAndLeavesNothingWhenTheTargetIsADirectory(test *testing.T) {
	path := Path(test.TempDir())
	if err := os.MkdirAll(path, 0o755); err != nil {
		test.Fatal(err)
	}
	if err := Write(path, map[string]string{}); err == nil {
		test.Fatal("Write over a directory gave no error")
	}
	listing, _ := os.ReadDir(filepath.Dir(path))
	if len(listing) != 1 {
		test.Errorf("the directory holds %d entries, want the directory only", len(listing))
	}
}
