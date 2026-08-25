package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestDoctorPrereqsReportMissingToolsWithRemediation(t *testing.T) {
	home := t.TempDir()
	isolateCLITestHome(t, home)
	var stdout, stderr bytes.Buffer
	a := app{stdout: &stdout, stderr: &stderr, lookPath: func(name string) (string, error) {
		if name == "git" {
			return "/usr/bin/git", nil
		}
		return "", errors.New("not found")
	}}
	if err := a.run([]string{"my", "doctor", "--no-fetch", "--home", home}); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	for _, want := range []string{
		"prereq\tgit\tok",
		"prereq\tgh\twarning",
		"gh auth login",
		"prereq\tharness\twarning",
		"my onboarding",
		"prereq\tpath\t",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor stdout missing %q:\n%s", want, out)
		}
	}
}

func TestDoctorPrereqsReportInstalledHarnessAndLoggedInGH(t *testing.T) {
	home := t.TempDir()
	isolateCLITestHome(t, home)
	var stdout, stderr bytes.Buffer
	a := app{stdout: &stdout, stderr: &stderr,
		lookPath: func(name string) (string, error) {
			switch name {
			case "git", "gh", "codex":
				return "/usr/local/bin/" + name, nil
			}
			return "", errors.New("not found")
		},
		publishRunner: func(name string, args ...string) ([]byte, error) {
			if name == "gh" && len(args) == 2 && args[0] == "auth" && args[1] == "token" {
				return []byte("gho_x"), nil
			}
			return nil, errors.New("unexpected")
		},
	}
	if err := a.run([]string{"my", "doctor", "--no-fetch", "--home", home}); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "prereq\tgh\tok\t/usr/local/bin/gh\tlogged in") || !strings.Contains(out, "prereq\tharness\tok\tinstalled: codex") {
		t.Fatalf("doctor stdout:\n%s", out)
	}
}
