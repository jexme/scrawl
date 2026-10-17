//go:build linux

package screencast

import "testing"

func TestInAppScope(t *testing.T) {
	const pre = "0::/user.slice/user-1000.slice/user@1000.service/"
	for _, tc := range []struct {
		cgroup string
		want   bool
	}{
		{pre + "app.slice/app-io.github.jexme.Scrawl-4242.scope\n", true},
		{pre + "app.slice/app-gnome-io.github.jexme.Scrawl-4242.scope\n", true},
		{pre + "app.slice/app-com.anthropic.Claude-31542.scope\n", false},
		{pre + "session.slice/org.gnome.Shell@wayland.service\n", false},
		{pre + "app.slice/io.github.jexme.Scrawl-1.scope\n", false},
		{"", false},
	} {
		if got := inAppScope(tc.cgroup); got != tc.want {
			t.Errorf("%q: %v, ожидали %v", tc.cgroup, got, tc.want)
		}
	}
}
