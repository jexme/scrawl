// Package update checks whether a new version has been released, by querying GitHub Releases.
// This is the only place where the program goes online: the check is off
// by default, is run by a button or (if enabled) at tray icon and settings window startup,
// and sends nothing except a User-Agent header with the version.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jexme/scrawl/internal/i18n"
)

// Repo is the repository with releases.
const Repo = "jexme/scrawl"

// apiURL is the list of releases, newest first; replaced in tests. The list is used instead
// of /releases/latest because GitHub does not return pre-releases there, and every release of
// the project is a pre-release until v0.1.0.
var apiURL = "https://api.github.com/repos/" + Repo + "/releases?per_page=30"

// Result is the outcome of a check.
type Result struct {
	Latest     string // tag of the latest release, for example "v0.2.0"
	URL        string // release page
	Newer      bool   // Latest is newer than the current version
	Comparable bool   // the current version could be compared (developer builds have none)
}

type release struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
}

// Check requests the releases and compares the best one with the current version current.
// Pre-releases count only for a build that is itself a pre-release or a developer build:
// somebody running v0.1.0 is not offered v0.2.0-rc1.
func Check(ctx context.Context, current string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "scrawl/"+current)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf(i18n.T("cannot reach GitHub: %w"), err)
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return Result{}, fmt.Errorf("%s", i18n.T("there are no releases yet"))
	case resp.StatusCode != http.StatusOK:
		return Result{}, fmt.Errorf(i18n.T("GitHub replied with code %d"), resp.StatusCode)
	}
	var rels []release
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 1<<20)).Decode(&rels); err != nil {
		return Result{}, fmt.Errorf(i18n.T("unexpected reply from GitHub: %w"), err)
	}
	if len(rels) == 0 {
		return Result{}, fmt.Errorf("%s", i18n.T("there are no releases yet"))
	}
	cur, curOK := parse(current)
	withPre := !curOK || cur.pre != ""
	var best *release
	tagged := false
	for i := range rels {
		r := &rels[i]
		if r.Draft || r.TagName == "" {
			continue
		}
		tagged = true
		if _, ok := parse(r.TagName); !ok || (r.Prerelease && !withPre) {
			continue
		}
		if best == nil {
			best = r
		} else if newer, _ := Newer(r.TagName, best.TagName); newer {
			best = r
		}
	}
	if !tagged {
		return Result{}, fmt.Errorf("%s", i18n.T("the GitHub reply has no version"))
	}
	if best == nil {
		// Only pre-releases exist and this build is a final release: nothing newer.
		return Result{Latest: current, Comparable: true}, nil
	}
	res := Result{Latest: best.TagName, URL: best.HTMLURL}
	if newer, ok := Newer(best.TagName, current); ok {
		res.Newer, res.Comparable = newer, true
	}
	return res, nil
}

// version is a parsed version vX.Y.Z[-prerelease].
type version struct {
	nums [3]int
	pre  string
}

func parse(v string) (version, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, _, _ = strings.Cut(v, "+") // build metadata is not important
	core, pre, _ := strings.Cut(v, "-")
	parts := strings.Split(core, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return version{}, false
	}
	var out version
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return version{}, false
		}
		out.nums[i] = n
	}
	out.pre = pre
	return out, true
}

// Newer reports that version latest is newer than current. The second value is false if
// one of the versions cannot be parsed (for example, a developer build "dev-1bff227").
// A prerelease (v1.0.0-rc1) is older than the release (v1.0.0) with the same numbers.
func Newer(latest, current string) (newer, ok bool) {
	a, okA := parse(latest)
	b, okB := parse(current)
	if !okA || !okB {
		return false, false
	}
	for i := range a.nums {
		if a.nums[i] != b.nums[i] {
			return a.nums[i] > b.nums[i], true
		}
	}
	switch {
	case a.pre == b.pre:
		return false, true
	case a.pre == "":
		return true, true // a release is newer than its own prerelease
	case b.pre == "":
		return false, true
	}
	return comparePre(a.pre, b.pre) > 0, true
}

// comparePre compares pre-release suffixes part by part (the parts are separated by dots);
// inside a part digit runs compare by value, so "rc10" is newer than "rc9". When one list
// of parts is a prefix of the other, the shorter one is older.
func comparePre(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		if c := compareNatural(pa[i], pb[i]); c != 0 {
			return c
		}
	}
	return len(pa) - len(pb)
}

// compareNatural compares two strings, taking runs of digits as numbers.
func compareNatural(a, b string) int {
	for a != "" && b != "" {
		ra, rb := splitRun(a), splitRun(b)
		da, db := isDigit(ra[0]), isDigit(rb[0])
		switch {
		case da && db:
			na, nb := strings.TrimLeft(ra, "0"), strings.TrimLeft(rb, "0")
			if len(na) != len(nb) {
				return len(na) - len(nb)
			}
			if c := strings.Compare(na, nb); c != 0 {
				return c
			}
		case da != db:
			if da {
				return -1 // numbers sort before letters, as in semver
			}
			return 1
		default:
			if c := strings.Compare(ra, rb); c != 0 {
				return c
			}
		}
		a, b = a[len(ra):], b[len(rb):]
	}
	return len(a) - len(b)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// splitRun returns the leading run of s made of digits only or of non-digits only.
func splitRun(s string) string {
	d := isDigit(s[0])
	i := 1
	for i < len(s) && isDigit(s[i]) == d {
		i++
	}
	return s[:i]
}

// InstallHint tells how to update a copy of the program that runs from the path exe on the
// operating system goos, or returns "" when the way of installing is not recognised.
func InstallHint(exe, goos string) string {
	exe = strings.ReplaceAll(exe, `\`, "/") // Windows paths, also when checked on another system
	switch goos {
	case "linux":
		switch {
		case strings.HasPrefix(exe, "/usr/bin/"):
			return i18n.T("To update: sudo apt update && sudo apt install --only-upgrade scrawl")
		case strings.Contains(exe, "/.local/bin/"):
			return i18n.T("To update: download the new archive and run its install.sh")
		}
	case "windows":
		switch {
		case strings.Contains(strings.ToLower(exe), "/scoop/"):
			return i18n.T("To update: scoop update scrawl (or install again from the new manifest address)")
		default:
			return i18n.T("To update: download the new installer or zip from the releases page; quit Scrawl from the tray icon menu first")
		}
	}
	return ""
}

// Message describes the result of a check in words for the window and notification.
func (r Result) Message() string {
	switch {
	case r.Newer:
		msg := fmt.Sprintf(i18n.T("A new version %s is available: %s"), r.Latest, r.URL)
		if exe, err := os.Executable(); err == nil {
			if hint := InstallHint(exe, runtime.GOOS); hint != "" {
				msg += "\n" + hint
			}
		}
		return msg
	case r.Comparable:
		return fmt.Sprintf(i18n.T("You have the latest version (%s)"), r.Latest)
	default:
		return fmt.Sprintf(i18n.T("Latest version on GitHub: %s (you run a developer build, cannot compare)"), r.Latest)
	}
}
