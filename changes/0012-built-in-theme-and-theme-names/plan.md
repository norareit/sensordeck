# A built-in theme and themes found by name

## Approach

One new package holds the built-in themes, a small package draws the background of `minimal-dark`, and the theme
commands of `internal/cli` learn to take a name and to list the themes.

* `internal/builtinthemes` embeds the directory `themes/` beside it with `//go:embed themes`, and gives the built-in
  themes as a set: `Names()` lists the directories under `themes/`, and `Folder(name)` gives one of them as an `fs.FS`
  through `fs.Sub`. Nothing in it or in the commands names `minimal-dark`, so a second built-in theme is a new folder
  (R1, D5). An `embed.FS` has no way to be written to (R1)
* `internal/builtinthemes/themes/minimal-dark/` holds `theme.yaml`, the text of R2, and `background.png`, and nothing
  else (R2, R3)
* `internal/builtinthemes/background` draws the picture of R4 with whole numbers only, and its subdirectory
  `generate` is the program that `go generate` runs to write it as `background.png` (R4, R5, D6, D7). The directive is
  in `internal/builtinthemes/builtinthemes.go`, so that `go generate ./internal/builtinthemes/` writes the file
* `internal/cli` gets a file of its own for the themes found by name: the rule of R8, the themes directory, the test of
  whether an entry is an installed theme, the lookup of R10 to R12 and the lines of the list. The lookup and the list
  both decide what is an installed theme through one function, so they agree by construction (R16, D3)

Before this plan was written, the `theme.yaml` of R2 was compiled in the sandbox with Go 1.27.4 and the current
`theme compile`, against a `background.png` drawn by the rule of R4: it compiles, `theme dump` of the result prints no
warning, and the tags file holds exactly the mapping of R6. The spec fits the tree.

None of the items of "Beyond the request" is built, as the record decided.

### The names and texts the implementation uses

These are decided here, so that the tests and the code agree:

| What | Exact text |
|---|---|
| Use line of `theme compile` | `compile <name-or-folder> <out.dat>` |
| Use line of `theme preview` | `preview <name-or-folder> <out.png>` |
| Short of `theme compile` | `Compile a theme, by name or folder, into a theme file and its tags file` |
| Short of `theme preview` | `Draw the preview of a theme, by name or folder, as a PNG` |
| Use and Short of `theme list` | `list`, `List the built-in and installed themes` |
| The empty argument (R8) | `a theme name or folder is needed` |
| A name found nowhere (R11) | `no installed or built-in theme is named <name>; a folder is named by a path, such as ./<name>` |
| No config directory (R12, R17) | `cannot find the config directory: <cause>` |
| An entry that cannot be examined (R12) | `cannot examine the installed theme <path>: <cause>` |
| A themes directory that cannot be read (R17) | `cannot read the themes directory <path>: <cause>` |
| An error of a built-in theme (R10) | `built-in theme <name>: <error of the compiler>` |

`<cause>` is the cause alone, through the existing `innerError`, so that a path is not written twice. An installed
theme is compiled with the existing `compileFolder`, whose errors already start with the path of its `theme.yaml`
(R10).

### The rule of R8

```go
// isFolderPath tells a folder path from a theme name by the text alone: . and .. and anything with a separator of a
// path on this platform is a path.
func isFolderPath(argument string) bool
```

It is true for `.` and `..`, for an argument that holds `/`, and, when `runtime.GOOS` is `windows`, for one that holds
`\` or `:`. The empty argument is handled before it is called.

### The installed themes

```go
// themesDirectory gives the directory sensordeck/themes in the config directory of the user.
func (current *session) themesDirectory() (string, error)

// installedTheme gives the path of the entry of the name in the themes directory and whether it is an installed theme:
// a directory, or a symbolic link that leads to one. An entry that does not exist, or a link that leads nowhere, is no
// theme and no error.
func installedTheme(directory, name string) (string, bool, error)
```

`installedTheme` calls `os.Stat`, which follows a symbolic link. `errors.Is(err, fs.ErrNotExist)` is no theme and no
error; any other error, such as `not a directory` when the themes directory is a file, is the error of R12. A regular
file of the name is no theme and no error, so the lookup goes on to the built-in themes.

### The lookup

```go
// compileTheme compiles the theme the first argument of theme compile and theme preview names: a folder path, or a
// name looked up among the installed themes and then the built-in ones.
func (current *session) compileTheme(argument string) (themecompile.Result, error)
```

The empty argument is the error of R8. A folder path goes to `compileFolder` unchanged (R9). A name asks for
`themesDirectory`, then `installedTheme`, and compiles an installed theme with `compileFolder(path)`; then
`builtinthemes.Folder(name)`, compiled with `themecompile.Compile` and its error prefixed with
`built-in theme <name>: `; then the error of R11. The themes directory is asked for only for a name, so a folder path
needs no config directory.

### The list

```go
// themeListLines gives the lines of theme list, sorted by name by their bytes: the built-in themes by name, and the
// installed themes by name with their folder.
func themeListLines(builtin []string, installed map[string]string) []string
```

A pure function, unit tested with plain values. Each line is the fields joined by a tab with no tab at the end:
`<name>\tbuilt-in`, `<name>\tinstalled\t<folder>` or `<name>\tinstalled, overrides built-in\t<folder>` (R15). The
command reads the themes directory with `os.ReadDir`, treats `fs.ErrNotExist` as an empty directory and any other error
as the error of R17, and puts in `installed` each entry for which `installedTheme` gives true, with the path it gives
(R16). An error of `installedTheme` for an entry fails the command, as it fails the lookup of that name.

### The background

```go
// MinimalDark draws the background of minimal-dark: 480 x 320 opaque pixels, by the rule of R4 of unit 0012.
func MinimalDark() *image.NRGBA
```

The package keeps the rule as data: the colours `#0b0f14`, `#141b24` and `#1f2a36`, the six panels as their top left
corners with the size 151 x 123 added, and the bar from (8, 272) to (471, 311). An unexported
`roundedRectangle{left, top, right, bottom, radius int}` with a method `holds(x, y int) bool` is the rule of R4, with the
centre clamped and the limit `radius * (radius + 1)`. A pixel is the fill when the panel shrunk by 1 pixel with radius 9
holds it, the outline when the panel of radius 10 holds it otherwise, and the background colour in no panel. The
package does no I/O.

`generate/main.go` is `package main`. It takes the output path as its one argument, encodes `background.MinimalDark()`
with `png.Encode` and writes it, failing with exit status 1 and a message on standard error. The directive in
`builtinthemes.go` is `//go:generate go run ./background/generate themes/minimal-dark/background.png`.

### The refactoring first

`configPath` in `internal/cli/run.go` joins the config directory with `sensordeck`, and the themes directory needs the
same directory. A first commit, which changes no behaviour, extracts:

```go
// ownConfigDirectory gives the directory sensordeck in the config directory of the user, with the error of
// userConfigDir unwrapped, so that each caller writes its own message.
func (current *session) ownConfigDirectory() (string, error)
```

`configPath` then joins it with `config.yaml` and keeps its message `cannot find the config directory, name the file
with --config: <cause>` as it is.

## Files

- `internal/cli/run.go`: `ownConfigDirectory`, and `configPath` through it
- `internal/builtinthemes/builtinthemes.go`: new. The embedded tree, `Names`, `Folder`, the `go:generate` directive
- `internal/builtinthemes/builtinthemes_test.go`: new. R1 to R3, R5 and R6
- `internal/builtinthemes/themes/minimal-dark/theme.yaml`: new. The text of R2
- `internal/builtinthemes/themes/minimal-dark/background.png`: new. Written by `go generate`
- `internal/builtinthemes/background/background.go`: new. `MinimalDark` and the rule of R4
- `internal/builtinthemes/background/background_test.go`: new. The rule of R4 at chosen pixels
- `internal/builtinthemes/background/generate/main.go`: new. The generator
- `internal/cli/themes.go`: new. `isFolderPath`, `themesDirectory`, `installedTheme`, `compileTheme`, `themeListLines`
- `internal/cli/themes_test.go`: new. R8, R10 to R18 through the harness, and `themeListLines`
- `internal/cli/theme.go`: `theme compile` and `theme preview` through `compileTheme`, their help, and `theme list`
- `internal/cli/theme_test.go`: the help of R14
- `e2e/builtin_theme_test.go`: new, `//go:build linux`. R23
- `README.md`: the quick start, the status line, "Planned usage", "Writing a theme" (R19, R20)
- `docs/theme-folder.md`: a section on naming a theme and the themes directory (R21)

## Tasks

### The refactoring

- [x] Extract `ownConfigDirectory` in `internal/cli/run.go` and make `configPath` use it, with its message unchanged.
  The existing tests of `run` with no config directory pass unchanged (R12)

### The background

- [x] Write `internal/builtinthemes/background/background_test.go`, which checks `MinimalDark()` is 480 x 320 and
  checks these pixels: (0, 0), a corner of the background, is `#0b0f14`; (80, 70), inside the first panel, is
  `#141b24`; (80, 8), the top edge of the first panel, and (8, 70), its left edge, are `#1f2a36`; (11, 11), on the
  rounded outline of the top left corner of the first panel, is `#1f2a36`; (8, 8) and (9, 9), just outside that rounded
  corner, are `#0b0f14`; (240, 290), inside the bar, is `#141b24`; (160, 70), between the first two panels, is
  `#0b0f14`; and every pixel has alpha 255. Each of these colours was worked out from the formula of R4 when the plan
  was written: at (11, 11) the distance to the centre (18, 18) squared is 98, within 110 for radius 10 and beyond 90
  for radius 9, and at (9, 9) it is 162 (R4, R22)
- [x] Write `internal/builtinthemes/background/background.go` as the approach says, until the test passes (R4, D7)
- [x] Write `internal/builtinthemes/background/generate/main.go` (R5)

### The theme and its package

- [x] Write `internal/builtinthemes/themes/minimal-dark/theme.yaml` as the text of R2, byte for byte, with a line feed
  after its last line (R2)
- [x] Write `internal/builtinthemes/builtinthemes_test.go` with these tests (R22):
    - `Names()` is exactly `["minimal-dark"]`, and the folder of `minimal-dark` read with `fs.ReadDir` holds exactly
      `background.png` and `theme.yaml`, both regular files (R1, R3)
    - the `theme.yaml` of the folder equals, byte for byte, a constant of the test that holds the text of R2 (R2)
    - `background.png` of the folder decodes to 480 x 320 and equals `background.MinimalDark()` at every pixel,
      compared as `color.NRGBA`, and the failure names the first pixel that differs (R4, R5)
    - `Folder` gives false for `missing`, `.` and `..` and for the name of a file, so that only a directory of
      `themes/` is a theme (R1)
    - `themecompile.Compile(Folder("minimal-dark"))` gives no error, `themefile.Parse` of its file gives no error and no
      warning, its `Tags` are the eleven mappings of R6 in order, and `themecompile.TagsFile` of them is the comment
      line followed by the block of R6 (R6, R7)
- [x] Write `internal/builtinthemes/builtinthemes.go` with the embed, `Names` (the directories of `themes/`, sorted),
  `Folder` and the `go:generate` directive (R1, D5)
- [x] Run `go generate ./internal/builtinthemes/` to write `background.png`, run it a second time and check with
  `git status` that the file did not change, and run the tests of the package until they pass (R5)

### The lookup

- [x] Write `internal/cli/themes_test.go` with the tests of the lookup, through the harness with `configDir` set to a
  temporary directory, each checking the exit status, standard output and the files written (R22):
    - `isFolderPath` is true for `.`, `..`, `./x`, `a/b` and `/abs` and false for `x`, `x.y` and `minimal-dark` on
      Linux, and `theme compile` and `theme preview` with the empty argument fail with the error of R8 and write
      nothing (R8)
    - an installed theme alone: a folder `night` in the themes directory, written with `writeThemeFolder` content,
      compiles by the name `night` to the same bytes as `compileFolder` of its path, and its fault is named by the path
      of its `theme.yaml` in the themes directory (R10)
    - a built-in theme alone: `theme compile minimal-dark` with no themes directory writes the bytes of
      `themecompile.Compile(builtinthemes.Folder("minimal-dark"))` and its tags file (R10, R12)
    - an installed theme that replaces a built-in one: a folder `minimal-dark` in the themes directory with the content
      of `compiledTheme` is the one compiled (R10, D2)
    - a symbolic link `linked` in the themes directory to a theme folder elsewhere compiles by its name, skipped where
      symbolic links cannot be made (R10, D10)
    - a regular file `minimal-dark` in the themes directory is passed over and the built-in theme is compiled, and a
      regular file `other` gives the error of R11 for `other` (R10, R11)
    - a name found nowhere fails with the error of R11, exit status 1 and no file written (R11)
    - a config directory that cannot be found, through `configDirErr`, fails with the error of R12 for a name, and a
      folder path still compiles (R12)
    - a themes directory that is a regular file fails a name with `cannot examine the installed theme <path>: not a
      directory` (R12)
    - R13: the files of the built-in folder copied with `fs.ReadFile` into a temporary folder compile, by path, to the
      same theme file and tags file, byte for byte, as `theme compile minimal-dark` (R13)
    - `theme preview minimal-dark` writes the preview of the compiled built-in theme (R10)
    - each command of the lookup, with `--device /nowhere` and without it, with `sourcesErr` set, opens no transport
      (R18)
- [x] Add to `internal/cli/theme_test.go` a test that the help of `theme compile` and of `theme preview` holds
  `<name-or-folder>` (R14)
- [x] Write `isFolderPath`, `themesDirectory`, `installedTheme` and `compileTheme` in `internal/cli/themes.go`, and make
  `theme compile` and `theme preview` in `internal/cli/theme.go` call `current.compileTheme(args[0])` with the use
  lines and short texts of the table above, until the tests and the existing tests of the folder pass (R8 to R14, R18)

### The list

- [x] Add to `internal/cli/themes_test.go` (R22):
    - `themeListLines` with no installed theme, with an installed theme whose name sorts before the built-in one, and
      with one that overrides it, against the exact lines of R15 (R15)
    - `theme list` through the harness for the cases of the lookup: no themes directory gives `minimal-dark\tbuilt-in`
      alone; an installed folder, a symbolic link to a folder, a regular file that is left out, and an installed
      `minimal-dark` listed once as `installed, overrides built-in` with its folder; a config directory that cannot be
      found and a themes directory that is a regular file fail with status 1 and the errors of the table; an argument
      is refused; no transport is opened, with `--device /nowhere` and without it (R15 to R18)
- [x] Write `themeListLines` in `internal/cli/themes.go` and the command `theme list` in `internal/cli/theme.go`, added
  to `themeCommand`, until the tests pass (R15 to R17)

### End to end

- [ ] Write `e2e/builtin_theme_test.go`, running the executable with `runWith` and `XDG_CONFIG_HOME` set to a temporary
  directory, on `fakescreen.New(test)` with no screen added (R23):
    - `theme list` with no themes directory, with an installed folder `night`, and with an installed `minimal-dark`,
      against the exact standard output
    - `theme compile minimal-dark <dir>/img.dat`: status 0, the line of R22 of unit 0011 on standard output, the theme
      file parsed by `themefile.Parse` with no warning, and `img.tags.yaml` equal to the comment line and the block of
      R6
    - `theme preview minimal-dark <dir>/preview.png` equal to the PNG encoding of `themeimage.Preview` of the theme
      file compiled in the test before
    - an installed theme compiled by its name, and the folder `night` compiled as `./night` with the working directory
      of the command set to the directory that holds it, the two giving equal files
    - `theme compile nowhere <dir>/img.dat` with the error of R11 on standard error, status 1 and no file in `<dir>`
    - `theme list`, `theme compile minimal-dark` and `theme preview minimal-dark` with `--device /nonexistent` succeed
      with no screen (R18)

  `runWith` sets no working directory; add a variant of it, or a field, that sets `command.Dir`, as the smallest change
  the test needs

### The documents

- [ ] Update `README.md` (R19, R20):
    - the status line names the built-in theme `minimal-dark` and `theme list` among what works, and says that
      `theme compile` and `theme preview` take a theme by name or folder
    - a section "Quick start" before "Planned usage", in the order of R19, with the tags block of R6 in full, every
      point of the list of R19, and the line that it is for Linux. It says nothing about what the device shows for a
      value it was never sent beyond what R19 allows, and does not say the theme has been seen on the device (D11)
    - "Planned usage" shows `sensordeck theme preview <name-or-folder> <out.png>`
    - "Writing a theme" names the folder as `./my-theme` in its two commands, and describes `theme list`, the rule of
      R8, the order of R10, the themes directory (`~/.config/sensordeck/themes`, or `$XDG_CONFIG_HOME/sensordeck/themes`,
      on Linux, and `%AppData%\sensordeck\themes` on Windows), and that a folder copied there under a name is found by
      that name and replaces a built-in theme of that name
- [ ] Update `docs/theme-folder.md` (R20, R21): a section "Naming a theme" after "The folder" with the rule of R8, the
  lookup order of R10, the themes directory on Linux and on Windows, the override of a built-in theme, `theme list` and
  its three kinds of line, and the built-in themes by name, `minimal-dark`; and `<theme-dir>` in the command of "The
  tags" written as `<name-or-folder>`. Check with `grep -n "theme \(compile\|preview\) [a-z]" README.md docs/*.md` that
  no command names a folder by a bare name

## Verification

- [ ] `go test ./...` passes
- [ ] `go vet ./...` reports nothing
- [ ] `gofmt -l .` prints no file name
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
- [ ] `go generate ./internal/builtinthemes/` leaves `git status` clean
- [ ] `npm run check-changes` passes on the committed work

## Commit Points

- [x] `feature-0012 Extract the config directory of sensordeck`, the refactoring alone
- [x] `feature-0012 Add the built-in theme minimal-dark`, the background, its generator, the theme folder and
  `internal/builtinthemes` with their tests
- [x] `feature-0012 Find a theme by name in theme compile and theme preview`, the lookup with its tests and the help
- [x] `feature-0012 Add theme list`, with its tests
- [ ] `feature-0012 Test the built-in theme and the theme names end to end`
- [ ] `feature-0012 Document the quick start and the theme names`, the README and `docs/theme-folder.md`
