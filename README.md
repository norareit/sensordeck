# sensordeck

Turn a cheap USB "sensor panel" screen into a hardware-monitoring dashboard on **Linux and Windows**.

> **Status: early development.** Nothing is usable yet. This README describes what sensordeck is meant to become.

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

hidraw is provided by the kernel's `usbhid` driver, so that driver must stay bound to the screen. A rule that unbinds
`usbhid` for this device, as some guides for other tools add, has to be removed.

## Credits

Thanks to the [turing-smart-screen-python-HIDdev2](https://github.com/Agentry433/turing-smart-screen-python-HIDdev2)
project for discovering the protocol these screens use.

## License

MIT
