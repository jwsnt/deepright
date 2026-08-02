package browserplaywrightsvc

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestPlaywrightDriverReadyUsesResolvedNodePath(t *testing.T) {
	originalLookupEnvFn := playwrightDriverLookupEnvFn
	originalLookPathFn := playwrightDriverLookPathFn
	originalDriverUsableFn := playwrightDriverExecutableUsableFn
	defer func() {
		playwrightDriverLookupEnvFn = originalLookupEnvFn
		playwrightDriverLookPathFn = originalLookPathFn
		playwrightDriverExecutableUsableFn = originalDriverUsableFn
	}()

	driverDir := filepath.Join(t.TempDir(), "playwright", "driver")
	if err := tMkdirAll(driverDir, 0o755); err != nil {
		t.Fatalf("mkdir driver dir: %v", err)
	}
	if err := tMkdirAll(filepath.Join(driverDir, "package"), 0o755); err != nil {
		t.Fatalf("mkdir package dir: %v", err)
	}
	if err := tWriteFile(filepath.Join(driverDir, "package", "cli.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatalf("write cli.js: %v", err)
	}
	if err := tWriteFile(filepath.Join(driverDir, "package", "package.json"), []byte(`{"name":"playwright-core"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	playwrightDriverLookupEnvFn = func(string) (string, bool) { return "", false }
	playwrightDriverLookPathFn = func(file string) (string, error) {
		if file != "node" {
			t.Fatalf("unexpected lookpath %q", file)
		}
		return "/usr/local/bin/node", nil
	}
	playwrightDriverExecutableUsableFn = func(path string) bool {
		return strings.TrimSpace(path) == "/usr/local/bin/node"
	}

	ready, err := PlaywrightDriverReady(driverDir)
	if err != nil {
		t.Fatalf("PlaywrightDriverReady: %v", err)
	}
	if !ready {
		t.Fatal("expected driver to be ready with resolved system node")
	}
}

func TestInstallPlaywrightDriverFallsBackToNPMTarball(t *testing.T) {
	originalInstallFn := playwrightInstallFn
	originalLookupEnvFn := playwrightDriverLookupEnvFn
	originalLookPathFn := playwrightDriverLookPathFn
	originalDriverUsableFn := playwrightDriverExecutableUsableFn
	originalSetenvFn := playwrightDriverSetenvFn
	originalRegistryURLFn := playwrightDriverRegistryURLFn
	originalHTTPClient := playwrightDriverHTTPClient
	defer func() {
		playwrightInstallFn = originalInstallFn
		playwrightDriverLookupEnvFn = originalLookupEnvFn
		playwrightDriverLookPathFn = originalLookPathFn
		playwrightDriverExecutableUsableFn = originalDriverUsableFn
		playwrightDriverSetenvFn = originalSetenvFn
		playwrightDriverRegistryURLFn = originalRegistryURLFn
		playwrightDriverHTTPClient = originalHTTPClient
	}()

	playwrightInstallFn = func(opts *playwright.RunOptions) error {
		return io.EOF
	}
	playwrightDriverLookupEnvFn = func(string) (string, bool) { return "", false }
	playwrightDriverLookPathFn = func(file string) (string, error) {
		if file != "node" {
			t.Fatalf("unexpected lookpath %q", file)
		}
		return "/usr/local/bin/node", nil
	}
	playwrightDriverExecutableUsableFn = func(path string) bool {
		return strings.TrimSpace(path) == "/usr/local/bin/node"
	}
	var envValue string
	playwrightDriverSetenvFn = func(key, value string) error {
		if key == "PLAYWRIGHT_NODEJS_PATH" {
			envValue = value
		}
		return nil
	}

	version := "1.57.0"
	tarball := buildPlaywrightCoreTarball(t, version)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/playwright-core/-/playwright-core-"+version+".tgz" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(tarball)
	}))
	defer server.Close()
	playwrightDriverRegistryURLFn = func(gotVersion string) string {
		if gotVersion != version {
			t.Fatalf("version = %q, want %q", gotVersion, version)
		}
		return server.URL + "/playwright-core/-/playwright-core-" + gotVersion + ".tgz"
	}
	playwrightDriverHTTPClient = server.Client()

	driverDir := filepath.Join(t.TempDir(), "playwright", "driver")
	if err := InstallPlaywrightDriver(driverDir); err != nil {
		t.Fatalf("InstallPlaywrightDriver: %v", err)
	}
	if envValue != "/usr/local/bin/node" {
		t.Fatalf("PLAYWRIGHT_NODEJS_PATH = %q", envValue)
	}
	for _, path := range []string{
		filepath.Join(driverDir, "package", "cli.js"),
		filepath.Join(driverDir, "package", "package.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected extracted file %s: %v", path, err)
		}
	}
}

func TestInstallPlaywrightDriverPromptsForManualInstallWhenAllDownloadsFail(t *testing.T) {
	originalInstallFn := playwrightInstallFn
	originalLookupEnvFn := playwrightDriverLookupEnvFn
	originalLookPathFn := playwrightDriverLookPathFn
	originalDriverUsableFn := playwrightDriverExecutableUsableFn
	originalSetenvFn := playwrightDriverSetenvFn
	originalDownloadFn := playwrightDriverDownloadFn
	defer func() {
		playwrightInstallFn = originalInstallFn
		playwrightDriverLookupEnvFn = originalLookupEnvFn
		playwrightDriverLookPathFn = originalLookPathFn
		playwrightDriverExecutableUsableFn = originalDriverUsableFn
		playwrightDriverSetenvFn = originalSetenvFn
		playwrightDriverDownloadFn = originalDownloadFn
	}()

	playwrightInstallFn = func(*playwright.RunOptions) error { return io.EOF }
	playwrightDriverLookupEnvFn = func(string) (string, bool) { return "", false }
	playwrightDriverLookPathFn = func(string) (string, error) { return "/usr/local/bin/node", nil }
	playwrightDriverExecutableUsableFn = func(path string) bool { return path == "/usr/local/bin/node" }
	playwrightDriverSetenvFn = func(string, string) error { return nil }
	playwrightDriverDownloadFn = func(context.Context, string) (io.ReadCloser, error) {
		return nil, io.ErrUnexpectedEOF
	}

	err := InstallPlaywrightDriver(filepath.Join(t.TempDir(), "playwright", "driver"))
	if err == nil {
		t.Fatal("expected Playwright installation failure")
	}
	if !strings.Contains(err.Error(), playwrightInstallRequiredMessage) {
		t.Fatalf("error = %q, want manual-install prompt %q", err, playwrightInstallRequiredMessage)
	}
}

func TestEnsurePlaywrightUsesSystemNodeAfterInstall(t *testing.T) {
	originalRunFn := playwrightRunFn
	originalInstallFn := playwrightInstallFn
	originalLookupEnvFn := playwrightDriverLookupEnvFn
	originalLookPathFn := playwrightDriverLookPathFn
	originalDriverUsableFn := playwrightDriverExecutableUsableFn
	originalSetenvFn := playwrightDriverSetenvFn
	defer func() {
		playwrightRunFn = originalRunFn
		playwrightInstallFn = originalInstallFn
		playwrightDriverLookupEnvFn = originalLookupEnvFn
		playwrightDriverLookPathFn = originalLookPathFn
		playwrightDriverExecutableUsableFn = originalDriverUsableFn
		playwrightDriverSetenvFn = originalSetenvFn
	}()

	driverDir := filepath.Join(t.TempDir(), "playwright", "driver")
	playwrightInstallFn = func(opts *playwright.RunOptions) error {
		if err := tMkdirAll(filepath.Join(driverDir, "package"), 0o755); err != nil {
			return err
		}
		if err := tWriteFile(filepath.Join(driverDir, "package", "cli.js"), []byte("console.log('ok')"), 0o644); err != nil {
			return err
		}
		return tWriteFile(filepath.Join(driverDir, "package", "package.json"), []byte(`{"name":"playwright-core"}`), 0o644)
	}
	playwrightDriverLookupEnvFn = func(string) (string, bool) { return "", false }
	playwrightDriverLookPathFn = func(file string) (string, error) {
		if file != "node" {
			t.Fatalf("unexpected lookpath %q", file)
		}
		return "/usr/local/bin/node", nil
	}
	playwrightDriverExecutableUsableFn = func(path string) bool {
		return strings.TrimSpace(path) == "/usr/local/bin/node"
	}
	var envValue string
	playwrightDriverSetenvFn = func(key, value string) error {
		if key == "PLAYWRIGHT_NODEJS_PATH" {
			envValue = value
		}
		return nil
	}
	playwrightRunFn = func(opts *playwright.RunOptions) (*playwright.Playwright, error) {
		return &playwright.Playwright{}, nil
	}

	svc := newDaemonService(Options{StateDir: t.TempDir(), DriverDir: driverDir}, log.New(io.Discard, "", 0))
	if err := svc.ensurePlaywright(); err != nil {
		t.Fatalf("ensurePlaywright: %v", err)
	}
	if envValue != "/usr/local/bin/node" {
		t.Fatalf("PLAYWRIGHT_NODEJS_PATH = %q", envValue)
	}
}

func buildPlaywrightCoreTarball(t *testing.T, version string) []byte {
	t.Helper()

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)

	entries := map[string]string{
		"package/cli.js":       "console.log('Version " + version + "')\n",
		"package/package.json": `{"name":"playwright-core","version":"` + version + `"}`,
	}
	for name, body := range entries {
		data := []byte(body)
		if err := tarWriter.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(data)),
		}); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tarWriter.Write(data); err != nil {
			t.Fatalf("write tar body: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return archive.Bytes()
}

func tMkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }

func tWriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
