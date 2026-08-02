package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseVoxCPMCheckConfig(t *testing.T) {
	config, err := parseVoxCPMCheckConfig(map[string]interface{}{"voxcpm": map[string]interface{}{"check": float64(24), "install": "请安装 voxcpm"}})
	if err != nil || config.CacheFor != 24*time.Hour || config.Install != "请安装 voxcpm" {
		t.Fatalf("parse valid config = %#v, %v", config, err)
	}
	if _, err := parseVoxCPMCheckConfig(map[string]interface{}{"voxcpm": map[string]interface{}{"check": 0, "install": "x"}}); err == nil {
		t.Fatal("zero cache duration must be rejected")
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

func TestVoxCPMCloneIgnoresExpressionStyle(t *testing.T) {
	root := t.TempDir()
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
