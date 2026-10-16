package editor

import "github.com/jexme/scrawl/internal/x11capture"

// isX11 сообщает, что это сеанс X11. Переменная нужна тестам: от WAYLAND_DISPLAY
// зависит и выбор платформы EGL, поэтому подменять окружение там нельзя.
var isX11 = x11capture.IsSession
