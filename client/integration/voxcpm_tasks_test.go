package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func useTestVoxCPMLocalModel(t *testing.T, root string) {
	t.Helper()
	model := filepath.Join(root, "test-voxcpm-model")
	if err := os.MkdirAll(model, 0o755); err != nil {
		t.Fatal(err)
	}
	previous := voxcpmPrepareLocalModel
	voxcpmPrepareLocalModel = func(*voxcpmTaskManager, context.Context, int64) (string, error) {
		return model, nil
	}
	t.Cleanup(func() { voxcpmPrepareLocalModel = previous })
}

func TestParseVoxCPMCheckConfig(t *testing.T) {
	config, err := parseVoxCPMCheckConfig(map[string]interface{}{"voxcpm": map[string]interface{}{"check": float64(24), "install": "请安装 voxcpm"}})
	if err != nil || config.CacheFor != 24*time.Hour || config.Install != "请安装 voxcpm" {
		t.Fatalf("parse valid config = %#v, %v", config, err)
	}
	if _, err := parseVoxCPMCheckConfig(map[string]interface{}{"voxcpm": map[string]interface{}{"check": 0, "install": "x"}}); err == nil {
		t.Fatal("zero cache duration must be rejected")
	}
}

func TestVoxCPMExecutableFindsManagedPythonInstallAddedAfterStartup(t *testing.T) {
	command := filepath.Join(t.TempDir(), "voxcpm")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	previousLookPath, previousGlob := voxcpmLookPath, voxcpmGlob
	previousCheckedAt, previousExecutable := time.Time{}, ""
	voxcpmCheckCache.Lock()
	previousCheckedAt, previousExecutable = voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable
	voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = time.Time{}, ""
	voxcpmCheckCache.Unlock()
	installed := false
	voxcpmLookPath = func(string) (string, error) { return "", os.ErrNotExist }
	voxcpmGlob = func(pattern string) ([]string, error) {
		if pattern == "/usr/local/share/python*/cpython-*/bin/voxcpm" && installed {
			return []string{command}, nil
		}
		return nil, nil
	}
	t.Cleanup(func() {
		voxcpmLookPath, voxcpmGlob = previousLookPath, previousGlob
		voxcpmCheckCache.Lock()
		voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = previousCheckedAt, previousExecutable
		voxcpmCheckCache.Unlock()
	})

	if _, ok := voxcpmExecutable(time.Hour); ok {
		t.Fatal("VoxCPM must be unavailable before installation")
	}
	installed = true
	if path, ok := voxcpmExecutable(time.Hour); !ok || path != command {
		t.Fatalf("VoxCPM added after startup = %q, %v; want %q, true", path, ok, command)
	}
}

func TestVoxCPMAllocateNamesSingleAndBatch(t *testing.T) {
	root, workspace := t.TempDir(), ""
	workspace = filepath.Join(root, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "audios"), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureVoxCPMTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	originalNow := voxcpmNow
	voxcpmNow = func() time.Time { return time.Date(2026, 8, 1, 16, 2, 3, 0, time.Local) }
	defer func() { voxcpmNow = originalNow }()
	manager := &voxcpmTaskManager{cfg: &Config{AgentDir: root}, db: db}
	if err := os.WriteFile(filepath.Join(workspace, "audios", "voice.wav"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	relative, _, err := manager.allocate("agent-a", workspace, "voice.wav", false, map[string]bool{}, 0)
	if err != nil || filepath.ToSlash(relative) != "audios/voice_20260801_160203.000000000.wav" {
		t.Fatalf("single collision allocation = %q, %v", relative, err)
	}
	batch, _, err := manager.allocate("agent-a", workspace, "voice.wav", true, map[string]bool{}, 0)
	if err != nil || filepath.ToSlash(batch) != "audios/voice_20260801_160203.000000000.wav" {
		t.Fatalf("batch allocation = %q, %v", batch, err)
	}
}

func TestVoxCPMDeleteFailedOrCancelledTask(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureVoxCPMTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	insert := func(status string) int64 {
		result, err := db.Exec(`INSERT INTO voxcpm_task(agent_id,text_path,text_saved_as,reference_audio_path,reference_audio_saved_as,requested_name,output_path,saved_as,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?, 'now','now')`, "agent-a", "tmp/input.txt", "tmp/input.txt", "", "", "output.wav", "audios/output.wav", "audios/output.wav", status)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	failedID, cancelledID, completedID := insert(voxcpmTaskFailed), insert(voxcpmTaskCancelled), insert(voxcpmTaskCompleted)
	manager := &voxcpmTaskManager{db: db}
	for _, id := range []int64{failedID, cancelledID} {
		if err := manager.deleteFailedOrCancelled("agent-a", id); err != nil {
			t.Fatalf("delete task %d: %v", id, err)
		}
	}
	if err := manager.deleteFailedOrCancelled("agent-a", completedID); err == nil {
		t.Fatal("completed task must not be deletable")
	}
}

func TestVoxCPMCloneIgnoresExpressionStyle(t *testing.T) {
	root := t.TempDir()
	useTestVoxCPMLocalModel(t, root)
	workspace := filepath.Join(root, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "tmp", "speech.txt"), []byte("你好，VoxCPM。"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "tmp", "reference.wav"), []byte("RIFF\x24\x00\x00\x00WAVEfmt "), 0o644); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(root, "voxcpm")
	script := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then exit 0; fi\nprintf '%s\\n' \"$@\" > \"$0.args\"\nout=''\nwhile [ $# -gt 0 ]; do if [ \"$1\" = \"--output\" ]; then out=$2; shift 2; else shift; fi; done\nprintf 'RIFF\\044\\000\\000\\000WAVEfmt ' > \"$out\"\n"
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	originalLookPath := voxcpmLookPath
	voxcpmLookPath = func(string) (string, error) { return command, nil }
	defer func() { voxcpmLookPath = originalLookPath }()
	voxcpmCheckCache.Lock()
	originalCheckedAt, originalExecutable := voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable
	voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = time.Time{}, ""
	voxcpmCheckCache.Unlock()
	defer func() {
		voxcpmCheckCache.Lock()
		voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = originalCheckedAt, originalExecutable
		voxcpmCheckCache.Unlock()
	}()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureVoxCPMTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	manager := &voxcpmTaskManager{cfg: &Config{AgentDir: root}, db: db, wake: make(chan struct{}, 1)}
	const control = "温和、平稳、自然的叙述语气"
	created, err := manager.create(voxcpmTaskCreateRequest{AgentID: "agent-a", OutputName: "clone", Tasks: []voxcpmTaskCreateItem{{TextPath: "tmp/speech.txt", ReferenceAudioPath: "tmp/reference.wav", Scenario: voxcpmScenarioQuality, Control: control}}})
	if err != nil || len(created) != 1 || created[0].OutputPath != "audios/clone.wav" || created[0].Scenario != voxcpmScenarioQuality || created[0].Control != "" {
		t.Fatalf("create task = %#v, %v", created, err)
	}
	if err := manager.generate(context.Background(), created[0]); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(workspace, "audios", "clone.wav"))
	if err != nil || !strings.HasPrefix(string(data), "RIFF") {
		t.Fatalf("saved WAV = %q, %v", data, err)
	}
	arguments, err := os.ReadFile(command + ".args")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--cfg-value\n2.5", "--inference-timesteps\n20", "--normalize", "--denoise"} {
		if !strings.Contains(string(arguments), want) {
			t.Fatalf("quality scenario missing %q in args %q", want, arguments)
		}
	}
	if strings.Contains(string(arguments), "--no-denoiser") {
		t.Fatalf("a requested clone denoiser must remain enabled: %q", arguments)
	}
	if strings.Contains(string(arguments), "--control") {
		t.Fatalf("clone arguments must not include expression style: %q", arguments)
	}
	var logs string
	if err := db.QueryRow(`SELECT logs FROM voxcpm_task WHERE id=?`, created[0].ID).Scan(&logs); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs, "表达风格仅适用于自由生成") || strings.Contains(logs, "已使用任务填写的表达风格：") {
		t.Fatalf("clone expression style must be ignored: %q", logs)
	}
}

func TestVoxCPMExpressionStyleOnlyAppliesToFreeGeneration(t *testing.T) {
	scenario, ok := voxcpmScenarioFor(voxcpmScenarioWarmNarration)
	if !ok {
		t.Fatal("warm narration scenario must exist")
	}
	if got := voxcpmEffectiveControl(scenario, "自定义表达", false); got != "自定义表达" {
		t.Fatalf("free generation custom control = %q", got)
	}
	if got := voxcpmEffectiveControl(scenario, "", false); got != scenario.Control {
		t.Fatalf("free generation scenario control = %q", got)
	}
	if got := voxcpmEffectiveControl(scenario, "自定义表达", true); got != "" {
		t.Fatalf("clone control = %q", got)
	}
}

func TestVoxCPMControlTextNormalizesAndBoundsInput(t *testing.T) {
	control, err := voxcpmControlText("  温和\n\t平稳  ")
	if err != nil || control != "温和 平稳" {
		t.Fatalf("normalized control = %q, %v", control, err)
	}
	if _, err := voxcpmControlText(strings.Repeat("声", voxcpmControlLimit+1)); err == nil {
		t.Fatal("overlong expression style must be rejected")
	}
}

func TestVoxCPMGenerateWithoutReferenceUsesDesignMode(t *testing.T) {
	root := t.TempDir()
	useTestVoxCPMLocalModel(t, root)
	workspace := filepath.Join(root, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "tmp", "speech.txt"), []byte("你好，VoxCPM。"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(root, "voxcpm")
	script := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then exit 0; fi\nprintf '%s\\n' \"$@\" > \"$0.args\"\nout=''\nwhile [ $# -gt 0 ]; do if [ \"$1\" = \"--output\" ]; then out=$2; shift 2; else shift; fi; done\nprintf 'RIFF\\044\\000\\000\\000WAVEfmt ' > \"$out\"\n"
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	originalLookPath := voxcpmLookPath
	voxcpmLookPath = func(string) (string, error) { return command, nil }
	defer func() { voxcpmLookPath = originalLookPath }()
	voxcpmCheckCache.Lock()
	originalCheckedAt, originalExecutable := voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable
	voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = time.Time{}, ""
	voxcpmCheckCache.Unlock()
	defer func() {
		voxcpmCheckCache.Lock()
		voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = originalCheckedAt, originalExecutable
		voxcpmCheckCache.Unlock()
	}()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureVoxCPMTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	manager := &voxcpmTaskManager{cfg: &Config{AgentDir: root}, db: db, wake: make(chan struct{}, 1)}
	const control = "清晰、自然、稍慢的表达"
	created, err := manager.create(voxcpmTaskCreateRequest{AgentID: "agent-a", OutputName: "design", Tasks: []voxcpmTaskCreateItem{{TextPath: "tmp/speech.txt", Scenario: voxcpmScenarioBalanced, Control: control}}})
	if err != nil || len(created) != 1 || created[0].ReferenceAudioPath != "" {
		t.Fatalf("create design task = %#v, %v", created, err)
	}
	if err := manager.generate(context.Background(), created[0]); err != nil {
		t.Fatalf("generate design task: %v", err)
	}
	arguments, err := os.ReadFile(command + ".args")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(arguments), "design\n") || strings.Contains(string(arguments), "--reference-audio") || !strings.Contains(string(arguments), "--control\n"+control) {
		t.Fatalf("expected design arguments without reference audio, got %q", arguments)
	}
	if !strings.Contains(string(arguments), "--no-denoiser") {
		t.Fatalf("design without requested denoising must disable ZipEnhancer: %q", arguments)
	}
	var logs string
	if err := db.QueryRow(`SELECT logs FROM voxcpm_task WHERE id=?`, created[0].ID).Scan(&logs); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs, "未选择参考音色") || !strings.Contains(logs, "使用 design 模式自由生成声音") || !strings.Contains(logs, "已使用任务填写的表达风格："+control) {
		t.Fatalf("optional reference and control logs = %q", logs)
	}
}

func TestVoxCPMScenarioValidation(t *testing.T) {
	for _, value := range []string{"", voxcpmScenarioBalanced, voxcpmScenarioQuality, voxcpmScenarioFast, voxcpmScenarioClean, voxcpmScenarioWarmNarration, voxcpmScenarioLively} {
		if _, ok := voxcpmScenarioFor(value); !ok {
			t.Fatalf("scenario %q must be valid", value)
		}
	}
	if _, ok := voxcpmScenarioFor("--arbitrary-command"); ok {
		t.Fatal("untrusted scenario must be rejected")
	}
}

func TestVoxCPMGenerateFallsBackFromAnyGPUError(t *testing.T) {
	root := t.TempDir()
	useTestVoxCPMLocalModel(t, root)
	workspace := filepath.Join(root, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "tmp", "speech.txt"), []byte("你好，VoxCPM。"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "tmp", "reference.wav"), []byte("RIFF\x24\x00\x00\x00WAVEfmt "), 0o644); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(root, "voxcpm")
	script := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then exit 0; fi\nout=''\ndevice='auto'\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--output\" ]; then out=$2; shift 2; elif [ \"$1\" = \"--device\" ]; then device=$2; shift 2; else shift; fi\ndone\nprintf '%s\\n' \"$device\" >> \"$0.attempts\"\nif [ \"$device\" != \"cpu\" ]; then printf 'an arbitrary GPU backend failure\\n' >&2; exit 1; fi\nprintf 'CPU fallback clone\\n'\nprintf 'RIFF\\044\\000\\000\\000WAVEfmt ' > \"$out\"\n"
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	originalLookPath := voxcpmLookPath
	voxcpmLookPath = func(string) (string, error) { return command, nil }
	defer func() { voxcpmLookPath = originalLookPath }()
	originalPreferredDevice := voxcpmTaskPreferredDevice
	voxcpmTaskPreferredDevice = func(context.Context, string) (string, string) {
		return "mps", "检测到 Apple MPS GPU，VoxCPM 优先使用 GPU。"
	}
	defer func() { voxcpmTaskPreferredDevice = originalPreferredDevice }()
	voxcpmCheckCache.Lock()
	originalCheckedAt, originalExecutable := voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable
	voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = time.Time{}, ""
	voxcpmCheckCache.Unlock()
	defer func() {
		voxcpmCheckCache.Lock()
		voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = originalCheckedAt, originalExecutable
		voxcpmCheckCache.Unlock()
	}()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureVoxCPMTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	manager := &voxcpmTaskManager{cfg: &Config{AgentDir: root}, db: db, wake: make(chan struct{}, 1)}
	created, err := manager.create(voxcpmTaskCreateRequest{AgentID: "agent-a", OutputName: "clone", Tasks: []voxcpmTaskCreateItem{{TextPath: "tmp/speech.txt", ReferenceAudioPath: "tmp/reference.wav"}}})
	if err != nil || len(created) != 1 {
		t.Fatalf("create task = %#v, %v", created, err)
	}
	if err := manager.generate(context.Background(), created[0]); err != nil {
		var logs string
		_ = db.QueryRow(`SELECT logs FROM voxcpm_task WHERE id=?`, created[0].ID).Scan(&logs)
		t.Fatalf("generate with GPU fallback: %v; logs=%q", err, logs)
	}
	data, err := os.ReadFile(filepath.Join(workspace, "audios", "clone.wav"))
	if err != nil || !strings.HasPrefix(string(data), "RIFF") {
		t.Fatalf("saved WAV = %q, %v", data, err)
	}
	var logs string
	if err := db.QueryRow(`SELECT logs FROM voxcpm_task WHERE id=?`, created[0].ID).Scan(&logs); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs, "MPS 配音失败，已自动回退到 CPU 重新尝试一次") || !strings.Contains(logs, "已使用 CPU 回退完成") || strings.Contains(logs, "file already closed") {
		t.Fatalf("unexpected fallback logs: %q", logs)
	}
	attempts, err := os.ReadFile(command + ".attempts")
	if err != nil || string(attempts) != "mps\ncpu\n" {
		t.Fatalf("GPU then one CPU fallback attempts = %q, %v", attempts, err)
	}
}

func TestVoxCPMGenerateFallsBackToFlatCLIForDesign(t *testing.T) {
	root := t.TempDir()
	useTestVoxCPMLocalModel(t, root)
	workspace := filepath.Join(root, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "tmp", "speech.txt"), []byte("你好，VoxCPM。"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(root, "voxcpm")
	script := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then exit 0; fi\nprintf '%s\\n' \"$@\" >> \"$0.args\"\nif [ \"$1\" = \"design\" ]; then printf 'usage: voxcpm [options]\\nvoxcpm: error: unrecognized arguments: design --control angry --device cpu\\n' >&2; exit 2; fi\nout=''\nwhile [ $# -gt 0 ]; do if [ \"$1\" = \"--output\" ]; then out=$2; shift 2; else shift; fi; done\nprintf 'RIFF\\044\\000\\000\\000WAVEfmt ' > \"$out\"\n"
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	originalLookPath := voxcpmLookPath
	voxcpmLookPath = func(string) (string, error) { return command, nil }
	defer func() { voxcpmLookPath = originalLookPath }()
	voxcpmCheckCache.Lock()
	originalCheckedAt, originalExecutable := voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable
	voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = time.Time{}, ""
	voxcpmCheckCache.Unlock()
	defer func() {
		voxcpmCheckCache.Lock()
		voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = originalCheckedAt, originalExecutable
		voxcpmCheckCache.Unlock()
	}()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureVoxCPMTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	manager := &voxcpmTaskManager{cfg: &Config{AgentDir: root}, db: db, wake: make(chan struct{}, 1)}
	created, err := manager.create(voxcpmTaskCreateRequest{AgentID: "agent-a", OutputName: "design", Tasks: []voxcpmTaskCreateItem{{TextPath: "tmp/speech.txt", Scenario: voxcpmScenarioBalanced, Control: "愤怒"}}})
	if err != nil || len(created) != 1 {
		t.Fatalf("create task = %#v, %v", created, err)
	}
	if err := manager.generate(context.Background(), created[0]); err != nil {
		t.Fatalf("generate with flat CLI fallback: %v", err)
	}
	arguments, err := os.ReadFile(command + ".args")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(arguments), "design\n") || strings.Count(string(arguments), "--text\n") != 2 || strings.Count(string(arguments), "--control\n") != 1 || strings.Count(string(arguments), "--device\n") != 1 {
		t.Fatalf("expected native then flat CLI arguments, got %q", arguments)
	}
	var logs string
	if err := db.QueryRow(`SELECT logs FROM voxcpm_task WHERE id=?`, created[0].ID).Scan(&logs); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs, "参数接口不兼容") || !strings.Contains(logs, "兼容直接合成模式完成") {
		t.Fatalf("expected compatibility fallback logs, got %q", logs)
	}
}

func TestVoxCPMGenerateUsesVerifiedModelScopeSnapshot(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "agent-a")
	if err := os.MkdirAll(filepath.Join(workspace, "tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "tmp", "speech.txt"), []byte("你好，VoxCPM。"), 0o644); err != nil {
		t.Fatal(err)
	}
	modelFiles := map[string][]byte{
		"config.json":       []byte(`{"architecture":"voxcpm2"}`),
		"model.safetensors": []byte("model-weights"),
		"audiovae.pth":      []byte("audio-vae"),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/files" {
			files := make([]voxcpmModelScopeFile, 0, len(modelFiles))
			for path, data := range modelFiles {
				sum := sha256.Sum256(data)
				files = append(files, voxcpmModelScopeFile{Path: path, SHA256: fmt.Sprintf("%x", sum[:]), Size: int64(len(data)), Type: "blob"})
			}
			_ = json.NewEncoder(w).Encode(voxcpmModelScopeFilesResponse{Code: http.StatusOK, Data: struct {
				Files []voxcpmModelScopeFile `json:"Files"`
			}{Files: files}})
			return
		}
		path := filepath.Base(request.URL.Path)
		data, ok := modelFiles[path]
		if !ok {
			http.NotFound(w, request)
			return
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()
	originalFilesURL, originalResolveBase, originalClient := voxcpmModelScopeFilesURL, voxcpmModelScopeResolveBase, voxcpmHTTPClient
	voxcpmModelScopeFilesURL = server.URL + "/files"
	voxcpmModelScopeResolveBase = server.URL + "/models/OpenBMB/VoxCPM2/resolve/master/"
	voxcpmHTTPClient = server.Client()
	defer func() {
		voxcpmModelScopeFilesURL, voxcpmModelScopeResolveBase, voxcpmHTTPClient = originalFilesURL, originalResolveBase, originalClient
	}()
	cacheRoot := filepath.Join(root, "model-cache")
	if err := os.Setenv("DEEPRIGHT_VOXCPM_MODEL_CACHE", cacheRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("DEEPRIGHT_VOXCPM_MODEL_CACHE")
	command := filepath.Join(root, "voxcpm")
	script := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then exit 0; fi\nif [ \"$HF_HUB_OFFLINE\" != \"1\" ]; then exit 3; fi\nout=''\nmodel=''\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--output\" ]; then out=$2; shift 2; elif [ \"$1\" = \"--model-path\" ]; then model=$2; shift 2; else shift; fi\ndone\nif [ -z \"$model\" ]; then exit 1; fi\nif [ ! -f \"$model/model.safetensors\" ]; then exit 2; fi\nprintf 'RIFF\\044\\000\\000\\000WAVEfmt ' > \"$out\"\n"
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	originalLookPath := voxcpmLookPath
	voxcpmLookPath = func(string) (string, error) { return command, nil }
	defer func() { voxcpmLookPath = originalLookPath }()
	voxcpmCheckCache.Lock()
	originalCheckedAt, originalExecutable := voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable
	voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = time.Time{}, ""
	voxcpmCheckCache.Unlock()
	defer func() {
		voxcpmCheckCache.Lock()
		voxcpmCheckCache.checkedAt, voxcpmCheckCache.executable = originalCheckedAt, originalExecutable
		voxcpmCheckCache.Unlock()
	}()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ensureVoxCPMTaskSchema(db); err != nil {
		t.Fatal(err)
	}
	manager := &voxcpmTaskManager{cfg: &Config{AgentDir: root}, db: db, wake: make(chan struct{}, 1)}
	created, err := manager.create(voxcpmTaskCreateRequest{AgentID: "agent-a", OutputName: "mirror", Tasks: []voxcpmTaskCreateItem{{TextPath: "tmp/speech.txt"}}})
	if err != nil || len(created) != 1 {
		t.Fatalf("create task = %#v, %v", created, err)
	}
	if err := manager.generate(context.Background(), created[0]); err != nil {
		t.Fatalf("generate with ModelScope fallback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheRoot, "OpenBMB--VoxCPM2", "model.safetensors")); err != nil {
		t.Fatalf("ModelScope model was not cached: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "audios", "mirror.wav")); err != nil {
		t.Fatalf("fallback output was not produced: %v", err)
	}
	var logs string
	if err := db.QueryRow(`SELECT logs FROM voxcpm_task WHERE id=?`, created[0].ID).Scan(&logs); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs, "ModelScope 国内源检查并准备") || !strings.Contains(logs, "ModelScope 国内源模型已校验") {
		t.Fatalf("expected controlled ModelScope logs, got %q", logs)
	}
}

func TestVoxCPMBackupManifestUsesSourceSpecificRuntimeMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/tree" {
			http.NotFound(writer, request)
			return
		}
		_ = json.NewEncoder(writer).Encode([]map[string]interface{}{
			{"type": "file", "path": ".gitattributes", "size": 1519},
			{"type": "file", "path": "config.json", "size": 81},
			{"type": "file", "path": "model.safetensors", "size": 4096, "lfs": map[string]interface{}{"size": 4096, "oid": strings.Repeat("a", 64)}},
			{"type": "file", "path": "audiovae.pth", "size": 2048, "lfs": map[string]interface{}{"size": 2048, "oid": strings.Repeat("b", 64)}},
		})
	}))
	defer server.Close()

	previousURL, previousClient := voxcpmHuggingFaceFilesURL, voxcpmHTTPClient
	voxcpmHuggingFaceFilesURL, voxcpmHTTPClient = server.URL+"/tree", server.Client()
	t.Cleanup(func() { voxcpmHuggingFaceFilesURL, voxcpmHTTPClient = previousURL, previousClient })

	resolver := &voxcpmModelBackupManifestResolver{}
	metadata, err := resolver.metadata(context.Background(), "model.safetensors")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.SourceID != "huggingface:openbmb/VoxCPM2" || metadata.Revision != "main" || metadata.ExpectedSize != 4096 {
		t.Fatalf("backup metadata = %#v", metadata)
	}
	if _, found := resolver.files[".gitattributes"]; found {
		t.Fatal("non-runtime .gitattributes must not take part in model verification")
	}
	if got := resolver.files["model.safetensors"].SHA256; got != strings.Repeat("a", 64) {
		t.Fatalf("backup SHA-256 = %q", got)
	}
}

func TestVoxCPMGPUDevice(t *testing.T) {
	for _, device := range []string{"cuda", "CUDA", "mps", " MPS "} {
		if !voxcpmGPUDevice(device) {
			t.Fatalf("%q must be a GPU device", device)
		}
	}
	for _, device := range []string{"", "cpu", "auto"} {
		if voxcpmGPUDevice(device) {
			t.Fatalf("%q must not be a GPU device", device)
		}
	}
}
