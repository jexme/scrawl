# Scrawl

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
scrawl -setup    # one-time: GNOME asks to pick each monitor
scrawl -grant-screenshot
scripts/gnome-hotkey.sh # binds Print to scrawl
```

- `-setup` shows one GNOME dialog **per monitor**: pick a different monitor each
  time and keep "Remember this choice" ticked. After that no dialogs appear.
- `-grant-screenshot` gives the app the permission it needs for the fallback
  capture path (see below). It writes one entry to the portal permission store.
- `-version` prints the build version and exits (also shown in the settings window).
- `scripts/gnome-hotkey.sh [KEY] [BINARY] [--dry-run]` creates a GNOME custom
  shortcut and refuses to overwrite one that is already taken.

### Upgrading from screen-editor

Scrawl is the new name of screen-editor. Settings are picked up from
`~/.config/screen-editor/config.json` until `~/.config/scrawl/config.json`
exists (the old file is left untouched). The portal ties monitor choices and
the screenshot permission to the app id, which changed, so run
`scrawl -setup` and `scrawl -grant-screenshot` once more and point your hotkey
at `scrawl` (`scripts/gnome-hotkey.sh`). The last area and the collection start
fresh.

## Usage

Press the hotkey, drag to select. Keys:

| Key | Action |
| --- | --- |
| `Enter` | save to file and copy (configurable in settings) |
| `Ctrl+C` / `Ctrl+S` | only copy / only save |
| `Ctrl+Z` | undo |
| `Esc` | cancel |
| `R` | restore the previous area |
| `A` / `G` / `D` | add to collection / glue collection / send collection as files |
| mouse wheel | thickness, text size or blur strength of the current tool |

Flags: `-v` prints timings, `-fresh` ignores the remembered area, `-setup` picks
monitors again, `-settings` opens the settings window.

Files: images in `~/Pictures/Screenshots`; state (monitor tokens, last area,
collection) in `~/.cache/scrawl/`.

## Settings

Open the settings window with the gear button on the toolbar (it closes the
overlay and opens a separate window), with `scrawl -settings`, or from the
application menu entry "Settings". Changes apply immediately. Settings are stored in
`~/.config/scrawl/config.json`, which you can also edit by hand:

```json
{
  "enter": "copy",
  "dim_percent": 60,
  "tools": { "marker": { "color": "#FDD835" } },
  "glue": { "before": "Before", "after": "After" }
}
```

| Key | Values | Meaning |
| --- | --- | --- |
| `double_click` | `off` (default), `copy`, `copy_save` | what a double click inside the selection does (with the *select* tool) |
| `enter` | `copy_save` (default), `copy`, `save` | what `Enter` does |
| `keep_open` | `false` | stay in the editor after copying or saving (leave with `Esc`) |
| `notifications` | `true` | show notifications after a shot; errors are always shown |
| `remember_region` | `true` | remember the last area and offer it on the next run |
| `include_cursor` | `false` | draw the mouse cursor into the shot (PipeWire capture only) |
| `save_dir` | directory, empty by default | where files go; empty means `~/Pictures/Screenshots` |
| `file_prefix` | text, empty by default | start of the file name, then date and time; empty means `scrawl` |
| `format` | `png` (default), `jpeg` | file format; the clipboard always gets PNG |
| `jpeg_quality` | 1-100, `90` | JPEG quality |
| `default_tool` | `select` (default), `pen`, `line`, `arrow`, `rect`, `marker`, `blur`, `text` | tool switched on once an area is selected |
| `tools` | object per tool: `{"arrow": {"color": "#E53935", "width": 6}}` | default colour (`#RRGGBB`) and thickness per tool; applies until you pick a colour in the toolbar |
| `palette` | list of 8 colours `#RRGGBB` | colours in the toolbar |
| `marker_opacity` | 5-100, `43` | marker opacity, % |
| `dim_percent` | 0-100, `86` | dimming outside the area, % |
| `border_width` | 0.5-4, `1` | selection frame: line thickness, px |
| `border_dash` | 1-20, `4.6` | selection frame: dash length, px |
| `border_gap` | 0-20, `2` | selection frame: gap between dashes, px |
| `border_color` | `#RRGGBB`, `#FFFFFF` | selection frame colour |
| `handle_radius` | 0-10, `3` | corner handle radius, px; `0` hides the handles |
| `show_size` | `true` | show the "W × H" label |
| `glue` | object: `before`, `after`, `step`, `layout`, `background`, `pad`, `label_size` | before/after collage: captions (empty means "Было", "Стало", "Шаг"), `layout` `side` (default) or `stack`, `background` `light` (default) or `dark`, `pad` 0-100 (`28`), `label_size` 12-64 (`28`) |

Missing or invalid values fall back to the defaults; numbers outside the allowed range are clamped. Unknown keys are kept when the file is saved.

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
- A monitor was added or replaced: run `scrawl -setup` again.
- The fallback path says the portal refused: run `scrawl -grant-screenshot`.

## Building

```bash
make build   # bin/scrawl; fetches and patches Gio on first run
make test
make notices # regenerate THIRD_PARTY_NOTICES.md
```

## License

MIT, see [LICENSE](LICENSE). Third-party components: [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
(icon shapes are from [Tabler Icons](https://tabler.io/icons), MIT).
