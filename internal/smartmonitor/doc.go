// Package smartmonitor speaks the wire protocol of the SmartMonitor screen with the USB ID 0483:0065, which is
// described in docs/protocol.md. It encodes reports and drives a theme upload over a Transport, and does no I/O of
// its own.
package smartmonitor
