//go:build linux

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"testing"

	"github.com/norareit/sensordeck/internal/fakescreen"
)

const alarmFolderTheme = "name: alarm\nbackground: \"#102030\"\nfonts:\n  body: {builtin: go-regular, size: 20, color: \"#ffffff\"}\n" +
	"widgets:\n" +
	"  - {type: number, font: body, metric: cpu.temp, x: 4, y: 4}\n" +
	"  - {type: number, font: body, metric: gpu.temp, x: 4, y: 40, alarm: {above: 80, color: \"#ff3030\"}}\n"

func TestThemeCompileWritesTheTagsFileAndTheTwoRecordsOfAnAlarmNumber(test *testing.T) {
	folder := writeThemeFolder(test, alarmFolderTheme)
	directory := test.TempDir()
	out := filepath.Join(directory, "alarm.dat")
	if got := run(test, fakescreen.New(test), "theme", "compile", folder, out); got.status != 0 {
		test.Fatalf("%+v", got)
	}
	tags, err := os.ReadFile(tagsFilePath(out))
	if err != nil {
		test.Fatal(err)
	}
	wantTags := "# Paste this block into the config file of sensordeck run.\ntags:\n  cpu.temp: 1\n  gpu.temp: {alarms: [{above: 80, normal: 2, alarm: 3}]}\n"
	if string(tags) != wantTags {
		test.Errorf("tags file %q", tags)
	}
	dumped := filepath.Join(directory, "dump")
	if got := run(test, fakescreen.New(test), "theme", "dump", out, dumped); got.status != 0 {
		test.Fatalf("%+v", got)
	}
	content, err := os.ReadFile(filepath.Join(dumped, "theme.json"))
	if err != nil {
		test.Fatal(err)
	}
	var theme struct {
		Records []struct {
			Slot     int    `json:"slot"`
			TypeName string `json:"type_name"`
			Tag      int    `json:"tag"`
			Colour   string `json:"colour"`
		} `json:"records"`
	}
	if err := json.Unmarshal(content, &theme); err != nil {
		test.Fatal(err)
	}
	type numberRecord struct {
		slot, tag int
		colour    string
	}
	var numbers []numberRecord
	for _, record := range theme.Records {
		if record.TypeName == "number" {
			numbers = append(numbers, numberRecord{record.Slot, record.Tag, record.Colour})
		}
	}
	// The colours are RGB565: white is ffff and #ff3030 is f986.
	want := []numberRecord{{2, 1, "ffff"}, {3, 2, "ffff"}, {4, 3, "f986"}}
	if len(numbers) != len(want) {
		test.Fatalf("numbers %v", numbers)
	}
	for position := range want {
		if numbers[position] != want[position] {
			test.Errorf("number %d is %v, want %v", position, numbers[position], want[position])
		}
	}
}

func TestRunSwitchesTheValueBetweenTheTagsOfAnAlarmPair(test *testing.T) {
	setup := newThemeSetup(test)
	level := filepath.Join(test.TempDir(), "level")
	writeFile(test, level, "47\n")
	folder := writeThemeFolder(test, "name: alarm\nbackground: \"#102030\"\nfonts:\n  body: {builtin: go-regular, size: 20, color: \"#ffffff\"}\n"+
		"widgets:\n  - {type: number, font: body, metric: level, x: 4, y: 4, alarm: {above: 80, color: \"#ff3030\"}}\n")
	config := writeConfigFile(test, "interval: 100ms\ntheme: '"+folder+"'\nmetrics:\n  level: {file: "+level+"}\n")
	out := filepath.Join(test.TempDir(), "alarm.dat")
	if compiling := runWith(test, setup.fake, setup.environment, "theme", "compile", folder, out); compiling.status != 0 {
		test.Fatalf("theme compile: %q", compiling.stderr)
	}
	tags, err := os.ReadFile(tagsFilePath(out))
	if err != nil {
		test.Fatal(err)
	}
	match := regexp.MustCompile(`normal: (\d+), alarm: (\d+)`).FindSubmatch(tags)
	if match == nil {
		test.Fatalf("no pair in %q", tags)
	}
	normalNumber, _ := strconv.Atoi(string(match[1]))
	alarmNumber, _ := strconv.Atoi(string(match[2]))
	normal, alarm := uint8(normalNumber), uint8(alarmNumber)
	compiled, err := os.ReadFile(out)
	if err != nil {
		test.Fatal(err)
	}
	setup.screen.AcceptUpload(compiled)
	running := start(test, setup.fake, setup.environment, "run", "--config", config)
	position := setup.expectUploadAt(0, compiled)
	// Each state is waited for from the report before, so that the three are seen in order.
	for _, state := range []struct {
		content               string
		normalWant, alarmWant uint16
		name                  string
	}{
		{"47\n", 47, 0xFFFF, "the normal state"},
		{"95\n", 0xFFFF, 95, "the alarm state"},
		{"52\n", 52, 0xFFFF, "the normal state again"},
	} {
		writeFile(test, level, state.content)
		reached, found := waitForReport(setup.screen, position, uploadLimit, func(pairs map[uint8]uint16) bool {
			normalValue, hasNormal := pairs[normal]
			alarmValue, hasAlarm := pairs[alarm]
			return hasNormal && hasAlarm && normalValue == state.normalWant && alarmValue == state.alarmWant
		})
		if !found {
			test.Fatalf("no value report of %s", state.name)
		}
		position = reached
	}
	running.stop(syscall.SIGTERM)
}
