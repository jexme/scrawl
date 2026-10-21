# Contributing to Scrawl

Thanks for helping. Scrawl is a small personal project, so the rules are short.

## Reporting a bug

Open an issue and fill in the form. The most useful things to include:

- the version: `scrawl -version`;
- the session type: `echo $XDG_SESSION_TYPE` (`wayland` or `x11`), and the desktop and distribution;
- what you did, what you expected and what happened;
- the output of `scrawl -v` if the problem is a failed or slow shot.

Screen capture depends heavily on the desktop, the portal and the compositor, so "works on my GNOME" is
real information in both directions.

## Building and testing

```bash
sudo apt install gcc pkg-config patch libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev libpipewire-0.3-dev
make build     # bin/scrawl; fetches and patches Gio on the first run
make test      # unit tests; X11 tests need Xvfb and xclip, Wayland ones need nothing
GOOS=windows CGO_ENABLED=0 go vet ./...    # the Windows build must keep compiling
```

CI (`.github/workflows/ci.yml`) runs the same checks plus `gofmt`. Please run `gofmt -w` and `go vet ./...` before
sending a change.

## Maintainer: keep personal e-mail out of the history

`make hooks` turns on `.githooks/`: a pre-commit check and a pre-push check that refuse any author or committer
address that is not a `...@users.noreply.github.com` one. CI repeats the check on every push to `main`.

## Where things are

| Path | What it does |
| --- | --- |
| `cmd/scrawl` | the program: flags, choosing the capture path, starting the overlay |
| `internal/editor` | the overlay: selection, toolbar, annotations, clipboard, collections |
| `internal/screencast`, `internal/pwcapture` | capture through the portal and PipeWire (Wayland) |
| `internal/x11capture` | direct capture in X11 sessions |
| `internal/settings`, `internal/settingsui` | the settings file and the settings window |
| `internal/i18n`, `internal/fonts` | interface translations and fallback fonts for CJK |
| `internal/tray`, `internal/hotkey`, `internal/autostart`, `internal/update` | tray icon, GNOME hotkey, autostart, update check |
| `docs/` | plan, settings reference, Windows notes, README translations |

## Texts and translations

Texts in the code are written in English and wrapped in `i18n.T(...)`. The other eight languages, Russian
included, live in `internal/i18n/` (one file per language, `lang_ru.go`, `lang_de.go` and so on; the key is the English text). When you add or change a text, add the
translation to **every** catalog: `go test ./internal/i18n` fails if a language misses a text, has a leftover
entry, or changes the `%d`/`%s` placeholders. If you do not speak a language, put an English text there and say
so in the pull request. Native-speaker corrections to existing translations (also the README translations in
`docs/translations/`) are very welcome.

## Pull requests

- One topic per pull request, with a test when the behaviour can be tested without a real screen.
- Do not add dependencies lightly; every new module goes into `THIRD_PARTY_NOTICES.md` (`make notices`).
- Settings: a new key needs a default, validation in `normalize`, a row in the settings window and a line in
  `docs/SETTINGS.md` and `docs/translations/SETTINGS.ru.md`.
- Commit messages in this repository are written in Russian; English is fine for pull requests from outside.
