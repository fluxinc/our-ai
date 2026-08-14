package worktreecheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePorcelainZeroDelimited(t *testing.T) {
	data := []byte("worktree /tmp/base\x00HEAD abc\x00branch refs/heads/main\x00\x00worktree /tmp/path with spaces\x00HEAD def\x00detached\x00locked agent active\x00prunable gitdir missing\x00")
	entries, err := parsePorcelain(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %#v", entries)
	}
	if entries[0].Branch != "main" || entries[1].Path != "/tmp/path with spaces" || !entries[1].Detached || !entries[1].Locked || entries[1].LockedReason != "agent active" || entries[1].PrunableReason != "gitdir missing" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestInspectClassifiesAndDeduplicatesCommonDir(t *testing.T) {
	repo := initRepo(t)
	leftover := filepath.Join(t.TempDir(), "leftover with spaces")
	active := filepath.Join(t.TempDir(), "active")
	gitRun(t, repo, "worktree", "add", "-b", "feature-leftover", leftover)
	gitRun(t, repo, "worktree", "add", "-b", "my/session/test", active)
	writeFile(t, filepath.Join(leftover, "draft.txt"), "draft\n")

	report := Inspect([]Checkout{
		{ID: "sample", Kind: "repo", Path: repo},
		{ID: "duplicate", Kind: "repo", Path: leftover},
	}, []Registration{{SessionID: "test", Status: "active", Path: active}})
	if len(report.Issues) != 0 {
		t.Fatalf("issues = %#v", report.Issues)
	}
	if len(report.Entries) != 3 {
		t.Fatalf("entries = %#v, want one common-dir inventory", report.Entries)
	}
	leftoverEntry := findEntry(t, report, leftover)
	if leftoverEntry.Class != ClassLeftover || len(leftoverEntry.Dirty) != 1 || leftoverEntry.Unlanded {
		t.Fatalf("leftover = %#v", leftoverEntry)
	}
	activeEntry := findEntry(t, report, active)
	if activeEntry.Class != ClassRegisteredActive || activeEntry.SessionID != "test" {
		t.Fatalf("active = %#v", activeEntry)
	}
}

func TestRemoveCleanPreservesBranchAndRefusesUnsafeTrees(t *testing.T) {
	repo := initRepo(t)
	leftover := filepath.Join(t.TempDir(), "clean")
	detached := filepath.Join(t.TempDir(), "detached")
	gitRun(t, repo, "worktree", "add", "-b", "feature-clean", leftover)
	writeFile(t, filepath.Join(leftover, "change.txt"), "change\n")
	gitRun(t, leftover, "add", ".")
	gitRun(t, leftover, "commit", "-m", "feature")
	gitRun(t, repo, "worktree", "add", "--detach", detached)

	report := Inspect([]Checkout{{ID: "sample", Kind: "repo", Path: repo}}, nil)
	cleanEntry := findEntry(t, report, leftover)
	if !cleanEntry.Unlanded || cleanEntry.Detached || len(cleanEntry.Dirty) != 0 {
		t.Fatalf("clean entry = %#v", cleanEntry)
	}
	if err := RemoveClean(cleanEntry); err != nil {
		t.Fatalf("remove clean: %v", err)
	}
	if _, err := os.Stat(leftover); !os.IsNotExist(err) {
		t.Fatalf("leftover still exists: %v", err)
	}
	gitRun(t, repo, "show-ref", "--verify", "refs/heads/feature-clean")

	detachedEntry := findEntry(t, Inspect([]Checkout{{ID: "sample", Path: repo}}, nil), detached)
	if err := RemoveClean(detachedEntry); err == nil || !strings.Contains(err.Error(), "detached") {
		t.Fatalf("detached close error = %v", err)
	}
}

func TestInspectReportsLockedFinishedAndPrunable(t *testing.T) {
	repo := initRepo(t)
	locked := filepath.Join(t.TempDir(), "locked")
	finished := filepath.Join(t.TempDir(), "finished")
	missing := filepath.Join(t.TempDir(), "missing")
	gitRun(t, repo, "worktree", "add", "-b", "locked-branch", locked)
	gitRun(t, repo, "worktree", "lock", "--reason", "agent active", locked)
	gitRun(t, repo, "worktree", "add", "-b", "finished-branch", finished)
	gitRun(t, repo, "worktree", "add", "-b", "missing-branch", missing)
	if err := os.RemoveAll(missing); err != nil {
		t.Fatal(err)
	}

	report := Inspect([]Checkout{{ID: "sample", Path: repo}}, []Registration{{SessionID: "old", Status: "finished", Path: finished}})
	lockedEntry := findEntry(t, report, locked)
	if lockedEntry.Class != ClassLeftover || !lockedEntry.Locked || lockedEntry.LockedReason != "agent active" {
		t.Fatalf("locked = %#v", lockedEntry)
	}
	if got := findEntry(t, report, finished); got.Class != ClassRegisteredFinishedResidue || got.SessionID != "old" {
		t.Fatalf("finished = %#v", got)
	}
	if got := findEntry(t, report, missing); got.Class != ClassPrunable || got.Exists {
		t.Fatalf("missing = %#v", got)
	}
}

func TestPrunableDetachedUniqueCommitIsNeverRemoved(t *testing.T) {
	repo := initRepo(t)
	missing := filepath.Join(t.TempDir(), "detached-missing")
	gitRun(t, repo, "worktree", "add", "--detach", missing)
	writeFile(t, filepath.Join(missing, "unique.txt"), "unique\n")
	gitRun(t, missing, "add", ".")
	gitRun(t, missing, "commit", "-m", "unique detached work")
	head := strings.TrimSpace(gitOutput(t, missing, "rev-parse", "HEAD"))
	if err := os.RemoveAll(missing); err != nil {
		t.Fatal(err)
	}

	entry := findEntry(t, Inspect([]Checkout{{ID: "sample", Path: repo}}, nil), missing)
	if entry.Class != ClassPrunable || !entry.Detached || entry.Head != head || !entry.Unlanded {
		t.Fatalf("entry = %#v", entry)
	}
	if err := RemoveClean(entry); err == nil {
		t.Fatal("RemoveClean accepted prunable detached worktree")
	}
	gitRun(t, repo, "cat-file", "-e", head+"^{commit}")
}

func findEntry(t *testing.T, report Report, path string) Entry {
	t.Helper()
	want := canonicalPath(path)
	for _, entry := range report.Entries {
		if entry.Path == want {
			return entry
		}
	}
	t.Fatalf("entry %s not found in %#v", want, report)
	return Entry{}
}

func initRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitRun(t, repo, "init", "-q")
	gitRun(t, repo, "config", "user.email", "test@example.com")
	gitRun(t, repo, "config", "user.name", "Test User")
	writeFile(t, filepath.Join(repo, "README.md"), "seed\n")
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-q", "-m", "seed")
	return repo
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
