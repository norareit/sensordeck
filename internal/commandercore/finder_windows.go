package commandercore

import (
	"github.com/norareit/sensordeck/internal/devicelock"
	"github.com/norareit/sensordeck/internal/winhid"
)

// controllerLockName is the lock file of the controller: one lock serves every controller of the user.
const controllerLockName = "controller.lock"

// Windows is the finder of Windows, which calls the HID API of the system.
func Windows() Finder {
	return HID(winhid.Windows(), func() (func(), error) { return devicelock.Acquire(controllerLockName) })
}
