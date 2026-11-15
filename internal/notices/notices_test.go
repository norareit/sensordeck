package notices

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLicenceFileNamesKeepsTheNamesOfLicenceFilesInByteOrder(test *testing.T) {
	names := []string{"README.md", "NOTICE", "license", "LICENSE.txt", "go.mod", "MyLICENSE", "LICENCE", "COPYING", "LICENSE"}
	before := append([]string(nil), names...)
	want := []string{"COPYING", "LICENCE", "LICENSE", "LICENSE.txt", "NOTICE"}
	if got := LicenceFileNames(names); !reflect.DeepEqual(got, want) {
		test.Errorf("got %v, want %v", got, want)
	}
	if !reflect.DeepEqual(names, before) {
		test.Errorf("the argument changed: %v", names)
	}
}

const separator = "\n================================================================================\n\n"

func TestRenderGivesTheWholeOutputOfAsmallInput(test *testing.T) {
	modules := []Module{
		{Path: "b.example/two", Version: "v2.0.0", Files: []File{{Name: "LICENSE", Text: "two\n"}, {Name: "NOTICE", Text: "note"}}},
		{Path: "a.example/one", Version: "v1.0.0", Files: []File{{Name: "LICENSE", Text: "one\n"}}},
	}
	want := "Go standard library\n\nLICENSE\n-------\n\ngo text\n" + separator +
		"a.example/one v1.0.0\n\nLICENSE\n-------\n\none\n" + separator +
		"b.example/two v2.0.0\n\nLICENSE\n-------\n\ntwo\n\nNOTICE\n------\n\nnote\n"
	got, err := Render("go text\n", modules)
	if err != nil {
		test.Fatal(err)
	}
	if got != want {
		test.Errorf("got\n%q\nwant\n%q", got, want)
	}
	if modules[0].Path != "b.example/two" {
		test.Error("the argument was reordered")
	}
}

func TestRenderKeepsATextByteForByte(test *testing.T) {
	text := "  indented\r\n\ttabbed\n\n"
	got, err := Render("x\n", []Module{{Path: "m", Version: "v1", Files: []File{{Name: "LICENSE", Text: text}}}})
	if err != nil {
		test.Fatal(err)
	}
	if !strings.HasSuffix(got, "\n\n"+text) || strings.HasSuffix(got, text+"\n") {
		test.Errorf("got %q", got)
	}
}

func TestRenderFailsForAModuleWithNoFile(test *testing.T) {
	got, err := Render("x\n", []Module{{Path: "empty.example/m", Version: "v1.0.0"}})
	if err == nil || !strings.Contains(err.Error(), "empty.example/m") || got != "" {
		test.Errorf("got %q, %v", got, err)
	}
}

func TestTheCommittedFileIsWhatTheGeneratorGives(test *testing.T) {
	const command = "go run ./internal/notices/generate THIRD_PARTY_LICENSES"
	want, err := Generate("../..")
	if err != nil {
		test.Fatalf("%v (the command that writes the file is %q at the module root)", err, command)
	}
	committed, err := os.ReadFile("../../THIRD_PARTY_LICENSES")
	if err != nil {
		test.Fatalf("%v: write the file with %q at the module root", err, command)
	}
	if string(committed) != want {
		test.Errorf("THIRD_PARTY_LICENSES is not what the generator gives now: write it again with %q at the module root", command)
	}
}
