# sensordeck

Turn a cheap USB "sensor panel" screen into a hardware-monitoring dashboard on **Linux**.

On Linux, sensordeck drives the screen with the low-level commands (`devices`, `clock`, `values`, `upload`, `listen`
and `raw`), shows live hardware readings with `sensordeck run`, lists them with `sensordeck sensors`, and runs `run` as
a systemd user service with `sensordeck service`. It also reads the temperature probes of a Corsair iCUE Commander Core
XT. `sensordeck theme` works with themes and needs no screen: `use` sets the theme of the config file, which `run`
uploads at its start, `compile` turns a theme, by name or folder, into a theme file, `preview` draws a
theme as a PNG, `dump` reads a theme file and `list` lists the themes. The built-in themes `minimal-dark` and `minimal-dark-probes` are there.

On Windows the screen and the Commander Core XT work with the commands that open a device, with `sensors` and with
`run`, and need no driver and no rule to install. The sensors of the machine are read on Windows too, natively for the
CPU load, the memory and the network, and through `nvidia-smi` and LibreHardwareMonitor for the rest, as
[Windows: the sensors of the machine](#windows-the-sensors-of-the-machine) says. `--device` takes a
device path on Windows, as `devices` writes it. Corsair iCUE, or the software of the screen's vendor, may hold the
device and make the opening fail, and running either at the same time as sensordeck is not supported. `sensordeck service`
runs `run` on Windows as a startup task of the Task Scheduler that starts at your logon, as
[Running as a service](#running-as-a-service) says. The Windows transport, the sensors of Windows and the startup task
have not yet been checked on a real Windows machine.

![The preview of the built-in theme minimal-dark: tiles for the CPU, the GPU, the memory, the disk, the network and the
clock, each with the number 42 as a sample value](docs/minimal-dark.png)

This image is the preview that `sensordeck theme preview` draws from the theme `minimal-dark` with sample values, and
not a photo of the screen.

Many small PC screens are sold as "Windows only" and depend on the vendor's own software. sensordeck is an open-source
replacement that works on Linux:

- Show live hardware readings: CPU and GPU temperature and load, VRAM and RAM usage, NVMe temperatures, the
  temperature probes of a Corsair iCUE Commander Core XT, network rates and a clock, as the table of metrics lists.
- Use themes that are plain folders (a `theme.yaml` layout, images and any TTF font). sensordeck compiles a theme into
  the device's format and uploads it. `run` uploads the theme at each start, and again when a reload changes it.
- Let you preview a theme as a PNG on your PC, so you can design without flashing the device each time.
- Run in the background as a service (a systemd user service on Linux, a startup task of the Task Scheduler on Windows).
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
3. Start `sensordeck run`. It uploads the theme to the screen at each start, which takes about 10 seconds and reboots
   the screen, and it sends the values after that

The tiles show the CPU temperature and load, the GPU temperature and load, the share of RAM in use and the GiB used, the
VRAM used and its total in MiB, the temperature of the first NVMe drive, the network rates up and down in KiB/s, and the
time and the date. The GPU tiles need `nvidia-smi` and show the first NVIDIA GPU. The disk tile is the first NVMe drive
in the order of the places of the hardware, as the table of metrics says.

When the host cannot read a metric of a tile, `run` does not send its value, so the tile keeps the last value it was
sent. A tile that was never sent one shows nothing: that a number that never received a value shows nothing was observed on
the device.

`sensordeck theme preview minimal-dark preview.png` draws the theme as a PNG, with no screen.

## Built-in themes

* `minimal-dark` shows the tiles that the [quick start](#quick-start) lists.

  ![The preview of the built-in theme minimal-dark: tiles for the CPU, the GPU, the memory, the VRAM with its total, the
  first NVMe drive, the network and the clock, each with the number 42 as a sample value](docs/minimal-dark.png)

* `minimal-dark-probes` is `minimal-dark` with the VRAM total replaced by the temperature of the second NVMe drive, and
  the disk tile replaced by the two probes of a Corsair iCUE Commander Core XT.

  ![The preview of the built-in theme minimal-dark-probes: tiles for the CPU, the GPU, the memory, the VRAM with the
  temperature of the second NVMe drive under it, the two probes GPU CABLE and INTAKE, the network and the clock, each with
  the number 42 as a sample value](docs/minimal-dark-probes.png)

The two images are previews that `sensordeck theme preview` draws with sample values, and not photos of the screen.

`minimal-dark-probes` expects a Commander Core XT with two temperature probes, `commander.1.temp` and
`commander.2.temp`, which the theme labels `GPU CABLE` and `INTAKE` because on the machine it was made for probe 1 is at
the GPU power cable and probe 2 at the front intake. It also expects a second NVMe drive: `nvme.2.temp` is the second
drive in the order of the places of the hardware, its PCI path, as the table of metrics says. For what reading the
controller does to its LEDs, see [The Corsair iCUE Commander Core XT](#the-corsair-icue-commander-core-xt). A tile whose
metric the host cannot read behaves as the [quick start](#quick-start) says.

`minimal-dark-probes` is an example to copy and adapt: copy its folder,
`internal/builtinthemes/themes/minimal-dark-probes` of the repository, into the themes directory under a name of your
own, and change the labels and the metrics to those of your own hardware.

## Commands

```bash
sensordeck devices                # list connected screens
sensordeck clock                  # set the clock of the screen to the current local time
sensordeck values                 # send the values of numeric tags to the screen
sensordeck upload                 # upload a theme file to the screen as it is
sensordeck listen                 # show the input reports that the screen sends
sensordeck raw                    # send one report of the bytes given and show the answer
sensordeck sensors [--config <path>] # list the sensor readings that can be read now, custom metrics included
sensordeck run                    # send live sensor values to the screen
sensordeck theme list             # show built-in and installed themes
sensordeck theme use minimal-dark # switch themes
sensordeck theme preview <name-or-folder> <out.png> # render a theme to PNG with sample values
sensordeck theme compile <name-or-folder> <out.dat> # compile a theme into a theme file and its tags file
sensordeck theme dump <file.dat> <out-dir>          # describe a theme file as JSON and PNG files
sensordeck service install        # run `run` as a systemd user service (Linux) or a startup task (Windows) that starts at login
sensordeck service status         # show whether the service or the task is installed, enabled and active or running
sensordeck service uninstall      # stop and remove the service or the task
sensordeck --version              # print the version
sensordeck --help                 # list the commands and flags
```

`sensordeck values` takes arguments `<tag>=<value>`, with a tag of 0 to 255 and a value in decimal or in hexadecimal
with `0x`, optionally after a `-`. A value from -32768 to 32767 is the number the screen shows. A value from 32768 to
65535 is the raw 16-bit pattern, which the screen shows as that value minus 65536, so `-1` and `65535` send the same
bytes.

## Linux: access to the screen

On Linux the screen is reached through hidraw, and a udev rule gives the user of the active session access to its
device node. Without the rule the node belongs to root, and the commands fail with a message that points here.

The rule file holds three rules:

```
SUBSYSTEM=="hidraw", ATTRS{idVendor}=="0483", ATTRS{idProduct}=="0065", MODE="0660", TAG+="uaccess"
SUBSYSTEM=="hidraw", ATTRS{idVendor}=="1b1c", ATTRS{idProduct}=="0c2a", MODE="0660", TAG+="uaccess"
SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTR{idVendor}=="0483", ATTR{idProduct}=="0065", MODE="0660", TAG+="uaccess"
```

It is the file `udev/70-sensordeck.rules` of this repository. To install it, copy that file to `/etc/udev/rules.d/`
and replug the screen. The file name starts with `70` because the `uaccess` tag is acted on by `73-seat-late.rules`,
and the rule must sort before it.

The second rule is of the same form, for the USB ID `1b1c:0c2a` of the Corsair iCUE Commander Core XT, so
that its temperature probes can be read without liquidctl. Its metrics are left out when the controller cannot be
opened, for lack of this rule for instance.

The third rule gives the user of the session access to the USB device node of the screen. `run` and the commands that
write use it to restart a hung screen, see [Hangs of the firmware](docs/protocol.md#hangs-of-the-firmware). A rule
file without that line is an old one, and is installed again. A file with a line that sets `power/control` is the old
rule of the version before. `grep 'SUBSYSTEM=="usb".*uaccess' /etc/udev/rules.d/70-sensordeck.rules` printing nothing
tells an old rule. A replug applies the rule.

hidraw is provided by the kernel's `usbhid` driver, so that driver must stay bound to the screen. A rule that unbinds
`usbhid` for this device, as some guides for other tools add, has to be removed.

## Showing sensor values on the screen

`sensordeck sensors` lists the readings that can be read now, one line each: the metric name, the value with one
decimal and the unit, separated by tabs. A metric that cannot be read is left out. When none can be read, it says so on
standard error. It lists the [custom metrics](#custom-metrics) of the config file after the built-in ones, and takes
`--config <path>` to read another config file than the one at the default place.

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
theme to the screen at its start, as [Writing a theme](#writing-a-theme) says. The key `theme` excludes the key
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
whole number from 0 to 255. A metric may also map to a mapping with the keys `tag`, the tag it is sent to, and `alarms`, a
list of alarm pairs, each with `normal` and `alarm`, the two tags, and `above`, `below` or both, as the tags file of a theme
with an alarm number writes it. At least one of `tag` and `alarms` is given, and every tag of the block is distinct. In
every round `run` sends, for each pair, the value to the normal tag and -1 to the alarm tag, or when the value is past
the threshold the reverse.

The key `metrics` defines [custom metrics](#custom-metrics), which a theme or the key `tags` can use as a built-in one.

`commander_interval` is the shortest time between two readings of the Commander Core XT, in the same syntax as
`interval`, such as `1m` or `30s`. It is at least `interval`, and is 1 minute without the key, or the interval when that
is longer. Each reading makes the LEDs of the fans on the controller flicker once, which is why it is read less often.
The period applies when `commander_color` is not given. With it, the controller is read in every round, as
[The Corsair iCUE Commander Core XT](#the-corsair-icue-commander-core-xt) says.
Between two readings `sensordeck run` sends the values of the last reading. `sensordeck sensors` reads the controller
once each time it runs.

Values are sent as whole numbers, rounded and clamped to -32768 to 32767, so 51.5 is sent as 52. A negative value shows
with its minus sign, and a reading above 32767, such as a network rate of 40000 KiB/s, shows 32767 while
`sensordeck sensors` shows the reading. A metric that cannot be read is not sent, so the
screen keeps its last value.

`sensordeck run` sends the values until Ctrl+C or SIGTERM. It sets the clock of the screen when it starts and once a
second, and it keeps running when the screen is unplugged and plugged in again.

On Linux `run` also recovers a hung screen, one whose writes time out, by restarting its firmware. It does so at the
first round of the hang and at most once a minute after. It writes `screen hung: ` and the error,
`restarting the screen to recover it`, `screen restarted at string descriptor <index>` and then `screen recovered` to
standard error, each once in a hang. When it cannot restart the screen, for lack of the third rule of the udev rule file,
it writes one line that says to install the current rule, and unplugging the screen once recovers it.

`run` reads the config file again when its modification time or its size changes, at the start of a round, so an edit
or `sensordeck theme use` takes effect within a round, with no restart. A reload applies the theme, the tags and the
`metrics`. A change
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
from the config file named with `--config`, or else at the place where `run` reads it without `--config`, so that it
reads the same address as `run`, and takes the default when it finds no file there. It writes one line to standard error and goes on with the default
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

The commands of [custom metrics](#custom-metrics) run through `cmd.exe`, and every program that sensordeck starts to read
a metric, `nvidia-smi` and those commands, is started with no console window, so that none opens while `run` runs in the
background.

The sensors of Windows have not yet been checked on a real Windows machine.

## Custom metrics

The metrics of the table above are the ones that sensordeck knows. The key `metrics` of the config file of
`sensordeck run` defines more, each read from a file, from a chip of hwmon or from a command, so that hardware that
sensordeck does not know can be shown: an AMD or Intel GPU, the fans and sensors of a board, a water loop, or a value that
a script or a web service gives. A theme or the key `tags` uses a custom metric as it uses a built-in one.

```yaml
metrics:
  water.temp:   {file: /sys/class/hwmon/hwmon5/temp1_input, scale: 0.001}
  gpu.amd.temp: {hwmon: amdgpu, input: temp1_input, scale: 0.001, unit: "°C"}
  fan.pump:     {hwmon: nct6799, input: fan2_input, unit: RPM}
  outside.temp: {command: "curl -s https://example.org/t | jq .temp", every: 5m}
```

A name is one or more segments joined by single dots, each of the characters `a` to `z`, `0` to `9`, `_` and `-`, and it
begins with a letter. A name that is a built-in metric, such as `gpu.temp`, or `nvme.9.temp` that no hardware backs on this
machine, cannot be defined: the config is refused with the name and the line that it is built in and cannot be replaced.
There is no prefix to write. To show an AMD GPU in a built-in theme, copy the theme and name the custom metric in it.
A theme compiles with a custom metric on any machine, because the compiler checks the form of the name only. `run` refuses
a theme whose metric is neither built in nor defined under `metrics`, and says which.

A definition has exactly one source key, and these keys besides:

| Key | Meaning |
|---|---|
| `file` | The absolute path of a file, as the system that runs sensordeck reads it. Read at each reading |
| `hwmon` | The name of a chip, as its `name` file holds it. It needs `input` |
| `input` | With `hwmon`: the file of the chip's directory to read, such as `temp1_input` |
| `index` | With `hwmon`: which chip of that name, counted from 1, 1 without the key |
| `command` | A command line, run through the shell of the system |
| `every` | With `command`: the time between two runs, of at least 1 second, such as `30s` or `5m`, 10 seconds without the key |
| `scale` | A number the reading is multiplied by, 1 without the key |
| `offset` | A number added after the scale, 0 without the key |
| `unit` | The text `sensordeck sensors` writes as the unit, empty without the key. It does not reach the theme |

The number of a reading is the first match in its text of an optional `-` or `+`, one or more digits and optionally a point
and one or more digits. Exponents, a comma as the decimal mark, `NaN` and `Inf` are not numbers, and a text with no number
gives the metric no value. The value is the number times `scale`, plus `offset`. `sensordeck sensors` writes it with one
decimal. The screen shows whole numbers from -32768 to 32767, as for every metric, so `run` rounds the value, a negative value
shows with its minus sign, and tenths are shown by a `scale` ten times larger.

### A chip of hwmon

The number N of a directory `hwmonN` of `/sys/class/hwmon` follows the order in which the drivers load, and changes between
boots, so a `file` path through it may read another chip, or none, after a reboot. The `hwmon` form finds the chip by its
name at each reading. To see the names of the chips, and the files of one:

```bash
cat /sys/class/hwmon/hwmon*/name
ls /sys/class/hwmon/hwmon5/
```

An AMD GPU is the chip `amdgpu`. This reads its edge temperature, its junction temperature, its power and its fan; the
files `temp1_input` and `temp2_input` hold millidegrees, `power1_average` holds microwatts:

```yaml
metrics:
  gpu.amd.temp:     {hwmon: amdgpu, input: temp1_input, scale: 0.001, unit: "°C"}
  gpu.amd.junction: {hwmon: amdgpu, input: temp2_input, scale: 0.001, unit: "°C"}
  gpu.amd.power:    {hwmon: amdgpu, input: power1_average, scale: 0.000001, unit: W}
  gpu.amd.fan:      {hwmon: amdgpu, input: fan1_input, unit: RPM}
```

A machine with two chips of one name, such as a Ryzen 7000 or 9000 desktop processor, whose integrated graphics has an
`amdgpu` chip of its own beside the one of an AMD card, takes the one that `index` names. The chips of a name are counted
from 1 in the order of their device paths, which follows the hardware and not the boot, and a chip with no device path comes
last. This lists the name and the resolved device path of every chip in that order:

```bash
for chip in /sys/class/hwmon/hwmon*; do printf '%s %s\n' "$(cat "$chip/name")" "$(readlink -f "$chip/device")"; done | LC_ALL=C sort
```

The device path of a GPU ends in its PCI address, which `lspci -s <address>` names, so the card and the graphics of the
processor are told apart. If the card is the second `amdgpu` line, this reads it:

```yaml
metrics:
  gpu.amd.temp: {hwmon: amdgpu, input: temp1_input, index: 2, scale: 0.001, unit: "°C"}
```

With fewer chips of the name than `index`, the metric has no value, and `run` and `sensordeck sensors` write the number of
chips found. The fan of a board is read through the chip of its controller, such as `nct6799`:

```yaml
metrics:
  fan.pump: {hwmon: nct6799, input: fan2_input, unit: RPM}
```

### A command

```yaml
metrics:
  outside.temp: {command: "curl -s https://example.org/t | jq .temp", every: 5m, unit: "°C"}
```

The command runs through the shell of the system, `/bin/sh -c` on Linux and `cmd.exe /d /s /c` on Windows, the program of
`ComSpec` when it is set, with the command line as you wrote it, so the quoting is yours. A user who wants PowerShell writes
`powershell -NoProfile -Command "..."` as the command. It runs in the directory of the config file, with the environment of
sensordeck and no standard input. A run that exits with status 0 and writes a number to standard output gives the value.

A command never holds up a round. `run` starts a run when the metric is mapped, none is in progress and none started within
the last `every`, and sends the value of the last run that finished, so the first rounds send no value for it. A run lasts
at most 10 seconds, or `every` when that is shorter, and is then killed. On Linux the command is the leader of a process
group that is killed whole, so that the programs of a pipeline end with it. Windows has no such group, and a program that a
killed command started may go on. When `run` ends, a run in progress is killed. `sensordeck sensors` runs each command once
and waits for it.

### Failures

A custom metric that has no value writes one line to standard error at the first reading of a streak of readings that
fail, and none while the streak lasts. The line starts with `metric <name>: ` and gives the cause: the file that cannot be
read and the error, the number of chips of the name, no number in the text, the exit status of the command with the first
line of its standard error, a command that could not start, or the time limit it outlasted. A line `metric <name>: read
again` says when the metric has a value again. The tile keeps its last value meanwhile. `sensordeck sensors` writes the line
of each custom metric that has no value, and lists the others.

A reload of the config applies `metrics`. A metric whose definition did not change keeps its value, its streak and its run in
progress. A new or changed one starts with no value, and the run of a changed or removed one is killed.

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

The controller also stores which of its ports have LEDs. iCUE rewrites this, and afterwards the LEDs stay dark in
software mode, on Linux and on Windows, although every command of the setup is accepted. `run` checks it at the start of
each session and repairs it when it does not match the fans the controller reports, writing
`controller LED ports configured: the stored configuration did not enable the ports with LEDs, iCUE may have changed it`
to standard error. The repair stays stored in the controller until iCUE changes it again. It is made only when every
port with LEDs has 4 of them, the one kind of fan it was tested with. Otherwise `run` writes
`controller LED ports left as stored` with the port and its LED count, once, and the LEDs may stay dark.

When the controller is unplugged or stops answering, `run` returns it to hardware mode if it can, writes
`controller lost` to standard error, leaves the metrics of the controller out and keeps the screen going. It tries
again after 10 seconds and then less often, up to every 5 minutes, and writes `controller back` and lights the colour
again when it succeeds.

Another program that talks to the controller at the same time, such as iCUE, liquidctl, OpenLinkHub or
`sensordeck sensors`, is not supported: the commands of both mix, readings fail, the LEDs may show the stored lighting or
go dark, and `run` sets the colour again at its next setup. Running iCUE and sensordeck at the same time is not
supported.

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

sensordeck runs `run` as a systemd user service on Linux and as a startup task of the Task Scheduler on Windows. Set the
config file first, as the [quick start](#quick-start) says, for instance with `sensordeck theme use minimal-dark`.

### Linux

On Linux sensordeck runs `run` as a systemd user service.

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

Each start of the service, at the login, at boot with lingering, and at each restart after an error, uploads the theme of
the config, so the values appear about 10 seconds after the start and the screen reboots once.
`sensordeck theme use` and edits of the config file reach the running service within a round, by the reload of `run`,
and a new theme is uploaded then. A change of `interval`, `commander_interval` or `commander_color` needs
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

### Windows

On Windows sensordeck runs `run` as a task of the Task Scheduler. None of what follows has been checked on a real
Windows machine yet. It is written from the documentation of Windows, and a wrong point is to be corrected.

`sensordeck service install` creates the task `sensordeck`, which runs `sensordeck run` at each logon of the user, hidden,
with no administrator rights, and starts it at once. The task runs as the user who ran `install`, with that user's
interactive token and the least privilege, with no time limit, and it is not stopped or kept from starting when the
computer runs on a battery. `install` loads the config file as `run` does at its start, theme included, and writes
nothing and makes no call when `run` would refuse it, as on Linux, and it opens no screen. The config file is the one
given with `--config`, or else the default one, and the task names the absolute path of the executable and of the config
file, so that it does not depend on the environment of the logon. A path that holds `"`, `%` or a control character is
refused, because it cannot be written safely into the task.

The task is written as the file `%LocalAppData%\sensordeck\startup-task.xml`, which `install` replaces each time and
hands to `schtasks.exe` with `/Create /XML`. The file is also the mark of an installed task for `status` and `uninstall`.
The task is named `sensordeck` for the whole machine, so a second user of the same machine cannot install one too.

The log is `%LocalAppData%\sensordeck\run.log`. It holds the lines that `run` writes to standard output and standard
error, each with the local time in front, since a task has no console to write them to. It is kept to two files of at
most 1 MiB each, `run.log` and the older `run.log.1`.

Running `sensordeck service install` again stops the running `run`, replaces the task and starts it again. That is how to
apply a moved or replaced executable, another `--config`, or a change of `interval`, `commander_interval` or
`commander_color`. Each start of the task, at each logon and at each restart after an error, uploads the theme of the
config, so the values appear about 10 seconds after the start and the screen reboots once. `sensordeck theme use` and
edits of the config file reach the running `run` within a round, by the reload of `run`, and a new theme is uploaded
then.

`sensordeck service status` writes four lines: `installed: yes, <task file>`, `enabled:` with `enabled`, `disabled` or
`unknown`, `active:` with `running`, `stopped` or `unknown`, and `log:` with the path of the log. The task is seen in the
Task Scheduler too. `enabled` is read from the definition of the task, and `active` from whether a background `run` is
running, and not from the text of `schtasks`, which Windows translates.

A `run` that the task starts writes the error of a failed start to the log, as when no screen is found, and starts again
every 10 seconds, with no limit, as the systemd service does on Linux.

`sensordeck service uninstall` stops the running `run` so that the Commander Core XT is back in hardware mode and the
screen keeps its last values, then deletes the task and removes the task file. It ends `run` with `schtasks /End` when it
has not stopped within 30 seconds. The same stop is made by `install` before it replaces the task.

`schtasks /End /TN sensordeck`, the Task Manager, logging off and shutting down end `run` with no teardown, so that the
Commander Core XT may stay in the colour of `commander_color` until `run` starts again. `schtasks /Run /TN sensordeck`
starts the task again.

The task starts `run` with a console, and `run` detaches from it at once, so a console window may flash at the logon
before it closes.

Running `sensordeck run`, or another command that writes to the screen, by hand while the task runs is not supported,
because both write to the screen. Stop the task first with `sensordeck service uninstall`.

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
  - {type: number, font: value, metric: cpu.temp, digits: 3, align: right, x: 70, y: 60}
  - {type: bar, metric: cpu.util, x: 20, y: 130, width: 200, height: 12, color: "#00c8ff"}
  - {type: datetime, font: label, format: "YYYY-MM-DD HH:mm:ss", x: 20, y: 280}
```

A number takes three more keys. `align` is `left`, `center` or `right`, and `center` without the key. `opacity` is a number
from 0 to 1, 1 without the key, but only 0 and 1 were seen on the device. `thousands: true` shows the value in thousands,
`1.2` for 1234. A bar shows its metric's value as a share from 0 (empty) to 100 (full), filled from the left, in its
`color`, with the empty part in its `background`, black without the key. The screen cuts a value that is wider than its
box, and the minus sign of a negative value takes a place in the box like a digit, so `digits` must count it. The
[theme folder document](docs/theme-folder.md) has the rest.

A number takes the key `alarm` to change its colour past a threshold, such as a GPU temperature that turns red:

```yaml
- {type: number, font: value, metric: gpu.temp, x: 176, y: 44, alarm: {above: 80, color: "#ff3030"}}
```

`above` and `below` are whole numbers, and either or both are given. The number shows in the alarm colour when its value is
greater than `above` or less than `below`, so with `above: 80` the value 80 is normal and 81 is the alarm. The value is
compared as it is sent, whole, so a reading of 80.4 is normal and 80.5 is the alarm. An alarm number is left-aligned and
cannot show a negative value or thousands, because it is made of two number widgets of which the one not shown is sent
-1, so it counts as two of the 62 widgets. `sensordeck theme preview` shows its normal colour. The
[theme folder document](docs/theme-folder.md#a-number-with-an-alarm-colour) has the rest.

A bar shows 0 to 100, which `cpu.util`, `gpu.util` and `ram.util` fit as they are. A metric of another range is shown
through a custom metric whose `scale` makes it 0 to 100. This one reads a temperature of a chip of hwmon in
millidegrees, so that 90 °C fills the bar, since 0.001 times 100 over 90 is 0.0011111:

```yaml
metrics:
  board.load: {hwmon: nct6799, input: temp1_input, scale: 0.0011111}
```

with `{type: bar, metric: board.load, x: 20, y: 150, width: 200, height: 12, color: "#ff8000"}` in the theme.

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

The compiler gives each metric of the number and bar widgets a tag and writes the mapping beside the theme file, in
`img.tags.yaml`:

```yaml
# Paste this block into the config file of sensordeck run.
tags:
  cpu.temp: 1
```

A metric with an alarm number gets a normal tag and an alarm tag for each threshold, and the tags file writes them in the
flow style of YAML, one line for the metric, as the key `tags` takes them:

```yaml
# Paste this block into the config file of sensordeck run.
tags:
  cpu.temp: 1
  gpu.temp: {tag: 2, alarms: [{above: 80, normal: 3, alarm: 4}]}
```

That file is for a config file with `tags`. A config file with the key `theme` needs no tags block, because `run` takes
the tags from the compiler itself.

To use a theme, set it with `sensordeck theme use <name-or-folder>`. The command compiles the theme, which fails with
the error of the compilation and changes nothing when the theme is not valid, and then sets the key `theme` of the
config file. It replaces the line of the key, or adds a line at the end of the file, and keeps every other byte of it,
and it creates the file when there is none. It writes the absolute path for a folder. It refuses a config file that holds
`tags`, one that is not valid and one whose root is written in the flow style of YAML, and changes none of them. A
running `sensordeck run` applies the new theme within a round.

`run` uploads the theme of the config at each start. The upload takes about 10 seconds and reboots the screen, and `run`
sends no value and no clock until it is done. It compiles the theme in memory and keeps no compiled file.

* Within one `run` the theme is uploaded again only when a reload gives a compiled theme that differs from the one it
  uploaded, so running `sensordeck theme use` again after you edited a theme folder puts the edit on the screen
* `run` uploads nothing when the screen is plugged in again or restarted after a hang, because the screen keeps its theme
  in its flash
* A theme whose upload failed is tried again after 10 seconds, and then after longer waits, up to 5 minutes
* A theme that other software, such as the vendor's, `sensordeck upload`, the other operating system of a dual-boot
  machine or another computer put on the screen is replaced at the next start of `run`
* A screen that received another theme while it was unplugged from a running `run` keeps it until `run` is started
  again
* Earlier releases kept the file `uploaded-themes.json` in the cache directory, at
  `$XDG_CACHE_HOME/sensordeck/uploaded-themes.json` or `~/.cache/sensordeck/uploaded-themes.json` on Linux and at
  `%LocalAppData%\sensordeck\uploaded-themes.json` on Windows. It is no longer used and may be deleted

The folder format, the tags and the limits are described in [docs/theme-folder.md](docs/theme-folder.md).

## Reading a theme file

`sensordeck theme dump <file.dat> <out-dir>` reads a theme file, the `img.dat` that `upload` sends to the screen, and
writes what is in it into the directory, which it creates when it does not exist:

* `theme.json`, a description of the file: the header and every record with its fields. A field whose meaning is a
  guess has a name that ends in `_guess`, and a byte whose meaning is unknown is named `byte_<n>_unknown`
* a PNG of each resource, named after the slot of its record, such as `slot03-image-frame1.png`
* `preview.png`, the 480 x 320 screen composed from the records alone

The preview shows every number as 42 and the date and time as 2026-11-15 09:41:07, draws each number as its record aligns it,
clipped to its box, and each bar filled to 42. The command touches no
device and works with no screen connected, on Linux and on Windows. It prints each warning of the reader, such as a
record of a type it does not know, to standard error. The format is described in
[docs/theme-format.md](docs/theme-format.md).

## Troubleshooting

### The screen shows its theme but no values after boot

The screen shows 12:00:00 and 2022-01-01, and the log of `run` has `screen hung`. The screen hung because the firmware
of the PC set it up before Linux, see [Hangs of the firmware](docs/protocol.md#hangs-of-the-firmware). With the current
udev rule installed, see [Linux: access to the screen](#linux-access-to-the-screen), `run` restarts it by itself. The
line `the screen did not restart, trying again every minute` says the restart did not make the screen leave the bus, and
`the screen is still hung after its restart` says it left and the writes time out still. In both `run` tries again
every minute, and unplugging the screen once and plugging it in again recovers it now.

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
