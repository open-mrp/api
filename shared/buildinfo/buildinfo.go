// Package buildinfo reports which source a binary was built from.
package buildinfo

import "runtime/debug"

// Release is the release tag the binary was built from (e.g. v4.6.0), stamped by the image build with
// -ldflags "-X github.com/open-mrp/api/shared/buildinfo.Release=<tag>". Empty in local and CI builds.
var Release string

// SourceRef returns the git ref of the binary's source: the release tag, else the clean VCS revision Go embedded,
// else "" when the source can't be pinned (a dev build, or one with uncommitted changes).
func SourceRef() string {
	if Release != "" {
		return Release
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var revision string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				return ""
			}
		}
	}
	return revision
}
