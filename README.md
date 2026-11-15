# sensordeck

Turn a cheap USB "sensor panel" screen into a hardware-monitoring dashboard on **Linux and Windows**.

> **Status: early development.** The screen can be driven on Linux: with the low-level commands (`devices`, `clock`,
> `values`, `upload`, `listen` and `raw`), with `sensordeck sensors` and with `sensordeck run`, which also reads the temperature probes of a Corsair iCUE
> Commander Core XT. `sensordeck theme dump` reads a theme file with no screen. Making themes, Windows and the
> service are still to come. This README describes what sensordeck is meant to become.

## What it will do

Many small PC screens are sold as "Windows only" and depend on the vendor's own software. sensordeck aims to be an
open-source replacement that also works on Linux:

- Show live hardware readings: CPU and GPU temperature and load, VRAM and RAM usage, NVMe temperatures, motherboard
  sensors and a clock.
- Use themes that are plain folders (a `theme.yaml` layout, images and any TTF font). sensordeck compiles a theme into
  the device's format and uploads it. It uploads again only when the theme changes.
- Let you preview a theme as a PNG on your PC, so you can design without flashing the device each time.
- Run in the background as a service (systemd on Linux, a startup task on Windows).
- Ship as a single executable with nothing else to install. Built-in themes are included.

## Supported devices

Version 1 targets a single device:

| | |
|---|---|
| Sold as | "3.5 inch PC CPU Data Monitor", "Windows only", under various Amazon/AliExpress brands |
| Vendor name | SmartMonitor (llhmi.com / llTechCo.,Ltd) |
| USB ID | `0483:0065` |
| Screen | 3.5", 480×320 |

The architecture is generic, so other screens can be added later.

## Planned usage

```bash
sensordeck devices                # list connected screens
sensordeck run                    # send live sensor values to the screen
sensordeck theme list             # show built-in and installed themes
sensordeck theme use minimal-dark # switch themes
sensordeck theme preview my-theme # render a theme to PNG with sample values
sensordeck service install        # run in the background at startup
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

The metrics are:

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

`sensordeck run` sends the metrics named in a config file to the screen. The tags are those of the theme on the
screen, so for a vendor theme only you know which tag shows what. The config file is `sensordeck/config.yaml` in your
config directory (`$XDG_CONFIG_HOME`, or `~/.config`), and `--config <path>` reads another file. This one is for a vendor
theme in which tag 1 shows the CPU temperature and tag 2 the CPU speed:

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
minute, and it keeps running when the screen is unplugged and plugged in again.

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

## Credits

Thanks to the [turing-smart-screen-python-HIDdev2](https://github.com/Agentry433/turing-smart-screen-python-HIDdev2)
project for discovering the protocol these screens use.

## License

MIT
