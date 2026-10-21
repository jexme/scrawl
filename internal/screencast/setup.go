package screencast

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/jexme/scrawl/internal/i18n"
)

// where describes the position of the single remaining monitor in words.
func where(m MonitorInfo) string {
	if m.Primary {
		return i18n.T("primary")
	}
	if m.X == 0 {
		return i18n.T("left")
	}
	return i18n.T("right")
}

// SetupMessage explains what to choose in the next GNOME dialog when setting up monitors:
// i is the dialog number from zero, n is their count, again means the previous choice
// repeated an already chosen monitor, left is the monitors not yet chosen.
func SetupMessage(i, n int, again bool, left []MonitorInfo) string {
	msg := fmt.Sprintf(i18n.T("Dialog %d of %d: "), i+1, n)
	if again {
		msg = i18n.T("This monitor is already chosen. ") + msg
	}
	if len(left) == 1 {
		msg += fmt.Sprintf(i18n.T("choose the monitor \"%s\" (%s)"), left[0].Name, where(left[0]))
	} else {
		msg += i18n.T("choose a monitor you have not chosen yet")
	}
	return msg + i18n.T(", keep \"Remember this choice\" ticked and press \"Share\"")
}

// SetupNotify shows a notification about the monitor setup: GNOME dialogs
// appear on top of everything, and the user needs a hint about what to choose in them.
func SetupNotify(msg string) {
	exec.Command("notify-send", "-a", "Scrawl", i18n.T("Monitor setup"), msg).Run()
}

// SetupMonitors re-selects the monitors: it shows GNOME dialogs, one per
// monitor, and saves the restore tokens. progress receives a hint for
// each dialog. It returns a summary to show the user; there is an error if
// the choice could not be saved.
func SetupMonitors(progress func(string)) (string, error) {
	p, err := New()
	if err != nil {
		return "", fmt.Errorf("portal: %w", err)
	}
	sessions, tokens, err := p.OpenAll(nil, func(i, n int, again bool, left []MonitorInfo) {
		msg := SetupMessage(i, n, again, left)
		progress(msg)
		SetupNotify(msg)
	})
	if err != nil {
		return "", fmt.Errorf(i18n.T("sessions: %w"), err)
	}
	for _, s := range sessions {
		s.Close()
	}
	if len(tokens) < len(sessions) {
		return "", errors.New(i18n.T("the portal did not issue a restore token: tick \"Remember this choice\" in the dialog, otherwise dialogs will appear on every run"))
	}
	if err := SaveTokens(tokens); err != nil {
		return "", fmt.Errorf(i18n.T("tokens not saved: %w"), err)
	}
	msg := fmt.Sprintf(i18n.T("Monitors set up: %d"), len(sessions))
	if n, ok := p.MonitorCount(); ok && n > len(sessions) {
		msg += fmt.Sprintf(i18n.T(" of %d connected, choose the rest again"), n)
	}
	return msg, nil
}
