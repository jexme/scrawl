// Package version хранит версию сборки. Makefile подставляет её через -ldflags
// из `git describe`; без этого берётся ревизия из данных сборки Go.
package version

import "runtime/debug"

// Version задаётся при сборке: -ldflags "-X .../internal/version.Version=v0.1.0".
var Version = ""

// Home — страница проекта для раздела «О программе».
const Home = "https://github.com/jexme/scrawl"

// String возвращает версию сборки: заданную явно, иначе по ревизии git, иначе "dev".
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
