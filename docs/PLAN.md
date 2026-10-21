# Project plan

A living document: mark what is done, add what is new. Size: S is quick (an hour or two), M is medium (half a day),
L is a separate big task. Statuses: `[x]` done, `[~]` in progress, `[ ]` not started.

Stages 1-3 of the settings window are done.

## Done

- [x] Screen capture through the ScreenCast portal and an own PipeWire client, a fallback through the Screenshot portal
- [x] An overlay on every monitor, a selection with a dashed frame, handles, moving
- [x] Tools: pen, line, arrow, box, marker, blur, text; colours; thickness with the wheel and buttons
- [x] Before/after collections: glued side by side, sent as separate files
- [x] Remembering the last area, the dock panel, protection against repeated launches
- [x] Tabler icons, the settings window (double click, the save folder)
- [x] Build (`make`), installation, the MIT license, the README in several languages, third-party licenses

## The settings window

All settings live in `~/.config/scrawl/config.json`, are changed in the settings window
(`scrawl -settings`) and are applied the next time the overlay starts.

### Stage 1. Foundation and quick settings

- [x] S Version: the `-version` flag, an "About" section in the window
- [x] S Settings schema: defaults, validation, unknown keys do not break the file
- [x] S A scrollable settings window with sections
- [x] S What Enter does: copy and save / only copy / only save
- [x] S Notifications after a shot: on/off (errors are always shown)
- [x] S Remember the last area: on/off
- [x] S File name template (date and time or a custom prefix)
- [x] S The "Open the screenshots folder" button
- [x] S Dimming outside the area: a slider 0-100%
- [x] S Frame: thickness, dash, gap, colour
- [x] S Size of the corner handles
- [x] S Show the size label "W × H"

### Stage 2. Tools and collections

- [x] M Default tool, colour and thickness (separate for each tool)
- [x] M Custom palette colours (entering #RRGGBB)
- [x] S Marker opacity
- [x] S Glue labels: own words instead of "Before/After/Step N"
- [x] M Glue layout: side by side or one under another
- [x] S Glue look: a white or dark background, padding, label size
- [x] M File format: PNG or JPEG with quality (the clipboard stays PNG)
- [x] M Do not close the overlay after copying, keep editing
- [x] M Include the mouse cursor in the shot (cursor_mode of the capture session)

### Stage 3. Buttons, keys, language

- [x] S The "Grant the fallback permission" button (instead of `-grant-screenshot`)
- [x] S The "Pick the monitors again" button (instead of `-setup`)
- [x] M The "Assign the launch hotkey" button (GNOME, through gsettings, on the user's click)
- [x] L Remapping the keys inside the overlay (R, A, G, D and others)
- [x] L Interface language: Russian or English (move all texts out)

## X11

- [x] M An X11 session: direct capture (RandR + GetImage), the clipboard through xclip, an overlay on every monitor (a Gio patch), the portal buttons hidden
- [ ] M Test on a real X11 machine (Ubuntu on Xorg, KDE): tested only on Xvfb with two RandR monitors and a stub EWMH manager, a real manager (mutter, kwin) was not tested
- [ ] S The mouse cursor in the shot on X11 (XFixes GetCursorImage)
- [ ] M X11 outside GNOME: the hotkey and notifications without gsettings

## Launch, tray and updates

- [x] M Autostart at login: a switch in the settings (a file in `~/.config/autostart/`)
- [x] L A tray icon (StatusNotifierItem): the menu "Take screenshot", "Settings", "Quit". It needs a resident process;
      on GNOME the icon is visible only with the AppIndicator extension (on by default in Ubuntu).
      The resident process can also be used for a future daemon (fast start)
- [x] M Update check: optional (off by default), a request to GitHub Releases,
      the "Check now" button, showing the new version. It needs a public repository and releases
- [x] S Showing the version: the `-version` flag and the "About" section

## Documentation and translations

- [x] The README in English and Russian rewritten, the settings reference moved to `docs/SETTINGS.md` and `docs/translations/SETTINGS.ru.md`
- [x] README translations: Chinese (Simplified), Spanish, Portuguese (Brazil), Japanese, German, French, Korean (`docs/translations/`)
- [x] L The program interface in these languages: catalogs `internal/i18n/lang_*.go` (English is the source text in the code, Russian is a catalog like the others), a choice in the settings, a system font with ideographs for Chinese, Japanese and Korean (`internal/fonts`)
- [ ] S Native speakers check the README and interface translations (they were made without a native speaker)

## Publishing

- [x] S The project name: Scrawl is chosen (there is no screenshot tool with that name on GitHub, in Debian or on Flathub)
- [x] M Prepare the history before the first push (remove the Gio copy from the old commits, replace the email); the repository was re-created clean and contains only noreply addresses, CI checks the addresses on every push
- [x] L Windows: capture, an overlay on the right monitor, the clipboard, the hotkey, the icon and a release archive, see docs/WINDOWS.md; what is left (scale above 100%, code signing) is listed there
- [x] M Packages and releases (CI on GitHub Actions, builds, checksums): releases are made by a `v*` tag (an archive, a `.deb`, Windows files)
- [x] M A signed apt repository on GitHub Pages (`scripts/build-apt-repo.sh`, `.github/workflows/apt-repo.yml`), updated after every release
- [ ] L Flathub: a manifest and a rework for the sandbox (the hotkey, the clipboard, notifications)
- [ ] M Code signing for Windows (see docs/WINDOWS.md)

## Decisions and questions

- Applying settings: the next time the overlay starts (the settings window is a separate process)
- Network: the update check is the only place where the program goes online; it is off by default
- The tray needs a resident process: the decision is postponed until the daemon work
