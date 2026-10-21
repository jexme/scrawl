// Package hotkey назначает горячую клавишу запуска через пользовательские
// ярлыки GNOME (gsettings). Меняется только ярлык самой программы; чужие ярлыки
// не трогаются, а занятая ими комбинация приводит к ошибке.
package hotkey

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jexme/scrawl/internal/i18n"
)

const (
	schema = "org.gnome.settings-daemon.plugins.media-keys"
	base   = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/"
	// ShortcutName — имя ярлыка в настройках клавиатуры GNOME.
	ShortcutName = "scrawl"
)

// Встроенные ярлыки GNOME, с которыми комбинация может пересекаться.
var builtin = [][2]string{
	{"org.gnome.shell.keybindings", "show-screenshot-ui"},
	{"org.gnome.shell.keybindings", "screenshot"},
	{"org.gnome.shell.keybindings", "screenshot-window"},
	{"org.gnome.shell.keybindings", "show-screen-recording-ui"},
}

// gsettings выполняет команду gsettings и возвращает её вывод. В тестах подменяется.
var gsettings = func(args ...string) (string, error) {
	out, err := exec.Command("gsettings", args...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return "", errors.New(text)
	}
	return text, nil
}

var accelRe = regexp.MustCompile(`(?i)^(<(Super|Control|Primary|Shift|Alt|Mod1|Ctrl)>)*[A-Za-z0-9_]+$`)

// ValidateAccel проверяет запись комбинации в формате GTK: Print, <Super>s,
// <Control><Alt>p.
func ValidateAccel(accel string) error {
	if !accelRe.MatchString(accel) {
		return errors.New(i18n.T("use the GNOME format: Print, <Super>s, <Control><Alt>p"))
	}
	return nil
}

// Result описывает итог назначения.
type Result struct {
	Updated bool   // обновлён существующий ярлык программы, а не создан новый
	Warning string // комбинация совпала со встроенным ярлыком GNOME
}

// Conflict — комбинация занята чужим пользовательским ярлыком.
type Conflict struct{ Name, Accel string }

func (c *Conflict) Error() string {
	return fmt.Sprintf(i18n.T("the combination %s is already taken by the shortcut \"%s\": free it in the keyboard settings"), c.Accel, c.Name)
}

type shortcut struct{ path, name, command, binding string }

// parseStrings разбирает список строк в записи GVariant: ['a', 'b'] или @as [].
func parseStrings(v string) []string {
	var out []string
	for i := 0; i < len(v); i++ {
		if v[i] != '\'' {
			continue
		}
		var sb strings.Builder
		for i++; i < len(v) && v[i] != '\''; i++ {
			if v[i] == '\\' && i+1 < len(v) {
				i++
			}
			sb.WriteByte(v[i])
		}
		out = append(out, sb.String())
	}
	return out
}

// quote записывает строку в виде GVariant.
func quote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}

func quoteList(items []string) string {
	q := make([]string, len(items))
	for i, it := range items {
		q[i] = quote(it)
	}
	return "[" + strings.Join(q, ", ") + "]"
}

func (s shortcut) key() string { return schema + ".custom-keybinding:" + s.path }

func listShortcuts() ([]shortcut, error) {
	out, err := gsettings("get", schema, "custom-keybindings")
	if err != nil {
		return nil, fmt.Errorf("gsettings: %w", err)
	}
	var list []shortcut
	for _, path := range parseStrings(out) {
		sc := shortcut{path: path}
		for _, f := range []struct {
			key string
			dst *string
		}{{"name", &sc.name}, {"command", &sc.command}, {"binding", &sc.binding}} {
			v, err := gsettings("get", sc.key(), f.key)
			if err != nil {
				return nil, fmt.Errorf("gsettings: %w", err)
			}
			if s := parseStrings(v); len(s) > 0 {
				*f.dst = s[0]
			}
		}
		list = append(list, sc)
	}
	return list, nil
}

// ours сообщает, что ярлык принадлежит этой программе (в том числе под прежним названием).
func (s shortcut) ours() bool {
	exe := filepath.Base(strings.Fields(s.command + " ")[0])
	return s.name == ShortcutName || exe == "scrawl" || exe == "screen-editor"
}

func sameAccel(a, b string) bool { return strings.EqualFold(a, b) }

// Current возвращает комбинацию ярлыка программы или "", если ярлыка нет.
func Current() string {
	if onWindows {
		return winCurrent()
	}
	list, err := listShortcuts()
	if err != nil {
		return ""
	}
	for _, sc := range list {
		if sc.ours() {
			return sc.binding
		}
	}
	return ""
}

// Assign назначает комбинацию accel на запуск bin. Есть ярлык программы:
// он обновляется. Нет: создаётся новый. Чужой ярлык с той же комбинацией даёт *Conflict.
func Assign(accel, bin string) (Result, error) {
	if onWindows {
		return winAssign(accel)
	}
	var res Result
	if err := ValidateAccel(accel); err != nil {
		return res, err
	}
	list, err := listShortcuts()
	if err != nil {
		return res, err
	}
	var mine *shortcut
	for i, sc := range list {
		if sc.ours() {
			mine = &list[i]
			continue
		}
		if sameAccel(sc.binding, accel) {
			return res, &Conflict{Name: sc.name, Accel: accel}
		}
	}
	for _, b := range builtin {
		if out, err := gsettings("get", b[0], b[1]); err == nil {
			for _, other := range parseStrings(out) {
				if sameAccel(other, accel) {
					res.Warning = fmt.Sprintf(i18n.T("the combination %s is also used by a built-in GNOME shortcut (%s)"), accel, b[1])
				}
			}
		}
	}

	var newList []string // список ярлыков с новым ярлыком; пусто, если обновляем свой
	if mine != nil {
		res.Updated = true
	} else {
		taken := map[string]bool{}
		for _, sc := range list {
			taken[sc.path] = true
			newList = append(newList, sc.path)
		}
		n := 0
		for taken[fmt.Sprintf("%scustom%d/", base, n)] {
			n++
		}
		mine = &shortcut{path: fmt.Sprintf("%scustom%d/", base, n)}
		newList = append(newList, mine.path)
	}
	for _, f := range [][2]string{{"name", ShortcutName}, {"command", bin}, {"binding", accel}} {
		if _, err := gsettings("set", mine.key(), f[0], quote(f[1])); err != nil {
			return res, fmt.Errorf("gsettings: %w", err)
		}
	}
	// Ярлык попадает в список последним: пока поля не заполнены, GNOME его не видит.
	if newList != nil {
		if _, err := gsettings("set", schema, "custom-keybindings", quoteList(newList)); err != nil {
			return res, fmt.Errorf("gsettings: %w", err)
		}
	}
	return res, nil
}

// PreferredBinary возвращает путь, который стоит прописать в ярлыке: постоянный
// ~/.local/bin/scrawl, если он ведёт на запущенный бинарник, иначе сам бинарник.
func PreferredBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if home, err := os.UserHomeDir(); err == nil {
		link := filepath.Join(home, ".local", "bin", "scrawl")
		a, errA := filepath.EvalSymlinks(link)
		b, errB := filepath.EvalSymlinks(exe)
		if errA == nil && errB == nil && a == b {
			return link, nil
		}
	}
	return exe, nil
}
