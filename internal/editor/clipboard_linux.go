//go:build linux

package editor

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"os/exec"

	"github.com/jexme/scrawl/internal/i18n"
)

// clipboardTool chooses the clipboard program for the session: wl-copy on Wayland,
// xclip on X11. Returns the command name and arguments for the data type.
//
// If Wayland has no wl-copy but has xclip and an X server (XWayland is nearly everywhere),
// use xclip: the compositor syncs the X11 clipboard with the Wayland one, and pasting
// works in both worlds. With neither program an error is returned with a hint on what to install.
func clipboardTool(mime string) (name string, args []string, err error) {
	xclip := []string{"-selection", "clipboard", "-t", mime}
	if !isX11() {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return "wl-copy", []string{"--type", mime}, nil
		}
		if _, err := exec.LookPath("xclip"); err == nil && os.Getenv("DISPLAY") != "" {
			return "xclip", xclip, nil
		}
		return "", nil, errors.New(i18n.T("the clipboard in Wayland needs wl-copy: sudo apt install wl-clipboard"))
	}
	if _, err := exec.LookPath("xclip"); err != nil {
		return "", nil, errors.New(i18n.T("the X11 clipboard needs xclip: sudo apt install xclip"))
	}
	return "xclip", xclip, nil
}

// clipboardCopy puts data on the clipboard. Both wl-copy and xclip go into the
// background on their own and hold the clipboard until it is replaced.
func clipboardCopy(mime string, data []byte) error {
	name, args, err := clipboardTool(mime)
	if err != nil {
		return err
	}
	cp := exec.Command(name, args...)
	cp.Stdin = bytes.NewReader(data)
	return cp.Run()
}

// copyPNG puts a PNG on the clipboard.
func copyPNG(data []byte) error { return clipboardCopy("image/png", data) }

// copyFiles puts a list of files (text/uri-list) on the clipboard: this way several
// images paste into a chat or an email as separate attachments.
func copyFiles(paths []string) error {
	var list bytes.Buffer
	for _, p := range paths {
		list.WriteString((&url.URL{Scheme: "file", Path: p}).String())
		list.WriteString("\r\n")
	}
	return clipboardCopy("text/uri-list", list.Bytes())
}
