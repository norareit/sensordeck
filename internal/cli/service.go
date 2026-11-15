package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// serviceCommand is service and its three subcommands. On a system other than Linux and Windows each of them fails at
// once.
func (current *session) serviceCommand() *cobra.Command {
	service := &cobra.Command{
		Use:   "service",
		Short: current.serviceText("Run sensordeck run as a systemd user service", "Run sensordeck run as a startup task of the Task Scheduler"),
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return current.onSystem(func() error { return command.Help() }, func() error { return command.Help() })
		},
	}
	service.AddCommand(current.serviceInstallCommand(), current.serviceStatusCommand(), current.serviceUninstallCommand())
	return service
}

// onSystem does the work of the system, and fails on a system other than Linux and Windows before it reads or writes
// anything.
func (current *session) onSystem(linux, windows func() error) error {
	switch current.env.system {
	case "linux":
		return linux()
	case "windows":
		return windows()
	}
	return fmt.Errorf("the service is not supported on %s", current.env.system)
}

// serviceText gives the text for Windows on Windows, and the text of Linux on every other system.
func (current *session) serviceText(linux, windows string) string {
	if current.env.system == "windows" {
		return windows
	}
	return linux
}

// unitFile gives the path of the unit file in the config directory of the user.
func (current *session) unitFile() (string, error) {
	directory, err := current.env.userConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot find the config directory: %w", err)
	}
	return unitFilePath(directory), nil
}

// systemctl makes one call and gives its standard output. A failure names the call and gives the standard error of
// the call as its cause, or the error of the call when it wrote none.
func (current *session) systemctl(arguments ...string) (string, error) {
	stdout, stderr, err := current.env.systemctl(arguments...)
	if err == nil {
		return stdout, nil
	}
	return "", callFailure("systemctl --user", arguments, stderr, err)
}

// callFailure names the call as "<program> <arguments joined by spaces>" and gives the standard error of the call,
// trimmed, as its cause, or the error of the call when it wrote none.
func callFailure(program string, arguments []string, stderr string, err error) error {
	cause := strings.TrimSpace(stderr)
	if cause == "" {
		cause = err.Error()
	}
	return fmt.Errorf("%s %s failed: %s", program, strings.Join(arguments, " "), cause)
}

// installed tells whether the file of the service exists, which kind names in the error, as "unit file". A link at its
// path counts as the file.
func installed(kind, file string) (bool, error) {
	_, err := os.Lstat(file)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cannot read the %s %s: %w", kind, file, innerError(err))
	}
	return true, nil
}

// serviceInstallCommand writes the file of the service, and starts it. It opens no device and reads no sensor.
func (current *session) serviceInstallCommand() *cobra.Command {
	var configPath string
	command := &cobra.Command{
		Use:   "install",
		Short: current.serviceText("Write, enable and start the systemd user service of run", "Write, create and start the startup task of run"),
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return current.onSystem(
				func() error { return current.unitInstall(configPath) },
				func() error { return current.startupTaskInstall(configPath) })
		},
	}
	command.Flags().StringVar(&configPath, "config", "", current.serviceText("name this config file in the service instead of the one of the user",
		"name this config file in the task instead of the one of the user"))
	return command
}

// unitInstall writes the unit file, and enables and starts the service.
func (current *session) unitInstall(configPath string) error {
	unitFile, err := current.unitFile()
	if err != nil {
		return err
	}
	content, config, err := current.unitFileFor(configPath)
	if err != nil {
		return err
	}
	if err := current.checkConfig(config); err != nil {
		return err
	}
	if err := writeUnitFile(unitFile, content); err != nil {
		return err
	}
	current.say("wrote " + unitFile)
	for _, arguments := range [][]string{{"daemon-reload"}, {"enable", unitName}, {"restart", unitName}} {
		if _, err := current.systemctl(arguments...); err != nil {
			return err
		}
	}
	current.say(unitName + " is enabled and started")
	return nil
}

// unitFileFor gives the content of the unit file for the running executable and the config file, which is the one named
// or the one of the user, and the path of the config file.
func (current *session) unitFileFor(named string) (content, config string, err error) {
	executable, err := current.executablePath()
	if err != nil {
		return "", "", err
	}
	config, err = current.serviceConfigPath(named)
	if err != nil {
		return "", "", err
	}
	content, err = unitFileContent(executable, config)
	return content, config, err
}

// serviceConfigPath gives the config file named, or the one of the user, made absolute.
func (current *session) serviceConfigPath(named string) (string, error) {
	config, err := current.configPath(named)
	if err != nil {
		return "", err
	}
	// The links of the config file are kept, so that a file that is a link into a repository is read through the link.
	absolute, err := filepath.Abs(config)
	if err != nil {
		return "", fmt.Errorf("cannot make the path %s absolute: %w", config, err)
	}
	return absolute, nil
}

// executablePath gives the path of the running executable, with its links resolved and made absolute.
func (current *session) executablePath() (string, error) {
	executable, err := current.env.executable()
	if err == nil {
		executable, err = filepath.EvalSymlinks(executable)
	}
	if err == nil {
		executable, err = filepath.Abs(executable)
	}
	if err != nil {
		return "", fmt.Errorf("cannot find the path of the executable: %w", innerError(err))
	}
	return executable, nil
}

// checkConfig loads the config file as run does at its start, so that a service is never installed for a config that
// run refuses.
func (current *session) checkConfig(path string) error {
	content, err := readNamedFile("config file", path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w; sensordeck theme use <theme> creates it", err)
	}
	if err != nil {
		return err
	}
	_, err = current.loadConfigFile(path, content)
	return err
}

// writeUnitFile writes the unit file, creating its directory and replacing a file that exists.
func writeUnitFile(unitFile, content string) error {
	if err := os.MkdirAll(filepath.Dir(unitFile), 0o755); err != nil {
		return fmt.Errorf("cannot write the unit file %s: %w", unitFile, innerError(err))
	}
	if err := os.WriteFile(unitFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cannot write the unit file %s: %w", unitFile, innerError(err))
	}
	return nil
}

// serviceUninstallCommand stops the service and removes its file.
func (current *session) serviceUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: current.serviceText("Stop, disable and remove the systemd user service", "Stop and delete the startup task and remove its file"),
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return current.onSystem(current.unitUninstall, current.startupTaskUninstall)
		},
	}
}

// unitUninstall stops and disables the service and removes the unit file.
func (current *session) unitUninstall() error {
	unitFile, err := current.unitFile()
	if err != nil {
		return err
	}
	present, err := installed("unit file", unitFile)
	if err != nil {
		return err
	}
	if !present {
		current.say(unitName + " is not installed")
		return nil
	}
	if _, err := current.systemctl("disable", "--now", unitName); err != nil {
		return err
	}
	if err := os.Remove(unitFile); err != nil {
		return fmt.Errorf("cannot remove the unit file %s: %w", unitFile, innerError(err))
	}
	if _, err := current.systemctl("daemon-reload"); err != nil {
		return err
	}
	current.say(unitName + " is stopped, disabled and removed")
	return nil
}

// serviceStatusCommand writes whether the service is installed, enabled and active.
func (current *session) serviceStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: current.serviceText("Show whether the service is installed, enabled and active", "Show whether the startup task is installed, enabled and running"),
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return current.onSystem(current.unitStatus, current.startupTaskStatus)
		},
	}
}

// unitStatus writes whether the unit file exists, and the state of the service that systemd gives.
func (current *session) unitStatus() error {
	unitFile, err := current.unitFile()
	if err != nil {
		return err
	}
	present, err := installed("unit file", unitFile)
	if err != nil {
		return err
	}
	if !present {
		current.say("installed: no")
		return nil
	}
	output, err := current.systemctl("show", "--property=UnitFileState", "--property=ActiveState", "--property=SubState", unitName)
	if err != nil {
		return err
	}
	for _, line := range statusLines(unitFile, showProperties(output)) {
		current.say(line)
	}
	return nil
}
