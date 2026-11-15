// Package winbackground holds the calls into Windows that a background run of sensordeck makes: the named event by which
// the service commands stop a running background run and tell whether one runs, and the detach from the console. Only
// the seam file is limited to Windows, so that the package exists on every system.
package winbackground
