package commandercore

import "github.com/norareit/sensordeck/internal/winhid"

// Windows is the finder of Windows, which calls the HID API of the system.
func Windows() Finder { return HID(winhid.Windows()) }
