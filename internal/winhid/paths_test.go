package winhid

import "testing"

func TestInterfaceZero(test *testing.T) {
	for path, want := range map[string]bool{
		`\\?\hid#vid_1b1c&pid_0c2a&mi_00#7&1&0&0000#{guid}`:       true,
		`\\?\HID#VID_1B1C&PID_0C2A&MI_00#7&1&0&0000#{guid}`:       true,
		`\\?\hid#vid_1b1c&pid_0c2a&mi_00&col01#7&1&0&0000#{guid}`: true,
		`\\?\hid#vid_1b1c&pid_0c2a&mi_01#7&1&0&0000#{guid}`:       false,
		`\\?\hid#vid_1b1c&pid_0c2a&mi_000#7&1&0&0000#{guid}`:      false,
		`\\?\hid#vid_1b1c&pid_0c2a&xmi_00#7&1&0&0000#{guid}`:      false,
		`\\?\hid#vid_0483&pid_0065#7&1&0&0000#{guid}`:             false,
		`\\?\hid#vid_1b1c&pid_0c2a#mi_00#{guid}`:                  false,
		`\\?\hid#vid_1b1c&pid_0c2a&mi_00`:                         false,
		`mi_00`:                                                   false,
		``:                                                        false,
	} {
		if got := InterfaceZero(path); got != want {
			test.Errorf("InterfaceZero(%q) = %v, want %v", path, got, want)
		}
	}
}
