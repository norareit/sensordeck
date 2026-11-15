// Package fakehid is a fake of the seam of the Windows HID layer, built from a list of collections. It records what
// was opened, closed, waited for and cancelled, and whether an operation is still pending, so that the tests of the
// layer on any system can check what must never happen: an operation left pending, a handle left open, a device
// opened that should not be.
package fakehid

import (
	"errors"
	"sync"
	"syscall"

	"github.com/norareit/sensordeck/internal/winhid"
)

// Collection is one device interface of the fake, with the error each call about it gives, if any.
type Collection struct {
	Path            string
	Vendor, Product uint16
	Name, Serial    string
	Caps            winhid.Caps

	QueryError, AttributesError, CapsError, ProductError, SerialError, OpenError error

	// Input is the input reports, each with the report ID first, that the reads take in turn. Respond, when set, gives
	// the report that follows a write, which is queued after those of Input.
	Input   [][]byte
	Respond func(written []byte) []byte
}

// Ending says how the next operation ends. The zero value is an operation that completes in time with all its bytes.
type Ending struct {
	// StartError makes the start fail, with nothing pending.
	StartError error
	// WaitsOut makes the wait end without completion. The reap then gives the Code and the Count, as the system does
	// once the operation completed or was cancelled.
	WaitsOut bool
	Code     syscall.Errno
	// Count is the byte count of the completion, and the length of the buffer when nil.
	Count *int
}

// Bytes gives a count for Ending.Count.
func Bytes(count int) *int { return &count }

// Opening is one opening of a path.
type Opening struct {
	Path      string
	ReadWrite bool
}

type handleState struct {
	collection *Collection
	closed     bool
}

// Fake is the seam. It is safe for use from several goroutines.
type Fake struct {
	mutex       sync.Mutex
	collections []Collection
	endings     []Ending
	handles     map[winhid.Handle]*handleState
	next        winhid.Handle
	openings    []Opening
	writes      [][]byte
	waits       []uint32
	cancels     int
	pending     int
	unreaped    int
}

var _ winhid.Seam = (*Fake)(nil)

// New gives a fake with the collections, listed in the order given.
func New(collections ...Collection) *Fake {
	return &Fake{collections: collections, handles: map[winhid.Handle]*handleState{}, next: 100}
}

// SetEndings queues how the next operations end, one for each. Operations after them complete in time.
func (fake *Fake) SetEndings(endings ...Ending) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.endings = endings
}

// SetCollections replaces the collections, as a replug does.
func (fake *Fake) SetCollections(collections ...Collection) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.collections = collections
}

// Openings gives every opening, in order.
func (fake *Fake) Openings() []Opening {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return append([]Opening(nil), fake.openings...)
}

// Writes gives the buffer of every write that was started, in order.
func (fake *Fake) Writes() [][]byte {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return append([][]byte(nil), fake.writes...)
}

// Waits gives the milliseconds of every wait, in order.
func (fake *Fake) Waits() []uint32 {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return append([]uint32(nil), fake.waits...)
}

// Cancels gives how many operations were cancelled.
func (fake *Fake) Cancels() int {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return fake.cancels
}

// Pending tells whether an operation was started and not reaped.
func (fake *Fake) Pending() bool {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return fake.pending > 0
}

// ReapedWithoutCancel counts the operations that were reaped after a wait that ended without completion and with no
// cancel, which on the system would wait for ever.
func (fake *Fake) ReapedWithoutCancel() int {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return fake.unreaped
}

// AllClosed tells whether every handle that was opened is closed.
func (fake *Fake) AllClosed() bool {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	for _, state := range fake.handles {
		if !state.closed {
			return false
		}
	}
	return true
}

func (fake *Fake) Paths() ([]string, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	paths := make([]string, len(fake.collections))
	for position, collection := range fake.collections {
		paths[position] = collection.Path
	}
	return paths, nil
}

var errNoSuchPath = errors.New("no such path")

func (fake *Fake) open(path string, readWrite bool) (winhid.Handle, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	for position, collection := range fake.collections {
		if collection.Path != path {
			continue
		}
		fake.openings = append(fake.openings, Opening{Path: path, ReadWrite: readWrite})
		if readWrite && collection.OpenError != nil {
			return 0, collection.OpenError
		}
		if !readWrite && collection.QueryError != nil {
			return 0, collection.QueryError
		}
		fake.next++
		fake.handles[fake.next] = &handleState{collection: &fake.collections[position]}
		return fake.next, nil
	}
	return 0, errNoSuchPath
}

func (fake *Fake) OpenQuery(path string) (winhid.Handle, error)     { return fake.open(path, false) }
func (fake *Fake) OpenReadWrite(path string) (winhid.Handle, error) { return fake.open(path, true) }

// collectionOf gives the collection behind a handle. It panics for a handle that is closed or unknown, as using one
// is a bug of the code under test.
func (fake *Fake) collectionOf(handle winhid.Handle) *Collection {
	state, known := fake.handles[handle]
	if !known || state.closed {
		panic("fakehid: a call on a handle that is closed or unknown")
	}
	return state.collection
}

func (fake *Fake) Attributes(handle winhid.Handle) (uint16, uint16, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	collection := fake.collectionOf(handle)
	return collection.Vendor, collection.Product, collection.AttributesError
}

func (fake *Fake) Product(handle winhid.Handle) (string, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	collection := fake.collectionOf(handle)
	return collection.Name, collection.ProductError
}

func (fake *Fake) Serial(handle winhid.Handle) (string, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	collection := fake.collectionOf(handle)
	return collection.Serial, collection.SerialError
}

func (fake *Fake) Caps(handle winhid.Handle) (winhid.Caps, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	collection := fake.collectionOf(handle)
	return collection.Caps, collection.CapsError
}

func (fake *Fake) Close(handle winhid.Handle) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.collectionOf(handle)
	fake.handles[handle].closed = true
	return nil
}

func (fake *Fake) nextEnding() Ending {
	if len(fake.endings) == 0 {
		return Ending{}
	}
	ending := fake.endings[0]
	fake.endings = fake.endings[1:]
	return ending
}

func (fake *Fake) StartRead(handle winhid.Handle, buffer []byte) (winhid.Operation, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	collection := fake.collectionOf(handle)
	ending := fake.nextEnding()
	if ending.StartError != nil {
		return nil, ending.StartError
	}
	if len(collection.Input) > 0 {
		copy(buffer, collection.Input[0])
		collection.Input = collection.Input[1:]
	}
	return fake.start(ending, len(buffer)), nil
}

func (fake *Fake) StartWrite(handle winhid.Handle, buffer []byte) (winhid.Operation, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	collection := fake.collectionOf(handle)
	ending := fake.nextEnding()
	if ending.StartError != nil {
		return nil, ending.StartError
	}
	fake.writes = append(fake.writes, append([]byte(nil), buffer...))
	if collection.Respond != nil {
		collection.Input = append(collection.Input, collection.Respond(append([]byte(nil), buffer...)))
	}
	return fake.start(ending, len(buffer)), nil
}

func (fake *Fake) start(ending Ending, length int) *operation {
	fake.pending++
	count := length
	if ending.Count != nil {
		count = *ending.Count
	}
	return &operation{fake: fake, ending: ending, count: uint32(count)}
}

type operation struct {
	fake      *Fake
	ending    Ending
	count     uint32
	cancelled bool
}

func (operation *operation) Wait(milliseconds uint32) bool {
	operation.fake.mutex.Lock()
	defer operation.fake.mutex.Unlock()
	operation.fake.waits = append(operation.fake.waits, milliseconds)
	return !operation.ending.WaitsOut
}

func (operation *operation) Cancel() {
	operation.fake.mutex.Lock()
	defer operation.fake.mutex.Unlock()
	operation.fake.cancels++
	operation.cancelled = true
}

func (operation *operation) Reap() (uint32, syscall.Errno) {
	operation.fake.mutex.Lock()
	defer operation.fake.mutex.Unlock()
	operation.fake.pending--
	if operation.ending.WaitsOut && !operation.cancelled {
		operation.fake.unreaped++
	}
	return operation.count, operation.ending.Code
}
