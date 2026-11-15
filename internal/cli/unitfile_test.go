package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestUnitFilePathIsUnderTheConfigDirectory(test *testing.T) {
	if got := unitFilePath("/home/user/.config"); got != "/home/user/.config/systemd/user/sensordeck.service" {
		test.Errorf("got %q", got)
	}
}

func TestQuoteUnitPathEscapesWhatSystemdExpands(test *testing.T) {
	for path, want := range map[string]string{
		"/usr/bin/sensordeck": `"/usr/bin/sensordeck"`,
		"/a b/c":              `"/a b/c"`,
		`/a"b`:                `"/a\"b"`,
		`/a\b`:                `"/a\\b"`,
		"/a%b":                `"/a%%b"`,
		"/a$b":                `"/a$$b"`,
		`/a\$b`:               `"/a\\$$b"`,
		"/a%%b":               `"/a%%%%b"`,
	} {
		got, err := quoteUnitPath(path)
		if err != nil || got != want {
			test.Errorf("%q: got %q, %v, want %q", path, got, err, want)
		}
	}
}

func TestQuoteUnitPathRefusesAControlCharacterAndNamesThePath(test *testing.T) {
	for _, path := range []string{"/a\nb", "/a\rb", "/a\tb"} {
		_, err := quoteUnitPath(path)
		want := "cannot write the path " + `"` + strings.NewReplacer("\n", `\n`, "\r", `\r`, "\t", `\t`).Replace(path) + `"` + " into the unit file: it holds a control character"
		if err == nil || err.Error() != want {
			test.Errorf("%q: error %v, want %q", path, err, want)
		}
	}
}

func TestUnitFileContentIsTheTextOfTheSpecByteForByte(test *testing.T) {
	want := "# Written by sensordeck service install, which overwrites it when run again.\n" +
		"# Put changes in a drop-in instead: systemctl --user edit sensordeck.service\n" +
		"\n" +
		"[Unit]\n" +
		"Description=sensordeck: sensor values on a USB screen\n" +
		"StartLimitIntervalSec=0\n" +
		"\n" +
		"[Service]\n" +
		"Type=exec\n" +
		"ExecStart=\"/opt/my tools/sensordeck\" run --config \"/home/u/.config/sensordeck/config.yaml\"\n" +
		"Restart=on-failure\n" +
		"RestartSec=10s\n" +
		"StandardOutput=journal\n" +
		"StandardError=journal\n" +
		"\n" +
		"[Install]\n" +
		"WantedBy=default.target\n"
	got, err := unitFileContent("/opt/my tools/sensordeck", "/home/u/.config/sensordeck/config.yaml")
	if err != nil || got != want {
		test.Errorf("got %q, %v", got, err)
	}
}

func TestUnitFileContentRefusesEitherPathWithAControlCharacter(test *testing.T) {
	if _, err := unitFileContent("/a\nb", "/c"); err == nil {
		test.Error("an executable with a line feed was accepted")
	}
	if _, err := unitFileContent("/a", "/c\nd"); err == nil {
		test.Error("a config file with a line feed was accepted")
	}
}

func TestShowPropertiesSplitsAtTheFirstEqualsSignAndLeavesOutOtherLines(test *testing.T) {
	got := showProperties("UnitFileState=enabled\nActiveState=active\nno equals sign\nSubState=a=b\n")
	want := map[string]string{"UnitFileState": "enabled", "ActiveState": "active", "SubState": "a=b"}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestStatusLines(test *testing.T) {
	const unitFile = "/c/systemd/user/sensordeck.service"
	full := map[string]string{"UnitFileState": "enabled", "ActiveState": "active", "SubState": "running"}
	for name, tc := range map[string]struct {
		properties map[string]string
		want       []string
	}{
		"all three":           {full, []string{"installed: yes, " + unitFile, "enabled: enabled", "active: active (running)"}},
		"no enabled state":    {map[string]string{"ActiveState": "active", "SubState": "running"}, []string{"installed: yes, " + unitFile, "enabled: unknown", "active: active (running)"}},
		"empty enabled state": {map[string]string{"UnitFileState": "", "ActiveState": "active", "SubState": "running"}, []string{"installed: yes, " + unitFile, "enabled: unknown", "active: active (running)"}},
		"no sub state":        {map[string]string{"UnitFileState": "enabled", "ActiveState": "active"}, []string{"installed: yes, " + unitFile, "enabled: enabled", "active: active (unknown)"}},
		"empty sub state":     {map[string]string{"UnitFileState": "enabled", "ActiveState": "active", "SubState": ""}, []string{"installed: yes, " + unitFile, "enabled: enabled", "active: active (unknown)"}},
		"no active state":     {map[string]string{"UnitFileState": "enabled", "SubState": "running"}, []string{"installed: yes, " + unitFile, "enabled: enabled", "active: unknown"}},
		"empty active state":  {map[string]string{"UnitFileState": "enabled", "ActiveState": "", "SubState": "running"}, []string{"installed: yes, " + unitFile, "enabled: enabled", "active: unknown"}},
		"nothing":             {nil, []string{"installed: yes, " + unitFile, "enabled: unknown", "active: unknown"}},
	} {
		if got := statusLines(unitFile, tc.properties); !reflect.DeepEqual(got, tc.want) {
			test.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}
