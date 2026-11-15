package fakehid_test

import (
	"errors"
	"testing"

	"github.com/norareit/sensordeck/internal/fakehid"
	"github.com/norareit/sensordeck/internal/winhid"
)

func newFake() *fakehid.Fake {
	return fakehid.New(fakehid.Collection{
		Path: "a", Vendor: 1, Product: 2, Name: "name", Serial: "serial", Caps: winhid.Caps{InputLength: 5, OutputLength: 5},
		Input:   [][]byte{{0, 1, 2, 3, 4}},
		Respond: func(written []byte) []byte { return []byte{0, written[1], 0, 0, 0} },
	}, fakehid.Collection{Path: "b", OpenError: errors.New("denied")})
}

func TestTheFakeListsAndOpensAndRecords(test *testing.T) {
	fake := newFake()
	paths, _ := fake.Paths()
	if len(paths) != 2 || paths[0] != "a" || paths[1] != "b" {
		test.Fatalf("paths %v", paths)
	}
	if _, err := fake.OpenReadWrite("b"); err == nil || err.Error() != "denied" {
		test.Fatalf("err %v", err)
	}
	handle, err := fake.OpenReadWrite("a")
	if err != nil {
		test.Fatal(err)
	}
	if fake.AllClosed() {
		test.Error("a handle is open")
	}
	if vendor, product, _ := fake.Attributes(handle); vendor != 1 || product != 2 {
		test.Errorf("attributes %d %d", vendor, product)
	}
	_ = fake.Close(handle)
	if !fake.AllClosed() {
		test.Error("every handle is closed")
	}
	want := []fakehid.Opening{{Path: "b", ReadWrite: true}, {Path: "a", ReadWrite: true}}
	got := fake.Openings()
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		test.Errorf("openings %v", got)
	}
}

func TestTheFakeTracksPendingOperationsAndTheirEndings(test *testing.T) {
	fake := newFake()
	handle, _ := fake.OpenReadWrite("a")
	buffer := make([]byte, 5)
	fake.SetEndings(fakehid.Ending{}, fakehid.Ending{WaitsOut: true, Code: 995, Count: fakehid.Bytes(0)},
		fakehid.Ending{StartError: errors.New("no")})
	operation, _ := fake.StartRead(handle, buffer)
	if !fake.Pending() || buffer[2] != 2 {
		test.Fatal("the read is pending and has filled its buffer")
	}
	if !operation.Wait(10) {
		test.Error("the first operation completes in time")
	}
	if count, code := operation.Reap(); count != 5 || code != 0 || fake.Pending() {
		test.Errorf("reap %d %d pending %v", count, code, fake.Pending())
	}
	operation, _ = fake.StartWrite(handle, []byte{0, 7, 0, 0, 0})
	if operation.Wait(20) {
		test.Error("the second operation waits out")
	}
	operation.Cancel()
	if count, code := operation.Reap(); count != 0 || code != 995 || fake.Cancels() != 1 || fake.ReapedWithoutCancel() != 0 {
		test.Errorf("reap %d %d", count, code)
	}
	if _, err := fake.StartRead(handle, buffer); err == nil || fake.Pending() {
		test.Error("the third start fails with nothing pending")
	}
	if waits := fake.Waits(); len(waits) != 2 || waits[0] != 10 || waits[1] != 20 {
		test.Errorf("waits %v", waits)
	}
	if writes := fake.Writes(); len(writes) != 1 || writes[0][1] != 7 {
		test.Errorf("writes %v", writes)
	}
}
