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

func TestParseWav2LipCheckConfig(t *testing.T) {
	config, err := parseWav2LipCheckConfig(map[string]interface{}{
		"wav2lip": map[string]interface{}{"check": float64(24), "install": "请安装 wav2lip"},
	})
	if err != nil || config.CacheFor != 24*time.Hour || config.Install != "请安装 wav2lip" {
		t.Fatalf("parse config = %#v, %v", config, err)
	}
	if _, err := parseWav2LipCheckConfig(map[string]interface{}{"wav2lip": map[string]interface{}{"check": float64(0)}}); err == nil {
		t.Fatal("invalid Wav2Lip configuration must be rejected")
	}
}

func TestWav2LipMediaAllowLists(t *testing.T) {
	for _, path := range []string{"videos/input.mp4", "videos/input.MOV", "tmp/input.webm"} {
		if !isWav2LipVideo(path) {
			t.Errorf("%s should be accepted as video", path)
		}
	}
	for _, path := range []string{"tmp/voice.wav", "tmp/voice.mp3", "tmp/voice.m4a", "tmp/voice.aac", "tmp/voice.flac", "tmp/voice.ogg"} {
		if !isWav2LipAudio(path) {
			t.Errorf("%s should be accepted as audio", path)
		}
	}
	if isWav2LipAudio("tmp/voice.mp4") || isWav2LipVideo("tmp/image.png") {
		t.Fatal("media allow lists must reject unsupported files")
	}
}

func TestWav2LipRuntimeCandidatesUseFixedGANPaths(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, "wav2lip")
	checkpoint := filepath.Join(home, "checkpoints", "wav2lip_gan.pth")
	if err := os.MkdirAll(filepath.Dir(checkpoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "inference.py"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkpoint, []byte("checkpoint"), 0o644); err != nil {
		t.Fatal(err)
	}
	candidates := wav2lipRuntimeCandidates(workspace)
	if len(candidates) != 1 || candidates[0].Checkpoint != checkpoint || candidates[0].Script != filepath.Join(home, "inference.py") {
		t.Fatalf("unexpected fixed Wav2Lip runtime candidates: %#v", candidates)
	}
}

func TestWav2LipInvocationUsesOnlyOfficialFixedArguments(t *testing.T) {
	runtime := wav2lipRuntime{Python: "python3", Script: "/agent/wav2lip/inference.py", Checkpoint: "/agent/wav2lip/checkpoints/wav2lip_gan.pth"}
	args := wav2lipInvocationArgs(runtime, "/agent/tmp/input.mp4", "/agent/tmp/voice.mp3", "/tmp/output.mp4")
	joined := strings.Join(args, " ")
	for _, expected := range []string{"-c", "--checkpoint_path " + runtime.Checkpoint, "--face /agent/tmp/input.mp4", "--audio /agent/tmp/voice.mp3", "--outfile /tmp/output.mp4"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("Wav2Lip invocation missing %q: %#v", expected, args)
		}
	}
	for _, forbidden := range []string{"--variant", "--output-alpha", "--output-foreground", "--device"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("Wav2Lip invocation must not pass %s: %#v", forbidden, args)
		}
	}
	if !strings.Contains(args[1], "DEEPRIGHT_WAV2LIP_DEVICE") || !strings.Contains(args[1], "compile(source") {
		t.Fatalf("runtime wrapper must select device without changing upstream files: %q", args[1])
	}
	if !strings.Contains(args[1], "_deepright_compatible_mel") || !strings.Contains(args[1], "kwargs.setdefault('sr', args[0])") {
		t.Fatalf("runtime wrapper must adapt newer librosa without editing Wav2Lip: %q", args[1])
	}
}

func TestWav2LipProbeUsesTheSameCompatibilityBootstrap(t *testing.T) {
	args := wav2lipProbeArgs("/agent/wav2lip/inference.py")
	if len(args) != 4 || args[0] != "-c" || args[1] != wav2lipRuntimeBootstrap || args[2] != "/agent/wav2lip/inference.py" || args[3] != "--help" {
		t.Fatalf("Wav2Lip probe args = %#v", args)
	}
	for _, required := range []string{"weights_only", "np.__dict__", "DEEPRIGHT_WAV2LIP_DEVICE', 'cpu'"} {
		if !strings.Contains(args[1], required) {
			t.Fatalf("Wav2Lip bootstrap missing %q: %q", required, args[1])
		}
	}
}

func TestWav2LipStagingUsesPathsWithoutWhitespace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Application Support")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(root, "input clip.mp4")
	audio := filepath.Join(root, "voice track.wav")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(audio, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	stage, err := wav2lipPrepareStaging(video, audio)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(stage.Directory) })
	for _, path := range []string{stage.Directory, stage.Video, stage.Audio, stage.Output} {
		if strings.ContainsAny(path, " \t\r\n") {
			t.Fatalf("staging path contains whitespace: %q", path)
		}
	}
	if target, err := os.Readlink(stage.Video); err != nil || target != video {
		t.Fatalf("video staging link = %q, %v", target, err)
	}
	if target, err := os.Readlink(stage.Audio); err != nil || target != audio {
		t.Fatalf("audio staging link = %q, %v", target, err)
	}
}

func TestRunWav2LipUsesStableUpstreamWorkingDirectory(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureWav2LipTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	result, err := db.Exec(`INSERT INTO wav2lip_task(agent_id,video_path,audio_path,output_path,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, "agent-a", "tmp/input.mp4", "tmp/voice.wav", "videos/result.mp4", wav2lipTaskRunning, "now", "now")
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	upstreamDirectory := filepath.Join(t.TempDir(), "wav2lip")
	if err := os.MkdirAll(upstreamDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "result.mp4")
	previous := wav2lipCommandContext
	wav2lipCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "pwd; printf generated > \"$1\"", "sh", output)
	}
	t.Cleanup(func() { wav2lipCommandContext = previous })
	manager := &wav2lipTaskManager{db: db}
	runtime := wav2lipRuntime{Python: "python3", Script: filepath.Join(upstreamDirectory, "inference.py")}
	if err := manager.runWav2Lip(context.Background(), id, runtime, "cpu", "input.mp4", "audio.wav", output); err != nil {
		t.Fatal(err)
	}
	task, err := manager.log("agent-a", id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(task.Logs, upstreamDirectory) {
		t.Fatalf("child working directory was not the upstream directory: %q", task.Logs)
	}
}

func TestCopyWav2LipOutputCopiesAcrossDirectories(t *testing.T) {
	sourceDirectory := t.TempDir()
	destinationDirectory := filepath.Join(t.TempDir(), "Application Support")
	if err := os.MkdirAll(destinationDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDirectory, "result.mp4")
	destination := filepath.Join(destinationDirectory, ".wav2lip-result.mp4")
	if err := os.WriteFile(source, []byte("generated video"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyWav2LipOutput(source, destination); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != "generated video" {
		t.Fatalf("copied output = %q, %v", got, err)
	}
}

func TestWav2LipPreferredDevices(t *testing.T) {
	devices := wav2lipPreferredDevices()
	if len(devices) != 2 || devices[1] != "cpu" {
		t.Fatalf("unexpected fallback devices: %#v", devices)
	}
}

func TestWav2LipReadableFailureReasonForCorruptedModel(t *testing.T) {
	reason := wav2lipReadableFailureReason("RuntimeError: unexpected EOF, expected more bytes. The file might be corrupted.")
	if !strings.Contains(reason, "s3fd-619a316812.pth") || !strings.Contains(reason, "缓存") {
		t.Fatalf("corrupted model reason = %q", reason)
	}
	if got := wav2lipReadableFailureReason("some unrelated error"); got != "" {
		t.Fatalf("unrelated failure must not be relabelled: %q", got)
	}
}

func TestWav2LipAllocateAddsTimestampForExistingOutput(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "videos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "videos", "clip_lip_sync.mp4"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureWav2LipTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	previous := wav2lipNow
	wav2lipNow = func() time.Time { return time.Date(2026, 8, 2, 16, 30, 45, 0, time.Local) }
	t.Cleanup(func() { wav2lipNow = previous })
	path, _, err := (&wav2lipTaskManager{db: db}).allocate("creator", workspace, "tmp/clip.mp4", map[string]bool{}, 0)
	if err != nil || filepath.ToSlash(path) != "videos/clip_lip_sync_20260802_163045.mp4" {
		t.Fatalf("allocate = %q, %v", path, err)
	}
}

func TestWav2LipDeleteFailedOrCancelledTask(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureWav2LipTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	insert := func(status string) int64 {
		r, err := db.Exec(`INSERT INTO wav2lip_task(agent_id,video_path,audio_path,output_path,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, "agent-a", "tmp/input.mp4", "tmp/voice.mp3", "videos/output.mp4", status, "now", "now")
		if err != nil {
			t.Fatal(err)
		}
		id, _ := r.LastInsertId()
		return id
	}
	m := &wav2lipTaskManager{db: db}
	for _, id := range []int64{insert(wav2lipTaskFailed), insert(wav2lipTaskCancelled)} {
		if err := m.deleteFailedOrCancelled("agent-a", id); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.deleteFailedOrCancelled("agent-a", insert(wav2lipTaskCompleted)); err == nil {
		t.Fatal("completed task must not be deletable")
	}
}

func TestWav2LipRuntimeForStartupFindsAgentWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "agent-a")
	home := filepath.Join(workspace, "wav2lip")
	checkpoint := filepath.Join(home, "checkpoints", "wav2lip_gan.pth")
	if err := os.MkdirAll(filepath.Dir(checkpoint), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "inference.py"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkpoint, []byte("checkpoint"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldLook, oldCommand, oldCache := wav2lipLookPath, wav2lipCommandContext, wav2lipCheckCache
	t.Cleanup(func() { wav2lipLookPath, wav2lipCommandContext, wav2lipCheckCache = oldLook, oldCommand, oldCache })
	wav2lipLookPath = func(string) (string, error) { return "python3", nil }
	wav2lipCommandContext = func(context.Context, string, ...string) *exec.Cmd { return exec.Command("true") }
	wav2lipCheckCache = struct {
		sync.Mutex
		entries map[string]wav2lipCachedRuntime
	}{}
	if _, ok := wav2lipRuntimeForStartup(time.Hour, &Config{AgentDir: root}); !ok {
		t.Fatal("startup preflight did not discover agent-local Wav2Lip")
	}
}

func TestWav2LipInstallRequestExpandsWorkspace(t *testing.T) {
	got := wav2lipInstallRequest("请安装 wav2lip，放到`$workspace/wav2lip`", "/agents/current")
	if !strings.Contains(got, "请安装 wav2lip，放到`/agents/current/wav2lip`") || !strings.Contains(got, wav2lipDomesticCheckpointURL) || !strings.Contains(got, wav2lipDomesticCheckpointSHA256) {
		t.Fatalf("install request missing expanded workspace or verified domestic source: %q", got)
	}
}
