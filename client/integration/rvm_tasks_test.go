package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
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

func TestRVMRuntimeCandidatesDiscoverNestedCloneInstallation(t *testing.T) {
	workspace := t.TempDir()
	rvmHome := filepath.Join(workspace, "rvm")
	cloneHome := filepath.Join(rvmHome, "RobustVideoMatting")
	checkpoint := filepath.Join(rvmHome, "weights", "rvm_mobilenetv3.pth")
	if err := os.MkdirAll(filepath.Dir(checkpoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cloneHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cloneHome, "inference.py"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkpoint, []byte("checkpoint"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range rvmRuntimeCandidates(workspace) {
		if candidate.Script == filepath.Join(cloneHome, "inference.py") && candidate.Checkpoint == checkpoint {
			return
		}
	}
	t.Fatal("nested git-clone RVM installation was not discovered")
}

func TestRVMRuntimesForRetainsFallbackLayouts(t *testing.T) {
	workspace := t.TempDir()
	rvmHome := filepath.Join(workspace, "rvm")
	cloneHome := filepath.Join(rvmHome, "RobustVideoMatting")
	checkpoint := filepath.Join(rvmHome, "weights", "rvm_mobilenetv3.pth")
	if err := os.MkdirAll(filepath.Dir(checkpoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cloneHome, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, script := range []string{filepath.Join(rvmHome, "inference.py"), filepath.Join(cloneHome, "inference.py")} {
		if err := os.WriteFile(script, []byte("# test"), 0o644); err != nil {
			t.Fatal(err)
		}
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
	runtimes := rvmRuntimesFor(time.Hour, workspace)
	if len(runtimes) != 2 {
		t.Fatalf("fallback runtimes = %#v, want direct and nested layouts", runtimes)
	}
	if runtimes[0].Script != filepath.Join(rvmHome, "inference.py") || runtimes[1].Script != filepath.Join(cloneHome, "inference.py") {
		t.Fatalf("fallback order = %#v", runtimes)
	}
	for _, runtime := range runtimes {
		if runtime.Checkpoint != checkpoint {
			t.Fatalf("runtime checkpoint = %q, want %q", runtime.Checkpoint, checkpoint)
		}
	}
}

func TestEnsureRVMRuntimesDownloadsSelectedResNet50(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "rvm")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "inference.py"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("resnet50-checkpoint"))
	}))
	defer server.Close()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE rvm_task (id INTEGER PRIMARY KEY, logs TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '')`); err != nil {
		t.Fatal(err)
	}
	oldLookPath, oldCommand, oldClient, oldArtifacts, oldCache := rvmLookPath, rvmCommandContext, rvmHTTPClient, rvmModelArtifacts, rvmCheckCache
	t.Cleanup(func() {
		rvmLookPath, rvmCommandContext, rvmHTTPClient, rvmModelArtifacts, rvmCheckCache = oldLookPath, oldCommand, oldClient, oldArtifacts, oldCache
	})
	rvmLookPath = func(string) (string, error) { return "python3", nil }
	rvmCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd { return exec.Command("true") }
	rvmHTTPClient = server.Client()
	artifacts := make(map[string]rvmModelArtifact, len(rvmModelArtifacts))
	for key, value := range rvmModelArtifacts {
		artifacts[key] = value
	}
	resnet := artifacts[rvmModelResNet50]
	resnet.OfficialURL, resnet.BackupURL = server.URL, ""
	artifacts[rvmModelResNet50] = resnet
	rvmModelArtifacts = artifacts
	rvmCheckCache = struct {
		sync.Mutex
		entries map[string]rvmCachedRuntime
	}{}
	manager := &rvmTaskManager{cfg: &Config{ModelTaskDownload: 1, ModelTaskRetry: 0, ModelTaskTimeout: 10}, db: db}
	runtimes, err := manager.ensureRVMRuntimes(context.Background(), 1, workspace, rvmScenarioQuality)
	if err != nil || len(runtimes) != 1 {
		t.Fatalf("ensureRVMRuntimes() = %#v, %v", runtimes, err)
	}
	checkpoint := filepath.Join(home, "weights", "rvm_resnet50.pth")
	if content, err := os.ReadFile(checkpoint); err != nil || string(content) != "resnet50-checkpoint" {
		t.Fatalf("downloaded checkpoint = %q, %v", content, err)
	}
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

func TestRVMProbeUsesTheSameCompatibilityBootstrap(t *testing.T) {
	args := rvmProbeArgs("/agent/rvm/inference.py")
	if len(args) != 4 || args[0] != "-c" || args[1] != rvmCompatibilityBootstrap || args[2] != "/agent/rvm/inference.py" || args[3] != "--help" {
		t.Fatalf("RVM probe args = %#v", args)
	}
	if !strings.Contains(args[1], "weights_only") {
		t.Fatalf("RVM bootstrap must cover current PyTorch checkpoint loading: %q", args[1])
	}
}

func TestRVMScenarioDefaultsAndValidation(t *testing.T) {
	value, scenario, ok := rvmScenarioFor("")
	if !ok || value != rvmScenarioFast || scenario.Name != "速度优先" || scenario.Model != rvmModelMobileNetV3 {
		t.Fatalf("empty scenario = (%q, %#v, %v), want fast MobileNetV3 scenario", value, scenario, ok)
	}
	if _, _, ok := rvmScenarioFor("unknown"); ok {
		t.Fatal("unknown RVM scenario must be rejected")
	}
}

func TestRVMScenarioLogDescriptionIncludesModeAndParameters(t *testing.T) {
	_, scenario, ok := rvmScenarioFor(rvmScenarioQuality)
	if !ok {
		t.Fatal("quality scenario must be valid")
	}
	got := rvmScenarioLogDescription(scenario)
	for _, value := range []string{"模式=质量优先", "模型=resnet50", "视频码率=24 Mbps", "序列分块=1", "下采样比例=1"} {
		if !strings.Contains(got, value) {
			t.Fatalf("log description %q is missing %q", got, value)
		}
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
		scenario, mbps, downsample, variant string
	}{
		{rvmScenarioStandard, "16", "", rvmModelMobileNetV3},
		{rvmScenarioQuality, "24", "1", rvmModelResNet50},
		{rvmScenarioFast, "16", "0.25", rvmModelMobileNetV3},
	} {
		args := rvmInvocationArgs(runtime, test.scenario, "cpu", "/input.mp4", "/foreground.mov", "/alpha.mov")
		if !containsPair(args, "--output-video-mbps", test.mbps) || !containsPair(args, "--seq-chunk", "1") {
			t.Fatalf("%s invocation args = %#v", test.scenario, args)
		}
		if got := containsPair(args, "--downsample-ratio", test.downsample); got != (test.downsample != "") {
			t.Fatalf("%s downsample args = %#v", test.scenario, args)
		}
		if !containsPair(args, "--variant", test.variant) {
			t.Fatalf("%s variant args = %#v", test.scenario, args)
		}
	}
	args := rvmInvocationArgs(runtime, "unknown", "cpu", "/input.mp4", "/foreground.mov", "/alpha.mov")
	if !containsPair(args, "--output-video-mbps", "16") || !containsPair(args, "--downsample-ratio", "0.25") || !containsPair(args, "--variant", rvmModelMobileNetV3) {
		t.Fatalf("unknown scenario must safely fall back to fast invocation: %#v", args)
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
	if !strings.Contains(joined, "-map [preview]") || !strings.Contains(joined, "-c:v libx264") || !strings.Contains(joined, "-crf 16") || !strings.Contains(joined, "-pix_fmt yuv420p") {
		t.Fatalf("unexpected preview conversion args: %q", args)
	}
}

func TestRVMFFmpegFallbackArgumentsPreserveAlphaAndPreview(t *testing.T) {
	mov := rvmTransparentMOVFFmpegArgs("foreground.mov", "alpha.mov", "result.mov", "qtrle")
	if got := strings.Join(mov, " "); !strings.Contains(got, "alphamerge") || !strings.Contains(got, "-c:v qtrle") || !strings.Contains(got, "-pix_fmt argb") {
		t.Fatalf("qtrle MOV args = %#v", mov)
	}
	preview := rvmPreviewFFmpegArgsWithCodec("result.mov", "result.mp4", "h264_nvenc")
	if got := strings.Join(preview, " "); !strings.Contains(got, "-c:v h264_nvenc") || !strings.Contains(got, "-b:v 16M") || strings.Contains(got, "mpeg4") {
		t.Fatalf("hardware H.264 preview args = %#v", preview)
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
	if !strings.Contains(got, "请安装 robust video matting，模型与代码存放至 `/agents/current/rvm` 目录下") || !strings.Contains(got, rvmDomesticCheckpointURL) || !strings.Contains(got, rvmResNet50BackupURL) || !strings.Contains(got, "rvm_resnet50.pth") || !strings.Contains(got, rvmDomesticCheckpointSHA256) {
		t.Fatalf("install request missing expanded workspace or both model downloads: %q", got)
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
