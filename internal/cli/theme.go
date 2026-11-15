package cli

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/norareit/sensordeck/internal/builtinthemes"
	"github.com/norareit/sensordeck/internal/themecompile"
	"github.com/norareit/sensordeck/internal/themedump"
	"github.com/norareit/sensordeck/internal/themefile"
	"github.com/norareit/sensordeck/internal/themeimage"
)

func (current *session) themeCommand() *cobra.Command {
	theme := &cobra.Command{
		Use:   "theme",
		Short: "Work with theme files without a screen",
	}
	theme.AddCommand(current.themeDumpCommand(), current.themeCompileCommand(), current.themePreviewCommand(),
		current.themeListCommand())
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

// themeCompileCommand compiles a theme, by name or folder, into a theme file and writes the tags file beside it. It opens no device
// and reads no sensor.
func (current *session) themeCompileCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "compile <name-or-folder> <out.dat>",
		Short: "Compile a theme, by name or folder, into a theme file and its tags file",
		Args:  cobra.ExactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			result, err := current.compileTheme(args[0])
			if err != nil {
				return err
			}
			out := args[1]
			tags := tagsPath(out)
			if err := writeOutput(out, result.File); err != nil {
				return err
			}
			if err := writeOutput(tags, themecompile.TagsFile(result.Tags)); err != nil {
				return err
			}
			current.say(fmt.Sprintf("wrote %s (%d bytes) and %s", out, len(result.File), tags))
			return nil
		},
	}
}

// themePreviewCommand compiles a theme, by name or folder, in memory and writes the preview of the compiled file as a PNG.
func (current *session) themePreviewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "preview <name-or-folder> <out.png>",
		Short: "Draw the preview of a theme, by name or folder, as a PNG",
		Args:  cobra.ExactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			result, err := current.compileTheme(args[0])
			if err != nil {
				return err
			}
			theme, _, err := themefile.Parse(result.File)
			if err != nil {
				return fmt.Errorf("cannot read the compiled theme: %w", err)
			}
			var encoded bytes.Buffer
			if err := png.Encode(&encoded, themeimage.Preview(theme)); err != nil {
				return fmt.Errorf("cannot encode the preview: %w", err)
			}
			if err := writeOutput(args[1], encoded.Bytes()); err != nil {
				return err
			}
			current.say("wrote " + args[1])
			return nil
		},
	}
}

// themeListCommand lists the built-in and the installed themes. It opens no device and reads no sensor.
func (current *session) themeListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the built-in and installed themes",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			installed, err := current.installedThemes()
			if err != nil {
				return err
			}
			for _, line := range themeListLines(builtinthemes.Names(), installed) {
				current.say(line)
			}
			return nil
		},
	}
}

// compileFolder opens the folder as a root, which refuses any name that leaves it, and compiles it. Its error starts
// with the path of theme.yaml.
func compileFolder(directory string) (themecompile.Result, error) {
	themePath := filepath.Join(directory, "theme.yaml")
	root, err := os.OpenRoot(directory)
	if err != nil {
		return themecompile.Result{}, fmt.Errorf("%s: cannot read the file: %w", themePath, innerError(err))
	}
	defer func() { _ = root.Close() }()
	result, err := themecompile.Compile(root.FS())
	if err != nil {
		return themecompile.Result{}, fmt.Errorf("%s: %w", themePath, err)
	}
	return result, nil
}

// tagsPath gives the path of the tags file of a theme file: the extension replaced by .tags.yaml, or .tags.yaml added
// when there is none.
func tagsPath(out string) string {
	return strings.TrimSuffix(out, filepath.Ext(out)) + ".tags.yaml"
}

func writeOutput(path string, content []byte) error {
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("cannot write the file %s: %w", path, innerError(err))
	}
	return nil
}
