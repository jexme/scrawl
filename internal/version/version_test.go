package version

import (
	"runtime/debug"
	"testing"
)

func TestStringPrefersExplicitVersion(t *testing.T) {
	old := Version
	defer func() { Version = old }()
	Version = "v1.2.3"
	if got := String(); got != "v1.2.3" {
		t.Errorf("задана версия v1.2.3, получено %q", got)
	}
}

func TestStringNeverEmpty(t *testing.T) {
	old := Version
	defer func() { Version = old }()
	Version = ""
	if String() == "" {
		t.Error("версия не должна быть пустой")
	}
}

func TestFromBuildInfo(t *testing.T) {
	cases := []struct {
		name     string
		settings []debug.BuildSetting
		want     string
	}{
		{"без git", nil, "dev"},
		{"ревизия", []debug.BuildSetting{{Key: "vcs.revision", Value: "1bff2271234567"}}, "dev-1bff227"},
		{"с изменениями", []debug.BuildSetting{
			{Key: "vcs.revision", Value: "1bff227"},
			{Key: "vcs.modified", Value: "true"},
		}, "dev-1bff227-dirty"},
	}
	for _, c := range cases {
		if got := fromBuildInfo(&debug.BuildInfo{Settings: c.settings}); got != c.want {
			t.Errorf("%s: ожидалось %q, получено %q", c.name, c.want, got)
		}
	}
}
