# The theme in the config file, uploaded when it changes

## Approach

The work falls into five parts, each with its own module, and `run` puts them together.

* **The state file** is a new package, `internal/uploadstate`, which holds the hash of R5, the path of R14, the
  decoding and encoding of the JSON object, and its reading and its writing through a temporary file and a rename. It
  knows no serial, transport or command, and gives the errors of the file system as they are, so that `internal/cli`
  writes every message (R5, R13 to R15)
* **The config file** learns the key `theme` in `internal/config/config.go` (R1, R2), and gets the pure function that
  edits it for `theme use` in a new file `internal/config/edit.go` (R25, R26). The resolving of a relative folder path
  against the directory of the config file is in `internal/cli`, where the rule of R8 of unit 0012 lives (R1, D2)
* **The serial** comes with the opened node: `hidraw.Transport` gets `Serial()`, set at each opening that succeeds, and
  the `transport` interface of `internal/cli` gets the same method, which the trace passes through (R6, D5)
* **The upload that keeps the state file true** is one type of `internal/cli`, `stateFile`, used by `upload` and by
  `run`, so that the rule of D7 and D20 is written once (R8, R35 to R37)
* **`run`** gets a `themeHolder`, which decides whether the theme of the config is on the screen, uploads it and keeps
  the waits of R9, and a `configWatcher`, which checks the file by its time and size at the start of each round and
  gives a reloaded config (R7 to R12, R17 to R21). The feeder asks the watcher, then reopens a lost screen, then asks
  the holder, then sends the clock and the values. `theme use` is a new command beside the other theme commands (R22 to
  R28)

The end-to-end tests need the fake screen to answer an upload that starts at a moment the test cannot count in
advance, because values are sent before it. The fake screen gets `AcceptUpload`, which arms the answers of the upload
of a given file at the next reset report that arrives (R33).

None of items 2 to 4 of "Beyond the request" is built, as the record decided. Item 1 is R35 to R38.

Before this plan was written, the YAML parser of the tree, `go.yaml.in/yaml/v3`, was checked in the sandbox for what the
edit of R25 relies on: an empty file and a file of comments only give a node of kind 0 with no content; a root in flow
style has `Style` `FlowStyle`; each key and value carries its line and column; `theme: 2024` gives a scalar with the
tag `!!int`; a block scalar `|` has the style `LiteralStyle` and starts on the line of the key; and a plain scalar
continued on the next line reads as one value joined by a space. The spec fits the tree.

### The names and texts the implementation uses

These are decided here, so that the tests and the code agree. `<cause>` is the cause alone, through the existing
`innerError`, so that a path is not written twice.

| What | Exact text |
|---|---|
| `theme` that is not a string scalar (R1) | `theme: want a theme name or folder path as a string` |
| `theme` that is empty (R1) | `theme: the theme name or folder path is empty` |
| Both keys (R2) | `theme and tags cannot both be given, the theme gives the tags` |
| Neither key (R2) | `one of theme and tags is needed` |
| Unknown key (R2) | `unknown key "<key>", the keys are commander_color, commander_interval, interval, tags and theme` |
| Not a mapping (R2) | `want a mapping with the keys commander_color, commander_interval, interval, tags and theme` |
| A theme of `run` that fails to compile (R3) | `config file <path>: theme: <error of the compilation>` |
| No cache directory (R15, R36) | `cannot find the cache directory: <cause>` |
| A state file that cannot be read or decoded (R15, R36) | `state file <path>: <cause>, taken as holding no entry` |
| The removal of R8 or R35 cannot be written | `cannot write the state file <path>: <cause>` |
| The record after a completed upload cannot be written (R8, R37) | `cannot record the theme in the state file <path>: <cause>` |
| Before an upload of `run` (R10) | `uploading theme <value>` |
| After an upload of `run` that completed (R10) | `theme uploaded: <n> bytes` |
| The check at the start finds the theme (R10) | `theme <value> is on the screen` |
| An upload of `run` that failed (R9) | `theme upload failed: <error>` |
| A reload that is not applied (R19) | `config file <path>: <error>, keeping the running config` |
| A reloaded file that cannot be read (R19) | `<error>` is `cannot read it: <cause>` |
| A key that needs a restart (R18) | `config file <path>: <key> changed, restart run to apply it` |
| Use line and Short of `theme use` (R22) | `use <name-or-folder>`, `Set the theme of the config file of run, by name or folder` |
| A config file `theme use` cannot edit (R25) | `config file <path>: cannot be edited, set the key theme in it by hand` |
| A config file `theme use` refuses by R27 | `config file <path>: <error of config.Parse, or of the YAML parser>` |
| A directory `theme use` cannot create | `cannot create the directory <path>: <cause>` |
| A config file `theme use` cannot write | `cannot write the config file <path>: <cause>` |
| The line of `theme use` (R28) | `theme <value> set in <path>` |

In the line of a state file that cannot be decoded, `<cause>` is
`not a JSON object of serials and hashes: <error of the decoder>`.

### The state file

```go
// Package uploadstate is the state file of the uploads: which theme file each screen holds, by the serial of the
// screen and the hash of the file. It is not limited to a platform.
package uploadstate

// Hash gives the SHA-256 of the theme file as 64 lower-case hexadecimal digits.
func Hash(file []byte) string

// Path gives the state file in the cache directory of the user.
func Path(cacheDirectory string) string // <cacheDirectory>/sensordeck/uploaded-themes.json

// Decode reads a JSON object whose keys are serials and whose values are hashes. Anything else is an error.
func Decode(content []byte) (map[string]string, error)

// Encode gives the object with its keys sorted and a final line feed.
func Encode(entries map[string]string) []byte

// Read gives the entries of the file. A file that does not exist holds none and is no error. A file that cannot be
// read or decoded gives no entry and the error.
func Read(path string) (map[string]string, error)

// Write creates the directory of the file, writes the entries to a temporary file in it and renames that over the
// file. On a failure the temporary file is removed.
func Write(path string, entries map[string]string) error
```

`Decode` refuses `null` and any content whose first byte that is not white space is not `{`, because the decoder of
`encoding/json` reads `null` into a map with no error; the decoder into `map[string]string` refuses a value that is not
a string. `Read` always gives a map that is not nil, so that the caller can change it. `Encode` uses
`json.MarshalIndent` with two spaces, which sorts the keys. `Write` uses `os.MkdirAll(directory, 0o755)` and
`os.CreateTemp(directory, ".uploaded-themes-*.json")`, closes the file before `os.Rename`, which replaces the target on
Linux and on Windows.

### The key `theme`

`config.Config` gets `Theme string`, the value as the file gives it, empty when the file does not give the key. A key
counts as given when it is present, whatever its value, so `tags:` with no value is still the error of the tags. The
exclusion of R2 is checked after the loop over the keys and before any value is parsed: both given is the error of
both, neither is the error of neither. The value of `theme` must be a scalar whose `ShortTag()` is `!!str` and whose
value is not empty; `theme: 2024` is therefore not a string, and a theme named so is written quoted, which R26 does.
With `theme`, `Mappings` is empty: the mappings come from the compiler, in `internal/cli`.

### The edit of `theme use`

```go
// ErrNotEditable is the config file whose key theme cannot be set by replacing or adding one line.
var ErrNotEditable = errors.New("cannot be edited, set the key theme in it by hand")

// SetTheme gives the content with the key theme set to the value, as R25 and R26 of unit 0013 say, every other byte
// kept. Its error is that of the YAML parser for content that is not YAML, or ErrNotEditable.
func SetTheme(content []byte, value string) ([]byte, error)
```

* The content is decoded into a `yaml.Node`. A node of kind 0 is a file with no document, and the line is added. A
  document whose root is a mapping with `Style&yaml.FlowStyle == 0` is a block mapping. Any other root is
  `ErrNotEditable`
* In a block mapping, the key `theme` is found by its `Value`. When it is present, its value must be a scalar whose
  style is neither `LiteralStyle` nor `FoldedStyle` and whose `Line` is the `Line` of the key, and the line of the key
  is replaced by `theme: <scalar>` with `Column-1` spaces in front. A line that ends in `\r\n` keeps its `\r`. When the
  key is absent, `theme: <scalar>\n` is added at the end, after a `\n` when the content does not end in one
* The scalar is the value as it is when the line `theme: <value>` reads back, through the parser, as a plain scalar
  with the tag `!!str` and exactly the value. Otherwise it is the value in single quotes, each `'` doubled, when the
  value holds no character below U+0020 and no U+007F. Otherwise it is in double quotes, with `\` and `"` escaped by a
  backslash and every other such character written as `\xNN`
* The new content is decoded again, and the value of the root key `theme` must be exactly the value. Anything else,
  such as a plain scalar continued on the next line, which the replaced line leaves behind, is `ErrNotEditable`

### The serial

`hidraw.Transport` gets a field `serial` and the method `Serial() string`. The function `open` it keeps becomes
`func() (*os.File, string, error)`, and `Reopen` sets the serial with the file. `OpenDiscovered` gives the `Serial` of
the screen it opened. `OpenPath` becomes `OpenPath(sysfsRoot, devRoot, path string)` and gives the serial of the entry
of `List(sysfsRoot, devRoot)` whose node is the same file as the path, by `os.SameFile` of `os.Stat` of both, which
follows the link of the fake screen and of a udev link alike, and the empty text when no entry is, or when a `Stat` or
the listing fails. The serial is looked up after the node opened, at each opening.

The `transport` interface of `internal/cli/main.go` gets `Serial() string`. `tracingTransport` gives the serial of the
inner transport and writes no line for it. The `fakeTransport` of the tests gets a field `serial`, which a test can
change between rounds.

### The upload that keeps the state file

```go
// stateFile puts theme files on a screen and keeps the state file true for it, for upload and for run.
type stateFile struct {
	path   string
	stderr io.Writer
}

// stateFile gives the state file of the user, or the error of a cache directory that cannot be found.
func (current *session) stateFile() (stateFile, error)

// holds tells whether the state file holds the hash for the serial. A file that cannot be read or decoded holds
// nothing, with its line.
func (state stateFile) holds(serial, hash string) bool

// upload removes the entry of the serial and writes the file, uploads the theme, and records its hash under the
// serial after an upload that completed. Its error is the failed removal, before any report was written, or the
// error of the upload. A failed record writes its line and is no error.
func (state stateFile) upload(opened smartmonitor.Transport, serial string, theme []byte, pause func(time.Duration),
	progress func(stage smartmonitor.Stage, block, blocks int)) error
```

`environment` gets `userCacheDir func() (string, error)`, which `Main` fills with `os.UserCacheDir`. The serial given
to `upload` is read from the transport once, before the removal, and the hash is recorded under that serial, the
entry that was removed, even when the reopening after the upload opened another node.

`upload` resolves the state file after reading the theme file and before opening the transport, reads the serial after
opening it, and calls `stateFile.upload` with its progress lines as they are.

### `run` and its theme

```go
// configTheme is the theme of the config, compiled.
type configTheme struct {
	value string // the value of the key theme as the file gives it
	file  []byte
	hash  string
}

// runConfig is a config of run as it is applied: its settings, with the mappings of the theme when it has one.
type runConfig struct {
	settings config.Config
	theme    *configTheme // nil with tags
}

// loadConfig parses the content of the config file at the path and, with theme, compiles the theme and takes its
// tags as the mappings. Its error is the text after the prefix naming the file.
func (current *session) loadConfig(path string, content []byte) (runConfig, error)
```

`loadConfig` resolves the value: a folder path by `isFolderPath` that is not absolute is joined to the directory of the
config file, and the folder is compiled with `compileFolder`; a name goes through the lookup by name. `compileTheme` is
split for that: its part after `isFolderPath`, the lookup of a name among the installed and the built-in themes,
becomes `compileNamed(name string)`, which `compileTheme` and `loadConfig` both call. The joined path is not passed
back through `compileTheme`, because a join can lose the separator that makes it a folder path. An error of the
compilation is `theme: <error>`.

At the start, `run` takes the stamp of the config file, reads it, calls `loadConfig`, and with `theme` resolves the
state file, all before `newSources` and before the transport is opened. With `tags` it resolves the state file too
but ignores its error, so that `run` behaves as today (R4), and a reload to `theme` with no state file is refused by
R19 with the error of the cache directory.

```go
// themeHolder decides whether the theme of the config is on the screen, and uploads it when it is not.
type themeHolder struct {
	theme    *configTheme
	state    stateFile
	pause    func(time.Duration)
	now      func() time.Time
	say      func(line string)
	stderr   io.Writer
	checkDue bool          // the state file is to be read at the next call of ensure
	starting bool          // the next check is the one at the start, which writes its line
	due      bool          // the theme is not on the screen
	next     time.Time     // no attempt before this moment
	wait     time.Duration // the wait after the next failure; 10 seconds, doubled up to 5 minutes
}

// ensure tells whether values may be sent, and whether an upload completed in this call. It reads the state file
// when a check is due and uploads the theme when it is not on the screen and the wait has passed. failed is an upload
// that returned an error, after which the screen is lost.
func (holder *themeHolder) ensure(stop context.Context, opened transport) (ready, uploaded, failed bool)

// setTheme replaces the theme of the config. A theme whose hash differs from the one before, or a first theme, makes a
// check due. No theme clears what is due.
func (holder *themeHolder) setTheme(theme *configTheme)

// screenBack makes a check due.
func (holder *themeHolder) screenBack()
```

With no theme, `ensure` is ready. Once stop is done it starts no upload and is not ready. A failure sets `next` to the
time after the failed attempt plus `wait`, and then doubles `wait` up to 5 minutes. A completed upload sets `wait`
back to 10 seconds. The constants `firstRetry` and `longestRetry` of `keeper.go` are those of D8, and the holder uses
them. The progress of an upload of `run` writes nothing.

The feeder gets the holder, the watcher, and `clockDue`, which replaces the clock written at the start of `feed`. A
round becomes:

1. ask the watcher, and apply a reloaded config: its mappings replace `feeding.config.Mappings`, its theme goes to
   `setTheme`, and a line of R18 is written for each key of `restartKeys`
2. when the screen is lost, try one `Reopen`; on success write `screen back`, call `screenBack` and set `clockDue`
3. call `ensure`: a failed upload marks the screen lost and ends the round, a holder that is not ready ends it, and a
   completed upload sets `clockDue`
4. write the clock when `clockDue` or when the period has passed, clearing `clockDue` when it was written
5. send the values

`feed` sets `clockDue` and starts the rounds, so the first round sends the clock as `feed` does today.

```go
// restartKeys gives the keys that changed and that a reload does not apply, in the order interval,
// commander_interval, commander_color.
func restartKeys(running, reloaded config.Config) []string
```

The keeper reads its probes when a commander probe is mapped. Today it takes that from the config at the start, so a
reload that maps a probe would not get its value under `commander_color`, and R18 would be false. `probesMapped`
becomes a `func() bool` that the keeper calls in `readings`, the feeder's goroutine, and `run` gives it a function of
the running mappings.

### The watcher

```go
// fileStamp is what os.Stat gives of the config file, or that it gave an error.
type fileStamp struct {
	failed  bool
	modTime time.Time
	size    int64
}

// configWatcher gives a reloaded config when the stamp of the file changed.
type configWatcher struct {
	path   string
	last   fileStamp
	load   func(content []byte) (runConfig, error)
	stderr io.Writer
}

// poll takes the stamp and, when it differs from the last one, reads and loads the file. It gives the config when it
// loaded, and writes the line of R19 when it did not. The stamp becomes the last one either way, so that a broken file
// is reported once.
func (watcher *configWatcher) poll() (runConfig, bool)
```

Two stamps are the same when both failed, or neither failed and their times are `Equal` and their sizes the same. The
`load` that `run` gives is `loadConfig` for the path, and refuses a config with `theme` when no state file was
resolved at the start.

### `theme use`

The command is in a new file `internal/cli/themeuse.go`. It takes the path of `configPath`, compiles the argument with
`compileTheme`, takes the value of R24 (`filepath.Abs` for a folder path, which cleans it, and the argument for a
name), reads the file (a file that does not exist is empty content to be created), calls `config.SetTheme`, parses the
result with `config.Parse`, and writes it with `os.WriteFile(path, content, 0o644)`, after `os.MkdirAll` of the
directory with `0o755` when the file did not exist. `os.WriteFile` truncates an existing file through a symbolic link
and keeps its permissions, which the mode given only sets on a file it creates. An error of `SetTheme` and of
`config.Parse` is prefixed with `config file <path>: `.

### The fake screen

```go
// AcceptUpload arms the answers of an upload of the theme: at the next reset report that arrives after the call, the
// screen answers ACK after each report at which smartmonitor.Upload reads, counted from that reset report.
func (screen *Screen) AcceptUpload(theme []byte)
```

It runs `smartmonitor.Upload` on a recorder in memory that answers ACK and reopens at once, and keeps the count of
reports written at each `Read`. The screen keeps how many complete reports it has looked at, and `receive` looks at the
new ones for the reset report; the reset report at position *p*, counted from 1, arms answer rules at *p* - 1 plus each
count. Calls queue, one upload each.

## Files

- `internal/uploadstate/uploadstate.go`, `internal/uploadstate/uploadstate_test.go`: new, the state file
- `internal/config/config.go`, `internal/config/config_test.go`: the key `theme`, and the texts of R2
- `internal/config/edit.go`, `internal/config/edit_test.go`: new, `SetTheme` and `ErrNotEditable`
- `internal/hidraw/transport.go`, `internal/hidraw/transport_test.go`: `Serial`, and `OpenPath` with the roots
- `internal/cli/device_linux.go`: the roots passed to `OpenPath`
- `internal/cli/main.go`: `Serial` in `transport`, `userCacheDir` in `environment`
- `internal/cli/trace.go`, `internal/cli/trace_test.go`: `Serial` passed through
- `internal/cli/statefile.go`, `internal/cli/statefile_test.go`: new, `stateFile`
- `internal/cli/commands.go`, `internal/cli/commands_test.go`: `upload` keeps the state file; the help names `theme use`
- `internal/cli/themes.go`: `compileNamed` split out of `compileTheme`
- `internal/cli/runconfig.go`, `internal/cli/runconfig_test.go`: new, `configTheme`, `runConfig`, `loadConfig`,
  `restartKeys`, `fileStamp` and `configWatcher`
- `internal/cli/themeholder.go`, `internal/cli/themeholder_test.go`: new, `themeHolder`
- `internal/cli/run.go`, `internal/cli/run_test.go`: the start, the round and the reload of `run`
- `internal/cli/keeper.go`, `internal/cli/keeper_test.go`: `probesMapped` as a function
- `internal/cli/theme.go`: `theme use` added to the theme command
- `internal/cli/themeuse.go`, `internal/cli/themeuse_test.go`: new, `theme use`
- `internal/cli/fake_test.go`: `serial` and a hook after each write in `fakeTransport`
- `internal/fakescreen/fakescreen.go`, `internal/fakescreen/fakescreen_test.go`: `AcceptUpload`
- `e2e/main_test.go`: `XDG_CACHE_HOME` set to a temporary directory for every run
- `e2e/theme_upload_test.go`: new, the end-to-end tests of R33
- `README.md`, `docs/theme-folder.md`: R29 to R31
- `changes/0013-theme-key-and-upload-on-change/record.md`: the log

## Tasks

### The state file

- [ ] Write `internal/uploadstate/uploadstate_test.go`: `Hash` of a known content against its SHA-256 written by hand;
  `Path` under a cache directory; `Decode` of an object of strings, of `{}`, and its error for content that is not
  JSON, for `null`, for `[]`, and for an object with a value that is not a string; `Encode` sorting the keys, ending
  in a line feed, and decoding back to the same map, with the empty serial as a key; `Read` of a file that does not
  exist giving an empty map and no error, and of a file that is not JSON giving an empty map and the error; `Write`
  creating the missing directory, leaving the state file as the only file of the directory, and replacing the file by
  a rename, which a hard link made to the old file before the write proves by keeping the old content (R5, R13 to R15)
- [ ] Write `internal/uploadstate/uploadstate.go` as the approach says, and make the tests pass (R5, R13 to R15)
- [ ] Commit

### The key `theme`

- [ ] Add to `internal/config/config_test.go`: `theme` alone giving `Theme` and no mapping; `theme` with `tags`, also
  with `tags:` with no value, giving the error of both; neither key giving its error, which replaces the expectation
  of the case "no tags"; `theme: 2024`, `theme: [a]` and `theme: {a: 1}` giving the error of a value that is not a
  string; `theme: ""` and `theme:` giving the error of an empty value; the texts of an unknown key and of a root that
  is not a mapping naming `theme`, which replaces the expectation of the existing case (R1, R2)
- [ ] Change `internal/config/config.go` as the approach says, and make the tests pass (R1, R2)
- [ ] Write `internal/config/edit_test.go` for `SetTheme`: an empty content; a file of comments only; a file whose key
  `theme` is replaced, with a comment on another line and an indented key kept byte for byte; a line of `theme` that
  ends in `\r\n`; a file with another key and comments, with and without a final line feed, where the line is added;
  a root in flow style, a root that is a list, a value in a block scalar, a value on the line after the key, and a
  plain value continued on the next line, each `ErrNotEditable`; content that is not YAML giving the error of the
  parser; and values that need quotes, `C:\Users\ada\my theme`, `a #b`, `it's`, `2024`, `true`, `- x`, and one with a
  tab, each read back by `config.Parse` as exactly the value (R25, R26)
- [ ] Write `internal/config/edit.go` as the approach says, and make the tests pass (R25, R26)
- [ ] Commit

### The serial

- [ ] Add to `internal/hidraw/transport_test.go`: `Serial` of a discovered screen with a serial, of one with an empty
  serial, of `OpenPath` of the node of a discovered screen, of `OpenPath` of the pseudo terminal behind that node, of
  `OpenPath` of a node with no entry under the sysfs root, and the serial after a `Reopen` by discovery that opened a
  screen moved to another number with another serial; and change the existing calls of `OpenPath` to pass the roots
  (R6)
- [ ] Change `internal/hidraw/transport.go` as the approach says, pass the roots in `internal/cli/device_linux.go`, and
  make the tests pass (R6)
- [ ] Add `Serial() string` to `transport` in `internal/cli/main.go`, pass it through in `tracingTransport` with a case
  in `internal/cli/trace_test.go` that the trace gives the serial of the inner transport and writes no line, and give
  `fakeTransport` the field `serial` (R6)
- [ ] Commit

### The state file kept by `upload`

- [ ] Add `userCacheDir` to `environment` and to `Main`. Give the harness of `internal/cli/commands_test.go` the fields
  `cacheDir` and `cacheDirErr`, with `cacheDirErr` set by `newHarness` to an error of its own, so that no test writes
  a state file outside a temporary directory; and give `fakeTransport` a hook called after each write with the count
  of writes, which the tests use to look at the state file at a given report and to stop the run (R35, R11)
- [ ] Write `internal/cli/statefile_test.go` for `stateFile`: `holds` for the hash stored for the serial, for another
  hash, for another serial, for the empty serial, for no state file, and for a file that is not JSON, with its one
  line; `upload` with the entry already removed and written when the first report is written, the hash recorded under
  the serial after an upload that completed and not after one that failed, no report when the removal cannot be
  written (the cache directory made a file), and the line and no error when the record cannot be written (the hook
  turns the directory of the state file into a place where the rename fails, by replacing the state file with a
  directory after the removal) (R8, R13, R15, R35, R37)
- [ ] Add to `internal/cli/commands_test.go`: `upload` recording the hash of the file it uploaded, a vendor file being
  any bytes; the entry removed before its first report; no report and exit status 1 with the error when the cache
  directory cannot be found and when the removal cannot be written; exit status 0 with the line of the record and the
  line `upload complete: ...` when the record cannot be written; and the existing tests of `upload` given a cache
  directory, their output otherwise unchanged (R35 to R37)
- [ ] Write `internal/cli/statefile.go` and change `upload` in `internal/cli/commands.go`, and make the tests pass (R35
  to R37)
- [ ] Commit

### `run` and its theme

- [ ] Split `compileNamed` out of `compileTheme` in `internal/cli/themes.go`, with the tests of `themes_test.go`
  passing unchanged, and commit it alone as a refactoring
- [ ] Write the tests of `loadConfig` in `internal/cli/runconfig_test.go`: `tags` alone giving its mappings and no
  theme; a built-in theme by name giving the tags `theme compile` writes for it and the hash of its file; a relative
  folder path resolved against the directory of the config file and not the working directory; `.` resolved to the
  directory of the config file; an absolute folder path; a theme with no number widget giving no mapping; and a theme
  that fails to compile giving `theme: <error>` (R1, R3, R5, R16)
- [ ] Write `internal/cli/themeholder_test.go` for `themeHolder` with a fake transport and the fake clock: an upload at
  the first check with no state file, with the line before and the line after; no upload and the line of a theme on
  the screen when the hash is stored for the serial; an upload when the hash stored differs and when it is stored for
  another serial; an upload for the empty serial and none after it; no line of a theme on the screen at a later check;
  a failed upload recording nothing, writing its line, giving failed, and the next attempt made no earlier than 10
  seconds after it, then 20, 40 and so on up to 5 minutes, and back to 10 seconds after an upload that completed; a
  check made due by `screenBack` and by `setTheme` with another hash and not with the same hash; no theme clearing
  what is due; and no upload started once stop is done (R7 to R10, R12, R13)
- [ ] Add to `internal/cli/run_test.go`: with `theme`, the upload before the first clock, then the clock and the values
  of the tags of the theme, in that order; no value and no clock while the theme is not on the screen after a failed
  upload, with the screen lost and reopened in the rounds that follow; the check after the screen came back, which
  uploads when the serial of the transport changed and not when it did not, before the clock; a stop that arrives
  during the upload, by the hook at the reset report, ending `run` after the upload with the hash recorded and with no
  report after the upload, and status 0; two starts with the same theme and state file making one upload, the second
  writing the line of a theme on the screen; the theme with `tags`, a theme that does not compile, and a cache
  directory that cannot be found with `theme`, each ending with status 1 and its error before the transport is
  opened; and with `tags`, no state file read or written when the cache directory cannot be found (R3, R4, R7 to R13,
  R15)
- [ ] Write `internal/cli/runconfig.go` with `configTheme`, `runConfig` and `loadConfig`, `internal/cli/themeholder.go`,
  and change the start and the round of `run` in `internal/cli/run.go` as the approach says, and make the tests pass
  (R3, R4, R7 to R13, R15, R16)
- [ ] Commit

### A change of the config file while `run` runs

- [ ] Add to `internal/cli/runconfig_test.go`: `restartKeys` for no change, for each of the three keys alone, for all
  three, and for `commander_color` given and removed; the stamps of two failed `Stat`s being the same, and of a time
  or a size that differs not; and `poll` with a time changed by `os.Chtimes` and the same content, with a size changed
  and the same time, with no change, with a file that becomes invalid, reported once and not again at the next poll
  with the same stamp, and read again after the next change, and with a file deleted (R17, R19, R21)
- [ ] Add to `internal/cli/run_test.go`, with the config file changed in the hook of a wait: new mappings sent from the
  round after the change; an invalid config keeping the running mappings with one line; a deleted file the same; a
  change of each of `interval`, `commander_interval` and `commander_color` writing its line and keeping the running
  value, the rounds staying on the interval of the start; `tags` to `theme` uploading the theme and then sending its
  tags; `theme` to `tags` uploading nothing and sending the new mappings; a rewrite with the same theme compiling it
  again and uploading nothing, and a theme folder edited before the rewrite uploading it; a reload to `theme` with no
  cache directory refused with its line; and under `commander_color`, a reload that maps a probe getting the probe's
  value from the keeper (R17 to R21)
- [ ] Add to `internal/cli/keeper_test.go` that the keeper asks the session for probes when the function of
  `probesMapped` turns true, and not while it is false, and change its existing constructions to pass a function (R18)
- [ ] Write `fileStamp`, `configWatcher` and `restartKeys` in `internal/cli/runconfig.go`, make `probesMapped` a
  function in `internal/cli/keeper.go`, wire the watcher into the round of `run`, and make the tests pass (R17 to R21)
- [ ] Commit

### `sensordeck theme use`

- [ ] Write `internal/cli/themeuse_test.go`: a missing file and its missing directories created with the one line and
  the line of R28; the default path in the config directory, and the error of a config directory that cannot be found;
  a file with `theme` replaced and every other byte kept; one with another key and comments where the line is added;
  one with no final line feed; one of comments only; a root in flow style refused and unchanged; a file with `tags`
  refused with the error of `config.Parse` and unchanged; a file that is not YAML refused and unchanged; a theme that
  does not compile refused and the file unchanged; a relative folder written as its absolute cleaned path; a folder
  whose path needs quotes, read back by `loadConfig` as the same folder; a symbolic link to the config file, which
  stays a link with the target changed; the permissions `0o600` of a file kept; and no device opened and no source
  made (R22 to R28)
- [ ] Write `internal/cli/themeuse.go`, add it to `themeCommand` in `internal/cli/theme.go`, and make the tests pass;
  add `use` to the names the help test of `internal/cli/commands_test.go` looks for under `theme --help` (R22 to R28)
- [ ] Commit

### The end-to-end tests

- [ ] Add a case to `internal/fakescreen/fakescreen_test.go` that `AcceptUpload` answers an upload made through
  `hidraw.OpenDiscovered` after a few other reports, and write `AcceptUpload` in `internal/fakescreen/fakescreen.go`
  (R33)
- [ ] Make `environmentOf` in `e2e/main_test.go` take the test and set `XDG_CACHE_HOME` to a temporary directory of it
  before the variables the test gives, so that no run of the suite writes into the cache of the user, and a test that
  runs several processes on one cache passes its own (R33)
- [ ] Write `e2e/theme_upload_test.go`, with `XDG_CONFIG_HOME` and `XDG_CACHE_HOME` set to temporary directories, a
  screen with the serial `2022100600`, and the compiled file of a theme taken from `theme compile` of the executable:
  `theme use minimal-dark` creating the config file with `theme: minimal-dark`; `run` making the reports of
  `smartmonitor.Upload` for the compiled `minimal-dark`, then the clock and a value report with the tag of `cpu.temp`,
  and the state file holding the hash under the serial, the cache directory holding no other file; a second `run`
  writing the line of a theme on the screen and starting with the clock; `theme use` of an installed theme that shows
  `cpu.clock` alone while `run` runs, followed by the reports of its upload and a value report with tag 1 and the value
  of `cpu.clock`; `theme` with `tags` refused before any report; a theme folder that does not compile refused by `theme
  use` with the config file unchanged and by `run` before any report; `upload` of another file recording its hash,
  after which `run` uploads `minimal-dark`; and `upload` of the compiled `minimal-dark` recording its hash, after which
  `run` uploads nothing (R12, R16, R33, R35, R38)
- [ ] Commit

### The documents

- [ ] Change the quick start of `README.md` to the steps of R29, keeping its text on the tiles, on the GPU and disk
  tiles, on a metric the host cannot read and on `theme preview`; name `theme use` in the status line; keep "Planned
  usage" (R29, R30)
- [ ] Describe in "Showing sensor values on the screen" of `README.md` the key `theme` and that it excludes `tags`, that
  `tags` stays for a vendor theme uploaded with `upload`, which keys a reload applies within a round and which need a
  restart, and that an upload is not interrupted by a stop; and in "Writing a theme" the upload only when the hash of
  the compiled file differs from the one stored for the serial of the screen, the path of the state file on Linux and
  on Windows, that deleting it makes the next start upload, that `sensordeck upload` keeps it true, that a theme put on
  the screen by other software is not known to `run`, that `theme use` sets the key, and that running it again after
  editing a theme folder puts the edit on the screen; replace the paragraph that tells to paste the tags block and
  upload by hand (R30)
- [ ] Say in "The tags" of `docs/theme-folder.md` that the tags file is for a config with `tags`, and that with the key
  `theme` the config needs no tags block (R31)
- [ ] Commit

### Handing over

- [ ] Check each case that R32 lists against the unit tests written above, and add a test for any case missing (R32)
- [ ] Run the five commands of the suite in `AGENTS.md` and `npm run check-changes`, all succeeding (R34)
- [ ] Log the implementation in `record.md`, tick the tasks, and commit

## Verification

- [ ] `go test ./...` passes
- [ ] `go vet ./...` reports nothing
- [ ] `gofmt -l .` prints no file name
- [ ] `GOOS=windows GOARCH=amd64 go build ./...` succeeds and reports nothing
- [ ] `GOOS=windows GOARCH=amd64 go vet ./...` succeeds and reports nothing
- [ ] `npm run check-changes` succeeds

## Commit Points

- [ ] `feature-0013 Add the state file of the uploads`, with `feature-0013 Take the key theme in the config file` after
  it, the first holding `internal/uploadstate`, the second `internal/config`
- [ ] `feature-0013 Give the serial of the opened screen`
- [ ] `feature-0013 Keep the state file true in upload`
- [ ] `feature-0013 Split the lookup of a theme by name out of compileTheme`
- [ ] `feature-0013 Upload the theme of the config in run when it is not on the screen`
- [ ] `feature-0013 Apply a changed config file while run runs`
- [ ] `feature-0013 Add theme use`
- [ ] `feature-0013 Test the theme of the config and its upload end to end`
- [ ] `feature-0013 Document the theme of the config and its upload`
- [ ] `feature-0013 Record the implementation and tick the plan`
