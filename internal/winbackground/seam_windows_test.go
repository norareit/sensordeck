package winbackground

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestTheEventIsCreatedFoundSignalledAndClosed(test *testing.T) {
	name := fmt.Sprintf(`Local\sensordeck-test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	event, err := Create(name)
	if err != nil {
		test.Fatal(err)
	}
	if _, err := Create(name); !errors.Is(err, ErrExists) {
		test.Errorf("a second creation gave %v", err)
	}
	if found, err := Exists(name); err != nil || !found {
		test.Errorf("exists: %v, %v", found, err)
	}
	select {
	case <-event.Signalled():
		test.Fatal("the event was signalled before Signal")
	default:
	}
	if signalled, err := Signal(name); err != nil || !signalled {
		test.Fatalf("signal: %v, %v", signalled, err)
	}
	select {
	case <-event.Signalled():
	case <-time.After(5 * time.Second):
		test.Fatal("the waiter did not end within 5 seconds")
	}
	if err := event.Close(); err != nil {
		test.Fatal(err)
	}
	if found, err := Exists(name); err != nil || found {
		test.Errorf("exists after Close: %v, %v", found, err)
	}
	if signalled, err := Signal(name); err != nil || signalled {
		test.Errorf("signal after Close: %v, %v", signalled, err)
	}
}

func TestCloseEndsTheWaiterOfAnEventThatWasNotSignalled(test *testing.T) {
	event, err := Create(fmt.Sprintf(`Local\sensordeck-test-close-%d-%d`, os.Getpid(), time.Now().UnixNano()))
	if err != nil {
		test.Fatal(err)
	}
	if err := event.Close(); err != nil {
		test.Fatal(err)
	}
	select {
	case <-event.Signalled():
		test.Error("a closed event reported a signal")
	default:
	}
}
