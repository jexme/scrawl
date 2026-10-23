package settings

import "testing"

// The prefix must be a valid file name part on Windows too, on every OS.
func TestValidateFilePrefixWindowsNames(t *testing.T) {
	for _, bad := range []string{
		"a:b", // an NTFS alternate data stream
		"a<b", "a>b", `a"b`, "a|b", "a?b", "a*b", "a/b", `a\b`,
		"shot.", "shot ", "shot. ", // Windows strips trailing dots and spaces
		".", " ",
	} {
		if ValidateFilePrefix(bad) == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
	for _, ok := range []string{"", "shot", "my shot 1", "a.b-c_d", ".hidden", "a b", "Скрин", "a(1)", "a,b", "a+b", "a@b"} {
		if err := ValidateFilePrefix(ok); err != nil {
			t.Errorf("%q must be accepted: %v", ok, err)
		}
	}
	// Such a prefix in the config file falls back to the default instead of being used.
	sandbox(t)
	writeConfig(t, `{"file_prefix":"a:b","save_dir":"`+xDir+`"}`)
	if got := Load(); got.FilePrefix != "" || got.SaveDir != xDir {
		t.Errorf("a prefix with a colon must be reset: %+v", got)
	}
}

func TestIsHomePath(t *testing.T) {
	cases := []struct {
		dir     string
		windows bool
		want    bool
	}{
		{"~", false, true},
		{"~", true, true},
		{"~/Pictures", false, true},
		{"~/Pictures", true, true},
		{`~\Pictures`, true, true},
		{`~\Pictures`, false, false}, // a backslash is an ordinary character on Linux
		{"~user/x", true, false},
		{`~user\x`, true, false},
		{"Pictures", true, false},
		{"", true, false},
	}
	for _, c := range cases {
		if got := isHomePath(c.dir, c.windows); got != c.want {
			t.Errorf("isHomePath(%q, windows=%v) = %v, want %v", c.dir, c.windows, got, c.want)
		}
	}
}
