package windowssensors

import "testing"

func TestBothProceduresOfKernel32Resolve(test *testing.T) {
	for _, procedure := range []interface {
		Find() error
	}{procGlobalMemoryStatusEx, procGetSystemTimes} {
		if err := procedure.Find(); err != nil {
			test.Error(err)
		}
	}
}

func TestTheMemoryOfTheSystemHasATotal(test *testing.T) {
	total, available, err := System().Memory()
	if err != nil || total == 0 {
		test.Errorf("total %d, available %d, %v", total, available, err)
	}
}

func TestTheTimesOfTheSystemHaveATotal(test *testing.T) {
	idle, kernel, user, err := System().SystemTimes()
	if err != nil || kernel+user == 0 {
		test.Errorf("idle %d, kernel %d, user %d, %v", idle, kernel, user, err)
	}
}

func TestTheInterfacesOfTheSystemAreGivenWhateverTheyAre(test *testing.T) {
	if _, err := System().Interfaces(); err != nil {
		test.Error(err)
	}
}
