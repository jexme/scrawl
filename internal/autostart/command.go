package autostart

import "strings"

// This file holds the pure string logic behind the Windows registry autostart. It has no build
// tag so that the unit tests run on every OS; the registry code itself is in reg_windows.go.

// runCommand is the command line stored in the HKCU Run value for the program bin. The
// -autostart flag tells the tray that this is the login start, which must stay silent when
// a tray is already running.
func runCommand(bin string) string {
	return `"` + bin + `" -tray -autostart`
}

// normalizeCommand reduces a command line to a form in which two spellings of the same
// command compare equal: Windows paths are case-insensitive, / and \ are interchangeable
// and the amount of white space between the arguments does not matter.
func normalizeCommand(cmd string) string {
	cmd = strings.ReplaceAll(cmd, "/", `\`)
	return strings.ToLower(strings.Join(strings.Fields(cmd), " "))
}

// sameCommand reports whether two stored commands start the same program the same way.
func sameCommand(a, b string) bool {
	return normalizeCommand(a) == normalizeCommand(b)
}

// inTempDir reports whether path lies inside one of the temporary directories. A program
// started from there (a zip opened in Explorer extracts to %TEMP%\Temp1_name.zip\) disappears
// soon, so an autostart entry pointing at it would be dead after the next cleanup.
//
// The directories passed by the caller are usually %TEMP%/%TMP%; the well-known layouts are
// checked as well because %TEMP% may be spelled in the 8.3 short form
// (C:\Users\JOHNSM~1\AppData\Local\Temp) while the program path is long, or the other way
// round.
func inTempDir(path string, tempDirs ...string) bool {
	p := dirForm(path)
	for _, d := range tempDirs {
		if d = dirForm(d); d != `\` && strings.HasPrefix(p, d) {
			return true
		}
	}
	for _, marker := range []string{`\appdata\local\temp\`, `\windows\temp\`} {
		if strings.Contains(p, marker) {
			return true
		}
	}
	return false
}

// dirForm lowercases a path, uses backslashes and ends it with one, so that a directory
// prefix test cannot match a sibling (C:\Temp2 is not inside C:\Temp).
func dirForm(path string) string {
	p := strings.ToLower(strings.ReplaceAll(path, "/", `\`))
	if !strings.HasSuffix(p, `\`) {
		p += `\`
	}
	return p
}

// needsRepair decides whether the autostart entry has to be rewritten for the program bin.
// current is the stored value and exists tells whether the value is there at all: an
// autostart that is off stays off, an entry that already starts bin is left alone, and a
// program in a temporary directory is never written.
func needsRepair(current string, exists bool, bin string, tempDirs ...string) bool {
	if !exists || inTempDir(bin, tempDirs...) {
		return false
	}
	return !sameCommand(current, runCommand(bin))
}
