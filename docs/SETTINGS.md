# Settings reference

[Русский](SETTINGS.ru.md)

Almost everything is set in the settings window (`scrawl -settings`, the gear
button on the toolbar, or "Settings" in the application menu). Changes are saved
at once. Everything below is stored in one JSON file, which you can also edit by
hand:

| System | File |
| --- | --- |
| Linux | `~/.config/scrawl/config.json` (or `$XDG_CONFIG_HOME/scrawl/config.json`) |

Rules the file follows:

- Every key is optional. A missing or invalid value falls back to the default,
  and one bad value never spoils the others.
- Numbers outside the allowed range are clamped to it.
- Unknown keys are kept when Scrawl saves the file.
- The overlay reads the file at launch. The settings window applies changes to
  itself immediately; the overlay and notifications use them from the next shot.
- If the file does not exist yet, Scrawl reads `~/.config/screen-editor/config.json`
  (the old name of the program) and creates the new file at the first change.

Example:

```json
{
  "enter": "copy",
  "dim_percent": 60,
  "tools": { "marker": { "color": "#FDD835" } },
  "glue": { "before": "Before", "after": "After", "layout": "stack" }
}
```

## Actions and behaviour

| Key | Values (default first) | Meaning |
| --- | --- | --- |
| `enter` | `copy_save`, `copy`, `save` | what `Enter` does: copy and save, only copy, only save |
| `double_click` | `off`, `copy`, `copy_save` | what a double click inside the selection does (with the *select* tool) |
| `keep_open` | `false`, `true` | stay in the editor after copying or saving; leave with `Esc` |
| `notifications` | `true`, `false` | notifications after a shot; errors are always shown |
| `remember_region` | `true`, `false` | remember the last area, show it at launch, `R` restores it |
| `include_cursor` | `false`, `true` | draw the mouse cursor into the shot (normal PipeWire capture only: not the Screenshot-portal fallback and not X11) |
| `default_tool` | `select`, `pen`, `line`, `arrow`, `rect`, `marker`, `blur`, `text` | tool switched on as soon as an area is selected |
| `keys` | object | overlay keys, see below |
| `language` | `auto`, `ru`, `en`, `zh`, `es`, `pt`, `ja`, `de`, `fr`, `ko` | interface language (Russian, English, Chinese, Spanish, Portuguese, Japanese, German, French, Korean); `auto` follows the system locale. Regional forms such as `zh-CN` or `pt_BR` are accepted |
| `check_updates` | `false`, `true` | ask GitHub Releases for a new version when the settings window and the tray icon start. This is the only network access of the program; the "Check now" button works either way |

### Keys

`keys` maps an action to a key. A key is written like `R`, `Enter`, `F5`,
`Ctrl+Z` or `Ctrl+Shift+S` (modifiers: `Ctrl`, `Shift`, `Alt`; keys: letters,
digits, `F1`-`F12`, `Enter`, `Escape`, `Space`, `Tab`, `Backspace`, `Delete`,
`Home`, `End`, `PageUp`, `PageDown`, `Up`, `Down`, `Left`, `Right`). An empty
value means the default. Two actions cannot share a key: if they do, all keys
go back to the defaults.

| Action | Default | Meaning |
| --- | --- | --- |
| `confirm` | `Enter` | copy and/or save (see `enter`) |
| `cancel` | `Escape` | close the overlay |
| `restore` | `R` | bring back the area of the previous shot |
| `add` | `A` | add the shot to the collection |
| `glue` | `G` | glue the collection into one image |
| `send` | `D` | save the collection as separate files and put the file list in the clipboard |
| `undo` | `Ctrl+Z` | undo the last annotation |
| `copy` | `Ctrl+C` | only copy |
| `save` | `Ctrl+S` | only save |

## Files

| Key | Values (default first) | Meaning |
| --- | --- | --- |
| `save_dir` | empty | folder for files; empty means `~/Pictures/Screenshots`; `~/Folder` and absolute paths work |
| `file_prefix` | empty | start of the file name, then the date and time; empty means `scrawl` (for example `scrawl-20260131-142501.png`) |
| `format` | `png`, `jpeg` | file format; the clipboard always gets PNG |
| `jpeg_quality` | `90` (1-100) | JPEG quality |

Two shots in the same second never overwrite each other: `-2`, `-3` is added to the name.

## Tools and colours

| Key | Values (default first) | Meaning |
| --- | --- | --- |
| `palette` | list of 8 colours `#RRGGBB` | the colours on the toolbar (default: red, orange, yellow, green, blue, purple, white, black) |
| `tools` | object per tool | default colour and thickness of each tool: `{"arrow": {"color": "#E53935", "width": 6}}`. Tools: `pen`, `line`, `arrow`, `rect`, `marker`, `blur`, `text`. Thickness 1-30; for text it is the size, for blur the strength. The colour applies until you pick one on the toolbar yourself |
| `marker_opacity` | `43` (5-100) | marker opacity, % |

## Appearance of the selection

| Key | Values (default first) | Meaning |
| --- | --- | --- |
| `dim_percent` | `86` (0-100) | dimming outside the area, % |
| `border_width` | `1` (0.5-4) | frame line thickness, px |
| `border_dash` | `4.6` (1-20) | length of a dash, px |
| `border_gap` | `2` (0-20) | gap between dashes, px |
| `border_color` | `#FFFFFF` | frame colour, `#RRGGBB` |
| `handle_radius` | `3` (0-10) | radius of the corner handles, px; `0` hides them |
| `show_size` | `true`, `false` | show the "width × height" label above the area |

## Before/after collage (`glue`)

`glue` is an object:

| Key | Values (default first) | Meaning |
| --- | --- | --- |
| `before`, `after` | empty | captions for two images; empty means the default words of the interface language |
| `step` | empty | the word before the number when there are three or more images ("Step 1", "Step 2", …) |
| `layout` | `side`, `stack` | side by side (equal height) or one under another (equal width) |
| `background` | `light`, `dark` | background of the collage |
| `pad` | `28` (0-100) | margin around and between the images, px |
| `label_size` | `28` (12-64) | caption size, px |

Captions are up to 30 characters, without control characters.
