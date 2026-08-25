package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A registered-but-unsynced manifest is the state right after
// `my manifests add`; setup and the deterministic onboarding walkthrough must
// clone it instead of telling the operator to run another command.
func TestSetupSelfHealsUnsyncedRegisteredManifest(t *testing.T) {
	home := t.TempDir()
	isolateCLITestHome(t, home)
	src := filepath.Join(t.TempDir(), "acme-manifest")
	writeCLITestFile(t, filepath.Join(src, "manifest.json"), `{
  "manifest_version": 1,
  "organization": { "id": "acme", "name": "Acme Example" },
  "umbrella": { "recommended_path": "~/acme" }
}`)
	runCLIGit(t, src, "init", "-q")
	runCLIGit(t, src, "add", ".")
	runCLIGit(t, src, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "init")

	var stdout, stderr bytes.Buffer
	a := app{stdout: &stdout, stderr: &stderr}
	if err := a.run([]string{"my", "manifests", "add", "acme", src, "--home", home}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "share", "my-cli", "manifests", "acme", "manifest.json")); err == nil {
		t.Fatal("manifests add must not clone by itself in this test")
	}
	stdout.Reset()
	stderr.Reset()
	if err := a.run([]string{"my", "setup", "--manifest", "acme", "--home", home, "--no-refresh", "--no-update-check"}); err != nil {
		t.Fatalf("setup: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "notice\tmanifest acme is not synced yet; cloning") {
		t.Fatalf("expected self-heal notice on stderr, got %q", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "share", "my-cli", "manifests", "acme", "manifest.json")); err != nil {
		t.Fatalf("manifest not cloned: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "acme", "AGENTS.md")); err != nil {
		t.Fatalf("umbrella guidance not written: %v", err)
	}
}

func TestSetupReportsSelfHealCloneFailureWithRemediation(t *testing.T) {
	home := t.TempDir()
	isolateCLITestHome(t, home)
	var stdout, stderr bytes.Buffer
	a := app{stdout: &stdout, stderr: &stderr}
	missing := filepath.Join(t.TempDir(), "nope.git")
	if err := a.run([]string{"my", "manifests", "add", "acme", missing, "--home", home}); err != nil {
		t.Fatal(err)
	}
	err := a.run([]string{"my", "setup", "--manifest", "acme", "--home", home, "--no-refresh", "--no-update-check"})
	if err == nil || !strings.Contains(err.Error(), `manifest "acme" could not be synced`) {
		t.Fatalf("err = %v", err)
	}
}
