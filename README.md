# sensordeck

Turn a cheap USB "sensor panel" screen into a hardware-monitoring dashboard on **Linux**.

On Linux, sensordeck drives the screen with the low-level commands (`devices`, `clock`, `values`, `upload`, `listen`
and `raw`), shows live hardware readings with `sensordeck run`, lists them with `sensordeck sensors`, and runs `run` as
a systemd user service with `sensordeck service`. It also reads the temperature probes of a Corsair iCUE Commander Core
XT. `sensordeck theme` works with themes and needs no screen: `use` sets the theme of the config file, which `run`
uploads when it is not on the screen, `compile` turns a theme, by name or folder, into a theme file, `preview` draws a
theme as a PNG, `dump` reads a theme file and `list` lists the themes. The built-in theme `minimal-dark` is there.

On Windows the screen and the Commander Core XT work with the commands that open a device, with `sensors` and with
`run`, and need no driver and no rule to install. The sensors of the machine are read on Windows too, natively for the
CPU load, the memory and the network, and through `nvidia-smi` and LibreHardwareMonitor for the rest, as
[Windows: the sensors of the machine](#windows-the-sensors-of-the-machine) says. `--device` takes a
device path on Windows, as `devices` writes it. Corsair iCUE, or the software of the screen's vendor, may hold the
device and make the opening fail, and running either at the same time as sensordeck is not supported. The service is
not available on Windows yet. The Windows transport and the sensors of Windows have not yet been checked on a real
Windows machine.

![The preview of the built-in theme minimal-dark: tiles for the CPU, the GPU, the memory, the disk, the network and the
clock, each with the number 42 as a sample value](docs/minimal-dark.png)

This image is the preview that `sensordeck theme preview` draws from the theme `minimal-dark` with sample values, and
not a photo of the screen.

Many small PC screens are sold as "Windows only" and depend on the vendor's own software. sensordeck is an open-source
replacement that works on Linux:

- Show live hardware readings: CPU and GPU temperature and load, VRAM and RAM usage, NVMe temperatures, the
  temperature probes of a Corsair iCUE Commander Core XT, network rates and a clock, as the table of metrics lists.
- Use themes that are plain folders (a `theme.yaml` layout, images and any TTF font). sensordeck compiles a theme into
  the device's format and uploads it. It uploads again only when the theme changes.
- Let you preview a theme as a PNG on your PC, so you can design without flashing the device each time.
- Run in the background as a service (systemd on Linux, with a startup task on Windows still to come).
- Ship as a single executable with nothing else to install. Built-in themes are included.

## Supported devices

sensordeck drives this one screen:

| | |
|---|---|
| Sold as | "3.5 inch PC CPU Data Monitor", "Windows only", under various Amazon/AliExpress brands |
| Vendor name | SmartMonitor (llhmi.com / llTechCo.,Ltd) |
| USB ID | `0483:0065` |
| Screen | 3.5", 480×320 (480x320) |

The one temperature controller whose probes sensordeck reads is the Corsair iCUE Commander Core XT, USB ID `1b1c:0c2a`.

The architecture is generic, so other screens can be added later.

## Installing

There are three ways, in this order. All of them give the executable `sensordeck`, which needs nothing else to run.
On Linux, install the udev rule of [Linux: access to the screen](#linux-access-to-the-screen) as well.

### A release archive

Take the archive for your platform from the
[releases page](https://github.com/norareit/sensordeck/releases) of `github.com/norareit/sensordeck`, where
`<version>` is the tag without its `v`:

* `sensordeck_<version>_linux_amd64.tar.gz` for Linux on 64-bit x86
* `sensordeck_<version>_linux_arm64.tar.gz` for Linux on 64-bit ARM
* `sensordeck_<version>_windows_amd64.zip` for Windows, where the screen and the Commander Core XT work with the commands that
  open a device, `sensors` and `run`, and the sensors of the machine are read natively and through LibreHardwareMonitor,
  as [Windows: the sensors of the machine](#windows-the-sensors-of-the-machine) says

Take `sensordeck_<version>_checksums.txt` from the same release, and check the archive you took against it:

```bash
sha256sum --check --ignore-missing sensordeck_<version>_checksums.txt
```

Extract the archive and put the executable on your `PATH`, for instance:

```bash
tar -xzf sensordeck_<version>_linux_amd64.tar.gz
install -m 0755 sensordeck ~/.local/bin/
```

The archive also holds `README.md`, `LICENSE`, `THIRD_PARTY_LICENSES` and the udev rule `udev/70-sensordeck.rules`, which
[Linux: access to the screen](#linux-access-to-the-screen) tells how to install.

### go install

With Go 1.27 or later, the version that `go.mod` names:

```bash
go install github.com/norareit/sensordeck@latest
```

The executable lands in `$(go env GOPATH)/bin`. The udev rule is not part of it, and comes from the repository, from
the file `udev/70-sensordeck.rules`.

### From source

Clone the repository and build with CGO disabled, as the releases are:

```bash
git clone https://github.com/norareit/sensordeck.git
cd sensordeck
CGO_ENABLED=0 go build -trimpath -o sensordeck .
```

## Quick start

This takes you from nothing to the built-in theme `minimal-dark` on the screen. On Windows the sensors of the machine are
read natively and through LibreHardwareMonitor, as [Windows: the sensors of the machine](#windows-the-sensors-of-the-machine)
says, and the first step is for Linux only.

1. On Linux, install the udev rule of [Linux: access to the screen](#linux-access-to-the-screen)
2. Set the theme: `sensordeck theme use minimal-dark`. It writes the key `theme` into the config file, which it creates
   when there is none, at the path that [Showing sensor values on the screen](#showing-sensor-values-on-the-screen)
   names
3. Start `sensordeck run`. At its first start it uploads the theme to the screen, which takes about 10 seconds and
   reboots the screen, and then it sends the values. A later start finds the theme on the screen and uploads nothing

The tiles show the CPU temperature and load, the GPU temperature and load, the share of RAM in use and the GiB used, the
VRAM used and its total in MiB, the temperature of the first NVMe drive, the network rates up and down in KiB/s, and the
time and the date. The GPU tiles need `nvidia-smi` and show the first NVIDIA GPU. The disk tile is the first NVMe drive
in the order of the places of the hardware, as the table of metrics says.

When the host cannot read a metric of a tile, `run` does not send its value, so the tile keeps the last value it was
sent. A tile that was never sent one shows what the screen shows for a number with no value, which has not been checked
on the device.

`sensordeck theme preview minimal-dark preview.png` draws the theme as a PNG, with no screen.

## Commands

```bash
sensordeck devices                # list connected screens
sensordeck clock                  # set the clock of the screen to the current local time
sensordeck values                 # send the values of numeric tags to the screen
sensordeck upload                 # upload a theme file to the screen as it is
sensordeck listen                 # show the input reports that the screen sends
sensordeck raw                    # send one report of the bytes given and show the answer
sensordeck sensors                # list the sensor readings that can be read now
sensordeck run                    # send live sensor values to the screen
sensordeck theme list             # show built-in and installed themes
sensordeck theme use minimal-dark # switch themes
sensordeck theme preview <name-or-folder> <out.png> # render a theme to PNG with sample values
sensordeck theme compile <name-or-folder> <out.dat> # compile a theme into a theme file and its tags file
sensordeck theme dump <file.dat> <out-dir>          # describe a theme file as JSON and PNG files
sensordeck service install        # run `run` as a systemd user service that starts at login
sensordeck service status         # show whether the service is installed, enabled and active
sensordeck service uninstall      # stop, disable and remove the service
sensordeck --version              # print the version
sensordeck --help                 # list the commands and flags
```

## Linux: access to the screen

On Linux the screen is reached through hidraw, and a udev rule gives the user of the active session access to its
device node. Without the rule the node belongs to root, and the commands fail with a message that points here.

The rule is one line:

```
SUBSYSTEM=="hidraw", ATTRS{idVendor}=="0483", ATTRS{idProduct}=="0065", MODE="0660", TAG+="uaccess"
```

It is the file `udev/70-sensordeck.rules` of this repository. To install it, copy that file to `/etc/udev/rules.d/`
and replug the screen. The file name starts with `70` because the `uaccess` tag is acted on by `73-seat-late.rules`,
and the rule must sort before it.

The same file holds a second rule of the same form, for the USB ID `1b1c:0c2a` of the Corsair iCUE Commander Core XT, so
that its temperature probes can be read without liquidctl. Its metrics are left out when the controller cannot be
opened, for lack of this rule for instance.

hidraw is provided by the kernel's `usbhid` driver, so that driver must stay bound to the screen. A rule that unbinds
`usbhid` for this device, as some guides for other tools add, has to be removed.

## Showing sensor values on the screen

`sensordeck sensors` lists the readings that can be read now, one line each: the metric name, the value with one
decimal and the unit, separated by tabs. A metric that cannot be read is left out. When none can be read, it says so on
standard error.

```
cpu.temp	51.4	°C
cpu.clock	4425.4	MHz
ram.used	21.8	GiB
```

The metrics are listed below with their sources on Linux. [Windows: the sensors of the machine](#windows-the-sensors-of-the-machine)
names those of Windows.

| Metric | Unit | Source |
|---|---|---|
| `cpu.temp` | °C | `k10temp`, the sensor `Tctl` |
| `cpu.ccd<n>.temp` | °C | `k10temp`, the sensor `Tccd<n>` |
| `cpu.util` | % | `/proc/stat` |
| `cpu.clock` | MHz | the highest clock of all CPUs |
| `gpu.temp` | °C | `nvidia-smi` |
| `gpu.util` | % | `nvidia-smi` |
| `gpu.mem.used` | MiB | `nvidia-smi` |
| `gpu.mem.total` | MiB | `nvidia-smi` |
| `ram.used` | GiB | `/proc/meminfo`, the total minus what is available |
| `ram.total` | GiB | `/proc/meminfo` |
| `ram.util` | % | `/proc/meminfo`, the share of the total that `ram.used` is |
| `ram.<n>.temp` | °C | `spd5118`, one for each memory module |
| `nvme.<n>.temp` | °C | `nvme`, the sensor `Composite`, one for each drive |
| `net.up` | KiB/s | `/proc/net/dev`, the bytes sent |
| `net.down` | KiB/s | `/proc/net/dev`, the bytes received |
| `commander.<n>.temp` | °C | the temperature probes of a Corsair iCUE Commander Core XT |

The GPU metrics need `nvidia-smi` and come from the first NVIDIA GPU only. Numbered metrics count from 1, in the order
of the places of the hardware in the system, which does not change when the drivers load in another order. For
`commander.<n>.temp`, *n* counts the probes of the controller from 1.

`net.up` and `net.down` are the sum over the network interfaces that belong to a device, so that loopback and virtual
interfaces such as those of Docker, bridges and VPNs are left out. A KiB is 1024 bytes. The rates are measured between
two readings, the first of which waits a moment for the second.

`sensordeck run` sends the metrics of the theme in a config file to the screen. The config file is
`sensordeck/config.yaml` in your config directory (`$XDG_CONFIG_HOME`, or `~/.config`), and `--config <path>` reads
another file. It names the theme with the key `theme`, a theme name or a folder path as `sensordeck theme compile` takes
it. A folder path that is not absolute is relative to the directory of the config file. `sensordeck theme use` sets the
key:

```yaml
interval: 1s
theme: minimal-dark
```

`run` compiles the theme, takes the tags that the compiler assigned to the metrics of its number widgets, and uploads the
theme to the screen when it is not there, as [Writing a theme](#writing-a-theme) says. The key `theme` excludes the key
`tags`, because the theme decides the tags, and a file that holds both is refused. A theme that fails to compile ends
`run` before it opens the screen.

The key `tags` stays for a vendor theme that you put on the screen with `sensordeck upload`. Its tags are those of the
theme on the screen, so only you know which tag shows what. This one is for a vendor theme in which tag 1 shows the CPU
temperature and tag 2 the CPU speed:

```yaml
interval: 1s
tags:
  cpu.temp: 1
  cpu.clock: 2
```

`interval` is the time between two rounds, of at least 100 milliseconds, and is 1 second without the key. A tag is a
whole number from 0 to 255.

`commander_interval` is the shortest time between two readings of the Commander Core XT, in the same syntax as
`interval`, such as `1m` or `30s`. It is at least `interval`, and is 1 minute without the key, or the interval when that
is longer. Each reading makes the LEDs of the fans on the controller flicker once, which is why it is read less often.
The period applies when `commander_color` is not given. With it, the controller is read in every round, as
[The Corsair iCUE Commander Core XT](#the-corsair-icue-commander-core-xt) says.
Between two readings `sensordeck run` sends the values of the last reading. `sensordeck sensors` reads the controller
once each time it runs.

Values are sent as whole numbers, rounded, so 51.5 is sent as 52. A metric that cannot be read is not sent, so the
screen keeps its last value.

`sensordeck run` sends the values until Ctrl+C or SIGTERM. It sets the clock of the screen when it starts and once a
second, and it keeps running when the screen is unplugged and plugged in again.

`run` reads the config file again when its modification time or its size changes, at the start of a round, so an edit
or `sensordeck theme use` takes effect within a round, with no restart. A reload applies the theme and the tags. A change
of `interval`, `commander_interval`, `commander_color` or `librehardwaremonitor_url` is not applied: `run` writes a line to standard error that
says a restart applies it, and keeps the value it runs with. A file that cannot be read, is not valid or names a theme
that does not compile is not applied: `run` writes one line to standard error and goes on with the config it has. An
upload that has begun is not interrupted by Ctrl+C or SIGTERM: `run` ends after the upload ended.

## Windows: the sensors of the machine

On Windows `sensordeck sensors` and `sensordeck run` give the same metric names, with the same units, as on Linux, so a
config written on one system feeds the same tiles on the other. They come from three places.

Windows gives these natively, with nothing to install: `cpu.util`, `ram.used`, `ram.total`, `ram.util`, `net.up` and
`net.down`. The GPU metrics come from `nvidia-smi`, which the NVIDIA driver installs, when it is on `PATH`. An old driver
may put it in `C:\Program Files\NVIDIA Corporation\NVSMI`, which is not on `PATH`, and then there are no GPU metrics until
you add that folder to `PATH`. `net.up` and `net.down` are the sum over the network adapters with a connector, so that
loopback, tunnels, filter modules and the virtual adapters of Hyper-V, WSL and VPNs are left out.

The temperatures and the clock of the CPU, and the temperatures of the drives and the memory modules, need a kernel
driver that Windows does not give a program. `cpu.temp`, `cpu.ccd<n>.temp`, `cpu.clock`, `ram.<n>.temp` and
`nvme.<n>.temp` come from [LibreHardwareMonitor](https://github.com/LibreHardwareMonitor/LibreHardwareMonitor), a program
that has such a driver and serves its readings over HTTP. sensordeck neither installs nor starts it, and the native
metrics do not depend on it.

To turn on its web server, start LibreHardwareMonitor as an administrator, which its driver needs, check
Options > Remote Web Server > Run, and keep it running. Options > Run On Windows Startup starts it with Windows. The
groups CPU, RAM and Storage Devices of its menu must be on, as they are by default.

The key `librehardwaremonitor_url` of the config file is the address of its document. Its default is
`http://localhost:8085/data.json`, which is right while the web server is left as it is. Set it when you changed the
port, the address or the authentication of the web server, with the user and the password in the URL when the
authentication is on:

```yaml
librehardwaremonitor_url: http://monitor:secret@localhost:9000/data.json
```

The key does nothing on Linux, and a config that has it is valid there. `sensordeck sensors` on Windows reads the key
from the config file at the place where `run` reads it without `--config`, so that it reads the same address as `run`,
and takes the default when it finds no file there. It writes one line to standard error and goes on with the default
when that file cannot be read or is not valid. The user and the password are left out of the lines that name the URL.

The sensors of LibreHardwareMonitor give the metrics as follows. The CPU is the first AMD CPU, or else the first Intel
CPU, of the document:

* `cpu.temp` is `Core (Tctl/Tdie)`, or without it `Core (Tctl)`, or without it `CPU Package`
* `cpu.ccd<n>.temp` is `CCD<n> (Tdie)`
* `cpu.clock` is the highest of the clocks `Core #<n>`, `P-Core #<n>`, `E-Core #<n>` and `CPU Core`
* `nvme.<n>.temp` is the temperature of the *n*-th NVMe drive, the sensor that LibreHardwareMonitor numbers 0, the
  composite temperature
* `ram.<n>.temp` is the temperature of the *n*-th memory module that has a thermal sensor

A CPU sensor that you renamed in LibreHardwareMonitor is not found, because the CPU sensors are told by their names. The
temperatures of memory modules need LibreHardwareMonitor 0.9.5 or newer. The numbers of `nvme.<n>.temp` and `ram.<n>.temp`
follow the numbers that LibreHardwareMonitor gives the drives and the modules, in order, counted from 1, and a drive keeps
its number when its temperature is missing. They may differ from the numbers on Linux on the same machine.

When the web server cannot be read, `sensordeck sensors` writes the line `LibreHardwareMonitor cannot be read at <URL>:
<error>. The temperatures and cpu.clock need it running, with Options > Remote Web Server > Run checked` to standard
error, and lists the other metrics. When it can be read and gives none of these metrics, it writes the line
`LibreHardwareMonitor at <URL> gives no temperature and no clock that sensordeck knows`. `sensordeck run` writes the first
line once when the reading fails, and not again while it keeps failing, and writes `LibreHardwareMonitor read again at
<URL>` when it works again. A reading waits at most half a second, and the tiles of the metrics that could not be read
keep their last value. The other metrics keep working throughout. `run` asks the web server only while a metric that
needs it is mapped.

The sensors of Windows have not yet been checked on a real Windows machine.

## The Corsair iCUE Commander Core XT

The controller has a hardware mode, in which it runs its fans and shows the lighting stored in it, and a software mode,
in which it shows only what a program sends it, and is dark when nothing is sent. Reading the probes needs software
mode, so without `commander_color` each reading of `sensordeck run` makes the LEDs flicker once, at most once per
period of `commander_interval`, of 1 minute by default. `sensordeck sensors` makes them flicker once each time it runs.

With `commander_color`, `run` holds software mode for as long as it runs. It lights the LEDs of every connected port in
that one colour, reads the probes in every round and keeps the controller awake with a reading once a second. The value
must be in quotes, because `#` starts a comment in YAML, and the key cannot be used with `commander_interval`. Choose one
of the two ways. Without the colour:

```yaml
interval: 1s
commander_interval: 1m
tags:
  cpu.temp: 1
  commander.1.temp: 18
  commander.2.temp: 19
```

With the colour:

```yaml
interval: 1s
commander_color: "#00c8ff"
tags:
  cpu.temp: 1
  commander.1.temp: 18
  commander.2.temp: 19
```

What you see with the colour: at the start a moment of darkness and then the colour, steady while `run` runs, and after
Ctrl+C or SIGTERM the lighting stored in the controller again.

When the controller is unplugged or stops answering, `run` returns it to hardware mode if it can, writes
`controller lost` to standard error, leaves the metrics of the controller out and keeps the screen going. It tries
again after 10 seconds and then less often, up to every 5 minutes, and writes `controller back` and lights the colour
again when it succeeds.

Another program that talks to the controller at the same time, such as iCUE, liquidctl, OpenLinkHub or
`sensordeck sensors`, is not supported: the commands of both mix, readings fail, the LEDs may show the stored lighting or
go dark, and `run` sets the colour again at its next setup.

What the controller does with fans that it runs, in software mode with no fan settings sent, is not known. sensordeck
sends no fan settings, and the fans of the machine it was tested on are not run by the controller.

At most 29 LEDs can be lit. A controller with more makes `run` write `controller setup failed`. A killed process, as
with SIGKILL, or a loss of power can leave the controller in software mode with the colour or dark, until a program
returns it to hardware mode or it is replugged.

The controller needs the second rule of `udev/70-sensordeck.rules`, which gives the user of the active session access
to its hidraw node:

```
SUBSYSTEM=="hidraw", ATTRS{idVendor}=="1b1c", ATTRS{idProduct}=="0c2a", MODE="0660", TAG+="uaccess"
```

## Running as a service

This is for Linux, where sensordeck runs `run` as a systemd user service. Set the config file first, as the
[quick start](#quick-start) says, for instance with `sensordeck theme use minimal-dark`.

`sensordeck service install` writes the unit file `~/.config/systemd/user/sensordeck.service`, or the file of that name
in `$XDG_CONFIG_HOME/systemd/user/` when the variable is set, and enables and starts the service at once. It starts at
each later login of the user. The unit runs `sensordeck run --config <file>` with the path of the executable that ran
`install`, symbolic links resolved. The config file is the one given with `--config`, or else the default one, and the
unit always names it, because the environment of the systemd user instance may not hold the `XDG_CONFIG_HOME` of your
shell. `install` first loads that file as `run` does at its start, theme included, and writes nothing and makes no
call when `run` would refuse it. It opens no screen, so it works with the screen unplugged.

Running `sensordeck service install` again overwrites the unit file and restarts the service. That is how to apply a
moved or replaced executable, or another `--config`. A change of the unit is made in a drop-in with
`systemctl --user edit sensordeck.service`, which `install` does not overwrite.

`sensordeck service status` writes whether the unit file exists, and whether the service is enabled and active.
`systemctl --user status sensordeck.service` gives more.

The log is the journal of the user: `journalctl --user -u sensordeck`, with `-f` to follow it. It holds the lines that
`run` writes to standard output and standard error.

`sensordeck theme use` and edits of the config file reach the running service within a round, by the reload of `run`.
A change of `interval`, `commander_interval` or `commander_color` needs
`systemctl --user restart sensordeck.service`.

The service is restarted 10 seconds after it ended with an error, as when no screen was found at its start, and there is
no limit to the number of restarts. A stop with `systemctl --user stop sensordeck.service` leaves the screen with its
last values and returns the Commander Core XT to hardware mode.

Running `sensordeck run`, or another command that writes to the screen, by hand while the service runs is not
supported, because both write to the screen. Stop the service first.

`sensordeck service uninstall` stops and disables the service, removes the unit file and reloads systemd.

A user service runs only while the user has a session. `loginctl enable-linger` makes it start at boot and keep running
after the user logged out, and `loginctl disable-linger` undoes it. Lingering is your choice, and `install` does not
make it. The udev rule of [Linux: access to the screen](#linux-access-to-the-screen) gives access to the screen to the
user of the active local session, so a service that starts at boot by lingering cannot open the screen until that user
logs in, and is restarted every 10 seconds until then.

## Writing a theme

A theme is a folder: a `theme.yaml` that lays out the screen, and the PNG pictures and TTF fonts it names. A theme can
also use the twelve Go fonts that are built into sensordeck. A complete example:

```yaml
name: Night
author: Ada
license: MIT
background: background.png        # a 480 x 320 PNG, or a colour such as "#101820"
fonts:
  label: {file: fonts/label.ttf, size: 22, color: "#8899aa"}
  value: {builtin: go-bold, size: 48, color: "#00c8ff"}
widgets:                          # drawn in this order
  - {type: image, file: icons/cpu.png, x: 20, y: 20}
  - {type: text, font: label, text: CPU, x: 70, y: 28}
  - {type: number, font: value, metric: cpu.temp, digits: 3, x: 70, y: 60}
  - {type: datetime, font: label, format: "YYYY-MM-DD HH:mm:ss", x: 20, y: 280}
```

Two commands work on a theme and need no screen. The first argument is a theme name or a folder:

```bash
sensordeck theme preview ./my-theme preview.png   # draw the theme as the compiled file would show it
sensordeck theme compile ./my-theme img.dat       # write img.dat and img.tags.yaml
```

A first argument is a folder path when it is `.` or `..` or holds `/` (on Windows also `\` or `:`), and a theme name
otherwise, so a folder in the working directory is named `./my-theme`. A name is looked up first among the installed
themes, the directories in the themes directory, and then among the built-in ones. The themes directory is
`~/.config/sensordeck/themes`, or `$XDG_CONFIG_HOME/sensordeck/themes`, on Linux and `%AppData%\sensordeck\themes` on
Windows. A theme folder copied there under a name is found by that name, and replaces a built-in theme of that name.
`sensordeck theme list` lists the built-in and the installed themes, one line each, with tabs between the name, the kind
and the folder of an installed theme.

The compiler gives each metric of the number widgets a tag and writes the mapping beside the theme file, in
`img.tags.yaml`:

```yaml
# Paste this block into the config file of sensordeck run.
tags:
  cpu.temp: 1
```

That file is for a config file with `tags`. A config file with the key `theme` needs no tags block, because `run` takes
the tags from the compiler itself.

To use a theme, set it with `sensordeck theme use <name-or-folder>`. The command compiles the theme, which fails with
the error of the compilation and changes nothing when the theme is not valid, and then sets the key `theme` of the
config file. It replaces the line of the key, or adds a line at the end of the file, and keeps every other byte of it,
and it creates the file when there is none. It writes the absolute path for a folder. It refuses a config file that holds
`tags`, one that is not valid and one whose root is written in the flow style of YAML, and changes none of them. A
running `sensordeck run` applies the new theme within a round.

`run` uploads the theme only when it is not on the screen. It keeps the SHA-256 of the compiled theme file that each
screen holds in a state file, by the serial of the screen, and uploads when the hash of the compiled theme differs from
the one stored for the screen. The upload takes about 10 seconds and reboots the screen, and `run` sends no value and no
clock until it is done. It compiles the theme in memory and keeps no compiled file. The state file is
`$XDG_CACHE_HOME/sensordeck/uploaded-themes.json`, or `~/.cache/sensordeck/uploaded-themes.json`, on Linux and
`%LocalAppData%\sensordeck\uploaded-themes.json` on Windows. The check is made at the start, after a reload that
changes the theme and after the screen came back, so a restart of `run` or of the machine uploads nothing. A theme whose
upload failed is tried again after 10 seconds, and then after longer waits, up to 5 minutes.

* Running `sensordeck theme use` again after you edited a theme folder puts the edit on the screen, because writing the
  config file again makes `run` compile the theme again, and the hash of the new file differs
* `sensordeck upload` keeps the state file true too. It removes the entry of the screen before it uploads and records the
  hash of the file it uploaded, so after you uploaded another file by hand, `run` puts its own theme back at its next
  start or reload
* A theme that other software, such as the vendor's, put on the screen is not known to `run`. Delete the state file to
  make the next start upload the theme of the config file. The same holds for a screen with no serial, or one that
  shares its serial with another: they have one entry, so a different screen in the place of the first one is taken to
  hold its theme

The folder format, the tags and the limits are described in [docs/theme-folder.md](docs/theme-folder.md).

## Reading a theme file

`sensordeck theme dump <file.dat> <out-dir>` reads a theme file, the `img.dat` that `upload` sends to the screen, and
writes what is in it into the directory, which it creates when it does not exist:

* `theme.json`, a description of the file: the header and every record with its fields. A field whose meaning is a
  guess has a name that ends in `_guess`, and a byte whose meaning is unknown is named `byte_<n>_unknown`
* a PNG of each resource, named after the slot of its record, such as `slot03-image-frame1.png`
* `preview.png`, the 480 x 320 screen composed from the records alone

The preview shows every number as 42 and the date and time as 2026-11-15 09:41:07, draws numbers left-aligned and bars
as magenta outlines, because the device's own alignment and the fields of a bar are not known yet. The command touches no
device and works with no screen connected, on Linux and on Windows. It prints each warning of the reader, such as a
record of a type it does not know, to standard error. The format is described in
[docs/theme-format.md](docs/theme-format.md).

## Making a release

This section is for the maintainer. CI runs the suite of five commands on every push and every pull request. Pushing a
tag of the form `v<major>.<minor>.<patch>` makes the release workflow run the suite and, only when it succeeds, build
the archives and the checksum file with GoReleaser into a draft release. The draft is checked and published by hand on
GitHub.

To try the configuration locally, run `goreleaser release --snapshot --clean`. It writes into `release/`, which git
ignores.

The file `THIRD_PARTY_LICENSES` holds the licence texts of the code in the executable, and a test fails when it differs
from what the generator gives. After a dependency changed, write it again at the module root with:

```bash
go run ./internal/notices/generate THIRD_PARTY_LICENSES
```

## Credits

Thanks to the [turing-smart-screen-python-HIDdev2](https://github.com/Agentry433/turing-smart-screen-python-HIDdev2)
project for discovering the protocol these screens use.

The theme compiler draws glyphs with [golang.org/x/image](https://pkg.go.dev/golang.org/x/image), and the Go fonts that
a theme can name are built into sensordeck. Both are by the Go authors, under a BSD licence.

sensordeck is built with these modules, each under the licence of its own licence file:

* [github.com/spf13/cobra](https://github.com/spf13/cobra), Apache 2.0
* [github.com/spf13/pflag](https://github.com/spf13/pflag), BSD 3-clause
* [github.com/inconshreveable/mousetrap](https://github.com/inconshreveable/mousetrap), Apache 2.0
* [go.yaml.in/yaml/v3](https://github.com/yaml/go-yaml), MIT and Apache 2.0
* [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys), BSD 3-clause
* [golang.org/x/image](https://pkg.go.dev/golang.org/x/image), BSD 3-clause, with the Go fonts
* [golang.org/x/text](https://pkg.go.dev/golang.org/x/text), BSD 3-clause

`THIRD_PARTY_LICENSES` holds their licence texts, and those of the Go standard library.

## License

sensordeck is under the MIT licence, in [LICENSE](LICENSE).
