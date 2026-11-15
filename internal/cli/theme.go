package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/norareit/sensordeck/internal/themedump"
)

func (current *session) themeCommand() *cobra.Command {
	theme := &cobra.Command{
		Use:   "theme",
		Short: "Work with theme files without a screen",
	}
	theme.AddCommand(current.themeDumpCommand())
	return theme
}

// themeDumpCommand reads a theme file and writes a JSON description, a PNG of each resource and a preview. It opens no
// device and reads no sensor.
func (current *session) themeDumpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "dump <file.dat> <out-dir>",
		Short: "Describe a theme file as JSON and PNG files",
		Args:  cobra.ExactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			path, directory := args[0], args[1]
			content, err := readTheme(path)
			if err != nil {
				return err
			}
			files, warnings, err := themedump.Dump(content)
			if err != nil {
				return fmt.Errorf("cannot read the theme file %s: %w", path, err)
			}
			// Nothing is created before the whole dump has succeeded, so a failed dump leaves no directory behind.
			if err := os.MkdirAll(directory, 0o755); err != nil {
				return fmt.Errorf("cannot create the output directory %s: %w", directory, innerError(err))
			}
			for _, file := range files {
				target := filepath.Join(directory, file.Name)
				if err := os.WriteFile(target, file.Content, 0o644); err != nil {
					return fmt.Errorf("cannot write the file %s: %w", target, innerError(err))
				}
			}
			for _, warning := range warnings {
				_, _ = fmt.Fprintf(current.env.stderr, "warning: %s\n", warning)
			}
			current.say(fmt.Sprintf("wrote %d files to %s", len(files), directory))
			return nil
		},
	}
}
