//go:build linux

package screencast

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

// scopeEnv marks a process that has already been restarted in its own scope: we do not do
// this again.
const scopeEnv = "SCRAWL_APP_SCOPE"

// inAppScope reports whether the process cgroup is the application's systemd scope for
// AppID of the form app-[<launcher>-]AppID-<number>.scope. This is how a portal older than
// 1.21 identifies the application.
func inAppScope(cgroup string) bool {
	for _, line := range strings.Split(cgroup, "\n") {
		base := path.Base(line[strings.LastIndex(line, ":")+1:])
		if strings.HasPrefix(base, "app-") && strings.HasSuffix(base, ".scope") && strings.Contains(base, "-"+AppID+"-") {
			return true
		}
	}
	return false
}

// EnterAppScope restarts the process in its own systemd scope app-AppID-PID.scope.
//
// It is needed for a portal older than 1.21: it has no Registry method and takes app_id
// from the scope of the calling process. A window launched by a GNOME hotkey sits in the
// gnome-shell service and stays anonymous to the portal: no persistence token is issued
// and the screen selection dialog is shown on every capture. On success the function does
// not return (the process is replaced), an error means "left as we were".
func (p *Portal) EnterAppScope() error {
	if !p.OldPortal || os.Getenv(scopeEnv) != "" {
		return nil
	}
	if b, err := os.ReadFile("/proc/self/cgroup"); err == nil && inAppScope(string(b)) {
		return nil
	}
	sd, err := exec.LookPath("systemd-run")
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	p.Close()
	args := append([]string{
		"systemd-run", "--user", "--scope", "--quiet",
		fmt.Sprintf("--unit=app-%s-%d.scope", AppID, os.Getpid()), "--", exe,
	}, os.Args[1:]...)
	err = syscall.Exec(sd, args, append(os.Environ(), scopeEnv+"=1"))
	return errors.Join(errors.New("systemd-run"), err)
}
