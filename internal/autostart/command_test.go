package autostart

import "testing"

func TestRunCommand(t *testing.T) {
	got := runCommand(`C:\Program Files\Scrawl\scrawl.exe`)
	if want := `"C:\Program Files\Scrawl\scrawl.exe" -tray -autostart`; got != want {
		t.Errorf("runCommand = %q, want %q", got, want)
	}
}

func TestSameCommand(t *testing.T) {
	const cmd = `"C:\Users\Me\AppData\Local\Programs\Scrawl\scrawl.exe" -tray -autostart`
	cases := []struct {
		a, b string
		want bool
	}{
		{cmd, cmd, true},
		{cmd, `"c:\users\me\appdata\local\programs\scrawl\SCRAWL.EXE" -tray -autostart`, true}, // case
		{cmd, `"C:/Users/Me/AppData/Local/Programs/Scrawl/scrawl.exe"   -tray   -autostart`, true},
		{cmd, `"C:\Users\Me\Downloads\scrawl.exe" -tray -autostart`, false},                               // moved
		{cmd, `"C:\Users\Me\AppData\Local\Programs\Scrawl\scrawl.exe" -tray`, false},                      // older flags
		{cmd, `"C:\Users\Me\AppData\Local\Programs\Scrawl\scrawl.exe" -tray -autostart -settings`, false}, // extra
		{"", cmd, false},
	}
	for _, c := range cases {
		if got := sameCommand(c.a, c.b); got != c.want {
			t.Errorf("sameCommand(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestInTempDir(t *testing.T) {
	temp := `C:\Users\Me\AppData\Local\Temp`
	cases := []struct {
		path string
		want bool
	}{
		{`C:\Users\Me\AppData\Local\Temp\Temp1_scrawl.zip\scrawl.exe`, true}, // opened from a zip in Explorer
		{`c:\users\me\appdata\local\temp\scrawl.exe`, true},                  // case
		{`C:/Users/Me/AppData/Local/Temp/scrawl.exe`, true},                  // forward slashes
		{`C:\Users\ME~1\AppData\Local\Temp\x\scrawl.exe`, true},              // 8.3 name, found by the layout
		{`D:\Scratch\scrawl.exe`, false},
		{`C:\Users\Me\AppData\Local\Programs\Scrawl\scrawl.exe`, false},
		{`C:\Users\Me\AppData\Local\Temporary\scrawl.exe`, false}, // a sibling with the same prefix
		{`C:\Users\Me\AppData\Local\Temp2\scrawl.exe`, false},
		{`C:\Windows\Temp\scrawl.exe`, true},
		{`C:\Users\Me\scoop\apps\scrawl\current\scrawl.exe`, false},
	}
	for _, c := range cases {
		if got := inTempDir(c.path, temp); got != c.want {
			t.Errorf("inTempDir(%q) = %v, want %v", c.path, got, c.want)
		}
	}
	// A custom %TEMP% outside the usual layout is honoured too.
	if !inTempDir(`D:\Scratch\x\scrawl.exe`, `D:\Scratch`) {
		t.Error("the directory passed by the caller must count")
	}
	// An empty directory must not turn everything into a temporary path.
	if inTempDir(`C:\Apps\scrawl.exe`, "") {
		t.Error("an empty temp dir must be ignored")
	}
}

func TestNeedsRepair(t *testing.T) {
	temp := `C:\Users\Me\AppData\Local\Temp`
	const bin = `C:\Users\Me\AppData\Local\Programs\Scrawl\scrawl.exe`
	cur := runCommand(bin)
	cases := []struct {
		name    string
		current string
		exists  bool
		bin     string
		want    bool
	}{
		{"already points here", cur, true, bin, false},
		{"same path, other case and slashes", `"c:/users/me/appdata/local/programs/scrawl/scrawl.exe" -tray -autostart`, true, bin, false},
		{"moved", runCommand(`C:\Users\Me\Downloads\scrawl.exe`), true, bin, true},
		{"old command without -autostart", `"` + bin + `" -tray`, true, bin, true},
		{"autostart is off: stays off", "", false, bin, false},
		{"never write a temp path", runCommand(`C:\Old\scrawl.exe`), true, `C:\Users\Me\AppData\Local\Temp\Temp1_s.zip\scrawl.exe`, false},
	}
	for _, c := range cases {
		if got := needsRepair(c.current, c.exists, c.bin, temp); got != c.want {
			t.Errorf("%s: needsRepair = %v, want %v", c.name, got, c.want)
		}
	}
}
