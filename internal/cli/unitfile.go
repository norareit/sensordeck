package cli

import (
	"fmt"
	"path/filepath"
	"strings"
)

// unitName is the name of the service.
const unitName = "sensordeck.service"

// unitFilePath gives the unit file in the config directory of the user.
func unitFilePath(configDirectory string) string {
	return filepath.Join(configDirectory, "systemd", "user", unitName)
}

// unitPathQuoter escapes what systemd expands inside double quotes. It replaces in one pass, so that the backslash it
// adds for a quote is not escaped again.
var unitPathQuoter = strings.NewReplacer(`\`, `\\`, `"`, `\"`, `%`, `%%`, `$`, `$$`)

// quoteUnitPath gives the path in double quotes, so that systemd reads back exactly the path, or an error naming the
// path when it holds a character below 0x20, which no quoting can write on one line.
func quoteUnitPath(path string) (string, error) {
	for _, character := range path {
		if character < 0x20 {
			return "", fmt.Errorf("cannot write the path %q into the unit file: it holds a control character", path)
		}
	}
	return `"` + unitPathQuoter.Replace(path) + `"`, nil
}

// unitFileContent gives the content of the unit file for the executable and the config file.
func unitFileContent(executable, config string) (string, error) {
	quotedExecutable, err := quoteUnitPath(executable)
	if err != nil {
		return "", err
	}
	quotedConfig, err := quoteUnitPath(config)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`# Written by sensordeck service install, which overwrites it when run again.
# Put changes in a drop-in instead: systemctl --user edit sensordeck.service

[Unit]
Description=sensordeck: sensor values on a USB screen
StartLimitIntervalSec=0

[Service]
Type=exec
ExecStart=%s run --config %s
Restart=on-failure
RestartSec=10s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
`, quotedExecutable, quotedConfig), nil
}

// showProperties reads the lines <property>=<value> of the output of systemctl show, split at the first =. A line with
// no = is left out.
func showProperties(output string) map[string]string {
	properties := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		name, value, found := strings.Cut(line, "=")
		if found {
			properties[name] = value
		}
	}
	return properties
}

// statusLines gives the three lines of status for the unit file and the properties, with unknown for a property that is
// missing or empty, and no parentheses when the active state is.
func statusLines(unitFile string, properties map[string]string) []string {
	active := "unknown"
	if state := properties["ActiveState"]; state != "" {
		active = fmt.Sprintf("%s (%s)", state, valueOrDefault(properties["SubState"], "unknown"))
	}
	return []string{
		"installed: yes, " + unitFile,
		"enabled: " + valueOrDefault(properties["UnitFileState"], "unknown"),
		"active: " + active,
	}
}
