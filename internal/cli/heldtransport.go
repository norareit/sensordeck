package cli

// heldTransport is a transport whose screen is locked by the process from before its opening. Its Close ends the lock
// after the close of the screen, so that a Reopen, which closes and opens the screen inside the inner transport,
// keeps the lock while the screen is lost.
type heldTransport struct {
	transport
	release  func()
	released bool
}

func (held *heldTransport) Close() error {
	if held.released {
		return nil
	}
	held.released = true
	err := held.transport.Close()
	held.release()
	return err
}
