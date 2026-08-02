package main

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParseRVMCheckConfig(t *testing.T) {
	config, err := parseRVMCheckConfig(map[string]interface{}{
		"rvm": map[string]interface{}{"check": float64(24), "install": "请安装 robust video matting"},
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if config.CacheFor != 24*time.Hour || config.Install != "请安装 robust video matting" {
		t.Fatalf("unexpected config: %#v", config)
	}
	if _, err := parseRVMCheckConfig(map[string]interface{}{"rvm": map[string]interface{}{"check": float64(0), "install": ""}}); err == nil {
		t.Fatal("invalid RVM configuration must be rejected")
	}
}

func TestRVMVideoAllowList(t *testing.T) {
	for _, path := range []string{"videos/input.mp4", "videos/input.MOV", "tmp/input.webm"} {
		if !isRVMVideo(path) {
			t.Errorf("%s should be accepted", path)
		}
	}
	if isRVMVideo("videos/input.png") {
		t.Fatal("image must not be accepted as an RVM source")
	}
}

func TestRVMRuntimeCandidatesDiscoverAgentInstallation(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "rvm")
	checkpoint := filepath.Join(home, "weights", "rvm_mobilenetv3.pth")
	if err := os.MkdirAll(filepath.Dir(checkpoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "inference.py"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkpoint, []byte("checkpoint"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range rvmRuntimeCandidates(workspace) {
		if candidate.Script == filepath.Join(home, "inference.py") && candidate.Checkpoint == checkpoint {
			return
		}
	}
	t.Fatal("agent-local RVM installation using weights/ was not discovered")
}

func TestRVMInvocationUsesCompatibilityWrapper(t *testing.T) {
	runtime := rvmRuntime{Python: "python3", Script: "/agent/rvm/inference.py", Checkpoint: "/agent/rvm/weights/rvm_mobilenetv3.pth"}
	args := rvmInvocationArgs(runtime, rvmScenarioStandard, "mps", "/agent/tmp/input.mp4", "/tmp/foreground.mov", "/tmp/alpha.mov")
	if len(args) < 4 || args[0] != "-c" || args[1] != rvmCompatibilityBootstrap || args[2] != runtime.Script {
		t.Fatalf("RVM must start through the compatibility wrapper: %#v", args)
	}
	if !strings.Contains(args[1], "Fraction") || !strings.Contains(args[1], "runpy.run_path") {
		t.Fatalf("wrapper must convert the PyAV rate before executing upstream RVM: %q", args[1])
	}
}

func TestRVMScenarioDefaultsAndValidation(t *testing.T) {
	value, standard, ok := rvmScenarioFor("")
	if !ok || value != rvmScenarioStandard || standard.Name != "标准主体提取" {
		t.Fatalf("empty scenario = (%q, %#v, %v), want standard scenario", value, standard, ok)
	}
	if _, _, ok := rvmScenarioFor("unknown"); ok {
		t.Fatal("unknown RVM scenario must be rejected")
	}
}

func TestRVMScenarioInvocationArgs(t *testing.T) {
	runtime := rvmRuntime{Python: "python3", Script: "/agent/rvm/inference.py", Checkpoint: "/agent/rvm/weights/rvm_mobilenetv3.pth"}
	containsPair := func(args []string, key, value string) bool {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == key && args[i+1] == value {
				return true
			}
		}
		return false
	}
	for _, test := range []struct {
		scenario, mbps, downsample string
	}{
		{rvmScenarioStandard, "4", ""},
		{rvmScenarioQuality, "8", "1"},
		{rvmScenarioFast, "2", "0.25"},
	} {
		args := rvmInvocationArgs(runtime, test.scenario, "cpu", "/input.mp4", "/foreground.mov", "/alpha.mov")
		if !containsPair(args, "--output-video-mbps", test.mbps) || !containsPair(args, "--seq-chunk", "1") {
			t.Fatalf("%s invocation args = %#v", test.scenario, args)
		}
		if got := containsPair(args, "--downsample-ratio", test.downsample); got != (test.downsample != "") {
			t.Fatalf("%s downsample args = %#v", test.scenario, args)
		}
	}
	args := rvmInvocationArgs(runtime, "unknown", "cpu", "/input.mp4", "/foreground.mov", "/alpha.mov")
	if !containsPair(args, "--output-video-mbps", "4") || containsPair(args, "--downsample-ratio", "1") {
		t.Fatalf("unknown scenario must safely fall back to standard invocation: %#v", args)
	}
}

func TestRVMAllocateAddsTimestampForExistingOutput(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "videos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "videos", "clip_subject.mp4"), []byte("existing preview"), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureRVMTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	previousNow := rvmNow
	rvmNow = func() time.Time { return time.Date(2026, 8, 2, 16, 30, 45, 0, time.Local) }
	t.Cleanup(func() { rvmNow = previousNow })
	manager := &rvmTaskManager{db: db}
	relative, _, err := manager.allocate("creator", workspace, "tmp/clip.mp4", map[string]bool{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := filepath.ToSlash(relative), "videos/clip_subject_20260802_163045.mov"; got != want {
		t.Fatalf("collision output = %q, want %q", got, want)
	}
}

func TestRVMPreviewMP4Path(t *testing.T) {
	if got, want := rvmPreviewMP4Path("/agent/videos/clip_subject.mov"), "/agent/videos/clip_subject.mp4"; got != want {
		t.Fatalf("preview path = %q, want %q", got, want)
	}
}

func TestRVMPreviewFFmpegArgsFlattensAlphaOnBlack(t *testing.T) {
	args := rvmPreviewFFmpegArgs("/agent/videos/clip_subject.mov", "/agent/videos/clip_subject.mp4")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "color=c=black") || !strings.Contains(joined, "overlay=shortest=1") {
		t.Fatalf("preview conversion must composite alpha over black: %q", args)
	}
	for index := 0; index+1 < len(args); index++ {
		if args[index] == "-map" && args[index+1] == "0:v:0" {
			t.Fatalf("preview conversion must not map the alpha-bearing MOV directly: %q", args)
		}
	}
	if !strings.Contains(joined, "-map [preview]") || !strings.Contains(joined, "-c:v libx264") || !strings.Contains(joined, "-pix_fmt yuv420p") {
		t.Fatalf("unexpected preview conversion args: %q", args)
	}
}

func TestRVMDeleteFailedOrCancelledTask(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureRVMTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	insert := func(status string) int64 {
		result, err := db.Exec(`INSERT INTO rvm_task(agent_id,source_path,output_path,status,created_at,updated_at) VALUES(?,?,?,?, 'now','now')`, "agent-a", "tmp/input.mp4", "videos/output.mov", status)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	failedID, cancelledID, completedID := insert(rvmTaskFailed), insert(rvmTaskCancelled), insert(rvmTaskCompleted)
	manager := &rvmTaskManager{db: db}
	for _, id := range []int64{failedID, cancelledID} {
		if err := manager.deleteFailedOrCancelled("agent-a", id); err != nil {
			t.Fatalf("delete task %d: %v", id, err)
		}
	}
	if err := manager.deleteFailedOrCancelled("agent-a", completedID); err == nil {
		t.Fatal("completed task must not be deletable")
	}
}

func TestRVMRuntimeForStartupFindsAgentWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "agent-a")
	home := filepath.Join(workspace, "rvm")
	checkpoint := filepath.Join(home, "weights", "rvm_mobilenetv3.pth")
	if err := os.MkdirAll(filepath.Dir(checkpoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "inference.py"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkpoint, []byte("checkpoint"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldLookPath, oldCommand, oldCache := rvmLookPath, rvmCommandContext, rvmCheckCache
	t.Cleanup(func() { rvmLookPath, rvmCommandContext, rvmCheckCache = oldLookPath, oldCommand, oldCache })
	rvmLookPath = func(string) (string, error) { return "python3", nil }
	rvmCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd { return exec.Command("true") }
	rvmCheckCache = struct {
		sync.Mutex
		entries map[string]rvmCachedRuntime
	}{}
	if _, ok := rvmRuntimeForStartup(time.Hour, &Config{AgentDir: root}); !ok {
		t.Fatal("startup RVM preflight did not discover the agent-local installation")
	}
}

func TestRVMInstallRequestExpandsWorkspace(t *testing.T) {
	workspace := "/agents/current"
	got := rvmInstallRequest("请安装 robust video matting，模型与代码存放至 `$workspace/rvm` 目录下", workspace)
	want := "请安装 robust video matting，模型与代码存放至 `/agents/current/rvm` 目录下"
	if got != want {
		t.Fatalf("install request = %q, want %q", got, want)
	}
}

func TestRVMWorkspaceForRequestResolvesAgentID(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "creator")
	if err := os.Mkdir(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	agentID, resolved, err := rvmWorkspaceForRequest(&Config{AgentDir: root}, "creator")
	if err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	wantWorkspace, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if agentID != "creator" || resolved != wantWorkspace {
		t.Fatalf("resolved = (%q, %q), want (%q, %q)", agentID, resolved, "creator", wantWorkspace)
	}
	if _, _, err := rvmWorkspaceForRequest(&Config{AgentDir: root}, "missing"); err == nil {
		t.Fatal("unknown Agent ID must be rejected")
	}
}
