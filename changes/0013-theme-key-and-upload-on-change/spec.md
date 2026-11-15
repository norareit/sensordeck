# The theme in the config file, uploaded when it changes

## Context

Step 8 of the roadmap of the initial plan is the service mode: `run` in the background, with the theme uploaded again
only when it changed, by a hash stored locally. This unit is its first half. It makes `run` take the theme itself and
keep it on the screen, and adds `sensordeck theme use`. Installing `run` as a service is the next unit.

Today, after unit `0012-built-in-theme-and-theme-names`, a user compiles a theme with `theme compile`, uploads the file
with `upload`, and pastes the `tags:` block of the tags file into the config file of `run`. The initial plan says how it
is meant to work instead, under "Where themes live" and "Switching and installing": `theme use` writes the active theme
to the config file, the running service picks the change up, compiles the theme, uploads it, which takes about 10
seconds and a reboot of the device, and resumes sending values; the hash of the theme on the device is stored per
serial of the device, so that a restart uploads nothing; and a compile cache keyed by a content hash goes in the
directory `sensordeck` in `os.UserCacheDir()`.

What the code gives today, which this unit builds on:

- `run` reads its config file once at the start, through `config.Parse`, which accepts the keys `interval`,
  `commander_interval`, `commander_color` and `tags`, requires `tags` with at least one entry and refuses any other
  key. The path is `--config` or `sensordeck/config.yaml` in `os.UserConfigDir()`. It then opens the screen and fails
  when it cannot
- The feeder of `run` sends the clock at the start, once a minute and after the screen came back, and the values of the
  mapped metrics in each round. A failed write marks the screen lost and writes `screen lost: <error>` to standard
  error; each later round tries one `Reopen`, and the first that succeeds writes `screen back` and sends the clock
- `smartmonitor.Upload` sends a theme file and, after the closing block, waits for the device to reboot and opens it
  again, with a pause of 1 second and up to 20 attempts 500 milliseconds apart, on the transport it was given. A
  completed `Upload` therefore leaves the transport open on the rebooted screen. `upload` is the command that calls it
- The theme lookup of unit 0012 takes a theme name or a folder path, by the rule that a folder path is `.`, `..` or
  holds a separator, and compiles it with the compiler of unit `0011-theme-compiler`, whose result is the theme file
  and the tags it assigned. The same folder gives byte-identical files run after run of the same executable, as
  `docs/theme-folder.md` says
- The discovery reads the serial of a screen from `HID_UNIQ` in the uevent of its hidraw entry, which may be empty, and
  `devices` shows it. The transport does not give it to the commands. The initial plan gives `2022100600` as an example
  of the serial of the device; whether two devices can have the same one is not known
- The fake screen of the end-to-end tests answers only with the reports set in advance for a count of reports
  received, and can move to another hidraw number; the end-to-end test of `upload` computes the counts of its answers
  from the reports of the upload

"The theme of the config" below is the theme the key `theme` names. "The state file" is the file of R14. "A theme is
on the screen" means that the state file holds the hash of its compiled file for the serial of the screen.

## Requirements

### The key `theme`

- **R1** The config file of `run` accepts the key `theme`, whose value is a string that names a theme as the first
  argument of `theme compile` does: a theme name, looked up as R10 of unit 0012 says, or a folder path, by the rule of
  R8 of unit 0012. A folder path that is not absolute is taken relative to the directory of the config file. A value
  that is not a string, or is empty, is an error that names the key `theme`
- **R2** A config file holds `theme` or `tags`, and not both. Both is the error
  `theme and tags cannot both be given, the theme gives the tags`, and neither is an error that says that one of
  `theme` and `tags` is needed. The text of the error of an unknown key, and of a file that is not a mapping, names
  `theme` among the keys
- **R3** With `theme`, `run` compiles the theme before it opens the screen, and the mappings it sends are the tags the
  compiler assigned, exactly the mapping `theme compile` writes into the tags file of that theme. A theme that fails to
  compile ends `run` before the screen is opened with exit status 1 and an error that names the config file, the key
  `theme` and the error of the compilation. A theme with no number widget is valid and gives no mapping, and `run`
  then sends the clock and no value
- **R4** With `tags` and no `theme`, `run` behaves as it does today: it compiles nothing, uploads nothing and reads and
  writes no state file

### The upload on change

- **R5** The hash of a compiled theme is the SHA-256 of the bytes of its theme file, written as 64 lower-case
  hexadecimal digits
- **R6** The serial of a screen is the `HID_UNIQ` of the hidraw entry of the node that the transport opened, the same
  text `devices` shows, and the empty text when the entry has none or the node has no entry under the sysfs root, as
  for a `--device` node that is not a discovered screen. After a reopening, it is the serial of the node opened then
- **R7** With `theme`, `run` checks whether the theme of the config is on the screen at each of these moments, and when
  it is not, uploads it before it sends any value:
    1. after it opened the screen at the start
    2. after it applied a reloaded config, R18, whose theme compiled to a file with another hash than the theme of the
       config before
    3. after the screen came back from being lost, before the clock is sent
- **R8** An upload of `run` goes through `smartmonitor.Upload` on the open transport, and so ends with the screen
  opened again after its reboot. Before the first report of the upload is written, `run` removes the entry of the
  serial from the state file and writes the file. When that write fails, the upload does not start and counts as
  failed. After an `Upload` that returned no error, `run` records the hash under the serial in the state file, sends
  the clock, and goes on to send the values of the round
- **R9** An upload that returned an error records nothing, writes `theme upload failed: <error>` to standard error,
  and marks the screen lost, so that the rounds that follow reopen it as they do for any lost screen. While the theme
  of the config is not on the screen, `run` sends the screen no value and no clock. The next attempt of the upload is
  made no earlier than 10 seconds after the attempt that failed, and each further failure doubles that wait, up to 5
  minutes. An upload that completes resets the wait to 10 seconds
- **R10** `run` writes these lines about the theme to standard output: `uploading theme <value>` before an upload,
  where `<value>` is the value of the key; `theme uploaded: <n> bytes` after one that completed; and
  `theme <value> is on the screen` when the check of R7 at the start finds it there. It writes no progress line for
  the data blocks
- **R11** An upload is not interrupted by Ctrl+C or SIGTERM: a stop that arrives during an upload ends `run` after the
  upload ended, with the hash recorded when it completed
- **R12** Two starts of `run` with the same theme of the config and the same screen, the first of which completed its
  upload, make one upload in all. The second writes the line of a theme on the screen and sends values at once
- **R13** A screen with an empty serial is keyed in the state file by the empty text, as any other serial is
- **R14** The state file is `sensordeck/uploaded-themes.json` in the directory that `os.UserCacheDir()` gives: on
  Linux `$XDG_CACHE_HOME/sensordeck/uploaded-themes.json`, or `~/.cache/sensordeck/uploaded-themes.json` without that
  variable, and on Windows `%LocalAppData%\sensordeck\uploaded-themes.json`. It is a JSON object whose keys are serials
  and whose values are hashes. `run` creates the directory when it writes the file, and writes the file by writing a
  temporary file in the same directory and renaming it over the state file, so that a stop at any moment leaves the old
  content or the new one
- **R15** A state file that does not exist holds no entry. A state file that cannot be read or is not such an object
  is taken as holding no entry, with one line on standard error that names its path and the cause, and is replaced at
  the next write. When `os.UserCacheDir()` fails and the config has `theme`, `run` ends before opening the screen with
  exit status 1 and an error that says the cache directory cannot be found
- **R16** No compiled theme file is kept on disk by `run`: it compiles the theme in memory at the start and at each
  reload, and the state file is the only file it writes

### A change of the config file while `run` runs

- **R17** At the start of each round, and while the screen is lost too, `run` reads the modification time and the size
  of the config file. When either differs from the last ones it read, it reads the file, parses it and, with `theme`,
  compiles the theme. The first ones are read before the file is read at the start
- **R18** A reloaded config that parses and, with `theme`, compiles, is applied: its mappings, which come from the
  theme or from `tags`, replace the running ones from that round on, and its theme becomes the theme of the config.
  A change of `interval`, `commander_interval` or `commander_color` against the running config is not applied, and
  writes `config file <path>: <key> changed, restart run to apply it` to standard error for each such key, the running
  value being kept
- **R19** A reloaded config that cannot be read, does not parse, or whose theme does not compile, is not applied:
  `run` writes `config file <path>: <error>, keeping the running config` to standard error once for that modification
  time and size, and goes on with the running config. A file that is deleted is such a change
- **R20** A reload that changes the config from `tags` to `theme` uploads the theme by R7 when it is not on the screen.
  One that changes it from `theme` to `tags` uploads nothing and sends the new mappings
- **R21** The detection of R17 uses only the modification time and size that `os.Stat` gives, so that it works the same
  on Linux and on Windows. A config file rewritten with the same content, as `theme use` does with the theme already
  set, is a change, and the theme is compiled again and uploaded when its hash differs

### `sensordeck theme use`

- **R22** `sensordeck theme use <name-or-folder>` takes one argument, a theme as `theme compile` takes it, and the flag
  `--config <path>`. Without the flag the config file is the one `run` reads without it, and the same error ends it
  when the config directory cannot be found. It opens no device and reads no sensor
- **R23** It compiles the theme, as `theme compile` does, and fails on the error of the compilation with exit status 1
  and the config file unchanged
- **R24** The value it writes is the argument for a theme name, and for a folder path the absolute and cleaned path of
  the folder, so that the value names the same folder whatever the working directory of `run` is
- **R25** When the config file does not exist, it creates it, and the directories it needs, holding the one line
  `theme: <value>`. When it exists:
    - when the root is a block mapping with a key `theme` whose value is a scalar on the line of the key, that line is
      replaced by `theme: <value>` with the indentation of the key, and every other byte of the file is kept
    - when the file holds no key `theme` and its root is a block mapping or the file holds no document, as a file of
      comments only, the line `theme: <value>` is added at its end, after a line feed when the file does not end in
      one, and every other byte is kept
    - in any other case, such as a root in flow style, the command fails with an error that says the file cannot be
      edited and that the key is to be set by hand, and the file is unchanged
- **R26** The value is written so that the YAML parser reads it back as exactly the text of R24, quoted when it is not
  a plain scalar of that text, as a Windows path with `:` and `\` or a path with `#` or a space needs
- **R27** Before it writes, it parses the new content with the parser of the config of `run`. When `run` would refuse
  it, as with a `tags` key beside the new `theme`, the command fails with that error, prefixed with the path of the
  config file, and the file is unchanged. A file that is not valid YAML before the edit fails the same way
- **R28** It writes the file in place at its path, following a symbolic link, and keeps the permissions of a file that
  exists. It writes `theme <value> set in <path>` to standard output

### The documents

- **R29** The quick start of `README.md` becomes, after the udev rule: `sensordeck theme use minimal-dark`, and then
  `sensordeck run`. It no longer compiles, uploads or shows a tags block, and says that `run` uploads the theme at its
  first start, which takes about 10 seconds and a reboot of the screen. Its text on the tiles, on the GPU and disk
  tiles, on a metric the host cannot read and on `theme preview` stays
- **R30** The README, in the section on themes and in the section on the config file of `run`, describes: the key
  `theme` and that it excludes `tags`; that `run` uploads the theme only when the hash of its compiled file differs from
  the one stored for the serial of the screen; the path of the state file on Linux and on Windows; that deleting the
  state file makes the next start upload; that a theme uploaded otherwise, as with `sensordeck upload`, is not known to
  `run`, which is why deleting the file is the way to put the theme of the config back; that `theme use` sets the key
  and that a running `run` applies it within a round; that running `theme use` again after editing a theme folder puts
  the edit on the screen; which keys a reload applies and which need a restart; that an upload is not interrupted by a
  stop; and that `tags` stays for a vendor theme uploaded with `upload`. The status line names `theme use` among what
  works, and "Planned usage" is kept as it is
- **R31** `docs/theme-folder.md` says that the tags file is for a config with `tags`, and that with the key `theme` the
  config needs no tags block

### Tests

- **R32** The unit tests cover at least:
    - R1 to R3: `theme` alone, `theme` with `tags`, neither, a `theme` that is not a string, an empty one, and a
      relative folder path resolved against the directory of the config file
    - R6: a discovered screen, a `--device` node that is a discovered screen, one that is not, and a screen with an
      empty serial
    - R7 to R13 with a fake transport: an upload at the start with no state file, none when the hash is stored for the
      serial, one when the hash stored differs, one for another serial, the entry removed before the first report of
      the upload, the hash recorded after an upload that completed and not after one that failed, no value and no
      clock while the theme is not on the screen, the waits of R9, and the check after the screen came back
    - R14 and R15: a state file that does not exist, one that is not JSON, one that is JSON but not an object of
      strings, the writing through a temporary file, and a cache directory that cannot be found
    - R17 to R21: a change of the time, of the size, a config that becomes invalid and the running config kept with
      one line, a deleted file, each of the three keys that need a restart, and both changes of R20
    - R24 to R28: a missing file, a file with `theme` replaced, one with another key and comments where `theme` is
      added, one with no final line feed, one of comments only, a root in flow style, a file with `tags`, a file that is
      not YAML, a value that needs quotes, a symbolic link to the config file and the permissions kept
- **R33** The end-to-end tests run the built executable against the fake screen, with `XDG_CONFIG_HOME` and
  `XDG_CACHE_HOME` set to temporary directories, and check: `theme use minimal-dark` creating the config file; `run`
  uploading the compiled `minimal-dark`, whose reports are those `smartmonitor.Upload` writes for it, then sending the
  clock and values, and recording the hash in the state file; a second `run` that uploads nothing; `theme use` of an
  installed theme while `run` runs, followed by its upload and its tags; `theme` with `tags` refused before the screen is
  opened; and a theme that does not compile, refused by `theme use` and by `run`. The fake screen may be extended to
  answer the items of an upload
- **R34** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1 `theme` and `tags` exclude each other.** The theme decides the tags: the compiler assigns them, and a `tags`
  block beside it could only repeat them or contradict them, and a contradiction sends a value to the wrong tile. An
  error that names both keys tells the user what to remove. Rejected: `tags` overriding or adding to the tags of the
  theme, which has no use, since a metric the theme does not show has no tile; and `theme` winning with a warning,
  which leaves a block in the file that does nothing. `tags` without `theme` stays for a vendor theme, whose tags only
  the user knows
- **D2 A relative folder is relative to the config file.** `run` will run as a service, whose working directory is
  not the one the user typed in. Relative to the config file is what the file means wherever `run` is started.
  `theme use` writes an absolute path, so the rule matters only for a value written by hand
- **D3 The hash is of the compiled file.** The file is what the device holds, so its hash changes exactly when the
  device would show something else, including when a new release of sensordeck compiles the same folder differently.
  A hash of the folder would miss that, and would need a rule for which files of the folder count. Compiling the same
  folder gives the same bytes, so the hash is stable across starts
- **D4 No cache of compiled files.** The hash is computed from the compiled file, so a cache cannot save the
  compilation without a key over every input: the files of the folder, the built-in fonts and the version of the
  compiler. Compiling is done in memory, needs no device, and happens at the start and at a reload only, so a cache
  would save little time and add a second copy of knowledge that can go stale. The state file is all the plan's
  directory in `os.UserCacheDir()` holds. A cache of compiled files is an item of "Beyond the request"
- **D5 The serial comes with the opened node.** The serial is in the sysfs entry of the node, which the discovery
  already reads, so the transport gives the serial of the node it opened, at the start and at each reopening. For a
  node given with `--device` the serial is that of the entry whose node it is, so that a screen named by its node keeps
  its entry. A reopening by discovery may open another screen, which is why R7 checks again after the screen came back
- **D6 An empty serial is a key like any other.** The supported usage is one screen per machine, and a screen with no
  serial, or with a serial shared by other devices, then has one entry, which is what the user expects. Uploading at
  each start when the serial is empty was rejected: it costs about 10 seconds and a write of the flash at every start
  or reboot of the machine, which is what the step of the roadmap is to avoid. The cost is that another screen with
  the same serial, or with none, put in its place is taken to hold the theme. The README says that deleting the state
  file makes the next start upload
- **D7 The entry is removed before an upload.** An upload that is cut short, by an error, a killed process or a loss of
  power, can leave the flash with neither the old theme nor the new one. With the entry removed first, the state file
  never claims a theme the screen may not hold, and the next check uploads again. Without it, a theme that failed to
  upload over theme A, followed by a return to A in the config, would leave the screen broken with no upload. The cost
  is one write of the state file per upload. When that write fails the upload does not start, because it could then
  leave the state file claiming the old theme
- **D8 A failed upload is retried with a growing wait.** `run` is to run unattended as a service, and a device that
  refuses a theme would otherwise be written again every round, for as long as the machine runs, which wears the flash
  and fills the log. The waits are those of the setup of the controller in unit 0009, 10 seconds doubling to 5
  minutes, so the project has one rule for retrying a device. The screen is marked lost because an error of the upload
  leaves the transport in an unknown state, and the reopening of the lost screen is the existing way back. No value is
  sent while the theme is not on the screen, because values for the tags of the new theme would land on the tiles of
  whatever theme the screen holds
- **D9 The check after a return of the screen.** The plan's goal is that a reboot uploads nothing, and the check costs
  a lookup when the theme is there. It is needed for the retry of D8, which goes through the lost screen, and it gives
  the theme to another screen plugged in while `run` runs
- **D10 An upload runs to its end.** Stopping an upload half way leaves the screen with a broken theme, by D7. The
  upload takes about 10 seconds, within the time a service manager waits for a stop, so a stop waits for it. The
  upload of `smartmonitor.Upload` takes no context, and this unit does not change it
- **D11 The config file is checked by its time and size.** `os.Stat` gives both on Linux and on Windows with no code of
  a platform, and checking once a round, at least every 100 milliseconds by the minimum interval, costs one system
  call. A file watch, inotify on Linux and another interface on Windows, was rejected as two implementations and a
  dependency for a change a person makes by hand. Comparing the content each round was rejected because `theme use`
  with the same theme writes the same bytes, and that is how a user puts an edited theme folder on the screen: a
  rewrite is a change by its time, and the theme is compiled again. The cost is that a write that keeps the size and
  falls in the same tick of the clock of the file system as the one before is missed, which matters only on file
  systems with coarse times, and the next write is seen
- **D12 A reload applies the theme and the tags only.** The mappings must be replaceable while running, because a new
  theme brings new tags, and whether they come from `theme` or from `tags` is then the same change. `interval` sets
  the grid of the rounds, `commander_color` decides whether the keeper of the controller runs at all, and
  `commander_interval` the period of the reading of the controller; applying them while running means restarting
  those parts, for keys a user sets once. They keep their running values, and a line says that a restart applies
  them, so that the change is not silently lost
- **D13 An invalid reload keeps the running config.** A service whose config is being edited must not stop or show
  wrong values because of a half-written or mistyped file. The line is written once per modification time and size,
  so that a broken file does not fill the log, and the next change is read again
- **D14 `theme use` edits a line and keeps every other byte.** The config file is the user's, with their comments and
  layout. Encoding it again through the YAML library would keep the comments but change the indentation and the
  quoting of keys the user wrote. Replacing one line or adding one keeps the rest exactly, and covers the files that
  `theme use`, the README and a hand-written file give. A root in flow style is refused rather than rewritten, because
  editing it needs the reformatting this rule avoids
- **D15 `theme use` writes only a config that `run` accepts.** Parsing the new content with the parser of `run` is one
  rule that covers every way the result could be refused: a `tags` key beside the theme, a key with a bad value, or a
  file that was invalid already. The user then learns at `theme use` and not at the next start of the service.
  Removing the `tags` key itself was rejected: the block is the user's work for a vendor theme, and deleting it is not
  what setting a theme asks for
- **D16 The config file is written in place, the state file by a rename.** The config file may be a symbolic link into
  a repository of the user's dotfiles, and a rename would replace the link with a file and drop the file's
  permissions. A `run` that reads it while it is written sees a broken file, reports it once by R19, and applies the
  file at the next change of its time. The state file is sensordeck's own, in its own directory, so a rename is safe
  there and keeps it whole at any moment
- **D17 The threat model.** The config file, the themes directory and the state file belong to the user who runs
  `run`, and are trusted as theirs. A state file that is not what R14 says is taken as empty, so a damaged file costs an
  upload and nothing else. Two processes of `run` that write the state file at the same time, as with two screens each
  named with `--device`, may lose the entry the other wrote, which costs an upload at a later check. The unit does not
  defend against another user who can write these files
- **D18 No device check in this unit.** Whether the device reboots and comes back after an upload made by `run` as the
  fake screen does, whether its serial is the one the discovery reads and whether that serial is unique, is for the
  human to check on the device later. The upload itself is the one of `upload`, which unit 0003 built. The unit is
  complete without that check and does not wait for it, and a finding of the check is a new unit. Until then the
  README says nothing about these as seen on the device
- **D19 Windows is built, not run.** The state file, the reload and `theme use` use the path, cache and stat functions
  of Go, which work on Windows with no code of their own, and the suite builds and vets for Windows. The transport of
  `run` is not supported on Windows yet, so the serial of R6 has no Windows source in this unit. Running it on Windows is
  the human's to check, and the unit does not wait for it

## Non-goals

- Installing `run` as a systemd user service or a Windows startup task, which is the next unit
- `theme install` and `theme new`
- The HID transport on Windows, and so the serial of a screen on Windows
- Marking the active theme in `theme list`
- Watching the files of a theme folder for edits: an edit is put on the screen by a restart or by `theme use` again
- Applying `interval`, `commander_interval` and `commander_color` without a restart
- A local IPC or HTTP call to a running `run`, which the initial plan names beside a file watch
- Following the screen by its serial when several screens are connected, which item 3 of "Beyond the request" of unit
  0003 names
- Changing `upload`, which neither reads nor writes the state file
- Changing `smartmonitor.Upload`, its pauses or its number of attempts
- Interrupting an upload with Ctrl+C or SIGTERM

## Beyond the request

1. `sensordeck upload` recording the hash of the file it uploaded under the serial of the screen, so that `run` sees
   that the screen holds another theme and uploads its own at its next check. Cost: the serial in `upload`, a write of
   the state file and tests. Without it, after a manual upload of another file `run` keeps sending values for its own
   theme to the other one until the state file is deleted, which the README says
2. A cache of compiled theme files in `os.UserCacheDir()/sensordeck/`, keyed by a hash of every input of the
   compilation and the version of sensordeck. Cost: the key over the folder, the fonts and the version, a rule for
   pruning old entries, and tests. Without it, `run` compiles the theme at each start and reload, in memory
3. A flag of `run`, such as `--force-upload`, that uploads the theme of the config at the start whatever the state file
   says. Cost: a flag and a test. Without it, the user deletes the state file, as the README says
4. Marking the active theme in `theme list`, as the initial plan sketches. Cost: reading the config file in `theme
   list`, a rule for an invalid one, and tests. Without it, the active theme is the `theme` line of the config file
