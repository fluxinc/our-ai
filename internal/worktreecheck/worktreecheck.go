// Package worktreecheck inventories linked Git worktrees belonging to My AI
// managed checkouts. It reports raw harness-created worktrees without taking
// ownership of their lifecycle.
package worktreecheck

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ClassBase                      = "base"
	ClassRegisteredActive          = "registered-active"
	ClassRegisteredFinishedResidue = "registered-finished-residue"
	ClassPrunable                  = "prunable"
	ClassLeftover                  = "leftover"
)

// Checkout is one exact umbrella checkout whose Git common directory should
// be scanned. Callers choose the managed scope; Inspect never walks the disk.
type Checkout struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Path string `json:"path"`
}

// Registration binds a worktree path to a My AI session registry record.
type Registration struct {
	SessionID string
	Status    string
	Path      string
}

// Entry is one worktree record returned by Git.
type Entry struct {
	RepoID          string   `json:"repo_id"`
	RepoKind        string   `json:"repo_kind,omitempty"`
	RepoPath        string   `json:"repo_path"`
	CommonDir       string   `json:"common_dir"`
	Path            string   `json:"path"`
	Class           string   `json:"class"`
	SessionID       string   `json:"session_id,omitempty"`
	Head            string   `json:"head,omitempty"`
	BaseHead        string   `json:"base_head,omitempty"`
	Branch          string   `json:"branch,omitempty"`
	Detached        bool     `json:"detached,omitempty"`
	Locked          bool     `json:"locked,omitempty"`
	LockedReason    string   `json:"locked_reason,omitempty"`
	PrunableReason  string   `json:"prunable_reason,omitempty"`
	Exists          bool     `json:"exists"`
	Dirty           []string `json:"dirty,omitempty"`
	Unlanded        bool     `json:"unlanded,omitempty"`
	InspectionError string   `json:"inspection_error,omitempty"`
}

// Issue is a checkout that could not be inventoried.
type Issue struct {
	RepoID   string `json:"repo_id"`
	RepoPath string `json:"repo_path"`
	Error    string `json:"error"`
}

// Report is the complete inventory for a caller-selected checkout set.
type Report struct {
	Entries []Entry `json:"entries"`
	Issues  []Issue `json:"issues,omitempty"`
}

type porcelainEntry struct {
	Path           string
	Head           string
	Branch         string
	Detached       bool
	Locked         bool
	LockedReason   string
	PrunableReason string
}

// Inspect inventories each unique Git common directory exactly once.
func Inspect(checkouts []Checkout, registrations []Registration) Report {
	return inspect(checkouts, registrations, true)
}

// InspectHeads inventories paths and ancestry without running git status in
// every linked worktree. It is intended for non-gating informational surfaces.
func InspectHeads(checkouts []Checkout, registrations []Registration) Report {
	return inspect(checkouts, registrations, false)
}

func inspect(checkouts []Checkout, registrations []Registration, includeDirty bool) Report {
	registered := map[string]Registration{}
	for _, registration := range registrations {
		registered[canonicalPath(registration.Path)] = registration
	}
	seenCommonDirs := map[string]bool{}
	var report Report
	for _, checkout := range checkouts {
		basePath := canonicalPath(checkout.Path)
		commonDir, err := gitCommonDir(checkout.Path)
		if err != nil {
			report.Issues = append(report.Issues, Issue{RepoID: checkout.ID, RepoPath: checkout.Path, Error: err.Error()})
			continue
		}
		if seenCommonDirs[commonDir] {
			continue
		}
		seenCommonDirs[commonDir] = true

		baseHead, err := gitText(checkout.Path, "rev-parse", "HEAD")
		if err != nil {
			report.Issues = append(report.Issues, Issue{RepoID: checkout.ID, RepoPath: checkout.Path, Error: err.Error()})
			continue
		}
		out, err := gitBytes(checkout.Path, "worktree", "list", "--porcelain", "-z")
		if err != nil {
			report.Issues = append(report.Issues, Issue{RepoID: checkout.ID, RepoPath: checkout.Path, Error: err.Error()})
			continue
		}
		parsed, err := parsePorcelain(out)
		if err != nil {
			report.Issues = append(report.Issues, Issue{RepoID: checkout.ID, RepoPath: checkout.Path, Error: err.Error()})
			continue
		}
		for _, worktree := range parsed {
			path := canonicalPath(worktree.Path)
			entry := Entry{
				RepoID:         checkout.ID,
				RepoKind:       checkout.Kind,
				RepoPath:       basePath,
				CommonDir:      commonDir,
				Path:           path,
				Head:           worktree.Head,
				BaseHead:       baseHead,
				Branch:         worktree.Branch,
				Detached:       worktree.Detached,
				Locked:         worktree.Locked,
				LockedReason:   worktree.LockedReason,
				PrunableReason: worktree.PrunableReason,
			}
			_, statErr := os.Stat(path)
			entry.Exists = statErr == nil
			switch {
			case path == basePath:
				entry.Class = ClassBase
			case registered[path].Status == "active":
				entry.Class = ClassRegisteredActive
				entry.SessionID = registered[path].SessionID
			case registered[path].SessionID != "":
				entry.Class = ClassRegisteredFinishedResidue
				entry.SessionID = registered[path].SessionID
			case worktree.PrunableReason != "" || errors.Is(statErr, os.ErrNotExist):
				entry.Class = ClassPrunable
			case statErr != nil:
				entry.Class = ClassLeftover
				entry.InspectionError = statErr.Error()
			default:
				entry.Class = ClassLeftover
			}
			if entry.Exists && includeDirty {
				entry.Dirty, entry.InspectionError = inspectDirty(path)
			}
			if entry.Class != ClassBase && entry.Head != "" && entry.BaseHead != "" {
				entry.Unlanded, err = isUnlanded(checkout.Path, entry.Head, entry.BaseHead)
				if err != nil && entry.InspectionError == "" {
					entry.InspectionError = err.Error()
				}
			}
			report.Entries = append(report.Entries, entry)
		}
	}
	sort.Slice(report.Entries, func(i, j int) bool {
		if report.Entries[i].RepoPath != report.Entries[j].RepoPath {
			return report.Entries[i].RepoPath < report.Entries[j].RepoPath
		}
		return report.Entries[i].Path < report.Entries[j].Path
	})
	return report
}

// RemoveClean removes one already-classified linked worktree with Git's safe,
// non-force path. It deliberately preserves the branch reference.
func RemoveClean(entry Entry) error {
	if entry.Class != ClassLeftover && entry.Class != ClassRegisteredFinishedResidue {
		return fmt.Errorf("worktree class %s cannot be closed", entry.Class)
	}
	if !entry.Exists {
		return fmt.Errorf("worktree path is missing; inspect before pruning Git metadata")
	}
	if entry.Locked {
		return fmt.Errorf("worktree is locked")
	}
	if entry.Detached || entry.Branch == "" {
		return fmt.Errorf("detached worktree cannot be closed safely without preserving its HEAD")
	}
	dirty, errText := inspectDirty(entry.Path)
	if errText != "" {
		return fmt.Errorf("inspect worktree before close: %s", errText)
	}
	if len(dirty) != 0 {
		return fmt.Errorf("worktree is dirty")
	}
	if _, err := gitBytes(entry.RepoPath, "worktree", "remove", entry.Path); err != nil {
		return err
	}
	return nil
}

func parsePorcelain(data []byte) ([]porcelainEntry, error) {
	var result []porcelainEntry
	var current *porcelainEntry
	flush := func() {
		if current != nil {
			result = append(result, *current)
			current = nil
		}
	}
	for _, raw := range bytes.Split(data, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		field := string(raw)
		if strings.HasPrefix(field, "worktree ") {
			flush()
			current = &porcelainEntry{Path: strings.TrimPrefix(field, "worktree ")}
			continue
		}
		if current == nil {
			return nil, fmt.Errorf("worktree porcelain field before path: %q", field)
		}
		switch {
		case strings.HasPrefix(field, "HEAD "):
			current.Head = strings.TrimPrefix(field, "HEAD ")
		case strings.HasPrefix(field, "branch "):
			current.Branch = strings.TrimPrefix(strings.TrimPrefix(field, "branch "), "refs/heads/")
		case field == "detached":
			current.Detached = true
		case field == "locked":
			current.Locked = true
		case strings.HasPrefix(field, "locked "):
			current.Locked = true
			current.LockedReason = strings.TrimPrefix(field, "locked ")
		case field == "prunable":
			current.PrunableReason = "missing worktree metadata target"
		case strings.HasPrefix(field, "prunable "):
			current.PrunableReason = strings.TrimPrefix(field, "prunable ")
		case field == "bare":
			// Bare repositories do not yield a closable linked worktree.
		default:
			return nil, fmt.Errorf("unknown worktree porcelain field %q", field)
		}
	}
	flush()
	for _, entry := range result {
		if entry.Path == "" {
			return nil, fmt.Errorf("worktree porcelain entry has no path")
		}
	}
	return result, nil
}

func gitCommonDir(repo string) (string, error) {
	value, err := gitText(repo, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(repo, value)
	}
	return canonicalPath(value), nil
}

func inspectDirty(path string) ([]string, string) {
	out, err := gitBytes(path, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err.Error()
	}
	var result []string
	for _, field := range bytes.Split(out, []byte{0}) {
		if len(field) != 0 {
			result = append(result, string(field))
		}
	}
	return result, ""
}

func isUnlanded(repo, head, baseHead string) (bool, error) {
	cmd := exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", head, baseHead)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return false, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, commandError(out, err)
}

func gitText(repo string, args ...string) (string, error) {
	out, err := gitBytes(repo, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitBytes(repo string, args ...string) ([]byte, error) {
	full := append([]string{"-C", repo}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		return out, commandError(out, err)
	}
	return out, nil
}

func commandError(out []byte, err error) error {
	message := strings.TrimSpace(string(out))
	if message == "" {
		message = err.Error()
	}
	return errors.New(message)
}

func canonicalPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(real)
	}
	// Missing worktree paths still need to compare equal to the path Git
	// reports. Resolve the nearest existing parent (notably /var -> /private/var
	// on macOS), then append the absent suffix again.
	current := filepath.Clean(abs)
	var suffix []string
	for {
		if _, err := os.Lstat(current); err == nil {
			if real, err := filepath.EvalSymlinks(current); err == nil {
				parts := append([]string{real}, suffix...)
				return filepath.Clean(filepath.Join(parts...))
			}
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		suffix = append([]string{filepath.Base(current)}, suffix...)
		current = parent
	}
	return filepath.Clean(abs)
}
