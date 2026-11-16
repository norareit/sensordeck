# The theme uploaded at every start of `run`

## Context

Unit `0013-theme-key-and-upload-on-change` made `sensordeck run` upload the theme of its config only when the theme is
not on the screen, and decided that by a state file: `sensordeck/uploaded-themes.json` in the cache directory of the
user, which maps the serial of a screen to the SHA-256 of the compiled theme file last uploaded to it. `run` reads it at
the start, after a reload that changes the theme and after the screen came back, and uploads when the hash it finds for
the serial differs from that of the compiled theme. `sensordeck upload` keeps the file true for the uploads it makes.

The state file belongs to one user on one operating system of one machine, while the screen keeps its theme in its own
flash and can be written by anything it is plugged into. The file therefore claims a theme that the screen may no
longer hold.

"The theme of the config" below is the theme the key `theme` of the config file names, as unit 0013 defines it. "The
last upload" of a `run` is the compiled theme file of the last upload that this process of `run` completed, as R3 says.

### Observed behavior

On 2026-11-16, on the human's PC, which boots Linux and Windows from the same disk with one screen attached, `run` on
Windows uploaded `minimal-dark`. The state file on Linux still recorded `minimal-dark-probes` for the serial of the
screen, so the next `run` on Linux uploaded nothing, wrote that its theme was on the screen, and sent the tags of
`minimal-dark-probes` to a screen that showed `minimal-dark`. The tiles showed the wrong values, such as the
temperature of a cable probe of the Commander Core XT on the CPU tile.

### Expected behavior

The screen shows the theme of the config of the `run` that feeds it, with each value on its own tile, whatever put a
theme on the screen before that `run` started.

### Steps to reproduce

1. Run `sensordeck run` with the theme `A` in its config, until it has uploaded `A` and recorded it in the state file
2. Stop it, and put the theme `B` on the same screen in a way that does not write that state file: from the other
   operating system of a dual-boot machine, from another computer, from the vendor's software, or on one machine with
   `sensordeck upload` or `run` under another `XDG_CACHE_HOME`
3. Start `sensordeck run` with `A` again. It writes `theme A is on the screen`, uploads nothing, and sends the tags of
   `A` to the tiles of `B`

The third way of step 2 reproduces it with the fake screen of the end-to-end tests. The observation on the device is the
one above.

### Root cause

The decision whether to upload rests on a record of what was uploaded that is kept per user, per operating system and
per machine, while the screen is shared by all of them. Any upload that does not go through that one file makes the
record false, and `run` trusts it. The protocol gives no way to read back which theme the screen holds, so no record
kept on the computer can be made true for every way a theme reaches the screen. The cause was established from the
observation above and the code of unit 0013, and not by a run on the device made for it.

### The fix

The human chose that `run` uploads the theme of its config at every start, unconditionally, and keeps no state file. The
human accepts the cost: about 10 seconds and one reboot of the screen at each start of `run`, and so at each start of
the service. Within one `run` the theme is uploaded again only when a reload changes it, which `run` decides from what
it uploaded itself, in memory. `sensordeck upload` no longer touches the state file and is otherwise unchanged.

This unit supersedes, in unit 0013: R5, R6, R7, R12, R13, R14, R15, R35, R36, R37 and R38; the parts of R8, R10, R15,
R16, R20 and R21 that name the state file, the hash, the serial, the check or the line `theme <value> is on the screen`;
the parts of R30, R32 and R33 that describe or test those; D3, D5, D6, D7 as far as it concerns the state file, D9 and
D20; and the parts of D16 and D17 on the state file. In unit `0016-windows-hid-transport` it supersedes R13, the serial
of the Windows transport, which served only the state file. What unit 0013 says about the key `theme`, the reload,
`theme use`, the lines `uploading theme <value>` and `theme uploaded: <n> bytes`, the retry of a failed upload (R9 and
D8), the upload that a stop does not interrupt (R11 and D10) and no value sent while an upload is due, stands.

## Requirements

### When `run` uploads

- **R1** With `theme`, `run` uploads the theme of the config after it opened the screen at its start, at every start,
  before it sends any clock or value. It reads no file to decide it, and the upload is made whatever theme the screen
  holds
- **R2** `run` writes `uploading theme <value>` before each upload and `theme uploaded: <n> bytes` after one that
  completed, as unit 0013 R10 says. It no longer writes `theme <value> is on the screen`
- **R3** A `run` keeps the compiled theme file of its last upload that completed in memory, as the last upload. An
  upload that begins clears it, and one that completes sets it to the file it uploaded. A reload from `theme` to `tags`
  clears it too
- **R4** After a reload is applied, by R18 of unit 0013, whose theme compiles to a file that differs byte for byte
  from the last upload, or when there is no last upload, `run` uploads the theme of the reloaded config before it sends
  any value. A reload whose theme compiles to the same bytes as the last upload uploads nothing, as for `theme use` with
  the theme already set and its folder unchanged
- **R5** A reload from `tags` to `theme` uploads the theme. A reload from `theme` to `tags` uploads nothing, sends the
  new mappings, and by R3 makes a later reload to any theme upload it
- **R6** When the screen comes back after it was lost, by a reopening after a failed write or a replug, or after a
  restart of a hung screen by unit `0021-restart-boot-hung-screen`, `run` uploads nothing for that return. It goes on
  as after a screen that came back, with the clock and then the values. An upload that was due before, because it failed
  or had not been attempted yet, stays due and is made by the rules of R9 of unit 0013 and of R21
- **R7** An upload at the start whose writes time out is the start of a hang as unit 0021 has it: the screen is
  restarted, and the upload is made again once the screen is open, at once by R21 after a restart that recovered it,
  and otherwise once the wait of R9 of unit 0013 has passed. No value and no clock is sent until it completed
- **R8** With `tags` and no `theme`, `run` uploads nothing, as unit 0013 R4 says

### The state file

- **R9** `run` and `upload` neither read, write, create nor delete `sensordeck/uploaded-themes.json` or any other file
  in the cache directory of the user for an upload. A state file that a release before this unit wrote is left as it
  is, and has no effect on what is uploaded
- **R10** `run` writes no file for the theme. It compiles the theme in memory, as unit 0013 R16 says
- **R11** `sensordeck upload <file>` writes the same reports and the same lines to standard output and standard error,
  and ends with the same exit statuses, as before this unit, except for what concerned the state file: it no longer
  looks up the cache directory, so a cache directory that cannot be found is no error of `upload`, and it writes no line
  about the state file. The same holds for `run`, whose start with `theme` no longer fails when the cache directory
  cannot be found
- **R12** The code that served only the state file is gone: the package of the state file, the state file of the
  command line, the hash of the compiled theme, and the serial of the transport that is open, which is the `Serial` of
  the transport of the command line, of its trace, of the Linux transport, of the Windows transport and of the Windows
  device, with the lookup of the sysfs entry of a node named with `--device` that gave it. The serial that discovery
  reads and `devices` shows, on Linux and on Windows, stays

### The documents

- **R13** The README no longer says that `run` uploads only when the theme changes or is not on the screen, that a
  later start uploads nothing, or that a state file is kept, except for the sentence of R16. This holds for the opening
  paragraph, the list of what sensordeck does, and the section on the config file of `run`
- **R14** Step 3 of the quick start says that `run` uploads the theme to the screen at each start, which takes about 10
  seconds and reboots the screen, and that it sends the values after that
- **R15** The sections of the service, on Linux and on Windows, say that each start of the service, at the login or
  logon, at boot with lingering on Linux, and at each restart after an error, uploads the theme of the config, so that
  the values appear about 10 seconds after the start and the screen reboots once. They keep saying that `theme use` and
  edits of the config file reach the running service within a round, and add that a new theme is uploaded then
- **R16** The section on themes says: `run` uploads the theme of the config at each start, which takes about 10 seconds
  and reboots the screen, with no value and no clock sent until it is done; within one `run` it uploads again only when
  a reload gives a compiled theme that differs from the one it uploaded, so that `theme use` again after an edit of the
  theme folder puts the edit on the screen; it uploads nothing when the screen is plugged in again or restarted after a
  hang, because the screen keeps its theme in its flash; a failed upload is tried again after 10 seconds and then after
  longer waits, up to 5 minutes; a theme put on the screen by other software, by `sensordeck upload`, by the other
  operating system of a dual-boot machine or by another computer is replaced at the next start of `run`; a screen that
  received another theme while it was unplugged from a running `run` keeps it until `run` is started again; and the
  file `uploaded-themes.json` that earlier releases kept in the cache directory, at the paths the README gave for Linux
  and Windows, is no longer used and may be deleted
- **R17** No other text of the README or of `docs/` mentions the state file, an upload on change, or a check whether
  the theme is on the screen

### Tests

- **R18** The unit tests cover at least: an upload at the start with no state file and with a state file that names
  the hash of the theme, the file left unchanged; a second start that uploads again; a reload to a theme of other bytes
  that uploads; a reload to the same bytes that does not; a reload from `tags` to `theme` that uploads; `theme` to
  `tags` and back to the same theme, which uploads; a failed upload of another theme followed by a reload back to the
  theme of the last upload, which uploads; a screen that came back after it was lost, and one restarted after a hang,
  with no upload; an upload that failed and stays due across the return of the screen; and `upload` and `run` with a
  cache directory that cannot be found
- **R19** The end-to-end tests run the built executable against the fake screen and check: `run` with
  `minimal-dark` uploading it, then sending the clock and the values; a second `run` that uploads it again; a `run`
  that uploads although a state file in `XDG_CACHE_HOME` names the hash of `minimal-dark` for the serial of the screen,
  and leaves that file unchanged; `theme use` of another theme while `run` runs, followed by its upload and its tags,
  as today; a screen moved to another hidraw number while `run` runs, after which `run` sends the clock and the values
  and uploads nothing; and no directory `sensordeck` created in `XDG_CACHE_HOME` by `run` or by `upload`
- **R20** The five commands of the suite in `AGENTS.md` succeed

### The upload after a restart that recovered the screen

These requirements were added by the amendment of the spec recorded in `record.md`, and are numbered after the last
one of the first version. A restart that recovered the screen is a restart of a hung screen, by unit 0021, in which the
screen left the bus and whose node `run` opened again: in the round of the restart when the screen was open again at its
end, or in the later round that opened it when it was not.

- **R21** When an upload is due as the screen is opened again after a restart that recovered it, `run` makes the upload
  in that round, without waiting for the time that R9 of unit 0013 set after the attempt that failed. A restart that
  did not make the screen leave the bus, a restart that was not possible, and a reopening of a screen that was lost
  without a hang keep that wait. The waits that follow a failure of the upload made at once are those of R9 of unit
  0013, doubling from where they were, so that a screen that hangs again at each upload is not written more often than
  that rule allows plus once per restart, which unit 0021 makes at most once a minute
- **R22** The unit tests cover R21: a timed-out upload followed by a restart in which the screen left the bus and was
  open again, after which the upload is made in the same round; the same with the screen opened in a later round; a
  restart that did not make the screen leave the bus, a restart that was not possible and a reopening after a failed
  write without a hang, after each of which the upload waits for the time of R9 of unit 0013; and the wait after a
  failure of the upload made at once

## Decisions

- **D1 Upload at every start.** The human chose it. It is the one rule that makes the screen show the theme of the
  config whatever happened while `run` did not run, because the protocol gives no way to read back what the screen
  holds, and nothing on one computer sees what another computer or operating system did. Rejected: keeping the state
  file and keying it otherwise, which any upload outside the one file still makes false; a state file shared between
  the operating systems of a dual-boot machine, which needs a common file system and still misses another computer and
  the vendor's software; and storing a mark of the theme on the device, which the protocol is not known to allow. The
  cost is about 10 seconds and one write of the flash at each start of `run`, which for the service is each login or
  boot and each restart after an error. The writes of the flash were the reason unit 0013 avoided it, and the human
  accepts them
- **D2 Within one `run`, what it uploaded is what the screen holds.** The screen keeps its theme in its flash: it shows
  it after a replug and after a hang, as fact 1 of unit 0021 records. While a `run` feeds the screen, the README already
  says that another command that writes to the screen is not supported, the other operating system of a dual-boot machine cannot run, and a second
  computer reaches the screen only through an unplug, which this rule does not cover, as D3 says. So within one `run`
  the last upload is a true record, and comparing a reloaded theme with it decides correctly whether to upload, with no
  file and no serial. Uploading again at each return of the screen was rejected: a replug or a restart of a hung screen
  would then cost 10 seconds and a reboot each time, and a hang at boot would be followed by an upload for nothing
- **D3 A screen given another theme while unplugged is not detected.** A screen unplugged from a running `run`, given
  another theme by another computer and plugged in again, keeps that theme, and `run` sends its tags to it, until `run`
  starts again. This is the one case of the bug that remains, it needs an unplug while `run` runs, and the README says
  that a start of `run` puts the theme back. Uploading at each return of the screen to cover it is item 2 of "Beyond the
  request"
- **D4 The comparison is of the bytes of the compiled file.** `run` holds the compiled file of the theme of the config
  already, so comparing it with the file of the last upload needs no hash. The same folder compiles to the same bytes, as
  unit 0013 D3 says, so a rewrite of the config with an unchanged theme uploads nothing, and an edit of the folder, or a
  new release that compiles it differently, does
- **D5 The last upload is forgotten when an upload begins, and at a reload to `tags`.** An upload cut short may leave
  the flash with neither theme, for the reason of D7 of unit 0013, so after it began, only its completion makes the
  screen's theme known again. Without this, a failed upload of theme `B` followed by a reload back to the theme `A`
  uploaded before would send values to a broken screen. A reload to `tags` says that the user put another theme on the
  screen by other means, a vendor theme with `upload`, so the theme of the last upload can no longer be assumed there
- **D6 An existing state file is left alone.** Deleting it at each start would keep in the code the path and the
  handling of the file whose removal this unit is for, and would write in the cache directory at every start, for a
  file of a few bytes in a directory that the operating system may clear. Nothing reads it any more, and the README says
  it may be deleted. Deleting it once is item 3 of "Beyond the request"
- **D7 The serial of the open transport goes with the state file.** It was added by unit 0013 R6, and on Windows by
  unit 0016 R13, only to key the state file. Kept, it would be code that nothing calls, and on Linux a lookup of sysfs at
  each opening of a node named with `--device`, which the standards rule out. The serial in the listing of `devices`
  was there before and stays
- **D8 A cold boot with a hung screen takes longer.** On the human's Linux machine the screen is hung after every cold
  boot, by unit 0021. With an upload at each start, the first upload of the service then times out, the screen is
  restarted, and by R21 the upload is made again as soon as the restart recovered the screen. Without R21 the rule of
  R9 of unit 0013 would make it wait until 10 seconds after the attempt that failed, which would add those seconds to
  every boot for nothing: the wait is there so that a screen that refuses the theme is not written at every round, and
  a restart that recovered the screen is new evidence that the next attempt can succeed. The restart is made at most
  once a minute by unit 0021, so the attempt at once adds at most one write of the flash per minute to a screen that
  hangs again. This decision, and R21 and R22, were taken by the amendment recorded in `record.md`, from item 1 of
  "Beyond the request"
- **D9 The device check is the human's.** That a cold boot shows the theme of the config, with the right values, about
  10 seconds after `run` started, is for the human to check on the device. The unit is complete without that check and
  does not wait for it, and a finding of it is a new unit
- **D10 Windows is built, not run.** The change on Windows is the removal of the serial of the device and of the state
  file, and the suite builds and vets for Windows. Running it on Windows is the human's, and the unit does not wait for
  it

## Non-goals

- Reading back which theme the screen holds, or storing a mark of it on the device
- Uploading when the screen comes back, or detecting a theme put on the screen by something else while `run` runs
- A flag or a key that makes `run` skip the upload at its start
- Deleting the state file that earlier releases wrote
- Any change to `upload` other than the removal of the state file, or to `smartmonitor.Upload`, its pauses or its
  attempts
- Changing the waits of a failed upload other than by R21, or the restart of a hung screen
- Following the screen by its serial when several screens are connected

## Beyond the request

1. Taken into the unit by the amendment recorded in `record.md`, as R21, R22 and D8. After a restart that recovered a
   hung screen, making the upload that is due at once, rather than at least 10 seconds
   after the attempt that failed. Cost: a rule in the retry of the upload that tells a failure by a hang from another
   one, and tests. Without it, after a cold boot with the boot hang of unit 0021, the values appear about 10 seconds
   later than they could
2. Uploading the theme of the config each time the screen comes back after it was lost, so that a screen given another
   theme while unplugged gets its own back. Cost: about 10 seconds and a reboot of the screen at each replug and each
   restart of a hung screen, and tests. Without it, such a screen keeps the other theme until `run` starts again, which
   the README says
3. Deleting the state file of earlier releases at the start of `run`, or once. Cost: the path and the removal kept in
   the code, a line on a failure, and tests. Without it, a file of a few bytes stays in the cache directory until the
   user or the operating system deletes it, which the README says may be done
