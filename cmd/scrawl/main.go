// scrawl takes a screenshot of a screen area with annotations.
//
// The first launch (or with -setup) shows GNOME dialogs, one per monitor;
// after that the saved restore_token values make questions unnecessary.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"gioui.org/app"

	"github.com/jexme/scrawl/internal/autostart"
	"github.com/jexme/scrawl/internal/editor"
	"github.com/jexme/scrawl/internal/fonts"
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
	"github.com/jexme/scrawl/internal/settingsui"
	"github.com/jexme/scrawl/internal/singleton"
	"github.com/jexme/scrawl/internal/tray"
	"github.com/jexme/scrawl/internal/update"
	"github.com/jexme/scrawl/internal/version"
	"github.com/jexme/scrawl/internal/x11capture"
)

func main() {
	i18n.SetLanguage(settings.Load().Language) // before the flags: they have descriptions too
	fonts.Preload(i18n.Language())             // the font for CJK glyphs loads while the capture is in progress
	setup := flag.Bool("setup", false, i18n.T("pick the monitors again (shows GNOME dialogs)"))
	verbose := flag.Bool("v", false, i18n.T("print timings"))
	fresh := flag.Bool("fresh", false, i18n.T("do not offer the area of the previous screenshot"))
	openSettings := flag.Bool("settings", false, i18n.T("open the settings window"))
	grant := flag.Bool("grant-screenshot", false, i18n.T("allow the app the fallback capture through the Screenshot portal and exit"))
	exitAfter := flag.Duration("exit-after", 0, i18n.T("exit normally after the given time (for testing)"))
	trayMode := flag.Bool("tray", false, i18n.T("keep an icon in the notification area (for starting at login)"))
	autostartRun := flag.Bool("autostart", false, i18n.T("set by the start-at-login entry: stay silent if the tray is already running"))
	flag.Bool("shot", false, i18n.T("take one screenshot (what the tray menu and its hotkey run)"))
	showVersion := flag.Bool("version", false, i18n.T("show the version and exit"))
	flag.Parse()

	if *showVersion {
		fmt.Println("scrawl", version.String())
		return
	}

	// On Windows a bare start (double click) runs the tray; if one is already running, the bare
	// start falls through and takes a screenshot, like on Linux.
	bare := bareStartIsTray && flag.NFlag() == 0 && flag.NArg() == 0
	if *trayMode || bare {
		if running := runTray(*exitAfter, bare, *autostartRun); !running || *trayMode {
			return
		}
	}

	if *grant && x11capture.IsSession() {
		fmt.Println(i18n.T("in an X11 session no permission is needed: capture goes straight to X11"))
		return
	}
	if *grant {
		if err := screencast.GrantScreenshotPermission(); err != nil {
			fatal(i18n.T("could not grant the permission: %v"), err)
		}
		fmt.Println(i18n.T("Screenshot permission granted to the app"), screencast.AppID)
		return
	}

	if *openSettings {
		// Its own lock: the settings window does not block the overlay and vice versa, but a
		// second settings window is not opened.
		if ok, err := singleton.Acquire("scrawl-settings"); err == nil && !ok {
			return
		}
		settingsui.Run(settings.NewStore())
		return
	}

	// Pressing the hotkey again must not open a second overlay over the first
	// (and send parallel requests to the portal).
	if ok, err := singleton.Acquire("scrawl"); err == nil && !ok {
		return
	}

	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		// Screen capture, the clipboard and the hotkey are not written for this system yet.
		fatal("%s", i18n.T("screen capture is only available on Linux (GNOME, Wayland) and Windows for now; the settings window works: scrawl -settings"))
	}

	var (
		frames []screencast.Frame
		t0     time.Time
	)
	if runtime.GOOS == "windows" {
		frames, t0 = grabWindows(*verbose)
	} else if x11capture.IsSession() {
		frames, t0 = grabX11(*setup, *verbose)
	} else {
		frames, t0 = grabPortal(*setup, *verbose)
	}

	finish := func() { os.Exit(0) }
	closeEditor := editor.Run(editor.Config{
		Frames:  frames,
		Finish:  finish,
		T0:      t0,
		Verbose: *verbose,
		Fresh:   *fresh,
	})
	if *exitAfter > 0 {
		time.AfterFunc(*exitAfter, closeEditor)
	}
	app.Main()
}

// grabX11 captures the screen directly from X11: no portal, dialogs or tokens.
func grabX11(setup, verbose bool) ([]screencast.Frame, time.Time) {
	if setup {
		fmt.Fprintln(os.Stderr, i18n.T("in an X11 session there is no need to pick monitors: capture goes straight to X11"))
	}
	t0 := time.Now() // the moment of the hotkey press
	frames, err := x11capture.GrabAll()
	if err != nil {
		fatal(i18n.T("frame: %v"), err)
	}
	if verbose {
		fmt.Printf("frames (%d) received in %s: X11\n", len(frames), time.Since(t0).Round(time.Millisecond))
	}
	return frames, t0
}

// grabPortal captures the screen through the ScreenCast portal (Wayland): first it starts
// sessions from the saved tokens, then takes one frame from each monitor.
func grabPortal(setup, verbose bool) ([]screencast.Frame, time.Time) {
	p, err := screencast.New()
	if err != nil {
		fatal("portal: %v", err)
	}
	if p.OldPortal {
		// An old portal identifies the application by its systemd scope: without it tokens do not work.
		if err := p.EnterAppScope(); err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("the portal did not accept the app_id: restore_token will not work, dialogs will appear on every run"))
		}
	} else if !p.Registered {
		fmt.Fprintln(os.Stderr, i18n.T("the portal did not accept the app_id: restore_token will not work, dialogs will appear on every run"))
	}

	p.IncludeCursor = settings.Load().IncludeCursor

	var tokens []string
	if !setup {
		tokens = screencast.LoadTokens()
	}
	// Without registration the portal will not accept the saved tokens and will show dialogs
	// nobody expects when launched by hotkey. Most often this is a hung portal.
	// A portal older than 1.21 has no Registry at all: that is not a hang, it accepts tokens
	// by the app_id from the systemd scope, so we carry on.
	if !p.Registered && !p.OldPortal && len(tokens) > 0 {
		fatal("%s", i18n.T("xdg-desktop-portal is not responding. Restart it: systemctl --user restart xdg-desktop-portal"))
	}

	// Starting the sessions is not part of the measurement: in the future a daemon will hold them.
	tSessions := time.Now()
	sessions, newTokens, err := p.OpenAll(tokens, func(i, n int, again bool, left []screencast.MonitorInfo) {
		msg := screencast.SetupMessage(i, n, again, left)
		fmt.Println(">>>", msg)
		screencast.SetupNotify(msg)
	})
	if err != nil {
		fatal(i18n.T("sessions: %v"), err)
	}
	if len(newTokens) > 0 {
		if err := screencast.SaveTokens(newTokens); err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("restore_token not saved: %v\n"), err)
		}
	}
	if setup && len(newTokens) < len(sessions) {
		fmt.Fprintln(os.Stderr, i18n.T("the portal did not issue a restore_token: tick \"Remember this choice\" in the dialog, otherwise dialogs will appear on every run"))
	}
	if n, ok := p.MonitorCount(); ok && n > len(sessions) {
		fmt.Fprintf(os.Stderr, i18n.T("monitors connected: %d, in sessions: %d. Run with -setup to add the rest\n"), n, len(sessions))
	}
	if verbose {
		fmt.Printf("sessions (%d) ready in %s\n", len(sessions), time.Since(tSessions).Round(time.Millisecond))
	}

	t0 := time.Now() // the moment of the hotkey press
	// Frames usually arrive within ~100 ms; if not, the screen is static and PipeWire is silent.
	frames, fallback, err := p.GrabAll(sessions, 250*time.Millisecond)
	if err != nil {
		fatal(i18n.T("frame: %v"), err)
	}
	// The frames are received: the capture sessions are no longer needed. While they are open,
	// GNOME has to composite the screen instead of direct output (this hurts FPS in games).
	for _, s := range sessions {
		s.Close()
	}
	if verbose {
		how := "PipeWire"
		if fallback {
			how = i18n.T("fallback path, Screenshot portal")
		}
		fmt.Printf("frames (%d) received in %s: %s\n", len(frames), time.Since(t0).Round(time.Millisecond), how)
	}

	return frames, t0
}

// fatal prints the error and shows a notification: when launched by hotkey there is no
// terminal, and without a notification the failure goes unnoticed.
func fatal(f string, a ...any) {
	msg := fmt.Sprintf(f, a...)
	fmt.Fprintln(os.Stderr, msg)
	notify(i18n.T("Could not take the screenshot"), msg)
	os.Exit(1)
}

// runTray keeps the tray icon in the notification area: screenshots and settings run as
// separate processes, so the icon stays light and does not hold the screen capture.
// It returns true without doing anything if another tray is already running (and tells the
// user so, unless this is a bare or autostart launch, which must stay quiet).
func runTray(exitAfter time.Duration, bare, autostartLaunch bool) (alreadyRunning bool) {
	if ok, err := singleton.Acquire("scrawl-tray"); err == nil && !ok {
		if !bare && !autostartLaunch {
			showAlreadyRunning(runningTrayVersion())
		}
		return true
	}
	exe, err := os.Executable()
	if err != nil {
		fatal("%v", err)
	}
	recordTrayVersion(version.String())
	// Self-healing autostart: if it is enabled but points elsewhere (the exe was moved or
	// updated in place), it is rewritten to this exe. A disabled autostart stays disabled.
	if _, err := autostart.Heal(exe); err != nil {
		fmt.Fprintln(os.Stderr, "autostart:", err)
	}
	spawn := func(args ...string) {
		cmd := exec.Command(exe, args...)
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		go cmd.Wait() // so as not to leave a zombie process
	}
	var stop chan struct{}
	if exitAfter > 0 {
		stop = make(chan struct{})
		time.AfterFunc(exitAfter, func() { close(stop) })
	}
	if settings.Load().CheckUpdates {
		go watchUpdates()
	}
	err = tray.Run(tray.Handlers{
		Shot:     func() { spawn("-shot") },
		Settings: func() { spawn("-settings") },
		Version:  version.String(),
		Welcome:  firstRunHint(),
	}, stop)
	if err != nil {
		fatal("%s: %v", i18n.T("tray icon"), err)
	}
	return false
}

// watchUpdates checks for a new version when the tray icon starts and then once a day;
// it announces each new version once. Works only if the check is enabled in the settings.
func watchUpdates() {
	for {
		if res, err := update.Check(context.Background(), version.String()); err == nil && res.Newer && !update.AlreadyNotified(res.Latest) {
			notify(i18n.T("A new version is available"), res.Message())
			update.MarkNotified(res.Latest)
		}
		time.Sleep(24 * time.Hour)
		if !settings.Load().CheckUpdates {
			return // turned off in the settings
		}
	}
}
