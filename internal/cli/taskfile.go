package cli

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// taskName is the name of the task in the root folder of the Task Scheduler.
const taskName = "sensordeck"

// taskFilePath gives sensordeck/startup-task.xml in the cache directory.
func taskFilePath(cacheDirectory string) string {
	return filepath.Join(cacheDirectory, "sensordeck", "startup-task.xml")
}

// checkTaskPath gives the error for a path that holds ", % or a character below 0x20. The path is written inside
// double quotes of the arguments, where Windows allows no ", and the Task Scheduler may expand a % with no escape.
func checkTaskPath(path string) error {
	if strings.ContainsAny(path, `"%`) || strings.IndexFunc(path, func(character rune) bool { return character < 0x20 }) >= 0 {
		return fmt.Errorf(`cannot write the path %q into the task file: it holds ", %% or a control character`, path)
	}
	return nil
}

const taskDefinition = `<?xml version="1.0" encoding="UTF-16"?>
<!-- Written by sensordeck service install, which overwrites it when run again. -->
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>sensordeck: sensor values on a USB screen</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      <UserId>{user}</UserId>
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>{user}</UserId>
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
      <Command>{executable}</Command>
      <Arguments>run --config "{config}" --background</Arguments>
    </Exec>
  </Actions>
</Task>
`

var xmlText = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// taskFileContent gives the bytes of the task file: the definition with the three values escaped for XML, every line
// ending in CR LF, encoded as UTF-16 little-endian behind the byte order mark. It refuses a path that checkTaskPath
// refuses.
func taskFileContent(user, executable, config string) ([]byte, error) {
	for _, path := range []string{executable, config} {
		if err := checkTaskPath(path); err != nil {
			return nil, err
		}
	}
	text := strings.NewReplacer("{user}", xmlText.Replace(user), "{executable}", xmlText.Replace(executable),
		"{config}", xmlText.Replace(config)).Replace(taskDefinition)
	units := utf16.Encode([]rune(strings.ReplaceAll(text, "\n", "\r\n")))
	content := []byte{0xFF, 0xFE}
	for _, unit := range units {
		content = binary.LittleEndian.AppendUint16(content, unit)
	}
	return content, nil
}

// taskDocument is the part of a task that status reads.
type taskDocument struct {
	XMLName  xml.Name `xml:"Task"`
	Settings struct {
		Enabled *string `xml:"Enabled"`
	} `xml:"Settings"`
}

// taskEnabled reads the output of /Query /XML, as UTF-16 little-endian behind FF FE and as UTF-8 otherwise, and gives
// enabled, disabled or unknown. The declaration is ignored, because the output is decoded before it is read. A task
// with no Enabled element is enabled, which is the default of the schema.
func taskEnabled(output string) string {
	text := output
	if strings.HasPrefix(output, "\xFF\xFE") {
		text = decodeUTF16(output[2:])
	}
	decoder := xml.NewDecoder(strings.NewReader(text))
	decoder.CharsetReader = func(label string, input io.Reader) (io.Reader, error) { return input, nil }
	var task taskDocument
	if err := decoder.Decode(&task); err != nil {
		return "unknown"
	}
	if task.Settings.Enabled == nil {
		return "enabled"
	}
	switch strings.TrimSpace(*task.Settings.Enabled) {
	case "true", "1":
		return "enabled"
	case "false", "0":
		return "disabled"
	}
	return "unknown"
}

// decodeUTF16 decodes UTF-16 little-endian bytes, leaving a last odd byte out.
func decodeUTF16(bytes string) string {
	units := make([]uint16, 0, len(bytes)/2)
	for index := 0; index+1 < len(bytes); index += 2 {
		units = append(units, uint16(bytes[index])|uint16(bytes[index+1])<<8)
	}
	return string(utf16.Decode(units))
}

// taskStatusLines gives the four lines of status for a task that is installed.
func taskStatusLines(taskFile, enabled, active, log string) []string {
	return []string{"installed: yes, " + taskFile, "enabled: " + enabled, "active: " + active, "log: " + log}
}
