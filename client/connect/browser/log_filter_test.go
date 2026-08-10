package main

import (
	"strings"
	"testing"
)

func TestBrowserRunLogFilterDropsChromeNoise(t *testing.T) {
	restore := stubBrowserRuntime()
	defer restore()

	tempDir := t.TempDir()
	browserPath := tempDir + "/browser"
	browserExecutablePathFn = func() (string, error) {
		return browserPath, nil
	}

	logPath, err := browserDefaultLogPath()
	if err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader(strings.Join([]string{
		`[71627:19191757:0603/211429.272537:VERBOSE1:chrome/updater/updater.cc:352] Version: 149.0.7814.5`,
		`[71625:19191753:0603/211429.282755:ERROR:third_party/crashpad/crashpad/util/file/file_io_posix.cc:145] open /Users/test/Library/Application Support/Google/RLZ/Crashpad/settings.dat: No such file or directory (2)`,
		`[70914:19189733:0603/211433.642190:ERROR:google_apis/gcm/engine/registration_request.cc:291] Registration response error message: DEPRECATED_ENDPOINT`,
		`{"event":"browser_create_trace","stage":"instance.create.new.ready"}`,
		`[123:456:0603/211500.000000:ERROR:net/socket.cc:123] real browser error`,
	}, "\n") + "\n")

	if err := browserRunLogFilter(input, logPath); err != nil {
		t.Fatal(err)
	}

	data, err := browserReadFileFn(logPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, unwanted := range []string{
		"chrome/updater/",
		"third_party/crashpad/crashpad/",
		"DEPRECATED_ENDPOINT",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("filtered log still contains %q: %s", unwanted, got)
		}
	}
	for _, wanted := range []string{
		`{"event":"browser_create_trace","stage":"instance.create.new.ready"}`,
		`real browser error`,
	} {
		if !strings.Contains(got, wanted) {
			t.Fatalf("filtered log missing %q: %s", wanted, got)
		}
	}
}

func TestBrowserShouldFilterChromeLogLine(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{line: "", want: false},
		{line: `[1:2:0603/211319.971807:VERBOSE2:chrome/updater/event_history.cc:265] noise`, want: true},
		{line: `[1:2:0603/211319.991308:ERROR:third_party/crashpad/crashpad/util/file/file_io_posix.cc:145] open settings.dat`, want: true},
		{line: `[1:2:0603/211433.642190:ERROR:google_apis/gcm/engine/registration_request.cc:291] Registration response error message: DEPRECATED_ENDPOINT`, want: true},
		{line: `[1:2:0603/211433.642190:ERROR:google_apis/gcm/engine/registration_request.cc:291] Registration response error message: OTHER`, want: false},
		{line: `{"event":"browser_instance_close","reason":"destroy"}`, want: false},
		{line: `{"event":"browser_instance_close","reason":"shutdown"}`, want: false},
	}

	for _, tc := range cases {
		if got := browserShouldFilterChromeLogLine(tc.line); got != tc.want {
			t.Fatalf("browserShouldFilterChromeLogLine(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}
