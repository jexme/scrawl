# Windows

Status: **works** on Windows 10 and 11: capture of all monitors, annotations, the clipboard, a notification-area
icon and a global hotkey. Tested on two monitors with different resolutions.
Not tested on monitors with a scale above 100% (see "What is left").

## Building and running

You need Go (the version from `go.mod`) and Git for Windows (it provides `sh` and `patch`).

```powershell
powershell -ExecutionPolicy Bypass -File scripts\build-windows.ps1
bin\scrawl.exe            # a double click: the notification-area icon and the hotkey
bin\scrawl.exe -shot      # one screenshot (what the icon menu and the hotkey run)
bin\scrawl.exe -tray      # the icon and the hotkey, explicitly
bin\scrawl.exe -settings  # settings window
```

On Windows a start without arguments runs the tray, because that is what a double click or the Start menu does. If a tray
is already running, a start without arguments takes one screenshot instead (on Linux it always does). The first time the
icon appears, a one-time balloon explains where Scrawl lives and which key takes a screenshot.

`scrawl.exe` is built as a GUI program (`-H=windowsgui`): there is no black console window. When started from a
terminal it attaches to that console, so `-version` and `-v` print as usual.

## How it works

| What | Where |
|---|---|
| Capture | `internal/wincapture`: `EnumDisplayMonitors`, `BitBlt` (`SRCCOPY\|CAPTUREBLT`) and `GetDIBits` for each monitor; the process declares itself Per-Monitor DPI aware V2 (`cmd/scrawl/dpi_windows.go`) |
| Overlay on the right monitor | the Gio patch `third_party/gio.patch` (`app/os_windows.go`): `FullscreenAt` moves the window to the monitor, puts it on top and takes focus; the window is exactly the size of the monitor and is not maximised (a maximised window would stop at the work area) |
| Clipboard | `internal/editor/clipboard_windows.go`: the `PNG` format (messengers, browsers) and `CF_DIB` (Paint, Word); several files as `CF_HDROP` |
| One instance | `internal/singleton`: a named mutex `Local\scrawl-<name>` |
| Icon and hotkey | `internal/tray/tray_windows.go`: `Shell_NotifyIcon` and `RegisterHotKey` in one process; the menu has "Take screenshot", "Settings" and "Quit"; the tooltip shows the version. If adding the icon fails (Explorer is not ready at logon), it is retried every 3 seconds and when `TaskbarCreated` arrives; the user is told only after 2 minutes |
| Second start of the tray | the named mutex is taken by the first one; a second `scrawl.exe -tray` shows "Scrawl is already running (version X). Quit it from the tray icon menu, then start the new version." (the running version is read from `%LocalAppData%\scrawl\tray-version`). A start with `-autostart`, or without arguments, stays silent |
| Key combination | the file `%AppData%\scrawl\hotkey` in the GTK format (`Print`, `<Control><Alt>a`); changed in the settings window, the icon picks it up within a couple of seconds |
| Autostart | `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, the value `Scrawl` (`"...\scrawl.exe" -tray -autostart`). When the tray starts and the value exists but points to another file, it is rewritten to the running exe (the exe was moved or updated); a disabled autostart is never switched on, and a path inside the temp folder is never written (`internal/autostart`) |
| Error notifications | a message box (`MessageBox`); a window that fails to open (a Gio `DestroyEvent` with an error) is reported through `internal/windowerr` instead of exiting silently |
| Fonts for Chinese, Japanese and Korean | system fonts from `%WINDIR%\Fonts` (YaHei, Yu Gothic, Malgun Gothic) |

The default hotkey is `Print`. Another program often holds it (Greenshot, the Snipping Tool): then the icon
shows a hint, and you pick another combination in `scrawl -settings`.

## Tests

The tests redirect the user folders through `internal/testenv` (on Windows these are `%AppData%` and
`%LocalAppData%`, not `XDG_*`). The `editor` tests with a fake `wl-copy` and the `install.sh` installer test are
skipped on Windows.

## Releasing

A tag `vX.Y.Z` (or `vX.Y.Z-rcN`, which is marked as a pre-release) starts `.github/workflows/release.yml`.
Next to the Linux packages it adds to the same release:

- `scrawl-vX.Y.Z-windows-amd64-setup.exe` and its `.sha256`: the installer, the recommended way to install (see below);
- `scrawl-vX.Y.Z-windows-amd64.zip`: `scrawl.exe`, the licenses, the README and this file;
- `scrawl-vX.Y.Z-windows-amd64.zip.sha256`;
- `scrawl-vX.Y.Z-windows-amd64.exe`: the same `scrawl.exe` on its own, for those who do not want an archive;
- `scrawl-vX.Y.Z-windows-amd64.exe.sha256`;
- `scrawl.json`: a Scoop manifest with the ready URL and hash.

### The installer

`packaging/windows/scrawl.iss` is an [Inno Setup](https://jrsoftware.org/isinfo.php) 6 script. The installer works per
user (`PrivilegesRequired=lowest`, no UAC prompt) and installs to `%LocalAppData%\Programs\Scrawl`. It adds a Start menu
entry that runs `scrawl.exe -tray` (and one for the settings), an optional task "Start Scrawl when I sign in" (the same
`HKCU\...\Run` value as the program writes), and offers to start Scrawl at the end. Before an upgrade, and on uninstall,
it stops any running `scrawl.exe` (`taskkill`, first politely, then forced); uninstalling also removes the autostart value
and leaves the settings in `%AppData%\scrawl`. Build it locally after `scripts\build-windows.ps1`:

```powershell
iscc /DAppVersion=0.1.0-rc3 /DNumVersion=0.1.0.0 packaging\windows\scrawl.iss   # dist\scrawl-v0.1.0-rc3-windows-amd64-setup.exe
```

**The installer step is not tested.** It was written without a Windows machine and has never run on a real runner. The
"Installer (Inno Setup)" step of the `windows` job in `release.yml` is therefore marked `continue-on-error: true`: if
`iscc` is missing (the step then runs `choco install innosetup -y`), the script has an error or anything else goes wrong,
the release is still published, only without the setup file (the upload and attestation steps skip it). After the first
release with it, check by hand: the step went green, a clean user can install without UAC, the Start menu entry starts the
icon, upgrading over a running tray works, uninstall removes the files and the autostart value. When it has proved itself,
remove `continue-on-error`.

SmartScreen shows "Windows protected your PC" for the installer too, until it is signed (see "Code signing"): *More info*,
then *Run anyway*.

### The archive and the bare exe

The archive is built on `windows-latest` by the same `scripts/build-windows.ps1 -Package` as locally, and comes with
a provenance attestation (`gh attestation verify scrawl-*.zip --repo jexme/scrawl`; the same for the bare exe and the setup). `scrawl.exe` has the icon
(`packaging/windows/scrawl.ico`, regenerated by `go run ./scripts/genicon`), the manifest `scrawl.manifest`
(Per-Monitor V2, long paths, `asInvoker`) and the file version from the tag built in.

CI (`ci.yml`, the `windows` job) builds the program with the same script on every push, runs `go vet`,
`go test ./...` and `scrawl.exe -version`.

### Code signing

There is none yet, and it is the main source of complaints: SmartScreen shows "Windows protected your PC", and
antivirus programs are more suspicious of unsigned Go programs that take screenshots, hold a global key and write to
autostart. Options:

- **SignPath Foundation**: free for open-source projects, an application is needed; signing is called as a workflow
  step after the build;
- **an open-source code-signing certificate** (for example Certum): inexpensive, valid for a year, the key lives on a
  token or in the cloud;
- **Azure Trusted Signing**: availability depends on the country and the status of the applicant.

The signature has to go on `scrawl.exe` before it is packed into the zip and the installer (and on the setup exe itself). SmartScreen reputation builds up
slowly, but with a signature the warnings and false positives go away sooner. Until then: the hash and the attestation
in the release, send the file as a false positive to Microsoft Defender with the first release, and do not pack the
`.exe` with packers.

### winget

`winget install` needs a manifest in [winget-pkgs](https://github.com/microsoft/winget-pkgs). The easiest way, after a
release: `wingetcreate new https://github.com/jexme/scrawl/releases/download/vX.Y.Z/scrawl-vX.Y.Z-windows-amd64.zip`
(type `portable`, command `scrawl`); it opens the pull request itself. For the next versions: `wingetcreate update`.

### What to check by hand before a release

CI cannot open the overlay or press keys, so before releasing:

- a scale of 125-200% and two monitors with different scales: toolbar sizes, the magnifier, the size label;
- pasting a shot into Telegram, Discord, a browser, Word, Paint, Slack;
- Windows 10 and 11 on a clean machine with neither Go nor Git;
- the Print key on a fresh Windows 11 ("Use the Print screen button to open screen capture" is on in Settings);
- the cursor leaves one monitor for another: the magnifier stays only where the cursor is;
- autostart: tick the box, sign out, sign in, the icon appears (and with a slow logon, still appears without an error);
- a double click on `scrawl.exe` starts the icon and shows the one-time balloon; a second double click takes a screenshot;
- starting a newer `scrawl.exe -tray` while an older one runs shows the "already running (version X)" message;
- moving the exe with autostart on: the next start of the icon rewrites the `Run` value;
- the installer: install, upgrade over a running tray, uninstall (see "The installer").

## What is left

- **A scale above 100%.** Capture and windows work in physical pixels, but the sizes of the toolbar and the labels
  on monitors with a scale of 125-200% have not been looked at.
- **Notifications.** The "saved" message is shown inside the overlay; Windows toasts (WinRT) are not done, the only
  system notification is the hint at the icon when the combination is taken.
- **Games in exclusive full-screen mode and protected video.** GDI may return a black frame; a fallback through
  Desktop Duplication (DXGI) or Windows.Graphics.Capture is not written.
- **The mouse cursor in the shot** (`include_cursor`) is not supported on Windows.
- **Code signing.** There is no signature yet (see "Releasing"); the installer is built by CI but has not been tried on a real
  Windows. There is no MSI, and no winget or Scoop-bucket package yet.
