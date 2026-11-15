package cli

import "runtime/debug"

const developmentVersion = "(devel)"

// resolveVersion gives the version to print: the one linked into the executable, else the version of the main module
// that the build information records, which is what go install sets, else (devel).
func resolveVersion(linked string, info *debug.BuildInfo, ok bool) string {
	if linked != "" {
		return linked
	}
	if ok && info != nil && info.Main.Version != "" {
		return info.Main.Version
	}
	return developmentVersion
}
