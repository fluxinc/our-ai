package manifest

import (
	"bytes"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloneFailureHintExplainsPrivateGitHubAuth(t *testing.T) {
	hint := cloneFailureHint("https://github.com/acme/acme-manifest.git", "fatal: could not read Username for 'https://github.com': terminal prompts disabled")
	if !strings.Contains(hint, "gh auth login") || !strings.Contains(hint, "git@github.com:acme/acme-manifest.git") {
		t.Fatalf("hint = %q", hint)
	}
	if cloneFailureHint("https://github.com/acme/acme-manifest.git", "fatal: destination path exists") != "" {
		t.Fatal("expected no hint for unrelated failure")
	}
	if cloneFailureHint("git@github.com:acme/acme-manifest.git", "Permission denied (publickey)") == "" {
		t.Fatal("expected ssh hint")
	}
}

func TestGitEnvInvokesGHCredentialHelperWithoutChangingGlobalConfig(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "gh.log")
	gh := "#!/bin/sh\nprintf '%s\\n' \"$*\" >>\"$GH_HELPER_LOG\"\ncat >/dev/null\nprintf 'username=x-access-token\\npassword=test-token\\n'\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(gh), 0o755); err != nil {
		t.Fatal(err)
	}
	globalConfig := filepath.Join(root, "gitconfig")
	initial := []byte("[user]\n\tname = Fixture\n")
	if err := os.WriteFile(globalConfig, initial, 0o600); err != nil {
		t.Fatal(err)
	}
	before := sha256.Sum256(initial)
	cmd := exec.Command(gitPath, "credential", "fill")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\n\n")
	cmd.Env = GitEnv([]string{
		"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + root,
		"GIT_CONFIG_GLOBAL=" + globalConfig,
		"GH_HELPER_LOG=" + logPath,
	})
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git credential fill: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("username=x-access-token")) || !bytes.Contains(out, []byte("password=test-token")) {
		t.Fatalf("credential output = %q", out)
	}
	if log := string(mustReadFile(t, logPath)); !strings.Contains(log, "auth git-credential get") {
		t.Fatalf("gh helper log = %q", log)
	}
	afterBytes := mustReadFile(t, globalConfig)
	after := sha256.Sum256(afterBytes)
	if before != after {
		t.Fatalf("global Git config changed:\nbefore: %s\nafter: %s", initial, afterBytes)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestGitHubCredentialEnvDisablesTerminalPromptInExec(t *testing.T) {
	env := GitEnv([]string{"PATH=/x"})
	if v, ok := envValue(env, "GIT_TERMINAL_PROMPT"); !ok || v != "0" {
		t.Fatalf("GIT_TERMINAL_PROMPT = %q, %v", v, ok)
	}
}

func TestSyncCloneFailureIncludesHint(t *testing.T) {
	home := t.TempDir()
	ref, err := Add(home, "acme", "https://github.com/acme/acme-manifest.git")
	if err != nil {
		t.Fatal(err)
	}
	runner := func(name string, args ...string) ([]byte, error) {
		if name == "gh" && len(args) >= 2 && args[1] == "user" {
			return []byte(`{"id":1,"node_id":"MDQ6VXNlcjE=","login":"acme-bot"}`), nil
		}
		if name == "gh" {
			return []byte("HTTP/2.0 200 OK\r\n\r\n" + `{"id":7,"node_id":"R_kgDO","full_name":"acme/acme-manifest","private":true,"permissions":{"admin":false,"push":false,"pull":true}}`), nil
		}
		if name == "git" && len(args) > 0 && args[0] == "clone" {
			return []byte("fatal: could not read Username for 'https://github.com': terminal prompts disabled"), errString("exit 128")
		}
		return nil, nil
	}
	res := syncOne(ref, false, runner)
	if res.Status != "failed" || !strings.Contains(res.Error, "gh auth login") {
		t.Fatalf("res = %+v", res)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
