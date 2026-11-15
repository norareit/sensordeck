package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/norareit/sensordeck/internal/config"
)

// themeUseCommand sets the key theme of the config file of run. It opens no device and reads no sensor.
func (current *session) themeUseCommand() *cobra.Command {
	var configPath string
	command := &cobra.Command{
		Use:   "use <name-or-folder>",
		Short: "Set the theme of the config file of run, by name or folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			path, err := current.configPath(configPath)
			if err != nil {
				return err
			}
			if _, err := current.compileTheme(args[0]); err != nil {
				return err
			}
			value, err := themeValue(args[0])
			if err != nil {
				return err
			}
			if err := setThemeInConfig(path, value); err != nil {
				return err
			}
			current.say(fmt.Sprintf("theme %s set in %s", value, path))
			return nil
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "edit this config file instead of the one of the user")
	return command
}

// themeValue gives what the config file holds for the argument: a theme name as it is, and a folder path as an absolute
// and cleaned path, so that it names the same folder whatever the working directory of run is.
func themeValue(argument string) (string, error) {
	if !isFolderPath(argument) {
		return argument, nil
	}
	absolute, err := filepath.Abs(argument)
	if err != nil {
		return "", fmt.Errorf("cannot make the path %s absolute: %w", argument, err)
	}
	return absolute, nil
}

// setThemeInConfig sets the key theme in the config file at the path, creating the file and its directories when it does
// not exist. It writes only a config that run accepts, and writes in place, so that a link to the file stays one.
func setThemeInConfig(path, value string) error {
	content, err := os.ReadFile(path)
	missing := errors.Is(err, fs.ErrNotExist)
	if err != nil && !missing {
		return fmt.Errorf("cannot read the config file %s: %w", path, innerError(err))
	}
	edited, err := config.SetTheme(content, value)
	if err != nil {
		return fmt.Errorf("config file %s: %w", path, err)
	}
	if _, err := config.Parse(edited); err != nil {
		return fmt.Errorf("config file %s: %w", path, err)
	}
	if missing {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("cannot create the directory %s: %w", filepath.Dir(path), innerError(err))
		}
	}
	// WriteFile truncates the file behind a symbolic link and keeps its permissions, which the mode only sets for a new file.
	if err := os.WriteFile(path, edited, 0o644); err != nil {
		return fmt.Errorf("cannot write the config file %s: %w", path, innerError(err))
	}
	return nil
}
