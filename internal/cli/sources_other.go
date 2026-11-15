//go:build !linux && !windows

package cli

func newSources(sysfsRoot, devRoot, procRoot string) (sourceSet, error) {
	return sourceSet{}, unsupported()
}
