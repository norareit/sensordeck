# Linux transport and a first command line

## Context

Unit `0002-device-protocol` built the package `internal/smartmonitor`: the encoders of the reports and a theme upload
that works on a `Transport` of three operations. Nothing can reach a device yet, and there is no executable. This unit
adds the Linux transport over hidraw, the executable `sensordeck` with a first set of low-level commands, the udev
rule, and the end-to-end suite that the executable makes possible.

After this unit is merged, the human does the first check on the real device, from the host, with the commands of
this unit. That check is no part of the unit. The unit's job is to make the check possible and informative: the
assumptions D5 to D10 of unit 0002 are expected to be partly wrong, for instance the device may answer `C`, `0x43`,
where an ACK is expected, as a standard YMODEM receiver does. The commands therefore show exactly which reports went
to the device and came from it, so that a following unit can correct the protocol.

The unit is built in a sandbox that holds only what is committed. It has Go 1.27.4, no USB device, no hidraw node of
the screen, and reaches no network but the Go module proxy. Every requirement below can be checked by automated
tests without hardware, and everything needed to build the unit stands in this spec.

### What is known of hidraw on Linux

- Every HID raw device has a directory `/sys/class/hidraw/hidrawN`, with *N* a decimal number, and a device node
  `/dev/hidrawN`
- The file `/sys/class/hidraw/hidrawN/device/uevent` holds lines of the form `KEY=value`. For the screen it looks
  like this:

  ```
  DRIVER=hid-generic
  HID_ID=0003:00000483:00000065
  HID_NAME=llhmi.com llTechCo.,Ltd
  HID_PHYS=usb-0000:00:14.0-3/input0
  HID_UNIQ=
  MODALIAS=hid:b0003g0001v00000483p00000065
  ```

- `HID_ID` is the bus as 4 hexadecimal digits, the vendor and the product as 8 hexadecimal digits each, separated by
  colons. Bus `0003` is USB. The kernel writes the digits in upper case
- `HID_NAME` holds the manufacturer and product strings, and `HID_UNIQ` the serial, which may be empty
- A write to the node is one output report: the report ID byte, which is `0x00` for this device because it uses no
  numbered reports, followed by the 64 bytes of the report, 65 bytes in one write
- A read from the node returns one input report. Because the device uses no numbered reports, the read gives the 64
  bytes without a report ID byte. A read never gives more than one report
- The node supports `poll`, so a read can be given a time limit
- After the device reboots it enumerates again, and its hidraw number may be another one
- hidraw is provided by the kernel's `usbhid` driver. When that driver is not bound to the device, there is no
  hidraw node
- A new device node belongs to root. A udev rule that sets `TAG+="uaccess"` gives the user of the active session
  access to it. The tag is acted on by the system rule file `73-seat-late.rules`, so a rule file that sets it must
  sort before that name

## Requirements

"The package of the protocol" is `internal/smartmonitor`. "A report" is its `Report`, 64 bytes. "A supported screen"
is a hidraw device with the USB ID `0483:0065`. "The node" is the device node of a hidraw device. Hexadecimal bytes
in output are two upper-case digits each, as `docs/protocol.md` writes them. `\t` stands for one tab character.

### Module, layout and platforms

- **R1** `go build .` in the root of the repository builds the executable, and the package in the root is `main`, so
  that `go install github.com/norareit/sensordeck@latest` gives an executable named `sensordeck`.
  `CGO_ENABLED=0 go build ./...` succeeds, and no Go file of the module imports `C`
- **R2** `go.mod` keeps the module path and the directive `go 1.27`. Its direct requirements are
  `github.com/spf13/cobra` at version 1.10.2 or newer and, when the fake screen of R29 uses it,
  `golang.org/x/sys`, and no others. `go.sum` exists, and `go mod tidy -diff` reports no difference
- **R3** `GOOS=windows GOARCH=amd64 go build ./...` and `GOOS=windows GOARCH=amd64 go vet ./...` succeed and report
  nothing. In such a build, each of the commands `devices`, `clock`, `values`, `upload`, `listen` and `raw` fails as R16
  says with the message `windows is not supported yet`, where the name is the value of `runtime.GOOS`
- **R4** The package of the protocol holds nothing of Linux and nothing of hidraw: R5 of unit 0002 still holds for
  it, with its test. The code of the commands names the package of the Linux transport only in files that a build
  constraint limits to Linux, and those files do nothing but choose the transport: they list the supported screens
  and open one. Every Go file of the Linux transport, of the fake screen and of the end-to-end tests is limited to
  Linux by a build constraint

### Changes to the package of the protocol

- **R5** The package of the protocol exports the vendor ID `0x0483` and the product ID `0x0065` of the screen as
  constants, and the discovery of R8 uses them
- **R6** The package of the protocol exports one error value that stands for a read whose time limit passed. The
  comment of `Transport` says that `Read` returns an error for which `errors.Is` with that value is true when the
  time limit passed, and only then
- **R7** `Upload` tells its caller where it is, through a function the caller supplies. It calls the function once
  for each of the stages reset, reopening after the reset, ymodem command, header block, end of transmission,
  closing block and reopening after the upload, and once for each data block, each time before its first call on
  the transport for that stage or block. The function is given the stage, the sequence number of the data block
  counted from 1, which is 0 at every other stage, and the count of data blocks of the theme. It is not called again
  when an item is written again after a NAK or when a reopening is tried again. The calls on the transport, R20 to
  R27 of unit 0002, do not change

### Finding the screen

- **R8** The discovery takes the root of sysfs and the root of the device nodes as values, and returns the supported
  screens. It reads `<sysfs root>/class/hidraw`, and for every entry whose name is `hidraw` followed by decimal
  digits it reads `<entry>/device/uevent`. The entry is a supported screen when the file has a `HID_ID` line whose
  three fields, read as hexadecimal numbers without regard to case, are bus 3, the vendor ID and the product ID of
  R5. For each supported screen it gives the node `<dev root>/<name of the entry>`, the vendor ID, the product ID,
  the value of `HID_NAME` and the value of `HID_UNIQ`, each of the last two empty when the line is missing or empty
- **R9** The discovery returns the screens ordered by the number in the entry name, as a number, so that `hidraw2`
  comes before `hidraw10`. An entry whose `uevent` file is missing, cannot be read or has no well-formed `HID_ID`
  line is passed over without an error. A `class/hidraw` directory that does not exist gives no screens and no
  error. Any other error of reading that directory is returned
- **R10** The discovery opens no node, so it works for a user who has no access to the nodes

### The hidraw transport

- **R11** The Linux transport implements `Transport` of the package of the protocol, and has a close operation
  besides. It is opened either by discovery or on a given node:
    - by discovery, it opens the node of the one supported screen found. No screen found is an error with the text
      `no supported screen found`. More than one is an error whose text is
      `more than one supported screen is connected, name one with --device:` followed by one line for each screen in
      the format of R15
    - on a given node, it opens that path as it is and does not use the discovery
- **R12** The operations of the transport:
    - **Write** writes 65 bytes to the node in one write: `0x00`, then the 64 bytes of the report. A write that
      takes fewer than 65 bytes is an error
    - **Read** returns one input report, or an error once the time limit has passed, for which `errors.Is` with the
      error value of R6 is true. It asks the node for at most 64 bytes in one read. A read that gives 1 to 63 bytes
      gives a report of those bytes filled up with `0x00`. A read that gives no bytes, or an error other than the
      passed time limit, is an error for which `errors.Is` with the value of R6 is false
    - **Reopen** closes the node and opens the device again the way it was opened first: by discovery, done anew,
      so that a screen that came back under another hidraw number is found, or on the same given path. An error of
      closing the old node does not stop it. When the opening fails, the transport holds no open node, a write or a
      read then returns an error, and a later Reopen tries again
    - **Close** closes the node
- **R13** When the node cannot be given a time limit for reading, the opening fails with an error that says so,
  and no read ever waits without a limit
- **R14** When opening a node fails because permission is denied, the text of the error is, with the path of the
  node in place of `/dev/hidraw3`:

  ```
  cannot open /dev/hidraw3: permission denied. Install the udev rule and replug the screen, see "Linux: access to the screen" in https://github.com/norareit/sensordeck#linux-access-to-the-screen
  ```

  Any other failure to open gives `cannot open <path>: ` followed by the error of the system. The error of the
  system stays reachable with `errors.Is` in both cases

### The executable

- **R15** `sensordeck devices` writes one line to standard output for each supported screen, in the order of R9:
  `<node>\t<vendor>:<product>\t<name>\t<serial>`. The vendor and the product are 4 lower-case hexadecimal digits
  each, and an empty name or serial is written as `-`. For the screen above on `hidraw3` the line is
  `/dev/hidraw3\t0483:0065\tllhmi.com llTechCo.,Ltd\t-`. When no screen is found it writes nothing to standard
  output and the line `no supported screen found` to standard error. The exit status is 0 in both cases. The command
  takes no account of `--device` and opens no node
- **R16** Every failure of a command writes `sensordeck: ` followed by the text of the error and a newline to
  standard error, and the exit status is 1. That holds for an unknown command, an unknown option and wrong arguments
  too. The usage text is not written after a failure. `sensordeck` without a command, and `--help` on it or on any
  command, write the help to standard output with the exit status 0, and the help names the six commands and the
  two options of R17 with a short description each. There is no `completion` command
- **R17** Two options hold for every command and are accepted before and after the name of the command:
    - `--device <path>` opens the transport on that node, where it is otherwise opened by discovery
    - `--trace` writes the trace of R23 to standard error
- **R18** The commands `clock`, `values`, `upload`, `listen` and `raw` check their arguments, and `upload` reads its
  file, before the transport is opened. With wrong arguments or an unreadable file they fail as R16 says, also when no
  screen is connected. They close the transport before they end
- **R19** `sensordeck clock` takes no arguments. It writes the one report that `EncodeClock` gives for the current
  time in the local time zone, reads nothing, and writes to standard output `clock sent: ` followed by that time as
  `2026-11-14 15:04:05`
- **R20** `sensordeck values <tag>=<value>...` takes one or more arguments. A number is either decimal, one or more
  digits read in base 10 whatever zeros it starts with, or hexadecimal, `0x` or `0X` followed by one or more
  hexadecimal digits. Nothing else is a number: no sign, no spaces, no other prefix. The tag is 0 to 255 and the
  value 0 to 65535. The command writes the reports that `EncodeValues` gives for the pairs in the order of the
  arguments, reads nothing, and writes to standard output
  `values sent: <count of pairs>, reports written: <count of reports>`. So `sensordeck values 1=0x1234 255=1`
  writes 65 bytes to the node, `00 02 02 01 12 34 FF 00 01` followed by 56 bytes of `0x00`, and prints
  `values sent: 2, reports written: 1`. The errors, each naming the whole argument in double quotes:
    - no arguments: `values needs at least one <tag>=<value>`
    - no `=` or more than one, or a side that is not a number: `invalid argument "<argument>": want <tag>=<value>`
    - `invalid argument "<argument>": the tag is outside 0 to 255`
    - `invalid argument "<argument>": the value is outside 0 to 65535`
- **R21** `sensordeck upload <file>` takes exactly one argument, reads the whole file and sends its bytes with
  `Upload`, with pauses that really wait. For each call of the function of R7 it writes one line to standard output:
  the name of the stage as the `String` of the stage gives it, and for a data block
  `data block <number> of <count>`. After a complete upload it writes
  `upload complete: <size> bytes in <count> data blocks`. For a theme of 2500 bytes the standard output of a complete
  upload is exactly:

  ```
  reset
  reopening after the reset
  ymodem command
  header block
  data block 1 of 3
  data block 2 of 3
  data block 3 of 3
  end of transmission
  closing block
  reopening after the upload
  upload complete: 2500 bytes in 3 data blocks
  ```

  When the upload fails, the lines up to the stage that failed are written, and the failure of R16 carries the text
  of the error of `Upload`, which names the stage, the block and the cause, as in
  `sensordeck: upload failed at the ymodem command: unexpected answer 0x43`. A file that cannot be read is the error
  `cannot read the theme file <path>: ` followed by the error of the system
- **R22** `sensordeck listen [--seconds N]` reads input reports for N seconds, 5 without the option. N is a decimal
  whole number of 1 or more, and anything else is the error
  `invalid --seconds "<text>": want a whole number of 1 or more`. It writes nothing to the device. Every report
  read is written to standard output as one line in the format of a read line of R23. A read whose time limit
  passed ends the listening and is no failure, and no read waits beyond the end of the N seconds. Any other error
  of a read is a failure as R16 says. At the end it writes `reports received: <count>`
- **R23** With `--trace`, every call on the transport writes one line to standard error when it returns. A line is
  the time since the command started, in seconds with exactly six decimals, padded on the left with spaces to 12
  characters, then one space and:
    - `> ` and the 64 bytes of the report, for a write that succeeded
    - `> `, the 64 bytes, ` ! ` and the text of the error, for a write that failed
    - `< ` and the 64 bytes of the report, for a read that succeeded
    - `< ! ` and the text of the error, for a read that failed, which includes the passed time limit
    - `* reopen`, for a reopening that succeeded
    - `* reopen ! ` and the text of the error, for a reopening that failed

  The 64 bytes are written as 64 groups of two upper-case hexadecimal digits separated by single spaces, all of
  them, with nothing left out. These two lines are exact examples, the reset report written and an ACK read:

  ```
      0.000412 > 01 72 65 73 65 74 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00
      1.503377 < 06 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00
  ```

  The trace shows the report and not the report ID byte. Without `--trace` none of these lines is written. The
  tracing works on `Transport` alone and knows nothing of hidraw, and the source of its time can be replaced in a
  unit test, which checks the lines above to the character
- **R24** The executable reads two environment variables, which exist for the tests and appear in no help text:
  `SENSORDECK_SYSFS_ROOT` replaces the root of sysfs, `/sys`, and `SENSORDECK_DEV_ROOT` the root of the device
  nodes, `/dev`. An empty or unset variable means the default. They are read in one place where the program starts,
  and everything past it takes the roots as values
- **R35** `sensordeck raw <hex bytes>... [--seconds N]` writes one report made of the bytes given and prints what
  the device sends afterwards. It takes one or more arguments. Each argument is an even number of hexadecimal
  digits, two or more, in upper or lower case, with no prefix and no separator, and stands for one byte for each
  pair of digits. The bytes of the arguments are joined in the order given, so `raw 03 01 15`, `raw 030115` and
  `raw 0301 15` give the same three bytes. There are 1 to 64 bytes in all, and the report is those bytes filled up
  with `0x00` to 64. The command writes that one report, then reads input reports for N seconds, 1 without the
  option, with the rule and the error text of R22 for N. Every report read is written to standard output as one line
  in the format of a read line of R23, with the time counted from the start of the command. A read whose time limit
  passed ends the reading and is no failure, no read waits beyond the end of the N seconds, and any other error of
  a read or of the write is a failure as R16 says. At the end it writes `reports received: <count>`. So
  `sensordeck raw 03 01 15` writes 65 bytes to the node, `00 03 01 15` followed by 61 bytes of `0x00`. The errors:
    - no arguments: `raw needs at least one byte`
    - an argument that is empty, has an odd number of characters or holds a character that is no hexadecimal digit:
      `invalid argument "<argument>": want pairs of hexadecimal digits`
    - more than 64 bytes: `raw takes at most 64 bytes, got <count of bytes>`

### The udev rule and the documents

- **R25** The repository holds the file `udev/70-sensordeck.rules`. Apart from comment lines and empty lines it
  holds exactly this one line:

  ```
  SUBSYSTEM=="hidraw", ATTRS{idVendor}=="0483", ATTRS{idProduct}=="0065", MODE="0660", TAG+="uaccess"
  ```

- **R26** `README.md` has a section with the heading `## Linux: access to the screen`. It says that the screen is
  reached through hidraw and that a udev rule gives the user access. It shows the rule line of R25 to the character,
  names the file of R25, and gives the installation as copying that file to `/etc/udev/rules.d/` and replugging the
  screen. It says that the kernel's `usbhid` driver must stay bound to the screen, because hidraw depends on it, and
  that a rule which unbinds `usbhid` for this device, as some guides for other tools add, has to be removed. No
  document of the repository tells the reader to run `udevadm trigger`
- **R27** `docs/protocol.md` says in its section on the transport, as established by this unit: that an input report
  is read from the node as 64 bytes without a report ID byte, because the device uses no numbered reports; that the
  screen is found through the `HID_ID` line of the `uevent` file in sysfs; and that hidraw needs the `usbhid` driver
  bound. Its section on the assumptions no longer says that this unit is where they are checked. It says that they
  are checked on the device by the human with the commands of this unit and stay unconfirmed until then, and it
  gains one assumption: an input report shorter than 64 bytes is filled up with `0x00`. Nothing else in the document
  changes
- **R28** The section "Testing" of `AGENTS.md` names the suite as five commands that must all succeed: the three it
  names now and the two of R3. In place of its statement that there is no end-to-end suite, it says where the
  end-to-end tests are, that `go test ./...` runs them, that they build the executable and run it against a fake
  screen made of a directory tree and pseudo terminals, that they need Linux with `/dev/ptmx`, and that the two
  variables of R24 exist for them

### Tests

- **R29** The fake screen is test support that the tests of the transport and the end-to-end tests share. In a
  temporary directory it makes a sysfs root and a dev root as R8 reads them. The node of a fake screen is a symbolic
  link to the slave of a pseudo terminal that is set to pass every byte unchanged, in both directions, without echo.
  A test scripts what the fake answers and when, and reads back every byte written to the node. The fake can take a
  screen away and bring it back under another hidraw number. When a pseudo terminal cannot be made, the tests fail
  and are not skipped
- **R30** Unit tests of the discovery, against directory trees made in a temporary directory, cover: no
  `class/hidraw` directory; no entries; entries of other devices only; one screen; two screens, with `hidraw10` and
  `hidraw2` to show the order; `HID_ID` in lower case; a missing `HID_NAME` and an empty `HID_UNIQ`; an entry
  without a `uevent` file; a malformed `HID_ID`; a device with the right vendor and product on another bus
- **R31** Unit tests of the transport, against the fake screen, cover: a write arriving as 65 bytes that start with
  `0x00`; a read of a report; two reports sent one after the other, read as two; a read of 1 byte, filled up; a read
  whose time limit passes, with a limit of 100 milliseconds or less, giving the error of R6; a read after the fake
  closed its end, giving an error that is not that of R6; a reopening that finds the screen under another number;
  a reopening with no screen, which fails, followed by one that succeeds; the two errors of opening by discovery of
  R11; and opening on a given node. The text of R14 is tested on the function that makes it, with a
  permission error given to it, because the tests may run as root, for whom no permission is denied
- **R32** Unit tests cover the argument rules of R20, R22 and R35 with every error named there, with `010=1` as tag
  10 and `0x1F=0XFFFF` as tag 31 and value 65535, with the three spellings of the example of R35 giving the same
  report, with `0x03` and `3` as wrong arguments of `raw`, with 64 bytes accepted and 65 refused, the line of R15,
  and the calls of R7 in a complete upload and in one with a NAK
- **R33** The end-to-end tests build the executable once, with `CGO_ENABLED=0`, and run it as a process against the
  fake screen through the variables of R24. They check standard output, standard error and the exit status, and
  cover at the least:
    - `devices` with no screen, with one and with two, with the exact output of R15
    - `values` with the example of R20, checking the 65 bytes that arrive, and with 21 pairs, checking two reports
    - a wrong argument of `values` with no screen connected: the message and the exit status 1
    - `clock`: the bytes at offsets 0, 1, 2 and 10 to 63 of the report, and that the bytes at offsets 3 to 9 give a
      time between the start and the end of the run, with the weekday that belongs to it
    - `upload` of a theme of 2500 bytes in which byte *i* is *i* modulo 256, against a fake that answers ACK and
      comes back under another hidraw number after the reset report: exit status 0, the standard output of R21, the
      bytes that arrived being those of the reports unit 0002 specifies in the order of its R20, and with `--trace`
      60 lines with `>`, 6 lines with `<` and 2 lines with `* reopen` on standard error, each matching R23
    - `upload` against a fake that answers `0x43` to the ymodem command: exit status 1, standard output ending with
      `ymodem command`, the last line of standard error being the example of R21, the trace holding a read line
      that starts with `43`, and nothing written to the node after the ymodem report
    - `listen --seconds 1` against a fake that sends two reports: both lines, `reports received: 2`, exit status 0
    - `raw 03 01 15` against a fake that answers the report with one report that starts with `0x43`: the 65 bytes
      that arrive are those of R35, standard output is one read line with that report and then
      `reports received: 1`, and the exit status is 0
    - two screens without `--device`: the error of R11 and exit status 1, and with `--device` a `values` that
      arrives at the screen named
- **R34** What the end-to-end tests do not cover, and where it is covered instead: a reopening that fails all 20
  times, which would take over 10 seconds, is covered by the tests of unit 0002 on the upload; a passed time limit
  of an answer, 10 seconds, is covered by R31 with a short limit and by the tests of unit 0002; the denied
  permission is covered by R31; the build for Windows is covered by the two commands of R3 only, and its message is
  not run

## Decisions

Those the human made before this spec:

- **D1** The transport is pure Go without cgo: hidraw is a file, and the screen is found through sysfs. Windows
  comes later as a second transport behind the same `Transport`, so nothing of Linux may enter the package of the
  protocol or the code of the commands beyond the choice of the transport
- **D2** A general driver interface over several kinds of device is not built yet. There is one device and one
  implementation
- **D3** The commands of this unit are low-level on purpose. They are the instruments of the first check on the
  device and not the final command set, which will have nested commands such as `theme list`, `theme use` and
  `service install`
- **D4** The udev rule is exactly the line of R25. The reader is told to replug the screen and is not told to run
  `udevadm trigger`

Those this spec makes:

- **D5** The executable is the package in the root of the module, and holds nothing but the call into the package
  of the commands. The README promises a single executable, and installing it with `go install` on the module path
  is meant to work one day and to give the name `sensordeck`. A `cmd/sensordeck` directory would make that path
  longer and is the layout for a module with several executables, which this one is not
- **D6** The packages are `internal/cli` for the commands, `internal/hidraw` for the discovery and the transport on
  Linux, a package of test support for the fake screen, and `e2e` for the end-to-end tests. The exact names of the
  last two are for the plan. `internal/hidraw` is the Linux transport of this screen and not a general hidraw
  library: it uses `Report` and the USB ID of the package of the protocol directly, because a general one would be
  an abstraction with one user
- **D7** The command line is built with `spf13/cobra` and not with the standard `flag` package. The final command
  set has nested commands and options that hold for every command, which `flag` leaves to hand-written dispatch,
  and cobra gives them with the help texts. Its cost is a dependency and a `go.sum`, which the human allowed. Its
  `completion` command is switched off, because nothing asks for it yet. Cobra's own printing of errors and of the
  usage after an error is switched off in favor of R16, so that the output of a failure is one known format
- **D8** There are two exit statuses, 0 and 1. A separate status for a wrong command line would need the errors of
  cobra told apart by kind, and nothing reads the difference yet
- **D9** `devices` exits 0 when no screen is found, because the command answers a question and "none" is an answer
  and no failure of the program. The line that says so goes to standard error, so that standard output holds device
  lines only and a script can count them. The fields are separated by tabs because the name holds spaces
- **D10** The time limit of a read is the read deadline of the opened file. Go puts a character device that
  supports `poll` under its network poller, so `SetReadDeadline` works on a hidraw node as on a pipe, without a
  second thread and without a system call of our own. This was tried for this spec on a pseudo terminal, with
  Go 1.22: the deadline held, and a read gave the 64 bytes of one write. It is not yet tried on a hidraw node, which
  the kernel documents as supporting `poll`. R13 turns a node on which it does not work into an error at opening
  rather than a read that hangs
- **D11** A short input report is filled up with `0x00` and is no error. The firmware may well send its one-byte
  answer as a short report, and an error would stop the first check on the device at the first answer, while
  filling up lets the upload go on and show the rest. The length is lost in the trace: item 2 of "Beyond the
  request" keeps it. A read of no bytes is an error, because it means the node has gone
- **D12** The error value for a passed time limit lives in the package of the protocol, R6, and not in the
  transport. `listen` has to tell the passed limit from a failure, the Windows transport will have to give the same
  signal, and the package of the protocol may not import `os`, where Go's own deadline error lives
- **D13** `Upload` reports its stages through a function, R7, which unit 0002 left out as item 4 of its "Beyond the
  request" for want of a caller. The caller is here now. Deriving the stage in the command by counting writes would
  repeat what the upload knows
- **D14** A reopening by discovery follows the rule of the first opening: exactly one supported screen. With several
  screens the user has named a node, and that path is opened again as it is. If the screen comes back under another
  number in that case, the reopening fails 20 times and the upload ends with the error of that stage. Item 3 of
  "Beyond the request" would follow the screen by its serial
- **D15** The upload reads no input reports away before it starts. If the screen sends reports unasked, the first
  one is taken as an answer and the upload fails with the byte, as D12 of unit 0002 wants it. `listen` is the
  command that shows what the screen sends unasked, and it is to be run first in the check on the device
- **D16** The tracing wraps `Transport` and is not built into the Linux transport. It then holds for the Windows
  transport without a change and is tested with a fake transport and a fixed clock. The price is that it sees
  reports and not the bytes of the node, so the report ID byte and the true length of a short read are not in it.
  The trace writes all 64 bytes of every report and shortens nothing, because the check on the device must not
  depend on what a shortening hides, and a line for a failed call, because a passed time limit is as much a finding
  as an answer
- **D17** The test roots are environment variables and not hidden options. They take effect for every command
  without a place in the command line or the help, and the tests set them once for a process. They open nothing
  that the user could not open anyway, since the executable runs with the rights of the user
- **D18** The fake node is the slave of a pseudo terminal behind a symbolic link. It is the one thing a test can
  make without privileges that is opened by its path, carries bytes in both directions and supports `poll`, so the
  transport runs its real code path: open, write, read with a deadline, close. A named pipe carries one direction
  only, and opened for both by one process it gives the process its own writes back. A socket is not opened with
  `open`, so the transport would need a second way in that exists for the tests alone. A pseudo terminal does not
  keep the bounds of a report as hidraw does, but a read of at most 64 bytes of reports that are all 64 bytes long
  gives one report, which the trial of D10 showed. A fake therefore sends a short report only when nothing follows
  it before it is read. Setting the terminal to pass bytes unchanged needs terminal calls, for which
  `golang.org/x/sys` may be used: it is pure Go, and the Windows transport will need the module anyway. The sandbox
  the unit is built in was checked: it has `/dev/ptmx` and can open a pseudo terminal
- **D19** The end-to-end tests are Go tests in the module, so that `go test ./...` stays the one command that runs
  everything, and the rule of the standards to run the end-to-end tests after every change is met by it. The two
  upload tests wait in real time, about 2 seconds for the complete upload and about 1 for the failing one, because
  the pauses are constants of the package of the protocol. That is accepted, and no option to shorten them is added.
  R6 of unit 0002, which forbids waiting, holds for the package of the protocol and not for these tests or for the
  short time limits of R31
- **D20** The build and the vet for Windows are part of the suite. They are the only check that nothing of Linux
  entered the code of the commands, they need no network beyond the module proxy and take seconds. Without them the
  constraint D1 would be found broken only when the Windows transport is written
- **D21** The rule file is named `70-sensordeck.rules`, because the `uaccess` tag works only when set before
  `73-seat-late.rules`. The README shows the rule line itself as well as the file, for the reader who installed the
  executable without the repository. It names no command to reload the rules: udev reads changed rule files by
  itself when the next device event comes, which the replugging is
- **D22** The text of R14 points to the README by its address, because the user who meets it may have nothing but
  the executable
- **D23** The commands check their arguments and read the theme file before they open the device, so that a wrong
  command line is reported the same with and without a screen and nothing is sent before everything is known to be
  usable
- **D24** A number of `values` is decimal or `0x` hexadecimal only. Go's own parsing of a number with a prefix
  would read `010` as 8, which nobody typing a tag means
- **D25** The command `raw` is part of the unit. It was item 3 of "Beyond the request", and the orchestrating
  session took it under the delegation in `AGENTS.md`. The device has commands that are not known, such as the
  brightness and reading back device information, and the check on the device after this unit is to be able to try
  byte sequences for them, or a known command out of its usual order, without another unit in between. Its bytes are
  pairs of hexadecimal digits without a prefix, because that is how `docs/protocol.md` and the trace write a report,
  so that a line of either can be copied into the command. A single digit is refused rather than read as one byte,
  because in a joined argument it cannot be told which pair lost its digit. It sends one report only: a sequence of
  reports with answers in between is what `upload` is, and a second report is a second call

## Non-goals

- Reading sensors
- The `run` command and the daemon
- Themes: the theme format, compiling a theme, the `theme` commands. `upload` sends a file as it is and does not
  look into it
- The Windows HID transport. On Windows the commands only say that it is not supported yet
- Installing a service
- The brightness and every other command that is not known
- Confirming or correcting the assumptions of the protocol. That is the unit after the check on the device, and
  this unit changes nothing of D5 to D12 of unit 0002
- The check on the real device itself, which the human does after the merge
- Installing the udev rule from the executable
- A driver interface over several kinds of device

## Beyond the request

1. Refuse an empty theme file, and one larger than the 4 MiB of the flash, before the device is opened. It costs a
   few lines and two tests. Without it an empty file is sent as a header block with the size 0 and no data blocks,
   and a file that is too large is sent in full, and what the device does in either case is not known. The check on
   the device is done with a real theme, so neither happens there
2. Show the true length of a short input report: the transport tells the command of it, and the command writes a
   line for it to standard error. It costs a function passed to the transport at opening, a line format and tests.
   Without it a short report looks in the trace like a full one that ends in zeros, and the upload works the same
   either way
3. Reopen by the serial of the screen that was open, when several screens are connected and the serial is not
   empty. It costs a rule for the choice and tests. Without it an upload with several screens connected needs
   `--device` and fails at the reopening if the number changes, with a clear error
4. Have `devices` say for each screen whether the user may open its node. It costs a check of the access and a
   fifth field. Without it a missing udev rule shows at the first command that opens the node, with the text of R14
5. Trace lines that name the node, for the first opening and after each reopening. It costs a way for the tracing to
   learn the path, which `Transport` does not give. Without it the node in use is found with `devices`
6. A document with the steps of the first check on the device: which commands in which order, and what to look for
   in the trace. It costs a page. Without it the human works from the help of the commands and this spec
7. A `--version` option. It costs the wiring of a version into the build. Without it a build is told apart by its
   commit
8. A status of its own, 2, for a wrong command line. It costs telling the errors of cobra apart by kind. Without it
   every failure is 1 and the text says which it was
9. Cobra's `completion` command for shell completion. It costs removing the switch of D7 and a line in the README.
   Without it there is no completion of command names in the shell
