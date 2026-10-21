package editor

import "github.com/jexme/scrawl/internal/x11capture"

// isX11 reports that this is an X11 session. The variable exists for tests: the EGL
// platform choice also depends on WAYLAND_DISPLAY, so the environment cannot be replaced there.
var isX11 = x11capture.IsSession
