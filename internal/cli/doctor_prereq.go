package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fluxinc/my-cli/internal/harness"
)

// doctorPrereqs reports the machine-level prerequisites a fresh install needs
// before any manifest work: git, the GitHub CLI (required for github.com
// manifests and mounts), the my binary directory on PATH, and at least one
// supported harness. Every non-ok line carries remediation so a new operator
// can fix it without reading docs.
func (a app) doctorPrereqs() []doctorItem {
	lookPath := a.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	var items []doctorItem

	if path, err := lookPath("git"); err == nil {
		items = append(items, doctorItem{Name: "git", Status: "ok", Path: path})
	} else {
		items = append(items, doctorItem{Name: "git", Status: "error", Message: "git is required; install it with " + installHint("git")})
	}

	if path, err := lookPath("gh"); err == nil {
		item := doctorItem{Name: "gh", Status: "ok", Path: path}
		if a.ghLoggedIn(path) {
			item.Message = "logged in"
		} else {
			item.Status = "warning"
			item.Message = "not logged in; run `gh auth login` (required for private github.com manifests and mounts)"
		}
		items = append(items, item)
	} else {
		items = append(items, doctorItem{Name: "gh", Status: "warning", Message: "GitHub CLI not installed; needed for github.com manifests and mounts. Install with " + installHint("gh") + " then run `gh auth login`"})
	}

	items = append(items, doctorBinaryOnPath())

	var found []string
	for _, h := range harness.All() {
		for _, cmd := range h.CommandCandidates() {
			if _, err := lookPath(cmd); err == nil {
				found = append(found, string(h))
				break
			}
		}
	}
	if len(found) != 0 {
		items = append(items, doctorItem{Name: "harness", Status: "ok", Message: "installed: " + strings.Join(found, ", ")})
	} else {
		items = append(items, doctorItem{Name: "harness", Status: "warning", Message: "no supported harness CLI found (" + harness.Names() + "); install one, then run `my onboarding`"})
	}
	return items
}

func (a app) ghLoggedIn(ghPath string) bool {
	if a.publishRunner != nil {
		_, err := a.publishRunner("gh", "auth", "token")
		return err == nil
	}
	cmd := exec.Command(ghPath, "auth", "token")
	cmd.Stdout = nil
	return cmd.Run() == nil
}

// doctorBinaryOnPath checks that the directory holding the running my binary
// is on PATH, so new shells can find `my` after the installer exits.
func doctorBinaryOnPath() doctorItem {
	exe, err := os.Executable()
	if err != nil {
		return doctorItem{Name: "path", Status: "unknown", Message: err.Error()}
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	if strings.HasPrefix(dir, os.TempDir()) || strings.Contains(dir, string(filepath.Separator)+"go-build") {
		return doctorItem{Name: "path", Status: "ok", Path: dir, Message: "temporary build; PATH check skipped"}
	}
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if entry == "" {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(entry); err == nil {
			entry = resolved
		}
		if filepath.Clean(entry) == dir {
			return doctorItem{Name: "path", Status: "ok", Path: dir}
		}
	}
	return doctorItem{Name: "path", Status: "warning", Path: dir, Message: "my binary directory is not on PATH; add `export PATH=\"" + dir + ":$PATH\"` to your shell rc file"}
}

func installHint(tool string) string {
	switch runtime.GOOS {
	case "darwin":
		return "`brew install " + tool + "`"
	case "linux":
		return "your package manager (for example `sudo apt install " + tool + "`)"
	default:
		return "your package manager"
	}
}
