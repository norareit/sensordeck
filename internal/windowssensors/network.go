package windowssensors

import (
	"time"

	"github.com/norareit/sensordeck/internal/sampling"
	"github.com/norareit/sensordeck/internal/sensors"
)

// The bits of InterfaceAndOperStatusFlags that decide whether an interface is a network adapter.
const (
	hardwareInterface = 0x01
	filterInterface   = 0x02
	connectorPresent  = 0x04
)

// counts tells whether a row is a physical adapter: one for hardware with a connector, which is not a filter module
// that NDIS stacks on an adapter and that would count the bytes of that adapter twice. Loopback, tunnels and most
// virtual adapters are left out.
func counts(row Interface) bool {
	return row.Flags&hardwareInterface != 0 && row.Flags&connectorPresent != 0 && row.Flags&filterInterface == 0
}

// NewNetwork gives the source of net.up and net.down. The interfaces are keyed by their LUID, which does not change
// while an interface exists, and whether a row counts is decided at every sample anew. The moment of a sample is taken
// right after the call that gave it.
func NewNetwork(seam Seam, wait func(time.Duration), now func() time.Time) sensors.Source {
	return &sampling.Network[uint64]{
		Take: func() (map[uint64]sampling.Counters, bool) {
			rows, err := seam.Interfaces()
			if err != nil {
				return nil, false
			}
			counting := map[uint64]sampling.Counters{}
			for _, row := range rows {
				if counts(row) {
					counting[row.LUID] = sampling.Counters{Received: row.InOctets, Transmitted: row.OutOctets}
				}
			}
			return counting, true
		},
		Wait: wait,
		Now:  now,
	}
}
