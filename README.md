# Scrawl

**Fast screenshots with annotations for GNOME on Wayland.** Press a key, select
an area on any monitor, draw arrows, boxes and text, blur out private data, and
the result is already in your clipboard and saved as a file.

[English](README.md) · [Русский](docs/translations/README.ru.md) · [简体中文](docs/translations/README.zh-CN.md) · [Español](docs/translations/README.es.md) · [Português (Brasil)](docs/translations/README.pt-BR.md) · [日本語](docs/translations/README.ja.md) · [Deutsch](docs/translations/README.de.md) · [Français](docs/translations/README.fr.md) · [한국어](docs/translations/README.ko.md)

![Scrawl overlay: a selected area with a box, an arrow, a caption, a marker stroke and a blurred spot, and the toolbar below it](docs/images/overlay.png)

*The picture is a real render of the overlay over a made-up desktop.*

## Why Scrawl

On GNOME with Wayland, apps cannot grab the screen directly. They ask the desktop
portal, and the portal asks **"which screen do you want to share?" on every shot**.
Scrawl asks once per monitor, remembers the answer, and after that a screenshot
is one key press: the frozen screen appears in about 150 ms.

## Quick start

On Ubuntu or Debian (Wayland or X11) it takes three steps.

**1. Install** from the signed apt repository (updates then come with `apt upgrade`):

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

**2. Set up once:** run `scrawl -settings`, open the tab "System". On Wayland press "Pick monitors again"
(GNOME asks once per monitor; tick "Remember this choice") and "Grant permission". Then type a hotkey
(`Print` by default) and press "Assign". On X11 only the hotkey is needed.

**3. Take a screenshot:** press the hotkey, drag an area, press `Enter`. The picture is in your clipboard and in
`~/Pictures/Screenshots`.

Other systems: [Windows](#windows-10-and-11), other Linux distributions (a release archive or a source build), see
[Install](#install) below.

## Features

**Capture**
- Works on every monitor at once; the overlay opens on all of them and you select on any.
- Selection with a dashed frame. Drag the corners to resize; with the *select*
  tool drag an edge to resize or drag inside to move.
- While you drag a frame, a magnifier next to the cursor shows the pixels 6× larger, with the frame's
  edge on the pixel grid, so you can line it up exactly (it can be turned off in the settings).
- Remembers the last area and brings it back with one key (`R`).
- Optional mouse cursor in the shot (Wayland only for now).

**Annotate**
- Pen, line, arrow, box, marker, **blur**, text and **numbered stamps** (a click puts a circle with 1, 2, 3 and so on, like in flameshot); eight colours.
- Mouse wheel (or the `−` / `+` buttons) changes the thickness, the text size or the blur strength.
- The **blur** has six levels (wheel or `−` / `+`): 1-3 soft Gaussian blur, 4-6 mosaic with growing squares. The mosaic cells start at the corner of the area, and the note that pops up when you change the level names the mode.
- Hold **Shift** while drawing: the pen and the marker make a straight line, a line or an arrow snaps to 45°, a box or a blur becomes a square.
- Undo with `Ctrl+Z`.

**Save and share**
- `Enter` copies the image to the clipboard and saves a PNG to `~/Pictures/Screenshots`.
  `Ctrl+C` only copies, `Ctrl+S` only saves. JPEG is available too.
- A **double click** inside the selected area copies it at once (in the settings, tab "General", section "Actions", it can also copy and save, or do nothing).
- **Before/after collections:** set shots aside with `A`, then glue them into one
  picture with captions ("Before", "After", or "Step 1…" for more), side by side
  or one under another. Or send the collection as separate files that paste into a chat as attachments.

**Make it yours**
- A settings window for almost everything: keys, colours, default tools, the look of the selection, file names and format, and the interface language (9 languages: English, Russian, Chinese, Spanish, Portuguese, Japanese, German, French, Korean).
- A tray icon with autostart, a hotkey button, and an optional update check (off by default).

## Status

A personal project under active development; the current version is a pre-release (`v0.1.0-rc2`). Ready-made packages are on the
[Releases](https://github.com/jexme/scrawl/releases) page; you can also build it from source (a few minutes).
Linux: GNOME on Wayland is the main target. **X11 sessions** (GNOME on Xorg, KDE,
XFCE and others) work too, but that support is new and has so far been tested only on a
virtual X server, so reports from real machines are welcome. **Windows 10 and 11** work too (capture, annotations,
clipboard, tray icon and a global hotkey); this port is new and has been tried only on a 100% scale, see
[docs/WINDOWS.md](docs/WINDOWS.md) for building and what is missing.

## Requirements

Installing with apt or the `.deb` pulls these in for you; the list is for the release archive and a source build.

- GNOME on Wayland with PipeWire, `xdg-desktop-portal` (1.21 or newer, it must
  support host app registration) and `xdg-desktop-portal-gnome`. Developed and tested on Ubuntu with GNOME 50.
- `wl-clipboard` (the `wl-copy` command) for the clipboard and `libnotify`
  (the `notify-send` command) for notifications. Without `wl-copy`, Scrawl falls back to
  `xclip` through XWayland if both are there; with neither you get a hint what to install.
- On **X11** there is no portal: the screen is captured directly, and the clipboard
  needs `xclip` instead of `wl-clipboard`.

```bash
sudo apt install wl-clipboard libnotify-bin   # Wayland
sudo apt install xclip libnotify-bin          # X11
```

## Install

### 1. Get the program

**From the apt repository** (Debian, Ubuntu; updates come with `apt upgrade`):

```bash
curl -fsSL https://jexme.github.io/scrawl/scrawl-archive-keyring.gpg | sudo tee /usr/share/keyrings/scrawl-archive-keyring.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/scrawl-archive-keyring.gpg] https://jexme.github.io/scrawl stable main" | sudo tee /etc/apt/sources.list.d/scrawl.list
sudo apt update && sudo apt install scrawl
```

The repository is signed with the key `E62C DAF5 A1C9 0DB7 A899 D24D FA43 15C7 154A 9EF0`.

**From a .deb package** (Debian, Ubuntu): download `scrawl_VERSION_amd64.deb`
from the [Releases](https://github.com/jexme/scrawl/releases) page, then
`sudo apt install ./scrawl_*.deb`. It pulls in the libraries and recommends `wl-clipboard`, `xclip`
and `libnotify-bin`. The hotkey is not assigned by the package (it is per user): do it in `scrawl -settings`.
Build it yourself with `make deb`.

**From a release archive**: download
`scrawl-VERSION-linux-amd64.tar.gz` and `SHA256SUMS` from the
[Releases](https://github.com/jexme/scrawl/releases) page, then:

```bash
sha256sum -c --ignore-missing SHA256SUMS   # the .deb is listed with "~" but GitHub saves it with "."; check it by hand
tar xzf scrawl-*-linux-amd64.tar.gz && cd scrawl-*-linux-amd64
./install.sh            # to ~/.local (PREFIX=/usr/local for system-wide)
sudo apt install libpipewire-0.3-0 libwayland-client0 libwayland-egl1 libwayland-cursor0 \
  libxkbcommon-x11-0 libegl1 libx11-6 libxcursor1 wl-clipboard libnotify-bin
```

`wl-clipboard` (the clipboard) and `libnotify-bin` (notifications) are needed at run time too: the
installer checks them and prints what is missing. On X11 use `xclip` instead of `wl-clipboard`.

**From source**:

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
git clone https://github.com/jexme/scrawl.git && cd scrawl
make install            # builds and installs to ~/.local (PREFIX=/usr/local for system-wide)
```

You need Go (the version from `go.mod`). No `sudo`? `scripts/fetch-pipewire-headers.sh`
unpacks the PipeWire headers into `.deps/`, and the Makefile picks them up.

Make sure `~/.local/bin` is in your `PATH`.

### Windows 10 and 11

Download `scrawl-VERSION-windows-amd64.zip` and its `.sha256` from the [Releases](https://github.com/jexme/scrawl/releases)
page, unpack it anywhere and run `scrawl.exe -tray`. It is a portable program: no installer, nothing else to install.
If you prefer a single file, download `scrawl-VERSION-windows-amd64.exe` (with its `.sha256`) instead and run it the same way.
With [Scoop](https://scoop.sh): `scoop install https://github.com/jexme/scrawl/releases/download/v0.1.0-rc2/scrawl.json` (the `releases/latest/...` address works only once there is a stable release, pre-releases are skipped).

- **The icon** is in the notification area; in Windows 11 it may hide behind the `^` arrow next to the clock, drag it out to keep it visible.
- **The Print key** is the default hotkey. Another program often holds it (Greenshot, ShareX, Lightshot). Windows itself may too:
  Settings, Accessibility, Keyboard, *Use the Print screen button to open screen capture*: turn it off. Or pick another key in
  `scrawl.exe -settings`, for example `<Control><Alt>a`. The icon tells you when the key is taken and keeps trying.
- **"Windows protected your PC"** (SmartScreen) may appear: the program is not code-signed yet. Check the SHA-256 against the
  `.sha256` file (`Get-FileHash scrawl-*.zip`), then *More info*, *Run anyway*. The build comes from public CI:
  `gh attestation verify scrawl-*.zip --repo jexme/scrawl` proves it. Antivirus programs may flag any new unsigned Go program;
  report such a case as an issue.
- **Autostart** is the checkbox in `scrawl.exe -settings`, tab *System* (a value in `HKCU\...\Run`). Keep the folder where it is, or toggle it again after moving.

Build from source and what is missing: [docs/WINDOWS.md](docs/WINDOWS.md).

### 2. First run: three one-time steps

*On X11 steps 1 and 2 are not needed (there is no portal), only the hotkey.*

Open the settings window (`scrawl -settings`, tab "System") and use these buttons, or run the commands:

1. **Pick the monitors:** button "Pick monitors again" or `scrawl -setup`.
   GNOME shows one dialog **per monitor**. In each, choose a *different* monitor,
   keep **"Remember this choice"** ticked and press the share button.
   After that, no dialogs.
2. **Grant the fallback permission:** button "Grant permission" or `scrawl -grant-screenshot`.
   This lets Scrawl use the slower Screenshot portal on the rare shot when
   PipeWire gives no frame.
3. **Set a hotkey:** on GNOME the archive installer (`install.sh`) and `make install` assign `Print` themselves unless another custom shortcut already uses it. After installing with apt or the `.deb`, or to change the key: the "Launch hotkey" field and the "Assign" button (default `Print`),
   or `scripts/gnome-hotkey.sh [KEY]`. It creates a GNOME custom shortcut and never
   overwrites one that is taken. If `Print` still opens GNOME's own screenshot tool, turn that
   off in *Settings → Keyboard → View and Customize Shortcuts → Screenshots*.

### Upgrading from screen-editor

Scrawl is the new name of screen-editor. Your settings are picked up from
`~/.config/screen-editor/config.json` until `~/.config/scrawl/config.json`
exists (the old file stays untouched). The portal ties monitor choices and
permissions to the app id, which changed, so repeat steps 1 to 3 once, and point your hotkey
at `scrawl`. The last area and the collection start fresh.

## Using Scrawl

1. Press your hotkey. The screen freezes and dims.
2. Drag to select an area (on any monitor). A toolbar appears under it.
3. Annotate if you want: pick a tool and a colour, drag on the area.
4. Press `Enter`. The image is in your clipboard and in `~/Pictures/Screenshots`. Paste it anywhere.

`Esc` cancels at any point.

### Keys

| Key | Action |
| --- | --- |
| `Enter` | copy to the clipboard **and** save to a file (what it does is configurable) |
| `Ctrl+C` / `Ctrl+S` | only copy / only save |
| `Ctrl+Z` | undo the last annotation |
| `Esc` | cancel |
| `R` | restore the area of the previous shot |
| `V` `P` `L` `W` `B` `M` `U` `T` `N` | tools: select, pen, line, arrow, box, marker, blur, text, number stamp (when an area is selected) |
| `+` / `-` | thickness, text size or blur strength of the current tool (like the mouse wheel) |
| `A` | add the shot to the collection |
| `G` | glue the collection (and the current area) into one image |
| `D` | save the collection as separate files and put the file list in the clipboard |
| mouse wheel | thickness, text size or blur strength of the current tool |

Every key can be changed in the settings window (the "Keys" tab).

### Toolbar

The main row, from left to right: select, pen, line, arrow, box, marker, blur, text, number
stamp · undo · copy · save · the `⋯` menu (settings, which closes the overlay and opens the settings
window, and add to collection) · close.

While a drawing tool is on, a second row appears under it: eight colours, a rainbow button for
your own colour (opens a palette; the colour is remembered) and `−` thickness `+`. The "select"
tool has neither, so the row is hidden.

The buttons that have a key show it under the icon (the keys come from the settings).

### Before/after collections

Take the first shot and press `A`. Take the next one: the previous area is already
selected, so you shoot the same place again. Press `G` to glue the shots into one
image (it is saved and copied), or `D` to get separate files. The small panel
at the bottom of the screen shows what is in the collection and has the same actions as buttons.

## Tray icon and autostart

`scrawl -tray` keeps an icon in the notification area: left click takes a screenshot,
the menu has "Screenshot", "Settings" and "Quit". To start it at login, switch on
*Settings → System → Autostart*. On GNOME the icon needs the AppIndicator
extension (enabled by default in Ubuntu).

## Settings

![The settings window](docs/images/settings-en.png)

Open it from the toolbar (gear), with `scrawl -settings`, or from the application menu.
Changes are saved at once to `~/.config/scrawl/config.json`, a plain JSON file
you can also edit by hand. Missing or invalid values fall back to the defaults.
Every key is described in the [settings reference](docs/SETTINGS.md).

## Files and privacy

- Images go to `~/Pictures/Screenshots` (change it in the settings), named like `scrawl-20260131-142501.png`.
- Settings: `~/.config/scrawl/`. Monitor tokens, the last area and the collection: `~/.cache/scrawl/`.
- Scrawl never goes online by itself. The only network request in the program is
  the optional update check (off by default); it asks GitHub for the latest release, and the only thing it sends besides the usual HTTP headers is the program version.
- Screenshots never leave your computer unless you paste them somewhere.

## Command line

| Flag | What it does |
| --- | --- |
| `-settings` | open the settings window |
| `-setup` | pick the monitors again (GNOME dialogs) |
| `-grant-screenshot` | grant the fallback capture permission and exit |
| `-tray` | keep the notification-area icon |
| `-fresh` | do not offer the area of the previous shot |
| `-v` | print timings |
| `-version` | print the version and exit |

## Troubleshooting

- **Nothing happens on the hotkey.** `xdg-desktop-portal` may be hung. Restart it:
  `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- **GNOME asks which screen to share on every shot.** The "Remember this choice" box was not ticked; repeat step 1.
- **A monitor was added or replaced.** Pick the monitors again (step 1).
- **The overlay opens on only one monitor.** Same: step 1 must be done for every monitor.
- **The message says the portal refused.** Grant the permission (step 2).
- **"Clipboard error" or pasting gives nothing.** Check that `wl-copy` is installed (`wl-clipboard`).
- **No notifications.** Install `libnotify-bin` (the `notify-send` command), and check that notifications are on in the settings.
- **No tray icon.** Enable the AppIndicator GNOME extension.
- **`Print` opens GNOME's screenshot tool.** Turn it off in the keyboard shortcuts (see step 3).

## How it works

Scrawl captures the screen through the `xdg-desktop-portal` ScreenCast interface
with its own small PipeWire client (`internal/pwcapture`). A saved restore token for each
monitor removes the dialog. The compositor (Mutter) delivers frames only when a monitor
repaints, and it delivered none over a fullscreen window in our tests (direct
scanout); if no frame arrives within 250 ms Scrawl falls back to the Screenshot portal,
which is slower (about 0.5-1 s). The overlays are fullscreen [Gio](https://gioui.org)
windows, one per monitor; Gio is patched (`third_party/gio.patch`) to place a window
on a chosen Wayland output. The result is rendered by the same code that draws it on screen,
so the saved image matches what you saw.

## Building and contributing

```bash
make build    # bin/scrawl; fetches and patches Gio on the first run
make test     # unit tests, including the tray icon on the session D-Bus
make dist     # release archive and checksum in dist/
make deb      # .deb package for Debian and Ubuntu in dist/
make notices  # regenerate THIRD_PARTY_NOTICES.md
```

The interface texts live in `internal/i18n/` (one file per language, `lang_ru.go`, `lang_de.go` and so on; the source texts in the code are English);
`go test ./...` checks that every text is translated into every language. How to report a bug and send a change: [CONTRIBUTING.md](CONTRIBUTING.md). Plan and ideas: [docs/PLAN.md](docs/PLAN.md).

## License

MIT, see [LICENSE](LICENSE). Third-party components: [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
(icon shapes are from [Tabler Icons](https://tabler.io/icons), MIT).
