package mycli

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallerPersistsPathAndRegistersManifestWithoutGo(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Unix installer test")
	}
	root := t.TempDir()
	home := filepath.Join(root, "home")
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(root, "my.log")
	fakeMy := []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$FAKE_MY_LOG\"\n")
	tarball := filepath.Join(root, "release.tar.gz")
	writeInstallerTarball(t, tarball, fakeMy)
	digest := sha256.Sum256(mustReadInstallerFile(t, tarball))
	archiveName := fmt.Sprintf("my-cli_1.2.3_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	checksums := filepath.Join(root, "checksums.txt")
	if err := os.WriteFile(checksums, []byte(fmt.Sprintf("%x  %s\n", digest, archiveName)), 0o644); err != nil {
		t.Fatal(err)
	}

	curlStub := `#!/bin/sh
out=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$url" in
  *releases/latest) printf '%s\n' 'https://github.com/fluxinc/my-cli/releases/tag/v1.2.3' ;;
  *checksums.txt) cp "$FAKE_CHECKSUMS" "$out" ;;
  *) cp "$FAKE_TARBALL" "$out" ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "curl"), []byte(curlStub), 0o755); err != nil {
		t.Fatal(err)
	}

	installer := filepath.Join(repositoryRootForInstallerTest(t), "install.sh")
	run := func() {
		cmd := exec.Command("sh", installer,
			"--manifest", "acme", "https://github.com/acme/acme-manifest.git",
			"--no-onboarding")
		cmd.Env = []string{
			"HOME=" + home,
			"SHELL=/bin/zsh",
			"PATH=" + bin + ":/usr/bin:/bin",
			"FAKE_TARBALL=" + tarball,
			"FAKE_CHECKSUMS=" + checksums,
			"FAKE_MY_LOG=" + logPath,
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("install failed: %v\n%s", err, out)
		}
	}
	run()
	run()

	profile := string(mustReadInstallerFile(t, filepath.Join(home, ".zshrc")))
	pathLine := `export PATH="$HOME/.local/bin:$PATH"`
	if strings.Count(profile, pathLine) != 1 {
		t.Fatalf("profile path line count = %d; profile:\n%s", strings.Count(profile, pathLine), profile)
	}
	installed := mustReadInstallerFile(t, filepath.Join(home, ".local", "bin", "my"))
	if string(installed) != string(fakeMy) {
		t.Fatalf("installed binary fixture mismatch: %q", installed)
	}
	log := string(mustReadInstallerFile(t, logPath))
	if !strings.Contains(log, "skills self install --all") ||
		!strings.Contains(log, "manifests add acme https://github.com/acme/acme-manifest.git") {
		t.Fatalf("installer did not initialize skill and manifest:\n%s", log)
	}
}

func TestInstallerPersistsPathEvenWhenInheritedPathAlreadyContainsInstallDir(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Unix installer test")
	}
	root := t.TempDir()
	home := filepath.Join(root, "home")
	stubBin := filepath.Join(root, "stubs")
	installDir := filepath.Join(home, ".local", "bin")
	for _, dir := range []string{home, stubBin, installDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fakeMy := []byte("#!/bin/sh\nexit 0\n")
	tarball := filepath.Join(root, "release.tar.gz")
	writeInstallerTarball(t, tarball, fakeMy)
	digest := sha256.Sum256(mustReadInstallerFile(t, tarball))
	archiveName := fmt.Sprintf("my-cli_1.2.3_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	checksums := filepath.Join(root, "checksums.txt")
	if err := os.WriteFile(checksums, []byte(fmt.Sprintf("%x  %s\n", digest, archiveName)), 0o644); err != nil {
		t.Fatal(err)
	}
	curlStub := `#!/bin/sh
out=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$url" in
  *releases/latest) printf '%s\n' 'https://github.com/fluxinc/my-cli/releases/tag/v1.2.3' ;;
  *checksums.txt) cp "$FAKE_CHECKSUMS" "$out" ;;
  *) cp "$FAKE_TARBALL" "$out" ;;
esac
`
	if err := os.WriteFile(filepath.Join(stubBin, "curl"), []byte(curlStub), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", filepath.Join(repositoryRootForInstallerTest(t), "install.sh"), "--no-onboarding")
	cmd.Env = []string{
		"HOME=" + home,
		"SHELL=/bin/zsh",
		"PATH=" + installDir + ":" + stubBin + ":/usr/bin:/bin",
		"FAKE_TARBALL=" + tarball,
		"FAKE_CHECKSUMS=" + checksums,
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	profile := string(mustReadInstallerFile(t, filepath.Join(home, ".zshrc")))
	if !strings.Contains(profile, `export PATH="$HOME/.local/bin:$PATH"`) {
		t.Fatalf("durable PATH line missing despite inherited PATH:\n%s", profile)
	}
}

func TestStableInstallerWrapperPropagatesDownloadFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX wrapper test")
	}
	root := t.TempDir()
	stubBin := filepath.Join(root, "bin")
	if err := os.MkdirAll(stubBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stubBin, "curl"), []byte("#!/bin/sh\nexit 22\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", filepath.Join(repositoryRootForInstallerTest(t), "site", "public", "install.sh"))
	cmd.Env = []string{"PATH=" + stubBin + ":/usr/bin:/bin"}
	if err := cmd.Run(); err == nil {
		t.Fatal("stable wrapper succeeded after curl failure")
	}
}

func TestStableInstallerWrapperPinsCanonicalInstallerToLatestTag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX wrapper test")
	}
	root := t.TempDir()
	stubBin := filepath.Join(root, "bin")
	if err := os.MkdirAll(stubBin, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, "curl.log")
	curlStub := `#!/bin/sh
printf '%s\n' "$*" >>"$CURL_LOG"
out=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -w) shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
case "$url" in
  *releases/latest) printf '%s' 'https://github.com/fluxinc/my-cli/releases/tag/v9.8.7' ;;
  *raw.githubusercontent.com*) printf '#!/bin/sh\nexit 0\n' >"$out" ;;
  *) exit 22 ;;
esac
`
	if err := os.WriteFile(filepath.Join(stubBin, "curl"), []byte(curlStub), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", filepath.Join(repositoryRootForInstallerTest(t), "site", "public", "install.sh"))
	cmd.Env = []string{"PATH=" + stubBin + ":/usr/bin:/bin", "CURL_LOG=" + logPath}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stable wrapper failed: %v\n%s", err, out)
	}
	log := string(mustReadInstallerFile(t, logPath))
	if !strings.Contains(log, "/v9.8.7/install.sh") || strings.Contains(log, "/master/install.sh") {
		t.Fatalf("stable wrapper did not pin latest release tag:\n%s", log)
	}
}

func writeInstallerTarball(t *testing.T, path string, my []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "my", Mode: 0o755, Size: int64(len(my))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(my); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func mustReadInstallerFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func repositoryRootForInstallerTest(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve installer test source")
	}
	return filepath.Dir(source)
}
