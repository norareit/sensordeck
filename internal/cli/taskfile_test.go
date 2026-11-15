package cli

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"
)

const (
	sampleUser = "S-1-5-21-1004336348-1177238915-682003330-1001"
	// wantTask is the definition of the spec with the placeholders of the user, the executable and the config file.
	wantTask = `<?xml version="1.0" encoding="UTF-16"?>
<!-- Written by sensordeck service install, which overwrites it when run again. -->
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>sensordeck: sensor values on a USB screen</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      <UserId>USER</UserId>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>USER</UserId>
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>99</Count>
    </RestartOnFailure>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>EXECUTABLE</Command>
      <Arguments>run --config "CONFIG" --background</Arguments>
    </Exec>
  </Actions>
</Task>
`
)

// expectedTask gives the bytes the spec describes: the text with CR LF, as UTF-16 little-endian behind FF FE.
func expectedTask(user, executable, config string) []byte {
	text := strings.NewReplacer("USER", user, "EXECUTABLE", executable, "CONFIG", config).Replace(wantTask)
	text = strings.ReplaceAll(text, "\n", "\r\n")
	bytes := []byte{0xFF, 0xFE}
	for _, unit := range utf16.Encode([]rune(text)) {
		bytes = append(bytes, byte(unit), byte(unit>>8))
	}
	return bytes
}

func TestTaskFilePathIsInTheSensordeckDirectoryOfTheCacheDirectory(test *testing.T) {
	if got, want := taskFilePath("/cache"), filepath.Join("/cache", "sensordeck", "startup-task.xml"); got != want {
		test.Errorf("task file %q, want %q", got, want)
	}
}

func TestTaskFileContentIsTheDefinitionOfTheSpecInUTF16WithCRLF(test *testing.T) {
	got, err := taskFileContent(sampleUser, `C:\Program Files\sensordeck\sensordeck.exe`, `C:\Users\a\AppData\Roaming\sensordeck\config.yaml`)
	if err != nil {
		test.Fatal(err)
	}
	want := expectedTask(sampleUser, `C:\Program Files\sensordeck\sensordeck.exe`, `C:\Users\a\AppData\Roaming\sensordeck\config.yaml`)
	if string(got) != string(want) {
		test.Fatalf("the bytes differ: %d bytes, want %d", len(got), len(want))
	}
	if got[0] != 0xFF || got[1] != 0xFE {
		test.Errorf("byte order mark % X", got[:2])
	}
	text := decodeUTF16(string(got[2:]))
	if !strings.HasSuffix(text, "</Task>\r\n") || strings.Count(text, "\n") != strings.Count(text, "\r\n") {
		test.Error("not every line ends in CR LF, the last one included")
	}
}

func TestTaskFileContentEscapesThePathsForXML(test *testing.T) {
	executable, config := `C:\a & b\<c>\sensordeck.exe`, `D:\x>y\config & more.yaml`
	got, err := taskFileContent(sampleUser, executable, config)
	if err != nil {
		test.Fatal(err)
	}
	want := expectedTask(sampleUser, `C:\a &amp; b\&lt;c&gt;\sensordeck.exe`, `D:\x&gt;y\config &amp; more.yaml`)
	if string(got) != string(want) {
		test.Errorf("the escaped file differs: %q", decodeUTF16(string(got[2:])))
	}
}

func TestTaskFileContentRefusesAPathWithAQuoteAPercentOrAControlCharacter(test *testing.T) {
	for _, path := range []string{`C:\a"b\x`, `C:\%APPDATA%\x`, "C:\\a\nb", "C:\\a\tb", "C:\\a\rb"} {
		want := `cannot write the path ` + strconv.Quote(path) + ` into the task file: it holds ", % or a control character`
		for name, arguments := range map[string][2]string{"executable": {path, `C:\c.yaml`}, "config": {`C:\e.exe`, path}} {
			content, err := taskFileContent(sampleUser, arguments[0], arguments[1])
			if err == nil || err.Error() != want || content != nil {
				test.Errorf("%s %q: %v, want %q", name, path, err, want)
			}
		}
	}
}

func taskOutput(enabled string) string {
	settings := "<Settings>" + enabled + "</Settings>"
	return `<?xml version="1.0" encoding="UTF-16"?><Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">` +
		`<Triggers><LogonTrigger><Enabled>false</Enabled></LogonTrigger></Triggers>` + settings + `</Task>`
}

func inUTF16(text string) string {
	bytes := []byte{0xFF, 0xFE}
	for _, unit := range utf16.Encode([]rune(text)) {
		bytes = append(bytes, byte(unit), byte(unit>>8))
	}
	return string(bytes)
}

func TestTaskEnabledReadsTheEnabledElementOfTheSettingsInUTF16AndUTF8(test *testing.T) {
	cases := map[string]string{
		"<Enabled>true</Enabled>":                 "enabled",
		"<Enabled>false</Enabled>":                "disabled",
		"<Enabled>1</Enabled>":                    "enabled",
		"<Enabled>0</Enabled>":                    "disabled",
		"<Enabled> false </Enabled>":              "disabled",
		"<Enabled>maybe</Enabled>":                "unknown",
		"":                                        "enabled",
		"<Enabled></Enabled>":                     "unknown",
		"<Other><Enabled>false</Enabled></Other>": "enabled",
	}
	for enabled, want := range cases {
		document := taskOutput(enabled)
		if got := taskEnabled(inUTF16(document)); got != want {
			test.Errorf("UTF-16 %q: %s, want %s", enabled, got, want)
		}
		if got := taskEnabled(document); got != want {
			test.Errorf("UTF-8 %q: %s, want %s", enabled, got, want)
		}
	}
}

func TestTaskEnabledIsUnknownForWhatIsNotATask(test *testing.T) {
	for _, output := range []string{"", "not xml", "<Other><Settings><Enabled>false</Enabled></Settings></Other>", "<Task>", inUTF16("<Task>")} {
		if got := taskEnabled(output); got != "unknown" {
			test.Errorf("%q: %s", output, got)
		}
	}
}

func TestTaskStatusLinesAreTheFourLines(test *testing.T) {
	got := taskStatusLines(`C:\t.xml`, "enabled", "running", `C:\run.log`)
	want := []string{`installed: yes, C:\t.xml`, "enabled: enabled", "active: running", `log: C:\run.log`}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		test.Errorf("lines %q", got)
	}
}
