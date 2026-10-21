// Package version holds the build version. The Makefile sets it through -ldflags from `git
// describe`; without that, the revision from the Go build data is used.
package version

import "runtime/debug"

// Version is set at build time: -ldflags "-X .../internal/version.Version=v0.1.0".
var Version = ""

// Home is the project page for the "About" section.
const Home = "https://github.com/jexme/scrawl"

// String returns the build version: the one set explicitly, otherwise from the git
// revision, otherwise "dev".
func String() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		return fromBuildInfo(info)
	}
	return "dev"
}

func fromBuildInfo(info *debug.BuildInfo) string {
	var rev string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "dev"
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if dirty {
		rev += "-dirty"
	}
	return "dev-" + rev
}
