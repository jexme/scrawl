// Package settingsui shows a separate settings window. Changes are applied
// immediately and written to config.json.
package settingsui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/jexme/scrawl/internal/fonts"
	"github.com/jexme/scrawl/internal/hotkey"
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/screencast"
	"github.com/jexme/scrawl/internal/settings"
	"github.com/jexme/scrawl/internal/version"
	"github.com/jexme/scrawl/internal/windowerr"
)

var (
	bgColor     = color.NRGBA{R: 32, G: 33, B: 36, A: 255}
	fgColor     = color.NRGBA{R: 235, G: 235, B: 235, A: 255}
	mutedColor  = color.NRGBA{R: 160, G: 163, B: 170, A: 255}
	accentColor = color.NRGBA{R: 66, G: 133, B: 244, A: 255}
	errorColor  = color.NRGBA{R: 242, G: 139, B: 130, A: 255}
	borderColor = color.NRGBA{R: 74, G: 75, B: 79, A: 255}
)

// state is the window state: widgets and a reference to the settings.
type state struct {
	th    *material.Theme
	store *settings.Store

	choice    widget.Enum // double click: off, copy, copy_save
	enter     widget.Enum // Enter: copy_save, copy, save
	notify    widget.Bool // notifications after a capture
	remember  widget.Bool // remember the previous area
	showSize  widget.Bool // area size label
	showLoupe widget.Bool // magnifier while selecting and resizing the area
	keepOpen  widget.Bool // do not close the overlay after copying
	cursor    widget.Bool // mouse cursor in the capture
	dir       widget.Editor
	prefix    widget.Editor // start of the file name
	reset     widget.Clickable
	openDir   widget.Clickable
	close     widget.Clickable
	list      widget.List // content scrolling
	sliders   []*slider   // all sliders of the window
	dim       *slider     // dimming outside the area
	bWidth    *slider     // border: thickness
	bDash     *slider     // border: dash
	bGap      *slider     // border: gap
	handle    *slider     // radius of the corner handles
	marker    *slider     // marker opacity

	palEd    [settings.PaletteSize]widget.Editor // panel colors
	palErr   string
	palReset widget.Clickable

	glueLayout widget.Enum      // glue layout: side, stack
	format     widget.Enum      // file format: png, jpeg
	quality    *slider          // JPEG quality
	glueBg     widget.Enum      // glue background: light, dark
	gluePad    *slider          // glue padding
	glueFont   *slider          // glue caption size
	glue       [3]widget.Editor // glue captions: Before, After, Step
	glueErr    [3]string

	toolChoice widget.Enum // tool after selecting an area
	toolRows   []*toolRow
	bColor     widget.Editor
	colorErr   string

	invalidate func() // redraw the window from another goroutine
	grant      task   // granting permission for the fallback capture
	monitors   task   // re-picking monitors
	hotkey     task   // assigning the hotkey
	freeKey    task   // turning off GNOME's own use of that key
	setup      task   // monitors and permission in one go
	welcome    bool   // first launch: open on the System tab with a short guide
	hotkeyEd   widget.Editor
	lang       widget.Enum        // UI language: auto, ru, en
	tab        int                // selected tab
	tabBtns    []widget.Clickable // tab buttons
	shownTab   int                // the tab whose content was drawn in the last frame
	checkUpd   widget.Bool        // check for updates at startup
	update     task               // update check
	autostart  widget.Bool        // launch the tray icon at login
	autoErr    string
	keyRows    []*keyRow
	capturing  *keyRow // the row waiting for a key press
	capTag     bool    // focus tag for key capture
	dirErr     string
	prefixErr  string
}

// newShaper builds the fonts for the UI language: for Chinese, Japanese and
// Korean, a system font with CJK glyphs is added to the built-in ones.
func newShaper() *text.Shaper {
	return text.NewShaper(text.WithCollection(fonts.Collection(i18n.Language())))
}

func newState(store *settings.Store) *state {
	th := material.NewTheme()
	th.Shaper = newShaper()
	th.Palette = material.Palette{Bg: bgColor, Fg: fgColor, ContrastBg: accentColor, ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}

	s := &state{th: th, store: store}
	s.list.Axis = layout.Vertical
	tabs := s.tabs()
	s.tabBtns = make([]widget.Clickable, len(tabs))
	s.tab = rememberedTab(tabs) // the window opens on the tab that was open when it was closed
	if welcomeMode {
		s.welcome = true
		for i, t := range tabs {
			if t.id == "system" {
				s.tab = i
			}
		}
	}
	px := func(v float64) string { return fmt.Sprintf("%.1f px", v) }
	s.dim = s.addSlider(0, 100, 1, func(v float64) string { return fmt.Sprintf("%.0f%%", v) },
		func(c settings.Settings) float64 { return float64(c.DimPercent) },
		func(c *settings.Settings, v float64) { c.DimPercent = int(v) })
	s.bWidth = s.addSlider(settings.MinBorderWidth, settings.MaxBorderWidth, 0.5, px,
		func(c settings.Settings) float64 { return c.BorderWidth },
		func(c *settings.Settings, v float64) { c.BorderWidth = v })
	s.bDash = s.addSlider(settings.MinBorderDash, 12, 0.2, px,
		func(c settings.Settings) float64 { return c.BorderDash },
		func(c *settings.Settings, v float64) { c.BorderDash = v })
	s.bGap = s.addSlider(0, 12, 0.5, px,
		func(c settings.Settings) float64 { return c.BorderGap },
		func(c *settings.Settings, v float64) { c.BorderGap = v })
	s.marker = s.addSlider(settings.MinMarkerOpacity, 100, 1, func(v float64) string { return fmt.Sprintf("%.0f%%", v) },
		func(c settings.Settings) float64 { return float64(c.MarkerOpacity) },
		func(c *settings.Settings, v float64) { c.MarkerOpacity = int(v) })
	s.handle = s.addSlider(0, settings.MaxHandleRadius, 0.5, func(v float64) string {
		if v == 0 {
			return i18n.T("none")
		}
		return fmt.Sprintf("%.1f px", v)
	},
		func(c settings.Settings) float64 { return c.HandleRadius },
		func(c *settings.Settings, v float64) { c.HandleRadius = v })
	cur := store.Get()
	for _, sl := range s.sliders {
		sl.Set(sl.get(cur))
	}
	s.initKeys()
	s.checkUpd.Value = store.Get().CheckUpdates
	s.autostart.Value = autostartOn()
	s.lang.Value = store.Get().Language
	s.hotkeyEd.SingleLine = true
	s.hotkeyEd.SetText("Print")
	if cur := currentHotkey(); cur != "" {
		s.hotkeyEd.SetText(cur)
	}
	s.initTools(cur)
	s.glueLayout.Value = string(cur.Glue.Layout)
	s.format.Value = string(cur.Format)
	s.quality = s.addSlider(1, 100, 1, func(v float64) string { return fmt.Sprintf("%.0f", v) },
		func(c settings.Settings) float64 { return float64(c.JPEGQuality) },
		func(c *settings.Settings, v float64) { c.JPEGQuality = int(v) })
	s.quality.Set(float64(cur.JPEGQuality))
	s.glueBg.Value = string(cur.Glue.Background)
	whole := func(v float64) string { return fmt.Sprintf("%.0f px", v) }
	s.gluePad = s.addSlider(0, settings.MaxGluePad, 1, whole,
		func(c settings.Settings) float64 { return float64(c.Glue.Pad) },
		func(c *settings.Settings, v float64) { c.Glue.Pad = int(v) })
	s.glueFont = s.addSlider(settings.MinGlueLabelSize, settings.MaxGlueLabelSize, 1, whole,
		func(c settings.Settings) float64 { return float64(c.Glue.LabelSize) },
		func(c *settings.Settings, v float64) { c.Glue.LabelSize = int(v) })
	for _, sl := range s.sliders[len(s.sliders)-2:] {
		sl.Set(sl.get(cur))
	}
	for i, v := range []string{cur.Glue.Before, cur.Glue.After, cur.Glue.Step} {
		s.glue[i].SingleLine = true
		s.glue[i].SetText(v)
	}
	for i := range s.palEd {
		s.palEd[i].SingleLine = true
		s.palEd[i].SetText(cur.Palette[i])
	}
	s.bColor.SingleLine = true
	s.bColor.SetText(cur.BorderColor)
	s.choice.Value = string(cur.DoubleClick)
	s.enter.Value = string(cur.Enter)
	s.notify.Value = cur.Notifications
	s.remember.Value = cur.RememberRegion
	s.showSize.Value = cur.ShowSize
	s.showLoupe.Value = cur.ShowLoupe
	s.keepOpen.Value = cur.KeepOpen
	s.cursor.Value = cur.IncludeCursor
	s.dir.SingleLine = true
	s.dir.SetText(cur.SaveDir)
	s.prefix.SingleLine = true
	s.prefix.SetText(cur.FilePrefix)
	return s
}

// Run opens the settings window and runs until it is closed. Call it from main.
// welcomeMode makes the next window a first-launch welcome: the System tab with a short guide.
var welcomeMode bool

// RunWelcome opens the settings window for the first launch of the program: on the System tab,
// with the three steps that make the first screenshot possible.
func RunWelcome(store *settings.Store) {
	welcomeMode = true
	Run(store)
}

func Run(store *settings.Store) {
	i18n.SetLanguage(store.Get().Language)
	app.ID = screencast.AppID
	go func() {
		s := newState(store)
		if store.Get().CheckUpdates {
			s.checkNow()
		}
		loop(s)
		os.Exit(0)
	}()
	app.Main()
}

func loop(s *state) {
	w := new(app.Window)
	s.invalidate = w.Invalidate
	w.Option(app.Title(i18n.T("Scrawl: settings")), app.Size(unit.Dp(820), unit.Dp(640)), app.MinSize(unit.Dp(700), unit.Dp(360)))
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			windowerr.Report(e.Err) // e.Err != nil: the window failed; do not exit silently
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			for {
				ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
				if !ok {
					break
				}
				if k, ok := ev.(key.Event); ok && k.State == key.Press {
					if s.capturing != nil {
						s.capturing = nil // Esc cancels key capture instead of closing the window
					} else {
						w.Perform(system.ActionClose)
					}
				}
			}
			if s.close.Clicked(gtx) {
				w.Perform(system.ActionClose)
			}
			paint.Fill(gtx.Ops, bgColor)
			s.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

// apply brings the saved settings in line with the widgets. It is called on
// every frame: it writes nothing until the values have changed.
func (s *state) apply(gtx layout.Context) {
	// A tab click is handled before the frame decides what to draw:
	// otherwise the content would change only on the next frame, which will not
	// come until there is a new event, and the click would seem delayed.
	for i := range s.tabBtns {
		if s.tabBtns[i].Clicked(gtx) {
			s.selectTab(i)
		}
	}
	s.checkUpd.Update(gtx)
	if want := s.checkUpd.Value; want != s.store.Get().CheckUpdates {
		if err := s.store.Update(func(c *settings.Settings) { c.CheckUpdates = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}
	if s.update.btn.Clicked(gtx) {
		s.checkNow()
	}
	s.autostart.Update(gtx)
	if want := s.autostart.Value; want != autostartOn() {
		s.autoErr = ""
		var err error
		if want {
			var bin string
			if bin, err = hotkeyBinary(); err == nil {
				err = autostartEnable(bin)
			}
		} else {
			err = autostartOff()
		}
		if err != nil {
			s.autoErr = fmt.Sprintf(i18n.T("Could not change autostart: %v"), err)
			s.autostart.Value = autostartOn() // restore the checkbox to the state in the file
		}
	}
	s.lang.Update(gtx)
	if want := s.lang.Value; want != s.store.Get().Language {
		if err := s.store.Update(func(c *settings.Settings) { c.Language = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
		i18n.SetLanguage(s.store.Get().Language) // the window is redrawn in the new language right away
		s.th.Shaper = newShaper()                // and gets a font for its script
	}
	s.updateKeys(gtx)
	if s.hotkey.btn.Clicked(gtx) {
		accel := hotkey.NormalizeAccel(s.hotkeyEd.Text())
		s.hotkeyEd.SetText(accel) // "Ctrl+Alt+P" turns into the GNOME form
		s.start(&s.hotkey, func(func(string)) (string, error) {
			if err := hotkey.ValidateAccel(accel); err != nil {
				return "", err
			}
			bin, err := hotkeyBinary()
			if err != nil {
				return "", fmt.Errorf(i18n.T("could not find the path to the program: %w"), err)
			}
			res, err := assignHotkey(accel, bin)
			if err != nil {
				return "", err
			}
			msg := i18n.T("Hotkey ") + accel + i18n.T(" assigned to ") + bin
			if res.Updated {
				msg = i18n.T("Shortcut updated: ") + accel + i18n.T(" launches ") + bin
			}
			if res.Warning != "" {
				msg += i18n.T(". Warning: ") + res.Warning
			}
			return msg, nil
		})
	}
	if s.freeKey.btn.Clicked(gtx) {
		accel := hotkey.NormalizeAccel(s.hotkeyEd.Text())
		s.hotkeyEd.SetText(accel)
		s.start(&s.freeKey, func(func(string)) (string, error) {
			changed, err := freeBuiltin(accel)
			if err != nil {
				return "", err
			}
			if len(changed) == 0 {
				return fmt.Sprintf(i18n.T("No built-in GNOME shortcut uses %s: nothing to change"), accel), nil
			}
			return fmt.Sprintf(i18n.T("Turned off in GNOME for %s: %s"), accel, strings.Join(changed, ", ")), nil
		})
	}
	if s.setup.btn.Clicked(gtx) {
		s.start(&s.setup, func(progress func(string)) (string, error) {
			progress(i18n.T("Waiting for the GNOME dialogs…"))
			msg, err := setupMonitors(progress)
			if err != nil {
				return "", err
			}
			if err := grantPermission(); err != nil {
				return "", fmt.Errorf(i18n.T("Could not grant the permission: %w"), err)
			}
			return msg + ". " + i18n.T("Permission granted to the app ") + screencast.AppID, nil
		})
	}
	if s.monitors.btn.Clicked(gtx) {
		s.start(&s.monitors, func(progress func(string)) (string, error) {
			progress(i18n.T("Waiting for the GNOME dialogs…"))
			return setupMonitors(progress)
		})
	}
	if s.grant.btn.Clicked(gtx) {
		s.start(&s.grant, func(func(string)) (string, error) {
			if err := grantPermission(); err != nil {
				return "", fmt.Errorf(i18n.T("Could not grant the permission: %w"), err)
			}
			return i18n.T("Permission granted to the app ") + screencast.AppID, nil
		})
	}

	s.choice.Update(gtx)
	if want := settings.DoubleClick(s.choice.Value); want != s.store.Get().DoubleClick {
		if err := s.store.Update(func(c *settings.Settings) { c.DoubleClick = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}

	s.enter.Update(gtx)
	if want := settings.Enter(s.enter.Value); want != s.store.Get().Enter {
		if err := s.store.Update(func(c *settings.Settings) { c.Enter = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}

	s.notify.Update(gtx)
	if want := s.notify.Value; want != s.store.Get().Notifications {
		if err := s.store.Update(func(c *settings.Settings) { c.Notifications = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}

	for _, sl := range s.sliders {
		sl.Update(gtx)
		if want := sl.Value(); want != sl.get(s.store.Get()) {
			if err := s.store.Update(func(c *settings.Settings) { sl.set(c, want) }); err != nil {
				s.dirErr = i18n.T("Could not save: ") + err.Error()
			}
		}
	}
	palChanged := s.palReset.Clicked(gtx)
	if palChanged {
		for i := range s.palEd {
			s.palEd[i].SetText(settings.DefaultPalette[i])
		}
	}
	for i := range s.palEd {
		if edited(gtx, &s.palEd[i]) {
			palChanged = true
		}
	}
	if palChanged {
		s.applyPalette()
	}

	s.glueLayout.Update(gtx)
	if want := settings.GlueLayout(s.glueLayout.Value); want != s.store.Get().Glue.Layout {
		if err := s.store.Update(func(c *settings.Settings) { c.Glue.Layout = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}
	s.format.Update(gtx)
	if want := settings.FileFormat(s.format.Value); want != s.store.Get().Format {
		if err := s.store.Update(func(c *settings.Settings) { c.Format = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}
	s.glueBg.Update(gtx)
	if want := settings.GlueBackground(s.glueBg.Value); want != s.store.Get().Glue.Background {
		if err := s.store.Update(func(c *settings.Settings) { c.Glue.Background = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}
	for i := range s.glue {
		if edited(gtx, &s.glue[i]) {
			s.applyGlueLabel(i)
		}
	}

	s.toolChoice.Update(gtx)
	if want := s.toolChoice.Value; want != s.store.Get().DefaultTool {
		if err := s.store.Update(func(c *settings.Settings) { c.DefaultTool = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}
	for _, row := range s.toolRows {
		if edited(gtx, &row.ed) {
			s.applyToolColor(row)
		}
	}
	if edited(gtx, &s.bColor) {
		s.applyBorderColor(s.bColor.Text())
	}

	s.cursor.Update(gtx)
	if want := s.cursor.Value; want != s.store.Get().IncludeCursor {
		if err := s.store.Update(func(c *settings.Settings) { c.IncludeCursor = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}

	s.keepOpen.Update(gtx)
	if want := s.keepOpen.Value; want != s.store.Get().KeepOpen {
		if err := s.store.Update(func(c *settings.Settings) { c.KeepOpen = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}

	s.showSize.Update(gtx)
	if want := s.showSize.Value; want != s.store.Get().ShowSize {
		if err := s.store.Update(func(c *settings.Settings) { c.ShowSize = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}

	s.showLoupe.Update(gtx)
	if want := s.showLoupe.Value; want != s.store.Get().ShowLoupe {
		if err := s.store.Update(func(c *settings.Settings) { c.ShowLoupe = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}

	s.remember.Update(gtx)
	if want := s.remember.Value; want != s.store.Get().RememberRegion {
		if err := s.store.Update(func(c *settings.Settings) { c.RememberRegion = want }); err != nil {
			s.dirErr = i18n.T("Could not save: ") + err.Error()
		}
	}

	if s.openDir.Clicked(gtx) {
		s.openOutputDir()
	}

	changed := edited(gtx, &s.dir)
	if s.reset.Clicked(gtx) {
		s.dir.SetText("")
		changed = true
	}
	if changed {
		s.applyDir(strings.TrimSpace(s.dir.Text()))
	}
	if edited(gtx, &s.prefix) {
		s.applyPrefix(s.prefix.Text())
	}
}

// addSlider creates a slider and adds it to the window's list.
func (s *state) addSlider(min, max, step float64, format func(float64) string,
	get func(settings.Settings) float64, set func(*settings.Settings, float64)) *slider {
	sl := &slider{min: min, max: max, step: step, format: format, get: get, set: set}
	s.sliders = append(s.sliders, sl)
	return sl
}

// applyBorderColor validates and saves the border color; on error the setting is unchanged.
func (s *state) applyBorderColor(txt string) {
	c, err := settings.ParseColor(txt)
	if err != nil {
		s.colorErr = err.Error()
		return
	}
	s.colorErr = ""
	if err := s.store.Update(func(cfg *settings.Settings) { cfg.BorderColor = settings.FormatColor(c) }); err != nil {
		s.colorErr = i18n.T("Could not save: ") + err.Error()
	}
}

// openOutputDir opens the screenshots folder in the file manager. The window does not
// wait for the command to finish while it runs.
func (s *state) openOutputDir() {
	dir, err := s.store.Get().OutputDir()
	if err == nil {
		err = openFolder(dir)
	}
	if err != nil {
		s.dirErr = i18n.T("Could not open the folder: ") + err.Error()
		return
	}
	s.dirErr = ""
}

// openFolder runs xdg-open for a folder and does not wait for it to finish.
var openFolder = func(dir string) error {
	cmd := exec.Command("xdg-open", dir)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // so as not to leave a zombie process
	return nil
}

// edited reports whether the text in the field has changed since the last frame.
func edited(gtx layout.Context, ed *widget.Editor) bool {
	changed := false
	for {
		ev, ok := ed.Update(gtx)
		if !ok {
			return changed
		}
		if _, ok := ev.(widget.ChangeEvent); ok {
			changed = true
		}
	}
}

// checkNow asks GitHub for the latest version in the background.
func (s *state) checkNow() {
	s.start(&s.update, func(progress func(string)) (string, error) {
		progress(i18n.T("Checking…"))
		res, err := checkUpdate()
		if err != nil {
			return "", err
		}
		return res.Message(), nil
	})
}

// applyPrefix validates and saves the start of the file name; on error the setting is
// unchanged.
func (s *state) applyPrefix(prefix string) {
	if err := settings.ValidateFilePrefix(prefix); err != nil {
		s.prefixErr = err.Error()
		return
	}
	s.prefixErr = ""
	if err := s.store.Update(func(c *settings.Settings) { c.FilePrefix = prefix }); err != nil {
		s.prefixErr = i18n.T("Could not save: ") + err.Error()
	}
}

// applyDir validates and saves the folder; on error the setting is unchanged.
func (s *state) applyDir(dir string) {
	if err := settings.ValidateSaveDir(dir); err != nil {
		s.dirErr = err.Error()
		return
	}
	s.dirErr = ""
	if err := s.store.Update(func(c *settings.Settings) { c.SaveDir = dir }); err != nil {
		s.dirErr = i18n.T("Could not save: ") + err.Error()
	}
}

// row is a single setting: a name, a hint and a control.
func (s *state) row(name, hint string, control layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(s.label(material.Body1, name, fgColor))}
		if hint != "" {
			children = append(children, layout.Rigid(s.label(material.Caption, hint, mutedColor)))
		}
		if control != nil {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout), layout.Rigid(control))
		}
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// section is a section heading with a line under it.
func (s *state) section(name string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.label(material.Subtitle1, name, accentColor)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				h := gtx.Dp(1)
				paint.FillShape(gtx.Ops, borderColor, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, h)}.Op())
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		)
	}
}

// label makes a text widget of the given style and color.
func (s *state) label(style func(*material.Theme, string) material.LabelStyle, txt string, c color.NRGBA) layout.Widget {
	l := style(s.th, txt)
	l.Color = c
	return l.Layout
}

func (s *state) radio(group *widget.Enum, key, label string) layout.Widget {
	rb := material.RadioButton(s.th, group, key, label)
	rb.Color = fgColor
	// The selected option is blue, the others light gray: otherwise they are invisible on a
	// dark background.
	rb.IconColor = mutedColor
	if group.Value == key {
		rb.IconColor = accentColor
	}
	return rb.Layout
}

// field is a single-line input field with a border.
func (s *state) field(ed *widget.Editor, hint string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return widget.Border{Color: borderColor, CornerRadius: unit.Dp(6), Width: unit.Dp(1)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					e := material.Editor(s.th, ed, hint)
					e.Color = fgColor
					e.HintColor = mutedColor
					return e.Layout(gtx)
				})
			})
	}
}

// swatchBox draws a square color swatch with a border.
func swatchBox(gtx layout.Context, c color.NRGBA, dp int) layout.Dimensions {
	sz := gtx.Dp(unit.Dp(dp))
	rr := clip.UniformRRect(image.Rect(0, 0, sz, sz), gtx.Dp(6))
	paint.FillShape(gtx.Ops, borderColor, rr.Op(gtx.Ops))
	inner := clip.UniformRRect(image.Rect(gtx.Dp(2), gtx.Dp(2), sz-gtx.Dp(2), sz-gtx.Dp(2)), gtx.Dp(4))
	paint.FillShape(gtx.Ops, c, inner.Op(gtx.Ops))
	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

// toolRow is the settings row of one tool: its default color and thickness.
type toolRow struct {
	name  string // name in the settings
	label string
	ed    widget.Editor
	sl    *slider
	err   string
}

// toolLabels are the tool names in the window, in the order of settings.ToolNames.
var toolLabels = []string{"Select", "Pen", "Line", "Arrow", "Box", "Marker", "Blur", "Text", "Number"}

// initTools creates the tool rows and the default tool choice.
func (s *state) initTools(cur settings.Settings) {
	s.toolChoice.Value = cur.DefaultTool
	for i, name := range settings.ToolNames {
		if name == "select" {
			continue
		}
		name := name
		row := &toolRow{name: name, label: toolLabels[i]} // Russian key: translated when shown
		row.ed.SingleLine = true
		row.ed.SetText(cur.Tools.Of(name).Color)
		row.sl = s.addSlider(0, settings.MaxWidthOf(name), 1, func(v float64) string {
			if v == 0 {
				return i18n.T("default")
			}
			return fmt.Sprintf("%.0f", v)
		},
			func(c settings.Settings) float64 { return c.Tools.Of(name).Width },
			func(c *settings.Settings, v float64) { c.Tools.Of(name).Width = v })
		// A slider made after the first sync must get its saved value too, or the first frame
		// writes its zero to the file and erases the width the user chose.
		row.sl.Set(row.sl.get(cur))
		s.toolRows = append(s.toolRows, row)
	}
}

// applyToolColor validates and saves the tool color; empty means "leave unchanged".
func (s *state) applyToolColor(row *toolRow) {
	txt := strings.TrimSpace(row.ed.Text())
	value := ""
	if txt != "" {
		c, err := settings.ParseColor(txt)
		if err != nil {
			row.err = err.Error()
			return
		}
		value = settings.FormatColor(c)
	}
	row.err = ""
	if err := s.store.Update(func(c *settings.Settings) { c.Tools.Of(row.name).Color = value }); err != nil {
		row.err = i18n.T("Could not save: ") + err.Error()
	}
}

// toolControls is the tool choice after selecting an area and the settings of each tool.
func (s *state) toolControls() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(s.toolRows)*2)
		for _, row := range s.toolRows {
			row := row
			c := color.NRGBA{}
			if parsed, err := settings.ParseColor(row.ed.Text()); err == nil {
				c = parsed
			}
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(84)
						return s.label(material.Body2, i18n.T(row.label), fgColor)(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if c == (color.NRGBA{}) {
							// No color set: an empty frame instead of a swatch.
							sz := gtx.Dp(28)
							return layout.Dimensions{Size: image.Pt(sz, sz)}
						}
						return swatchBox(gtx, c, 28)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Max.X = gtx.Dp(120)
						gtx.Constraints.Min.X = gtx.Dp(120)
						return s.field(&row.ed, "#RRGGBB")(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Flexed(1, s.slider(row.sl)),
				)
			}))
			if row.err != "" {
				children = append(children, layout.Rigid(s.label(material.Caption, i18n.T(row.label)+": "+row.err, errorColor)))
			}
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// borderControls are the border sliders and the color field with a swatch.
func (s *state) borderControls() layout.Widget {
	labeled := func(name string, sl *slider) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Dp(80)
					return s.label(material.Body2, name, mutedColor)(gtx)
				}),
				layout.Flexed(1, s.slider(sl)),
			)
		})
	}
	return func(gtx layout.Context) layout.Dimensions {
		cfg := s.store.Get()
		swatch := func(gtx layout.Context) layout.Dimensions {
			c := cfg.BorderNRGBA()
			if parsed, err := settings.ParseColor(s.bColor.Text()); err == nil {
				c = parsed
			}
			return swatchBox(gtx, c, 36)
		}
		children := []layout.FlexChild{
			labeled(i18n.T("Thickness"), s.bWidth),
			labeled(i18n.T("Dash"), s.bDash),
			labeled(i18n.T("Gap"), s.bGap),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(80)
						return s.label(material.Body2, i18n.T("Colour"), mutedColor)(gtx)
					}),
					layout.Rigid(swatch),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Flexed(1, s.field(&s.bColor, "#FFFFFF")),
				)
			}),
		}
		if s.colorErr != "" {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, s.colorErr, errorColor)))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

func (s *state) checkbox(b *widget.Bool, label string) layout.Widget {
	cb := material.CheckBox(s.th, b, label)
	cb.Color = fgColor
	cb.IconColor = mutedColor
	if b.Value {
		cb.IconColor = accentColor
	}
	return cb.Layout
}

// applyPalette saves the panel colors. Empty fields take the default color; while
// any field has an invalid color, the setting is unchanged.
func (s *state) applyPalette() {
	var pal [settings.PaletteSize]string
	for i := range s.palEd {
		txt := strings.TrimSpace(s.palEd[i].Text())
		if txt == "" {
			pal[i] = settings.DefaultPalette[i]
			continue
		}
		c, err := settings.ParseColor(txt)
		if err != nil {
			s.palErr = fmt.Sprintf(i18n.T("Colour %d: %v"), i+1, err)
			return
		}
		pal[i] = settings.FormatColor(c)
	}
	s.palErr = ""
	if err := s.store.Update(func(c *settings.Settings) { c.Palette = pal }); err != nil {
		s.palErr = i18n.T("Could not save: ") + err.Error()
	}
}

// paletteControls are the panel colors in two columns and a reset button.
func (s *state) paletteControls() layout.Widget {
	cell := func(i int) layout.Widget {
		return func(gtx layout.Context) layout.Dimensions {
			c, _ := settings.ParseColor(settings.DefaultPalette[i])
			if parsed, err := settings.ParseColor(s.palEd[i].Text()); err == nil {
				c = parsed
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(s.label(material.Body2, fmt.Sprintf("%d", i+1), mutedColor)),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return swatchBox(gtx, c, 28) }),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Flexed(1, s.field(&s.palEd[i], settings.DefaultPalette[i])),
			)
		}
	}
	return func(gtx layout.Context) layout.Dimensions {
		const half = settings.PaletteSize / 2
		var children []layout.FlexChild
		for r := 0; r < half; r++ {
			r := r
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(gtx,
						layout.Flexed(1, cell(r)),
						layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
						layout.Flexed(1, cell(r+half)),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		if s.palErr != "" {
			children = append(children, layout.Rigid(s.label(material.Caption, s.palErr, errorColor)), layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			b := material.Button(s.th, &s.palReset, i18n.T("Default"))
			b.Background = borderColor
			return b.Layout(gtx)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// glueField returns a pointer to a glue caption in the settings by field number.
func glueField(g *settings.Glue, i int) *string {
	return [...]*string{&g.Before, &g.After, &g.Step}[i]
}

// applyGlueLabel validates and saves a glue caption; on error the setting is unchanged.
func (s *state) applyGlueLabel(i int) {
	txt := strings.TrimSpace(s.glue[i].Text())
	if err := settings.ValidateGlueLabel(txt); err != nil {
		s.glueErr[i] = err.Error()
		return
	}
	s.glueErr[i] = ""
	if err := s.store.Update(func(c *settings.Settings) { *glueField(&c.Glue, i) = txt }); err != nil {
		s.glueErr[i] = i18n.T("Could not save: ") + err.Error()
	}
}

// glueControls are three fields with the words of the glue captions.
func (s *state) glueControls() layout.Widget {
	names := [3]string{i18n.T("First of two"), i18n.T("Second of two"), i18n.T("If more than two")}
	hints := [3]string{i18n.T(settings.DefaultGlueBefore), i18n.T(settings.DefaultGlueAfter), i18n.T(settings.DefaultGlueStep)}
	return func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild
		for i := range s.glue {
			i := i
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(130)
						return s.label(material.Body2, names[i], mutedColor)(gtx)
					}),
					layout.Flexed(1, s.field(&s.glue[i], hints[i])),
				)
			}))
			if s.glueErr[i] != "" {
				children = append(children, layout.Rigid(s.label(material.Caption, names[i]+": "+s.glueErr[i], errorColor)))
			}
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// autostartControl is the autostart switch and the error message under it.
func (s *state) autostartControl() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(s.checkbox(&s.autostart, i18n.T("Start the icon at login")))}
		if s.autoErr != "" {
			children = append(children, layout.Rigid(s.label(material.Caption, s.autoErr, errorColor)))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// updateControl is the startup check switch and the "Check now" button.
func (s *state) updateControl() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.checkbox(&s.checkUpd, i18n.T("Check when the settings window and the tray icon start"))),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(s.taskControl(&s.update, i18n.T("Check now"))),
		)
	}
}

// hotkeyControl is the combination field and the assign button.
func (s *state) hotkeyControl() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.field(&s.hotkeyEd, "Print")),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(s.taskControl(&s.hotkey, i18n.T("Assign"))),
		)
	}
}

// toolChoiceRadios is the choice of the default tool.
func (s *state) toolChoiceRadios() layout.Widget {
	items := make([]layout.Widget, len(settings.ToolNames))
	for i, name := range settings.ToolNames {
		items[i] = s.radio(&s.toolChoice, name, i18n.T(toolLabels[i]))
	}
	return radioColumns(2, items...)
}

// languageRadios is the language choice in two columns: "as in the system" and all
// languages under their own names (they must be recognizable even when the rest of
// the window is not understood). If a language with CJK glyphs is selected and there
// is no system font for them, a hint on what to install is shown under the list.
func (s *state) languageRadios() layout.Widget {
	items := []layout.Widget{s.radio(&s.lang, i18n.Auto, i18n.T("Same as the system"))}
	for _, l := range i18n.Languages {
		name := l.Name
		if l.Hint != "" {
			name += "  ·  " + l.Hint // without a font for CJK glyphs the label is still readable
		}
		items = append(items, s.radio(&s.lang, l.Code, name))
	}
	columns := radioColumns(2, items...)
	return func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(columns)}
		if lang := i18n.Language(); !fontAvailable(lang) {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				layout.Rigid(s.label(material.Caption, i18n.T("This language needs a font with ideographs. Install it: sudo apt install fonts-noto-cjk"), errorColor)))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// radioColumns lays the options out in columns evenly, left to right.
func radioColumns(n int, items ...layout.Widget) layout.Widget {
	per := (len(items) + n - 1) / n
	return func(gtx layout.Context) layout.Dimensions {
		var cols []layout.FlexChild
		for i := 0; i < len(items); i += per {
			cols = append(cols, layout.Flexed(1, radios(items[i:min(i+per, len(items))]...)))
		}
		return layout.Flex{}.Layout(gtx, cols...)
	}
}

// radios stacks the choice options in a column.
func radios(items ...layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, len(items))
		for i, w := range items {
			children[i] = layout.Rigid(w)
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

// settingsTab is a window tab: its name and the settings rows in it.
type settingsTab struct {
	id    string // stable tab name: the choice is remembered by it
	label string
	rows  []layout.Widget
}

// tabs are all the window's tabs and their content. The window is rebuilt on every frame,
// so the texts immediately follow the selected language.
func (s *state) tabs() []settingsTab {
	th := s.th

	dirField := func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(s.field(&s.dir, "~/Pictures/Screenshots"))}
		if s.dirErr != "" {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, s.dirErr, errorColor)))
		}
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout), layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := func(c *widget.Clickable, txt string) layout.Widget {
				b := material.Button(th, c, txt)
				b.Background = borderColor
				return b.Layout
			}
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(btn(&s.reset, i18n.T("Default"))),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(btn(&s.openDir, i18n.T("Open folder"))),
			)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}

	prefixField := func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{layout.Rigid(s.field(&s.prefix, settings.DefaultFilePrefix))}
		if s.prefixErr != "" {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, s.prefixErr, errorColor)))
		}
		current := s.store.Get()
		current.FilePrefix = s.prefix.Text()
		if settings.ValidateFilePrefix(current.FilePrefix) != nil {
			current = s.store.Get()
		}
		example := current.FileName(time.Now().Format(settings.StampLayout))
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout), layout.Rigid(s.label(material.Caption, i18n.T("For example: ")+example, mutedColor)))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}

	general := []layout.Widget{
		s.row(i18n.T("Interface language"), i18n.T("The settings window changes at once; the overlay and notifications change on the next shot."),
			s.languageRadios()),
		s.section(i18n.T("Actions")),
		s.row(i18n.T("Double click inside the selection"), i18n.T("Works with any tool except text and stamp, not on the frame."),
			radios(
				s.radio(&s.choice, string(settings.DoubleClickOff), i18n.T("Nothing")),
				s.radio(&s.choice, string(settings.DoubleClickCopy), i18n.T("Copy to the clipboard")),
				s.radio(&s.choice, string(settings.DoubleClickCopySave), i18n.T("Copy and save to a file")),
			)),
		s.row(i18n.T("Enter key"), i18n.T("Ctrl+C, Ctrl+S and the toolbar buttons do not depend on this setting."),
			radios(
				s.radio(&s.enter, string(settings.EnterCopySave), i18n.T("Copy and save to a file")),
				s.radio(&s.enter, string(settings.EnterCopy), i18n.T("Only copy to the clipboard")),
				s.radio(&s.enter, string(settings.EnterSave), i18n.T("Only save to a file")),
			)),
		s.row(i18n.T("Mouse cursor"), i18n.T("Works with the normal screen capture. The fallback through the Screenshot portal has no cursor in the shot."),
			s.checkbox(&s.cursor, i18n.T("Show the cursor in the shot"))),
		s.row(i18n.T("After copying"), i18n.T("Enter, Ctrl+C, Ctrl+S, double click and the toolbar buttons. Leave with Esc."),
			s.checkbox(&s.keepOpen, i18n.T("Keep editing instead of closing"))),
		s.row(i18n.T("Notifications"), i18n.T("After a shot and collection actions. Errors are always shown."),
			s.checkbox(&s.notify, i18n.T("Show notifications"))),
		s.row(i18n.T("Previous area"), i18n.T("Shown right at launch and with the R key. Without it, before/after will not fill in the area by itself."),
			s.checkbox(&s.remember, i18n.T("Remember the shot area"))),
	}
	tools := []layout.Widget{
		s.row(i18n.T("Tool after selecting an area"), i18n.T("Switches on by itself once an area is selected. \"Select\": switch nothing on."),
			s.toolChoiceRadios()),
		s.row(i18n.T("Default colour and thickness"), i18n.T("Colour #RRGGBB: empty means keep as is. The colour applies until you pick a colour in the toolbar yourself. For text, thickness is the size; for blur, the strength."),
			s.toolControls()),
		s.row(i18n.T("Toolbar colours"), i18n.T("Eight colours in the toolbar, #RRGGBB. Empty field: the default colour. Applies on the next shot."),
			s.paletteControls()),
		s.row(i18n.T("Marker opacity"), i18n.T("The higher the percentage, the denser the fill."), s.slider(s.marker)),
	}
	look := []layout.Widget{
		s.row(i18n.T("Selection frame"), i18n.T("Dashed line around the selected area."), s.borderControls()),
		s.row(i18n.T("Area size"), i18n.T("A \"width × height\" label above the top-left corner."),
			s.checkbox(&s.showSize, i18n.T("Show size"))),
		s.row(i18n.T("Magnifier"), i18n.T("Magnified pixels next to the cursor while you pick an area and drag a frame."),
			s.checkbox(&s.showLoupe, i18n.T("Show the magnifier"))),
		s.row(i18n.T("Corner handles"), i18n.T("Circles in the frame corners; drag them to resize the area. \"none\" hides them."), s.slider(s.handle)),
		s.row(i18n.T("Dimming outside the area"), i18n.T("0% does not dim at all, 100% paints it black."), s.slider(s.dim)),
	}
	glue := []layout.Widget{
		s.row(i18n.T("Layout"), i18n.T("Images are aligned to the lowest one (side by side) or the narrowest one (stacked)."),
			radios(
				s.radio(&s.glueLayout, string(settings.GlueSide), i18n.T("Side by side, left to right")),
				s.radio(&s.glueLayout, string(settings.GlueStack), i18n.T("Stacked, top to bottom")),
			)),
		s.row(i18n.T("Background"), "", radios(
			s.radio(&s.glueBg, string(settings.GlueLight), i18n.T("White")),
			s.radio(&s.glueBg, string(settings.GlueDark), i18n.T("Dark")),
		)),
		s.row(i18n.T("Margins"), i18n.T("From the edges and between images."), s.slider(s.gluePad)),
		s.row(i18n.T("Caption size"), "", s.slider(s.glueFont)),
		s.row(i18n.T("Captions above images"), i18n.T("Empty: default words. With three or more images a number is added to the word: \"Step 1\"."),
			s.glueControls()),
	}
	files := []layout.Widget{
		s.row(i18n.T("Screenshot folder"), i18n.T("Empty: ~/Pictures/Screenshots. You can write ~/Folder or a full path."), dirField),
		s.row(i18n.T("File name"), i18n.T("Start of the name, followed by date and time. Empty: ")+settings.DefaultFilePrefix+".", prefixField),
		s.row(i18n.T("File format"), i18n.T("The clipboard always gets PNG."), radios(
			s.radio(&s.format, string(settings.FormatPNG), i18n.T("PNG: lossless")),
			s.radio(&s.format, string(settings.FormatJPEG), i18n.T("JPEG: smaller file, lossy")),
		)),
		s.row(i18n.T("JPEG quality"), i18n.T("The higher, the better the picture and the bigger the file."), s.slider(s.quality)),
	}
	keys := []layout.Widget{
		s.row(i18n.T("Keys"), i18n.T("Press a button and then the new combination. Esc cancels recording. Applies on the next shot."),
			s.keyControls()),
	}
	system := append([]layout.Widget{s.section(i18n.T("Screen capture"))}, s.captureRows()...)
	system = append(system,
		s.section(i18n.T("Startup")),
		s.row(i18n.T("Autostart"), i18n.T("At login a notification-area icon is started (scrawl -tray): left click takes a screenshot. GNOME needs the AppIndicator extension."),
			s.autostartControl()),
	)
	about := []layout.Widget{
		s.row("Scrawl "+version.String(), version.Home, nil),
		s.row(i18n.T("Updates"), i18n.T("The only place where the program goes online: a request to GitHub Releases. Off by default; the button works even when the check is off."),
			s.updateControl()),
	}

	return []settingsTab{
		{id: "general", label: i18n.T("General"), rows: general},
		{id: "tools", label: i18n.T("Tools"), rows: tools},
		{id: "look", label: i18n.T("Appearance"), rows: look},
		{id: "glue", label: i18n.T("Collage"), rows: glue},
		{id: "files", label: i18n.T("Files"), rows: files},
		{id: "keys", label: i18n.T("Keys"), rows: keys},
		{id: "system", label: i18n.T("System"), rows: system},
		{id: "about", label: i18n.T("About"), rows: about},
	}
}

// captureRows are the capture settings. An X11 session has no portal: there is no need to
// pick monitors or grant permission, so a short explanation replaces those buttons.
func (s *state) captureRows() []layout.Widget {
	var rows []layout.Widget
	if s.welcome {
		guide := i18n.T("Set up once. 1. Press \"Pick monitors again\" and \"Grant permission\" below (GNOME asks once per monitor: tick \"Remember this choice\"). 2. Type the hotkey (Print by default), press \"Assign\" and, if GNOME's own screenshot tool uses that key, \"Free the key\". 3. Press the hotkey, drag an area and press Enter.")
		if onWindows {
			guide = i18n.T("Set up once. 1. Type the hotkey (Print by default) and press \"Assign\". 2. Press the hotkey, drag an area and press Enter.")
		} else if x11Session() {
			guide = i18n.T("Set up once. 1. Type the hotkey (Print by default), press \"Assign\" and, if GNOME's own screenshot tool uses that key, \"Free the key\" (on KDE, Xfce and others bind the command \"scrawl\" to a key in the system keyboard settings). 2. Press the hotkey, drag an area and press Enter.")
		}
		rows = append(rows, s.row(i18n.T("Welcome to Scrawl"), guide, nil))
	}
	hotkeyRow := s.row(i18n.T("Launch hotkey"), i18n.T("Creates or updates a shortcut in the GNOME keyboard settings. Format: Print, <Super>s, <Control><Alt>p. Other shortcuts are not changed."),
		s.hotkeyControl())
	if onWindows {
		// Windows has neither a portal nor GNOME: the screen is captured directly via GDI, and the
		// combination is held by the tray icon (scrawl -tray).
		return append(rows, s.row(i18n.T("Launch hotkey"), i18n.T("Works while the tray icon is running. Format: Print, <Super>s, <Control><Alt>p. Another program may hold Print (Greenshot, Snipping Tool): then choose another combination."),
			s.hotkeyControl()))
	}
	freeRow := s.row(i18n.T("Free the key from GNOME"),
		i18n.T("GNOME's own screenshot tool takes Print before Scrawl does. This turns that built-in shortcut off for the key typed above; nothing else is changed. To undo: gsettings reset org.gnome.shell.keybindings show-screenshot-ui"),
		s.taskControl(&s.freeKey, i18n.T("Free the key")))
	if x11Session() {
		// An X11 session has no portal: no monitors to pick and no permission to grant.
		return append(rows, hotkeyRow, freeRow,
			s.row(i18n.T("X11 session"), i18n.T("The screen is captured directly, without a portal: there is no need to pick monitors or grant a permission. The clipboard needs xclip."), nil))
	}
	// Wayland: the capture setup comes first, it is what a new user has to do before anything else.
	status := i18n.T("No monitors saved yet")
	if n := savedMonitors(); n > 0 {
		status = fmt.Sprintf(i18n.T("Monitors saved: %d"), n)
	}
	return append(rows,
		s.row(i18n.T("Screen capture setup"),
			fmt.Sprintf(i18n.T("Picks the monitors (GNOME shows a dialog for each one: choose a different monitor in each and tick \"Remember this choice\") and grants the permission for the fallback capture, all in one go. Needed once, and again after connecting or replacing a monitor. %s"), status),
			s.taskControl(&s.setup, i18n.T("Set up capture"))),
		s.row(i18n.T("Monitors"), i18n.T("Pick the monitors again: GNOME shows a dialog for each monitor. Needed after connecting or replacing a monitor. Formerly: scrawl -setup."),
			s.taskControl(&s.monitors, i18n.T("Pick monitors again"))),
		s.row(i18n.T("Fallback capture permission"), i18n.T("Needed by the fallback capture through the Screenshot portal when PipeWire gave no frame. Formerly: scrawl -grant-screenshot."),
			s.taskControl(&s.grant, i18n.T("Grant permission"))),
		hotkeyRow, freeRow,
	)
}

// rows returns the rows of the selected tab.
func (s *state) rows() []layout.Widget {
	tabs := s.tabs()
	return tabs[min(max(s.tab, 0), len(tabs)-1)].rows
}

// selectTab switches the tab and scrolls back to the top.
func (s *state) selectTab(i int) {
	if i != s.tab {
		s.tab = i
		s.list.Position = layout.Position{}
		if tabs := s.tabs(); i >= 0 && i < len(tabs) {
			saveTab(tabs[i].id)
		}
	}
}

// tabBar draws the tabs in a row and a thin full-width line under it.
func (s *state) tabBar(gtx layout.Context, tabs []settingsTab) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return s.tabRow(gtx, tabs) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := gtx.Dp(1)
			w := gtx.Constraints.Max.X
			paint.FillShape(gtx.Ops, borderColor, clip.Rect{Max: image.Pt(w, h)}.Op())
			return layout.Dimensions{Size: image.Pt(w, h)}
		}),
	)
}

// tabRow is the tabs themselves: the selected one is underlined in the accent color, the one
// under the cursor is highlighted.
func (s *state) tabRow(gtx layout.Context, tabs []settingsTab) layout.Dimensions {
	children := make([]layout.FlexChild, 0, len(tabs))
	for i := range tabs {
		i := i
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			selected := i == s.tab
			return s.tabBtns[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx layout.Context) layout.Dimensions {
						size := gtx.Constraints.Min
						if s.tabBtns[i].Hovered() && !selected {
							paint.FillShape(gtx.Ops, borderColor, clip.UniformRRect(image.Rectangle{Max: size}, gtx.Dp(6)).Op(gtx.Ops))
						}
						if selected {
							h := gtx.Dp(2)
							paint.FillShape(gtx.Ops, accentColor, clip.Rect{Min: image.Pt(0, size.Y-h), Max: size}.Op())
						}
						return layout.Dimensions{Size: size}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						c := mutedColor
						if selected || s.tabBtns[i].Hovered() {
							c = fgColor
						}
						return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(9), Left: unit.Dp(11), Right: unit.Dp(11)}.Layout(gtx,
							s.label(material.Body2, tabs[i].label, c))
					}),
				)
			})
		}))
	}
	return layout.Flex{}.Layout(gtx, children...)
}

func (s *state) layout(gtx layout.Context) layout.Dimensions {
	s.apply(gtx)
	th := s.th
	tabs := s.tabs()
	s.tab = min(max(s.tab, 0), len(tabs)-1)
	s.shownTab = s.tab
	items := tabs[s.tab].rows

	return layout.UniformInset(unit.Dp(22)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(s.label(material.H6, i18n.T("Settings"), fgColor)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return s.tabBar(gtx, tabs) }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				list := material.List(th, &s.list)
				return list.Layout(gtx, len(items), func(gtx layout.Context, i int) layout.Dimensions {
					// On the right there is room for the scrollbar, so that it does not lie over the controls.
					return layout.Inset{Right: unit.Dp(14)}.Layout(gtx, items[i])
				})
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(s.label(material.Caption, i18n.T("Changes apply immediately. File: ")+settings.Path(), mutedColor)),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				b := material.Button(th, &s.close, i18n.T("Close"))
				b.Background = accentColor
				return b.Layout(gtx)
			}),
		)
	})
}
