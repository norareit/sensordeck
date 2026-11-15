//go:build !linux

package cli

func newSources(sysfsRoot, devRoot, procRoot string) (sourceSet, error) {
	return sourceSet{}, unsupported()
}
