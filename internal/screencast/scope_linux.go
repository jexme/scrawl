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

// scopeEnv помечает процесс, уже перезапущенный в своём scope: повторно этого не делаем.
const scopeEnv = "SCRAWL_APP_SCOPE"

// inAppScope сообщает, что cgroup процесса — systemd-scope приложения AppID
// вида app-[<лаунчер>-]AppID-<номер>.scope. Так portal старше 1.21 узнаёт приложение.
func inAppScope(cgroup string) bool {
	for _, line := range strings.Split(cgroup, "\n") {
		base := path.Base(line[strings.LastIndex(line, ":")+1:])
		if strings.HasPrefix(base, "app-") && strings.HasSuffix(base, ".scope") && strings.Contains(base, "-"+AppID+"-") {
			return true
		}
	}
	return false
}

// EnterAppScope перезапускает процесс в своём systemd-scope app-AppID-PID.scope.
//
// Нужна portal старше 1.21: метода Registry в нём нет, и app_id он берёт из scope
// вызывающего процесса. Окно, запущенное хоткеем GNOME, сидит в службе gnome-shell и
// остаётся для portal безымянным: токен сохранения не выдаётся и диалог выбора экрана
// показывается при каждом снимке. При успехе функция не возвращается (процесс
// заменяется), ошибка значит «остались как были».
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
