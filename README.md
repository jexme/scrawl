# screen-editor

[Русская версия](README.ru.md)

A fast screenshot tool with annotations for **GNOME on Wayland**. Press a key,
select an area on any monitor, draw arrows, boxes, text, blur out sensitive
data, and the result goes to the clipboard and to a file. No "which screen do
you want to share?" dialog on every shot.

> Status: personal project under active development, not released yet.
> Linux (GNOME, Wayland) only for now; a Windows port is planned. The interface
> text (buttons, notifications, captions) is Russian only at the moment.

## Features

- Frozen screen on every monitor appears in about 150 ms after the hotkey.
- Selection with a dashed frame: drag the corners to resize; with the *select*
  tool drag any edge to resize, or drag inside to move.
- Tools: pen, line, arrow, box, marker, **blur**, text. Eight colors; the mouse
  wheel or the `−`/`+` buttons change thickness, text size or blur strength.
- **Before/after collections:** put shots aside, then glue them side by side with
  captions "Было"/"Стало" ("Was"/"Became", or "Шаг N" for three and more; captions
  are Russian only for now), or send them as separate files.
- Remembers the last area and brings it back with one key (`R`).
- Saves PNG to `~/Pictures/Screenshots` and copies the image to the clipboard.
- A separate settings window (gear button): for example, a double click inside
  the selection can copy it to the clipboard.

## Requirements

- GNOME on Wayland, PipeWire, and `xdg-desktop-portal` with host app registration
  (`org.freedesktop.host.portal.Registry`; developed and tested on 1.21 with GNOME
  50), plus `xdg-desktop-portal-gnome`.
- `wl-clipboard` (`wl-copy`) for the clipboard, `libnotify` (`notify-send`) for
  notifications.

Build dependencies: Go (version from `go.mod`), a C compiler, `pkg-config`,
`patch`, and development headers:

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
```

No `sudo`? `scripts/fetch-pipewire-headers.sh` unpacks the PipeWire headers into
`.deps/` and the Makefile picks them up.

## Install

```bash
make install            # to ~/.local (PREFIX=/usr/local for system-wide)
screen-editor -setup    # one-time: GNOME asks to pick each monitor
screen-editor -grant-screenshot
scripts/gnome-hotkey.sh # binds Print to screen-editor
```

- `-setup` shows one GNOME dialog **per monitor**: pick a different monitor each
  time and keep "Remember this choice" ticked. After that no dialogs appear.
- `-grant-screenshot` gives the app the permission it needs for the fallback
  capture path (see below). It writes one entry to the portal permission store.
- `scripts/gnome-hotkey.sh [KEY] [BINARY] [--dry-run]` creates a GNOME custom
  shortcut and refuses to overwrite one that is already taken.

## Usage

Press the hotkey, drag to select. Keys:

| Key | Action |
| --- | --- |
| `Enter` | save to file and copy |
| `Ctrl+C` / `Ctrl+S` | only copy / only save |
| `Ctrl+Z` | undo |
| `Esc` | cancel |
| `R` | restore the previous area |
| `A` / `G` / `D` | add to collection / glue collection / send collection as files |
| mouse wheel | thickness, text size or blur strength of the current tool |

Flags: `-v` prints timings, `-fresh` ignores the remembered area, `-setup` picks
monitors again, `-settings` opens the settings window.

Files: images in `~/Pictures/Screenshots`; state (monitor tokens, last area,
collection) in `~/.cache/screen-editor/`.

## Settings

Open the settings window with the gear button on the toolbar (it closes the
overlay and opens a separate window), with `screen-editor -settings`, or from the
application menu entry "Settings". Changes apply immediately. Settings are stored in
`~/.config/screen-editor/config.json`, which you can also edit by hand:

```json
{
  "double_click": "copy",
  "save_dir": "/home/me/Pictures/Shots"
}
```

| Key | Values | Meaning |
| --- | --- | --- |
| `double_click` | `off` (default), `copy`, `copy_save` | what a double click inside the selection does (with the *select* tool): nothing, copy to the clipboard, copy and save to a file |
| `save_dir` | a directory, empty by default | where PNG files go; empty means `~/Pictures/Screenshots` |

Missing or invalid values fall back to the defaults.

## How it works

The screen is captured through the `xdg-desktop-portal` ScreenCast interface
with a PipeWire client (`internal/pwcapture`). A saved restore token per monitor
removes the dialog. Mutter delivers frames only when the monitor repaints, and in
our tests it delivered none over a fullscreen window (direct scanout); when no
frame arrives within 250 ms the app falls back to the Screenshot portal (slower,
about 0.5–1 s). Overlays are
fullscreen [Gio](https://gioui.org) windows, one per monitor; Gio is patched
(`third_party/gio.patch`) to put a window on a chosen Wayland output.

## Troubleshooting

- Nothing happens on the hotkey: `xdg-desktop-portal` may be hung. Restart it:
  `systemctl --user restart xdg-desktop-portal xdg-desktop-portal-gnome`.
- A monitor was added or replaced: run `screen-editor -setup` again.
- The fallback path says the portal refused: run `screen-editor -grant-screenshot`.

## Building

```bash
make build   # bin/screen-editor; fetches and patches Gio on first run
make test
make notices # regenerate THIRD_PARTY_NOTICES.md
```

## License

MIT, see [LICENSE](LICENSE). Third-party components: [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
(icon shapes are from [Tabler Icons](https://tabler.io/icons), MIT).
