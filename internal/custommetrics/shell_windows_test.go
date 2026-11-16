package custommetrics

import (
	"context"
	"testing"
)

func TestTheShellOfWindows(test *testing.T) {
	cases := map[string]string{`C:\Windows\System32\cmd.exe`: `C:\Windows\System32\cmd.exe`, "": "cmd.exe"}
	for comspec, program := range cases {
		test.Setenv("ComSpec", comspec)
		command := shellCommand(context.Background(), `echo "a b"`)
		if want := `"` + program + `" /d /s /c "echo "a b""`; command.SysProcAttr.CmdLine != want {
			test.Errorf("CmdLine %q, want %q", command.SysProcAttr.CmdLine, want)
		}
		if command.SysProcAttr.CreationFlags&0x08000000 == 0 {
			test.Error("the flag CREATE_NO_WINDOW is not set")
		}
	}
}
