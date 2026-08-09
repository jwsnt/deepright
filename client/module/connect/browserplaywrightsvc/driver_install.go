package browserplaywrightsvc

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"connect/sharedutil"

	"github.com/playwright-community/playwright-go"
)

const playwrightDriverFallbackDownloadTimeout = 90 * time.Second
const playwrightDriverFallbackDownloadAttempts = 3
const playwrightInstallRequiredMessage = "Playwright is unavailable. Please install Playwright first."

var (
	playwrightDriverLookupEnvFn   = os.LookupEnv
	playwrightDriverSetenvFn      = os.Setenv
	playwrightDriverLookPathFn    = exec.LookPath
	playwrightDriverRegistryURLFn = func(version string) string {
		version = strings.TrimSpace(version)
		return fmt.Sprintf("https://registry.npmjs.org/playwright-core/-/playwright-core-%s.tgz", version)
	}
	playwrightDriverHTTPClient = &http.Client{
		Timeout: playwrightDriverFallbackDownloadTimeout,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          16,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	playwrightDriverDownloadFn = func(ctx context.Context, version string) (io.ReadCloser, error) {
		url := playwrightDriverRegistryURLFn(version)
		var joinedErr error
		for attempt := 1; attempt <= playwrightDriverFallbackDownloadAttempts; attempt++ {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return nil, err
			}
			resp, err := playwrightDriverHTTPClient.Do(req)
			if err != nil {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("attempt %d: %w", attempt, err))
				continue
			}
			if resp.StatusCode != http.StatusOK {
				_ = resp.Body.Close()
				joinedErr = errors.Join(joinedErr, fmt.Errorf("attempt %d: unexpected status %s from %s", attempt, resp.Status, url))
				continue
			}
			return resp.Body, nil
		}
		return nil, joinedErr
	}
)

// InstallPlaywrightDriver ensures the Playwright driver is installed. It keeps
// the legacy playwright-go installer as the primary path, and falls back to the
// official npm playwright-core package when the legacy driver CDN is unavailable.
func InstallPlaywrightDriver(driverDir string) error {
	driverDir = strings.TrimSpace(driverDir)
	if driverDir == "" {
		return fmt.Errorf("playwright driver directory is empty")
	}
	absDriverDir, err := filepath.Abs(driverDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absDriverDir, 0o755); err != nil {
		return err
	}
	if ready, err := playwrightDriverReady(absDriverDir); err != nil {
		return err
	} else if ready {
		return nil
	}

	runOptions := &playwright.RunOptions{
		DriverDirectory:     absDriverDir,
		SkipInstallBrowsers: true,
	}
	if err := playwrightInstallFn(runOptions); err == nil {
		return nil
	} else {
		fallbackErr := installPlaywrightDriverFromNPM(absDriverDir)
		if fallbackErr == nil {
			return nil
		}
		return fmt.Errorf("%s: %w", playwrightInstallRequiredMessage,
			errors.Join(err, fmt.Errorf("playwright npm fallback install failed: %w", fallbackErr)))
	}
}

func installPlaywrightDriverFromNPM(driverDir string) error {
	version, err := playwrightDriverVersion(driverDir)
	if err != nil {
		return err
	}
	if _, err := ensurePlaywrightNodeEnvironment(driverDir); err != nil {
		return fmt.Errorf("resolve node runtime for playwright-core@%s: %w", version, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), playwrightDriverFallbackDownloadTimeout)
	defer cancel()

	body, err := playwrightDriverDownloadFn(ctx, version)
	if err != nil {
		return fmt.Errorf("download playwright-core@%s: %w", version, err)
	}
	defer body.Close()

	tmpRoot, err := os.MkdirTemp(filepath.Dir(driverDir), ".playwright-driver-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpRoot)

	if err := extractPlaywrightPackageTarball(body, tmpRoot); err != nil {
		return err
	}

	extractedPackageDir := filepath.Join(tmpRoot, "package")
	if err := validatePlaywrightPackageDir(extractedPackageDir); err != nil {
		return err
	}

	targetPackageDir := filepath.Join(driverDir, "package")
	if err := os.RemoveAll(targetPackageDir); err != nil {
		return err
	}
	if err := os.Rename(extractedPackageDir, targetPackageDir); err != nil {
		return err
	}

	ready, err := playwrightDriverReady(driverDir)
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("playwright driver install incomplete: %s", driverDir)
	}
	return nil
}

func playwrightDriverVersion(driverDir string) (string, error) {
	driver, err := playwright.NewDriver(&playwright.RunOptions{
		DriverDirectory:     strings.TrimSpace(driverDir),
		SkipInstallBrowsers: true,
	})
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(driver.Version)
	if version == "" {
		return "", fmt.Errorf("playwright driver version is empty")
	}
	return version, nil
}

func ensurePlaywrightNodeEnvironment(driverDir string) (string, error) {
	nodePath, err := resolvePlaywrightNodePath(driverDir)
	if err != nil {
		return "", err
	}
	defaultNodePath := filepath.Join(strings.TrimSpace(driverDir), playwrightDriverNodeFileName())
	if strings.TrimSpace(nodePath) == "" || filepath.Clean(nodePath) == filepath.Clean(defaultNodePath) {
		return nodePath, nil
	}
	if err := playwrightDriverSetenvFn("PLAYWRIGHT_NODEJS_PATH", nodePath); err != nil {
		return "", err
	}
	return nodePath, nil
}

func resolvePlaywrightNodePath(driverDir string) (string, error) {
	driverDir = strings.TrimSpace(driverDir)
	if envPath, ok := playwrightDriverLookupEnvFn("PLAYWRIGHT_NODEJS_PATH"); ok && playwrightDriverExecutableUsableFn(strings.TrimSpace(envPath)) {
		return strings.TrimSpace(envPath), nil
	}
	defaultNodePath := filepath.Join(driverDir, playwrightDriverNodeFileName())
	if playwrightDriverExecutableUsableFn(defaultNodePath) {
		return defaultNodePath, nil
	}

	sharedutil.ApplySystemPath()
	for _, candidate := range []string{"node", "nodejs"} {
		path, err := playwrightDriverLookPathFn(candidate)
		if err == nil && playwrightDriverExecutableUsableFn(path) {
			return path, nil
		}
	}
	return "", fmt.Errorf("could not locate a usable Node.js runtime")
}

func extractPlaywrightPackageTarball(src io.Reader, dstDir string) error {
	gzipReader, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("open playwright package archive: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read playwright package archive: %w", err)
		}
		name := filepath.Clean(strings.TrimSpace(header.Name))
		if name == "." || name == "" {
			continue
		}
		targetPath := filepath.Join(dstDir, name)
		if !strings.HasPrefix(targetPath, filepath.Clean(dstDir)+string(os.PathSeparator)) && filepath.Clean(targetPath) != filepath.Clean(dstDir) {
			return fmt.Errorf("invalid path in playwright package archive: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

func validatePlaywrightPackageDir(packageDir string) error {
	required := []string{
		filepath.Join(packageDir, "cli.js"),
		filepath.Join(packageDir, "package.json"),
	}
	for _, path := range required {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("expected file, got directory: %s", path)
		}
	}
	return nil
}
