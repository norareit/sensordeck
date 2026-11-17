# One sensordeck process at a time on the controller and on the screen

## Context

`sensordeck run`, usually as the systemd user service of unit `0014-systemd-user-service` or the startup task of unit
`0019-windows-startup-task`, holds the Corsair iCUE Commander Core XT in a session of software mode when the config has
`commander_color` (units `0009-commander-led-colour` and `0025-commander-led-port-configuration`), and writes to the
screen for as long as it runs. Every other command that opens a device opens it as well, with nothing to tell it that
another sensordeck process has it: `sensors` reads the controller (unit `0005-commander-core-xt-probes`), and `clock`,
`values`, `upload`, `listen`, `raw` and a second `run` open the screen.

On Linux an opening of a hidraw node is never exclusive, and on Windows the screen and the controller are opened with a
share mode of read and write (D12 of unit `0016-windows-hid-transport`). The README says that another program talking to
the controller at the same time is not supported, and names `sensordeck sensors` among them, and that running `run` or
another command that writes to the screen by hand while the service runs is not supported. Nothing enforces either.

"R*n* of unit 00*nn*" below names a requirement of the spec of that unit in `changes/`.

### What was observed

The orchestrating session reported these from the human's Linux machine, on 2026-11-16 and 2026-11-17, while the
service ran with `commander_color`:

- A second sensordeck process that read the controller, `sensordeck sensors` or a second `run`, broke into the session
  of the service. The commands of the two processes interleaved on the controller
- The service wrote `controller lost: close of endpoint 0x1a on channel 1: echo 01`, or
  `controller lost: read of endpoint 0x17 on channel 1: status 03`
- The teardown of the second process, which returns the controller to hardware mode (step 8 of the sequence of unit
  0005), ended the software mode of the service. Its LEDs went dark or changed until it set the session up again about
  10 seconds later, at its first retry, and wrote `controller back`
- liquidctl, run at the same time, failed with `response does not match command`

Two processes writing to the screen at once were not observed. The same cause applies to them: a second `run`, or
`clock`, `values` or `upload` while the service runs, writes its reports between those of the service, and an upload of
a theme, a sequence of reports with answers, can be broken by them.

### Steps to reproduce

1. On Linux, run `sensordeck run` with a config that has `commander_color` and maps a probe of the controller, as the
   service or by hand, with the controller and the screen plugged in
2. Wait for the LEDs to show the colour
3. Run `sensordeck sensors` in another terminal
4. The LEDs go dark or show the stored lighting, the log of `run` has `controller lost: ` and about 10 seconds later
   `controller back`, and the colour comes back

This is the sequence the human saw. The specifier did not reproduce it, as the sandbox has no device. Each step of it
follows from the code as described below.

### Root cause

Every reading of the controller opens its hidraw node anew and runs the whole sequence, from the entry into software
mode to the return to hardware mode (`commandercore.Source`), and a session of `run` keeps the node open and sends a
reading once a second (`commandercore.Session`, `cli.sessionKeeper`). The protocol of the controller is a sequence of
commands, each answered in turn, with an endpoint opened on a channel before it is read and closed after. Two processes
with the node open each write their commands and each read whatever answer comes next, so each gets answers to the
other's commands: the echo and status errors above, and liquidctl's `response does not match command`. The second
process ends with the return to hardware mode, which ends the software mode the session of `run` depends on, so the
next keepalive of `run` fails, the keeper writes `controller lost` and sets the session up again after
`firstRetry`, 10 seconds.

No sensordeck process can tell that another one has the device: neither system makes the opening exclusive, and
sensordeck takes no lock of its own.

### The fix

A sensordeck process that has the controller or the screen open holds a lock on it, which another sensordeck process
tries to take before it writes anything, and does not wait for. The lock is advisory: it binds sensordeck processes and
no other program.

- `sensors` that finds the controller held leaves out the probes, writes one line that says so, and lists the other
  readings as it does now
- `run` and the one-shot screen commands that find the screen held refuse, with a line that names the other process
  when it can be found and says how to stop the service
- `run` holds the screen from its opening at its start to its end, and the controller for each reading and for the
  whole of each session
- A lock ends with the process that holds it, however the process ends

## Requirements

"The controller" is the node of interface 0 of the Commander Core XT that the finder of the system gives, and "the
screen" is the screen a command opens, discovered or named with `--device`. A process "holds" a lock while the
requirements below say it does. "Held by another process" is said of a lock that another sensordeck process holds.

### The locks

- **R1** On Linux, a sensordeck process holds an exclusive lock on a hidraw node it opened, the screen's or the
  controller's, from the opening to the close of that node. It is the lock of `flock(2)` with `LOCK_EX`, on the opened
  node, so that two processes that open the same device, by whatever path, take the same lock
- **R2** On Windows, a sensordeck process holds the lock of the screen from before it opens a screen to after it closes
  it, and the lock of the controller from before it opens the controller to after it closes it. The lock of the screen
  is an exclusive lock taken with `LockFileEx` on the file `screen.lock`, and that of the controller on the file
  `controller.lock`, both in the directory `sensordeck` of the cache directory of the user, as `os.UserCacheDir` gives
  it (`%LocalAppData%\sensordeck\`), which is created when it does not exist. One lock serves every screen and one
  every controller
- **R3** A sensordeck process tries a lock once and does not wait: a lock held by another process is a refusal at once.
  A process that is refused writes nothing to the device. On Linux it closes the node it opened for the try
- **R4** A lock ends when the process that holds it ends, whatever the way: a normal end, Ctrl+C, SIGTERM, SIGKILL or a
  crash. No file left behind by an ended process makes a lock held. A process that a sensordeck process starts, as a
  command of a custom metric, does not inherit a lock
- **R5** A failure to take a lock other than its being held by another process, such as a lock file that cannot be
  created, is an error that names the node or the file and the failure, and the device is not used. The command fails
  with it as with a failure to open the device
- **R6** The holder of a lock is named, when it can be found, as ` (process <pid>: <command line>)`, with the process ID
  in decimal and the command line of that process, its arguments joined by one space. Where the command line cannot be
  read it is ` (process <pid>)`, and where the holder cannot be found it is nothing. Below, `<holder>` stands for this
  text
    - On Linux the holder is the process that `<proc root>/locks` gives for a `FLOCK` lock whose device and inode are
      those of the node, and its command line is read from `<proc root>/<pid>/cmdline`, where the proc root is the one
      the executable reads, `/proc` or `SENSORDECK_PROC_ROOT`
    - On Windows the holder writes its process ID and its command line into its lock file once it holds the lock,
      replacing what the file held, and the refused process reads them from the file. The content stays readable by
      another process while the lock is held. A file whose content is not a process ID and a command line gives no
      holder

### The controller

- **R7** A reading of the controller, as `sensors` and `run` without `commander_color` make it, holds the lock of the
  controller from before its first command to after its return to hardware mode
- **R8** A session of the controller, as `run` with `commander_color` holds it, holds the lock of the controller from
  before the first command of its setup to after its teardown: through every keepalive and every reading of the
  session, and until the teardown at a stop, at a loss and at a setup that failed
- **R9** `sensors`, when the lock of the controller is held by another process, writes this line to standard error once
  and lists every other reading as it does now, custom metrics included, with no `commander.<n>.temp`:

  `the controller is in use by another sensordeck process<holder>, its probes are left out`

  It writes nothing to the controller, so the session of the other process goes on with its colour. When no other
  reading has a value it also writes `no sensor could be read`, as now. Its exit status is that of today for the
  readings it has
- **R10** A setup of a session of `run` that finds the lock of the controller held by another process is a setup that
  failed: the keeper writes `controller setup failed: the controller is in use by another sensordeck process<holder>`
  with the rule of today that writes the line once until a setup succeeds, and tries again after the wait of today,
  10 seconds and doubling up to 5 minutes. It writes nothing to the controller
- **R11** A reading of `run` without `commander_color` that finds the lock of the controller held by another process
  gives no probes and writes no line, as a failed reading does now. It counts as a reading, so the next one comes a
  period of `commander_interval` later, and the screen keeps the last values of the probes in between

### The screen

- **R12** `clock`, `values`, `upload`, `listen` and `raw`, when the lock of the screen is held by another process, end
  with status 1, write nothing to the screen and nothing to standard output, and write this line to standard error, on
  Linux:

  `sensordeck: the screen is in use by another sensordeck process<holder>, stop it first, or stop the service with systemctl --user stop sensordeck`

  and on Windows:

  `sensordeck: the screen is in use by another sensordeck process<holder>, stop it first, or stop the startup task with sensordeck service uninstall`

  There is no option that overrides the refusal
- **R13** `run`, when the lock of the screen is held by another process at its start, ends with status 1 before it
  writes anything to the screen or to the controller and before it writes `running:`, with the line of R12. As the
  service or the startup task it is started again after 10 seconds as after any failed start, and writes the line at each
  start that is refused
- **R14** `run` holds the lock of the screen from its first opening of the screen to its end. On Linux, while the screen
  is lost, its node is closed and the lock is not held; a reopening that finds the lock held by another process fails as
  a reopening fails now, with no line, and the next round tries again. On Windows `run` holds the lock for the whole of
  its run, while the screen is lost too
- **R15** `devices`, `sensors`, the `theme` commands and the `service` commands take no lock of the screen, and work
  while another process holds it as they do now

### The documents

- **R16** The README says, in the section of the Commander Core XT, in place of its paragraph on other programs that
  talk to the controller:
    - that one sensordeck process at a time uses the controller, that `sensordeck sensors` leaves out the probes, with
      its line, while `run` holds a session or reads, and that a `run` whose setup finds the controller in use tries
      again as after a failed setup
    - that other programs, such as iCUE, liquidctl and OpenLinkHub, are not kept out: talking to the controller from one
      of them at the same time as sensordeck is not supported, the commands of both mix, readings fail and the LEDs may
      show the stored lighting or go dark
- **R17** The README says, in place of the paragraphs on running `run` or another command that writes to the screen by
  hand while the service runs, in the Linux and the Windows parts of running as a service, that such a command refuses
  while the service has the screen, with the line of R12, and that a service or a task that starts while another
  sensordeck process has the screen refuses and starts again 10 seconds later. It says where a sentence on `sensors`
  reading the controller each time it runs, or making the LEDs flicker each time, no longer holds while `run` has the
  controller
- **R18** The README says that on Linux the lock is on the device node, so that it holds between every sensordeck
  process that opens the device whatever its user or its environment, and that on Windows it is one lock for the
  screens and one for the controllers, for the processes of one user, so that two processes cannot drive two screens
  at once on Windows

### Tests

- **R19** End-to-end tests run the executable twice at once against the fake screen and the fake controller:
    - `sensors` while `run` holds a session with `commander_color`: `sensors` succeeds, lists the readings of the fake
      tree with no `commander.<n>.temp`, writes the line of R9, and after it ended the fake controller is still in
      software mode with the colour and received no return to hardware mode since the setup of the session
    - a second `run` while the first runs: it ends with status 1 and the line of R12, writes no `running:`, and the
      first goes on sending values
    - `values` while `run` runs: it ends with status 1, the line of R12, and the screen receives no report with its
      value
    - the locks end at the end of the holder: after a `run` that ended on SIGTERM, and after one killed with SIGKILL
      while it held a session, `values` succeeds and `sensors` lists the probes
- **R20** Unit tests cover:
    - the finding of the holder on Linux from a fake proc root: a line of `locks` that matches the device and the inode
      gives the process and its command line, a process whose `cmdline` cannot be read gives the process alone, and no
      matching line gives no holder
    - the content of the lock file of Windows, written and read back, and a content that is not valid giving no holder,
      on every system where the code of the content builds
    - the line of R9 of `sensors` and the line of R10 of the keeper, with fakes that report the controller in use
    - the refusal of R12 and of R13 with a fake transport that reports the screen in use, with both texts of the
      systems
- **R21** The five commands of the suite in `AGENTS.md` succeed

## Decisions

- **D1 An advisory lock of sensordeck, not an exclusive opening.** Linux has no exclusive opening of a hidraw node.
  Windows has one, a share mode of none, which was rejected: it reverses D12 of unit 0016, it makes the opening fail
  whenever iCUE or the software of the vendor has the device open shared, which works today, it cannot name the other
  process, and it would turn the refusal into a sharing violation that does not tell a sensordeck process from another
  program. The lock binds the processes the bug is about, sensordeck's own, and the README keeps the rule for the rest
- **D2 On Linux, `flock` on the device node itself.** The request suggested a lock file under `$XDG_RUNTIME_DIR` with a
  fallback. That was rejected because the file would depend on the environment: a process without the variable, under
  `sudo`, from cron or from a login without logind, would use another file and not see the lock, and any fallback has
  the same gap. A lock on the opened node is the same for every process that opens the device, whatever its user, its
  environment or the path it opened, so the key is the device: a replug gives a new node and needs nothing, and two
  screens are two locks. The kernel ends the lock when the last descriptor of the opening is closed, which a process
  that ends does, SIGKILL included, and Go opens files close-on-exec, so a command of a custom metric does not inherit
  it. The end-to-end tests lock the nodes of the fake tree, so a service of the developer that runs while the suite runs
  does not change their outcome. The cost is that the try comes after the opening, which writes nothing to the device
- **D3 On Windows, `LockFileEx` on a file per kind of device.** A handle of the HID API cannot be locked, so the lock is
  on a file. A named mutex was rejected: it is owned by a thread of the system, and a goroutine moves between threads,
  so its release would need the goroutine pinned to its thread for the whole of the hold, and it carries no content to
  name the holder. The system releases the locks of a process when it ends and its handles close, so a crashed process
  does not keep one, and handles that Go opens are not inherited. The directory is the cache directory, as the log of
  `run` uses on Windows, because it is the same for every process of the user, the startup task included. One lock per
  kind is enough for the service, which drives one screen and one controller, and avoids a key made from a device path
  that a replug can change; the cost is R18's limit on two screens
- **D4 The holder is named.** The question a user has at a refusal is whether the other process is the service or a
  `run` started by hand, and the process ID and the command line answer it; the process ID is the one that
  `systemctl --user status sensordeck` shows. On Linux the kernel already knows the holder, through `/proc/locks`, and
  the node cannot hold content. On Windows the lock file holds it. The proc root of the tests makes the Linux lookup
  testable with a fake tree. A holder that cannot be found leaves the line without it, which still says what to do
- **D5 No `--force`.** Overriding the refusal brings back the bug it prevents, and stopping the service, the one other
  process to expect, is one command that the line names
- **D6 `sensors` does not wait.** A session of `run` holds the lock for as long as `run` runs, so waiting would not end.
  A reading of `run` without `commander_color` holds it for under a second, once a period, and a `sensors` that
  happens to come at that moment writes the line, and is run again
- **D7 The cases of `run` follow its existing lines.** A setup refused for the lock is a failed setup (R10), and a
  reading refused for the lock is a failed reading (R11), so `run` gains no new state and no new line of its own
- **D8 `listen` is refused like the others.** It writes nothing, but the request lists it, and one rule for every
  command that opens the screen is simpler to state and to test than an exception
- **D9 A failure of the lock is not ignored (R5).** A process that went on without its lock would bring the bug back
  silently. A lock file that cannot be created is visible and cheap to fix
- **D10 The checks outside the sandbox are the human's.** That `sensors` leaves the LEDs of the running service alone on
  the device, and every behaviour on Windows, are checked by the human. The automated loop does not wait for them, and
  the unit does not depend on them

## Non-goals

- Keeping out programs other than sensordeck, such as iCUE, liquidctl, OpenLinkHub or the software of the screen's
  vendor
- Versions of sensordeck from before this unit, which take no lock
- On Windows, the processes of two users of the same machine, whose cache directories differ
- On Windows, two sensordeck processes that drive two screens at once
- `sensors` getting the probes from the running `run`, through a socket or a file
- Queueing a command until the device is free
- A process that means harm, which can ignore an advisory lock

## Beyond the request

1. `sensors` waits up to two seconds for a lock of the controller held by a reading, as opposed to a session, before it
   leaves the probes out. It needs the holder to tell a reading from a session, through its command line or the content
   of the lock, and a test with a holder that releases. Without it, a `sensors` that meets the minute reading of a
   `run` without `commander_color` writes the line of R9 and is run again
2. `run` without `commander_color` writes a line when a reading was refused for the lock. It is one line in the
   periodic reading and a test. Without it, the probes on the screen are a period old once, with nothing in the log
3. `sensordeck service status` says whether a process holds the screen or the controller, and which. It is a try of
   each lock and a line each, on both systems. Without it, the user runs a command that refuses, which names the holder
