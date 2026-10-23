//go:build !windows

package windowerr

import "os/exec"

// show uses notify-send like the rest of the program; without it nothing is shown, and the
// message stays on stderr.
func show(title, msg string) {
	exec.Command("notify-send", "-a", "Scrawl", title, msg).Run()
}
