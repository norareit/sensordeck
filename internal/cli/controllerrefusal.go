package cli

import (
	"fmt"
	"io"
	"sync"
)

// controllerRefusal keeps the error of the controller that another sensordeck process holds, so that sensors says
// once that it left the probes out.
type controllerRefusal struct {
	mutex sync.Mutex
	err   error
}

func (refusal *controllerRefusal) keep(err error) {
	refusal.mutex.Lock()
	defer refusal.mutex.Unlock()
	refusal.err = err
}

func (refusal *controllerRefusal) say(stderr io.Writer) {
	refusal.mutex.Lock()
	defer refusal.mutex.Unlock()
	if refusal.err != nil {
		_, _ = fmt.Fprintf(stderr, "%v, its probes are left out\n", refusal.err)
	}
}
