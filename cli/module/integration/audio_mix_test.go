package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHandleAudioMixBuildsRestrictedWAVProject(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command shim uses POSIX sh")
	}
	root := t.TempDir()
	sourcePath := filepath.Join(root, "agent-a", "sources", "voice.any")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("source-audio"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	previousLookPath := videoTrimLookPathFn
	previousCommand := videoTrimCommandContextFn
	t.Cleanup(func() {
		videoTrimLookPathFn = previousLookPath
		videoTrimCommandContextFn = previousCommand
	})
	var ffmpegArgs []string
	videoTrimLookPathFn = func(name string) (string, error) {
		if name != "ffmpeg" && name != "ffprobe" {
			return "", errors.New("unexpected executable")
		}
		return name, nil
	}
	videoTrimCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "ffprobe" {
			return exec.CommandContext(ctx, "sh", "-c", "printf 'audio\\n12.5\\n'")
		}
		ffmpegArgs = append([]string(nil), args...)
		return exec.CommandContext(ctx, "sh", "-c", "cp \"$1\" \"$2\"", "audio-mix-test", sourcePath, args[len(args)-1])
	}

	request := AudioMixRequest{
		AgentID:    "agent-a",
		OutputName: "final.wav",
		SampleRate: 48000,
		BitDepth:   24,
		Tracks: []AudioMixTrack{{
			Path: "sources/voice.any", TrimStart: 1, TrimEnd: 3, Start: 2, Speed: 2,
			Loop: true, Duration: 3, Volume: 1, LeftVolume: 0.8, RightVolume: 0,
			FadeIn: 0.2, FadeOut: 0.3,
			Effects: []AudioMixEffect{{Type: "equalizer", Amount: 3}, {Type: "delay", Amount: 0.4}},
		}},
	}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	recorder := httptest.NewRecorder()
	handleAudioMix(&Config{AgentDir: root}).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/audio_mix", bytes.NewReader(body)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var response AudioMixResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if response.Status != 0 || response.Path != "audios/final.wav" || response.SavedAs == "" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if saved, err := os.ReadFile(response.SavedAs); err != nil || !bytes.Equal(saved, []byte("source-audio")) {
		t.Fatalf("saved WAV = %q, err=%v", saved, err)
	}
	joined := strings.Join(ffmpegArgs, " ")
	for _, expected := range []string{"-filter_complex", "amix=inputs=1", "atempo=2", "pcm_s24le", "-ar 48000", "-ac 2", "equalizer=", "aecho="} {
		if !strings.Contains(joined, expected) {
			t.Errorf("ffmpeg args missing %q: %q", expected, joined)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "agent-a", "audios"))
	if err != nil {
		t.Fatalf("read output directory: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp.wav") {
			t.Errorf("temporary WAV remains: %s", entry.Name())
		}
	}
}

func TestHandleAudioMixRejectsInvalidProject(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "agent-a"), 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	request := AudioMixRequest{
		AgentID: "agent-a", OutputName: "invalid.wav", SampleRate: 44100, BitDepth: 16,
		Tracks: []AudioMixTrack{{Path: "../../outside.wav", TrimStart: 0, TrimEnd: 1, Speed: 1, Volume: 1, LeftVolume: 1, RightVolume: 1}},
	}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	recorder := httptest.NewRecorder()
	handleAudioMix(&Config{AgentDir: root}).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/audio_mix", bytes.NewReader(body)))
	if recorder.Code != http.StatusBadRequest {
		payload, _ := io.ReadAll(recorder.Body)
		t.Fatalf("status = %d, body=%s", recorder.Code, payload)
	}
	if _, err := os.Stat(filepath.Join(root, "outside.wav")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid request wrote outside workspace: %v", err)
	}
}

func TestHandleAudioMixRendersRealStereoWAV(t *testing.T) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed")
	}
	root := t.TempDir()
	sourcePath := filepath.Join(root, "agent-a", "sources", "tone.wav")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if output, err := exec.Command(ffmpegPath, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100", "-t", "0.5", "-c:a", "pcm_s16le", "-y", sourcePath).CombinedOutput(); err != nil {
		t.Fatalf("create source WAV: %v: %s", err, output)
	}
	request := AudioMixRequest{AgentID: "agent-a", OutputName: "real.wav", SampleRate: 44100, BitDepth: 16, Tracks: []AudioMixTrack{{
		Path: "sources/tone.wav", TrimStart: 0, TrimEnd: 0.5, Start: 0, Speed: 1, Loop: true, Duration: 1,
		Volume: 1, LeftVolume: 1, RightVolume: 0.5, FadeIn: 0.05, FadeOut: 0.05,
		Effects: []AudioMixEffect{{Type: "compressor", Amount: 0.25}, {Type: "reverb", Amount: 0.15}},
	}}}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	recorder := httptest.NewRecorder()
	handleAudioMix(&Config{AgentDir: root}).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/audio_mix", bytes.NewReader(body)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var response AudioMixResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, err := os.ReadFile(response.SavedAs)
	if err != nil || !isValidPCMContainerWAV(data) {
		t.Fatalf("output is not a valid 16-bit WAV: err=%v", err)
	}
}
