package main

// Wav2Lip lip-sync tasks use the official Wav2Lip inference script. Keeping
// this queue separate from image tasks means a long video never blocks them.

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	wav2lipTaskQueued    = "queued"
	wav2lipTaskRunning   = "running"
	wav2lipTaskCompleted = "completed"
	wav2lipTaskCancelled = "cancelled"
	wav2lipTaskFailed    = "failed"
	wav2lipTaskLogLimit  = 128 * 1024
	wav2lipProbeTimeout  = time.Minute
	// The checkpoint is kept separate from the source tree so the existing
	// runtime probe can verify it before scheduling a lip-sync task.
	wav2lipDomesticCheckpointURL    = "https://hf-mirror.com/camenduru/Wav2Lip/resolve/main/checkpoints/wav2lip_gan.pth"
	wav2lipDomesticCheckpointSHA256 = "ca9ab7b7b812c0e80a6e70a5977c545a1e8a365a6c49d5e533023c034d7ac3d8"
)

type wav2lipCheckConfig struct {
	CacheFor time.Duration
	Install  string
}
type wav2lipCheckResponse struct {
	Available bool   `json:"available"`
	Install   string `json:"install,omitempty"`
	Content   string `json:"content,omitempty"`
	AgentID   string `json:"agentId,omitempty"`
	Status    int    `json:"status"`
}
type wav2lipRuntime struct{ Python, Script, Checkpoint string }
type wav2lipStagingFiles struct {
	Directory string
	Video     string
	Audio     string
	Output    string
}

// wav2lipRuntimeBootstrap executes the unmodified upstream script in a
// separate process. It adapts only process-local APIs whose signatures changed
// between supported Python dependency releases; the downloaded Wav2Lip tree
// and checkpoint are never written or patched.
const wav2lipRuntimeBootstrap = `
import os
import sys
import socket
import hashlib
import json
import concurrent.futures
import threading
import time
import urllib.request

# The upstream S3FD detector downloads its checkpoint through urllib on first
# use. Bound a stalled socket so a missing download never leaves a task running
# indefinitely without producing output. The downloader below additionally
# applies a per-part total and no-progress deadline.
socket.setdefaulttimeout(30)

# Wav2Lip's older dependency pins pre-date NumPy 1.24, which removed these
# aliases. Restore them only for this child process before librosa imports.
import numpy as np
for _deepright_name, _deepright_value in (
    ('complex', complex), ('float', float), ('int', int), ('bool', bool),
):
    if _deepright_name not in np.__dict__:
        setattr(np, _deepright_name, _deepright_value)

# PyTorch 2.6 changed torch.load's default to weights_only=True. Official
# Wav2Lip checkpoints contain the conventional state-dict wrapper and need
# the earlier loader behavior. Do not pass an unknown keyword to old PyTorch.
import inspect
import torch
try:
    _deepright_supports_weights_only = 'weights_only' in inspect.signature(torch.load).parameters
except (TypeError, ValueError):
    _deepright_supports_weights_only = False
if _deepright_supports_weights_only:
    _deepright_torch_load = torch.load
    def _deepright_compatible_torch_load(*args, **kwargs):
        kwargs.setdefault('weights_only', False)
        return _deepright_torch_load(*args, **kwargs)
    torch.load = _deepright_compatible_torch_load

# Wav2Lip's bundled audio.py uses the pre-0.10 positional signature
# librosa.filters.mel(sample_rate, n_fft, ...).  Newer librosa releases make
# these keyword-only. Adapt only this child-process module object; no third-
# party file is changed on disk.
import librosa
_deepright_librosa_mel = librosa.filters.mel
def _deepright_compatible_mel(*args, **kwargs):
    if len(args) > 2:
        return _deepright_librosa_mel(*args, **kwargs)
    if len(args) > 0:
        kwargs.setdefault('sr', args[0])
    if len(args) > 1:
        kwargs.setdefault('n_fft', args[1])
    return _deepright_librosa_mel(**kwargs)
librosa.filters.mel = _deepright_compatible_mel

script = os.path.abspath(sys.argv[1])
script_dir = os.path.dirname(script)
if script_dir not in sys.path:
    sys.path.insert(0, script_dir)
os.chdir(script_dir)

# Wav2Lip otherwise lets face_detection download S3FD lazily from a single
# overseas URL. Populate its documented local file location ourselves. Every
# source first uses HTTP Range parts and falls back to one ordinary transfer
# only when Range is unavailable or that transfer fails. A failed primary then
# repeats the same order with the domestic backup source.
_deepright_s3fd = os.path.join(script_dir, 'face_detection', 'detection', 'sfd', 's3fd.pth')
if not os.path.isfile(_deepright_s3fd):
    _deepright_tmp = _deepright_s3fd + '.deepright-part'
    _deepright_state = _deepright_tmp + '.json'
    try:
        _deepright_workers = max(1, int(os.environ.get('DEEPRIGHT_MODEL_TASK_DOWNLOAD', '10')))
    except ValueError:
        _deepright_workers = 10
    try:
        _deepright_part_timeout = max(1, int(os.environ.get('DEEPRIGHT_MODEL_DOWNLOAD_PART_TIMEOUT', '900')))
    except ValueError:
        _deepright_part_timeout = 900
    try:
        _deepright_retries = max(0, int(os.environ.get('DEEPRIGHT_MODEL_DOWNLOAD_RETRY', '3')))
    except ValueError:
        _deepright_retries = 3
    _deepright_idle_timeout = 90
    _deepright_socket_timeout = min(30, _deepright_part_timeout)
    _deepright_chunk_size = 4 * 1024 * 1024
    _deepright_sources = (
        ('主下载源', 'https://www.adrianbulat.com/downloads/python-fan/s3fd-619a316812.pth', None),
        ('备用下载源', 'https://hf-mirror.com/irishavmishra/s3fd-face-detector/resolve/main/sfd_face.pth', 'd54a87c2b7543b64729c9a25eafd188da15fd3f6e02f0ecec76ae1b30d86c491'),
    )

    def _deepright_range_size(url):
        request = urllib.request.Request(url, headers={'Range': 'bytes=0-0'})
        with urllib.request.urlopen(request, timeout=_deepright_socket_timeout) as response:
            content_range = response.headers.get('Content-Range') or ''
            if response.getcode() != 206 or '/' not in content_range:
                return None
            try:
                return int(content_range.rsplit('/', 1)[1])
            except ValueError:
                return None

    def _deepright_save_state(state):
        temporary = _deepright_state + '.new'
        with open(temporary, 'w', encoding='utf-8') as file:
            json.dump(state, file)
        os.replace(temporary, _deepright_state)

    def _deepright_range_download(url, total):
        chunks = (total + _deepright_chunk_size - 1) // _deepright_chunk_size
        if chunks < 2:
            return False
        state = {'url': url, 'size': total, 'chunks': [False] * chunks}
        try:
            with open(_deepright_state, 'r', encoding='utf-8') as file:
                stored = json.load(file)
            if stored.get('url') == url and stored.get('size') == total and len(stored.get('chunks', [])) == chunks:
                state = stored
            else:
                os.remove(_deepright_tmp)
        except FileNotFoundError:
            pass
        except Exception:
            try:
                os.remove(_deepright_tmp)
            except FileNotFoundError:
                pass
        with open(_deepright_tmp, 'a+b') as file:
            file.truncate(total)
        _deepright_save_state(state)
        copied = sum(min(_deepright_chunk_size, total - index * _deepright_chunk_size) for index, complete in enumerate(state['chunks']) if complete)
        next_percent = (copied * 100 // total // 10 + 1) * 10
        lock = threading.Lock()
        thread_ids = {}
        thread_ids_lock = threading.Lock()
        def thread_id():
            ident = threading.get_ident()
            with thread_ids_lock:
                if ident not in thread_ids:
                    thread_ids[ident] = len(thread_ids) + 1
                return thread_ids[ident]
        def download_part(index):
            worker = thread_id()
            start = index * _deepright_chunk_size
            end = min(total - 1, start + _deepright_chunk_size - 1)
            part_total = end - start + 1
            last_error = None
            for retry in range(_deepright_retries + 1):
                if retry > 0:
                    print('Wav2Lip S3FD 下载线程 %d，分段 %d/%d 第 %d 次重试（最多 %d 次）：%s' % (worker, index + 1, chunks, retry, _deepright_retries, last_error), flush=True)
                try:
                    started_at = time.monotonic()
                    last_progress_at = started_at
                    part_copied = 0
                    part_next_percent = 10
                    request = urllib.request.Request(url, headers={'Range': 'bytes=%d-%d' % (start, end)})
                    with urllib.request.urlopen(request, timeout=_deepright_socket_timeout) as response:
                        if response.getcode() != 206:
                            raise RuntimeError('分段请求返回 HTTP %s' % response.getcode())
                        with open(_deepright_tmp, 'r+b') as file:
                            file.seek(start)
                            while True:
                                now = time.monotonic()
                                if now - started_at >= _deepright_part_timeout:
                                    raise RuntimeError('下载线程 %d 的分段 %d 超过总时限 %d 秒' % (worker, index + 1, _deepright_part_timeout))
                                if now - last_progress_at >= _deepright_idle_timeout:
                                    raise RuntimeError('下载线程 %d 的分段 %d 超过 %d 秒没有进度' % (worker, index + 1, _deepright_idle_timeout))
                                data = response.read(256 * 1024)
                                if not data:
                                    break
                                file.write(data)
                                part_copied += len(data)
                                last_progress_at = time.monotonic()
                                while part_copied * 100 >= part_total * part_next_percent and part_next_percent <= 100:
                                    print('Wav2Lip S3FD 下载线程 %d，分段 %d/%d 已下载 %d%%。' % (worker, index + 1, chunks, part_next_percent), flush=True)
                                    part_next_percent += 10
                    if part_copied != part_total:
                        raise RuntimeError('分段大小不正确')
                    return index, part_copied
                except Exception as error:
                    last_error = error
                    if retry == _deepright_retries:
                        raise
        pending = [index for index, complete in enumerate(state['chunks']) if not complete]
        if not pending:
            return True
        with concurrent.futures.ThreadPoolExecutor(max_workers=min(_deepright_workers, len(pending))) as pool:
            for index, size in pool.map(download_part, pending):
                with lock:
                    state['chunks'][index] = True
                    copied += size
                    _deepright_save_state(state)
                    progress = copied * 100 // total
                    while progress >= next_percent and next_percent <= 100:
                        print('Wav2Lip S3FD 人脸检测模型已下载 %d%%。' % next_percent, flush=True)
                        next_percent += 10
        return True

    def _deepright_plain_download(url):
        try:
            os.remove(_deepright_state)
        except FileNotFoundError:
            pass
        last_error = None
        for retry in range(_deepright_retries + 1):
            if retry > 0:
                print('Wav2Lip S3FD 普通下载第 %d 次重试（最多 %d 次）：%s' % (retry, _deepright_retries, last_error), flush=True)
            try:
                started_at = time.monotonic()
                last_progress_at = started_at
                with urllib.request.urlopen(url, timeout=_deepright_socket_timeout) as response, open(_deepright_tmp, 'wb') as file:
                    total = int(response.headers.get('Content-Length') or 0)
                    copied = 0
                    next_percent = 10
                    next_bytes = 10 * 1024 * 1024
                    while True:
                        now = time.monotonic()
                        if now - started_at >= _deepright_part_timeout:
                            raise RuntimeError('普通下载超过总时限 %d 秒' % _deepright_part_timeout)
                        if now - last_progress_at >= _deepright_idle_timeout:
                            raise RuntimeError('普通下载超过 %d 秒没有进度' % _deepright_idle_timeout)
                        data = response.read(1024 * 1024)
                        if not data:
                            break
                        file.write(data)
                        copied += len(data)
                        last_progress_at = time.monotonic()
                        if total > 0:
                            while copied * 100 >= total * next_percent and next_percent <= 100:
                                print('Wav2Lip S3FD 人脸检测模型已下载 %d%%。' % next_percent, flush=True)
                                next_percent += 10
                        elif copied >= next_bytes:
                            print('Wav2Lip S3FD 人脸检测模型已下载 %d MiB（下载源未提供总大小）。' % (copied // (1024 * 1024)), flush=True)
                            next_bytes = copied + 10 * 1024 * 1024
                return
            except Exception as error:
                last_error = error
                if retry == _deepright_retries:
                    raise

    _deepright_errors = []
    for _deepright_name, _deepright_url, _deepright_sha256 in _deepright_sources:
        try:
            print('正在从%s下载 Wav2Lip S3FD 人脸检测模型。下载源：%s；下载目标绝对路径：%s' % (_deepright_name, _deepright_url, _deepright_s3fd), flush=True)
            _deepright_total = _deepright_range_size(_deepright_url)
            if _deepright_total is None:
                print('%s 不支持 HTTP Range，回退到普通下载。' % _deepright_name, flush=True)
                _deepright_plain_download(_deepright_url)
            else:
                print('%s 支持 HTTP Range，使用 %d 路分段断点续传。' % (_deepright_name, _deepright_workers), flush=True)
                try:
                    if not _deepright_range_download(_deepright_url, _deepright_total):
                        print('%s 文件较小，回退到普通下载。' % _deepright_name, flush=True)
                        _deepright_plain_download(_deepright_url)
                except Exception as _deepright_range_error:
                    print('%s 多线程断点续传失败：%s；回退到普通下载。' % (_deepright_name, _deepright_range_error), flush=True)
                    _deepright_plain_download(_deepright_url)
            if _deepright_sha256 is not None:
                with open(_deepright_tmp, 'rb') as _deepright_file:
                    _deepright_actual = hashlib.sha256(_deepright_file.read()).hexdigest()
                if _deepright_actual != _deepright_sha256:
                    raise RuntimeError('SHA-256 校验失败')
            os.replace(_deepright_tmp, _deepright_s3fd)
            try:
                os.remove(_deepright_state)
            except FileNotFoundError:
                pass
            print('Wav2Lip S3FD 人脸检测模型下载完成。', flush=True)
            break
        except Exception as _deepright_error:
            _deepright_errors.append('%s：%s' % (_deepright_name, _deepright_error))
            if _deepright_name == '主下载源':
                print('主下载源普通下载失败，回退到多线程断点备用源。', flush=True)
    else:
        raise RuntimeError('Wav2Lip S3FD 人脸检测模型下载失败；' + '；'.join(_deepright_errors))
sys.argv = [script] + sys.argv[2:]
device = os.environ.get('DEEPRIGHT_WAV2LIP_DEVICE', 'cpu')
with open(script, 'rb') as f:
    source = f.read().decode('utf-8')
old = "device = 'cuda' if torch.cuda.is_available() else 'cpu'"
if old not in source:
    raise RuntimeError('不支持的 Wav2Lip inference.py：未找到设备选择代码')
source = source.replace(old, "device = " + repr(device), 1)
scope = {'__name__': '__main__', '__file__': script, '__package__': None}
exec(compile(source, script, 'exec'), scope)
`

type wav2lipCachedRuntime struct {
	checkedAt time.Time
	runtime   wav2lipRuntime
}
type wav2lipTask struct {
	ID            int64  `json:"id"`
	AgentID       string `json:"agentId"`
	VideoPath     string `json:"videoPath"`
	AudioPath     string `json:"audioPath"`
	OutputPath    string `json:"outputPath,omitempty"`
	SavedAs       string `json:"savedAs,omitempty"`
	FailureReason string `json:"failureReason,omitempty"`
	Status        string `json:"status"`
	Progress      int    `json:"progress"`
	StartedAt     string `json:"startedAt,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	Logs          string `json:"logs,omitempty"`
	CancelAsked   bool   `json:"cancelRequested"`
}
type wav2lipTaskPage struct {
	Tasks                 []wav2lipTask
	Total, Page, PageSize int
	Status                string
}
type wav2lipTaskCreateRequest struct {
	AgentID string                  `json:"agentId"`
	Tasks   []wav2lipTaskCreateItem `json:"tasks"`
}
type wav2lipTaskCreateItem struct {
	VideoPath string `json:"videoPath"`
	AudioPath string `json:"audioPath"`
}
type wav2lipTaskActionRequest struct {
	AgentID string `json:"agentId"`
	ID      int64  `json:"id"`
}
type wav2lipTaskManager struct {
	cfg          *Config
	db           *sql.DB
	createMu, mu sync.Mutex
	runningID    int64
	cancel       context.CancelFunc
	wake         chan struct{}
}

var (
	wav2lipTasks          *wav2lipTaskManager
	wav2lipLookPath       = exec.LookPath
	wav2lipCommandContext = exec.CommandContext
	wav2lipNow            = time.Now
	wav2lipCheckCache     struct {
		sync.Mutex
		entries map[string]wav2lipCachedRuntime
	}
)

func wav2lipTimestamp() string { return wav2lipNow().Format(time.RFC3339Nano) }

func parseWav2LipCheckConfig(raw map[string]interface{}) (wav2lipCheckConfig, error) {
	value, ok := raw["wav2lip"]
	if !ok {
		return wav2lipCheckConfig{}, errors.New("config/config.json.wav2lip 配置缺失")
	}
	section, ok := value.(map[string]interface{})
	if !ok || section == nil {
		return wav2lipCheckConfig{}, errors.New("config/config.json.wav2lip 必须是对象")
	}
	hours, ok := ffmpegCheckPositiveHours(section["check"])
	if !ok {
		return wav2lipCheckConfig{}, errors.New("config/config.json.wav2lip.check 必须是正整数（小时）")
	}
	install, ok := section["install"].(string)
	install = strings.TrimSpace(install)
	if !ok || install == "" {
		return wav2lipCheckConfig{}, errors.New("config/config.json.wav2lip.install 必须是非空字符串")
	}
	return wav2lipCheckConfig{CacheFor: time.Duration(hours) * time.Hour, Install: install}, nil
}

func wav2lipRuntimeFor(cacheFor time.Duration, workspace string) (wav2lipRuntime, bool) {
	for _, candidate := range wav2lipRuntimeCandidates(workspace) {
		key := candidate.Script + "\x00" + candidate.Checkpoint
		now := wav2lipNow()
		wav2lipCheckCache.Lock()
		cached, ok := wav2lipCheckCache.entries[key]
		wav2lipCheckCache.Unlock()
		if ok && cached.runtime.Python != "" && now.Before(cached.checkedAt.Add(cacheFor)) {
			return cached.runtime, true
		}
		for _, python := range wav2lipPythonCandidates() {
			probe, cancel := context.WithTimeout(context.Background(), wav2lipProbeTimeout)
			// Probe with exactly the compatibility bootstrap used for the real
			// job. A modern Python may reject the upstream script's old librosa or
			// PyTorch assumptions even though the isolated wrapper can run it.
			err := wav2lipCommandContext(probe, python, wav2lipProbeArgs(candidate.Script)...).Run()
			timedOut := probe.Err() != nil
			cancel()
			if err != nil || timedOut {
				continue
			}
			value := wav2lipRuntime{Python: python, Script: candidate.Script, Checkpoint: candidate.Checkpoint}
			wav2lipCheckCache.Lock()
			if wav2lipCheckCache.entries == nil {
				wav2lipCheckCache.entries = make(map[string]wav2lipCachedRuntime)
			}
			wav2lipCheckCache.entries[key] = wav2lipCachedRuntime{checkedAt: now, runtime: value}
			wav2lipCheckCache.Unlock()
			return value, true
		}
	}
	return wav2lipRuntime{}, false
}

func wav2lipRuntimeCandidates(workspace string) []wav2lipRuntime {
	values := make([]wav2lipRuntime, 0, 1)
	add := func(home, checkpoint string) {
		home, checkpoint = strings.TrimSpace(home), strings.TrimSpace(checkpoint)
		if home == "" || checkpoint == "" {
			return
		}
		script, err := filepath.Abs(filepath.Join(home, "inference.py"))
		if err != nil {
			return
		}
		checkpoint, err = filepath.Abs(checkpoint)
		if err != nil {
			return
		}
		scriptInfo, scriptErr := os.Stat(script)
		checkpointInfo, checkpointErr := os.Stat(checkpoint)
		if scriptErr != nil || scriptInfo.IsDir() || checkpointErr != nil || checkpointInfo.IsDir() || checkpointInfo.Size() == 0 {
			return
		}
		for _, value := range values {
			if value.Script == script && value.Checkpoint == checkpoint {
				return
			}
		}
		values = append(values, wav2lipRuntime{Script: script, Checkpoint: checkpoint})
	}
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return values
	}
	home := filepath.Join(workspace, "wav2lip")
	// Installation, checking and task execution use this same fixed layout.
	add(home, filepath.Join(home, "checkpoints", "wav2lip_gan.pth"))
	return values
}

func wav2lipRequiredPaths(workspace string) string {
	home := filepath.Join(workspace, "wav2lip")
	return filepath.Join(home, "inference.py") + " 与 " + filepath.Join(home, "checkpoints", "wav2lip_gan.pth")
}

func wav2lipPythonCandidates() []string {
	values := make([]string, 0, 4)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range values {
			if existing == value {
				return
			}
		}
		values = append(values, value)
	}
	if python, err := wav2lipLookPath("python3"); err == nil {
		add(python)
	}
	if runtime.GOOS == "darwin" {
		if candidates, err := filepath.Glob("/Library/Frameworks/Python.framework/Versions/*/bin/python3"); err == nil {
			for index := len(candidates) - 1; index >= 0; index-- {
				add(candidates[index])
			}
		}
	}
	return values
}

func wav2lipRuntimeForStartup(cacheFor time.Duration, cfg *Config) (wav2lipRuntime, bool) {
	if cfg == nil {
		return wav2lipRuntime{}, false
	}
	root, err := resolveIntegrationAgentDir(cfg.AgentDir)
	if err != nil {
		return wav2lipRuntime{}, false
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return wav2lipRuntime{}, false
	}
	var first wav2lipRuntime
	available := false
	for _, entry := range entries {
		if !entry.IsDir() || !isValidMediaPreviewAgentID(entry.Name()) {
			continue
		}
		workspace, workspaceErr := getWorkspaceByAgentID(cfg, entry.Name())
		if workspaceErr != nil {
			continue
		}
		if value, ok := wav2lipRuntimeFor(cacheFor, workspace); ok {
			if !available {
				first = value
			}
			available = true
		}
	}
	return first, available
}

func wav2lipInstallRequest(template, workspace string) string {
	request := strings.ReplaceAll(template, "$workspace", workspace)
	checkpoint := filepath.Join(workspace, "wav2lip", "checkpoints", "wav2lip_gan.pth")
	return request + "\n\n模型下载要求：先按配置中的官方权重地址下载 `wav2lip_gan.pth` 到 `" + checkpoint + "`。官方源失败后回退国内源 `" + wav2lipDomesticCheckpointURL + "`。下载连接超时最多 30 秒，连续 90 秒无字节进度必须终止并报告。完成后校验 SHA-256 必须为 `" + wav2lipDomesticCheckpointSHA256 + "`。"
}

func wav2lipWorkspaceForRequest(cfg *Config, agentID string) (string, string, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return "", "", errors.New("agentId is required")
	}
	workspace, err := getWorkspaceByAgentID(cfg, agentID)
	if err != nil {
		return "", "", err
	}
	return agentID, workspace, nil
}

func checkWav2LipDependency(integrationConfig *Config, agentID string) (int, wav2lipCheckResponse) {
	raw, _, err := readIntegrationStartupConfigRaw()
	if err != nil {
		return http.StatusInternalServerError, wav2lipCheckResponse{Content: "读取 config/config.json 失败: " + err.Error(), Status: 1}
	}
	checkConfig, err := parseWav2LipCheckConfig(raw)
	if err != nil {
		return http.StatusBadRequest, wav2lipCheckResponse{Content: err.Error(), Status: 1}
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		_, available := wav2lipRuntimeForStartup(checkConfig.CacheFor, integrationConfig)
		if available {
			return http.StatusOK, wav2lipCheckResponse{Available: true, Status: 0}
		}
		return http.StatusOK, wav2lipCheckResponse{Content: "未检测到已安装的 Wav2Lip", Status: 0}
	}
	agentID, workspace, err := wav2lipWorkspaceForRequest(integrationConfig, agentID)
	if err != nil {
		return http.StatusNotFound, wav2lipCheckResponse{Content: "Agent 工作目录不存在", Status: 1}
	}
	if _, available := wav2lipRuntimeFor(checkConfig.CacheFor, workspace); !available {
		return http.StatusOK, wav2lipCheckResponse{
			Install: wav2lipInstallRequest(checkConfig.Install, workspace),
			Content: "未检测到可用的 Wav2Lip（需要 " + wav2lipRequiredPaths(workspace) + "）",
			AgentID: agentID,
			Status:  0,
		}
	}
	return http.StatusOK, wav2lipCheckResponse{Available: true, AgentID: agentID, Status: 0}
}
func writeWav2LipJSON(w http.ResponseWriter, code int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func handleWav2LipCheck(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeWav2LipJSON(w, http.StatusMethodNotAllowed, wav2lipCheckResponse{Content: "仅支持 GET 请求", Status: 1})
			return
		}
		status, result := checkWav2LipDependency(cfg, r.URL.Query().Get("agentId"))
		writeWav2LipJSON(w, status, result)
	}
}

func ensureWav2LipTaskSchema(db *sql.DB) error {
	if db == nil {
		return errors.New("wav2lip task database is not ready")
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS wav2lip_task (
		id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT NOT NULL, video_path TEXT NOT NULL, audio_path TEXT NOT NULL, output_path TEXT NOT NULL DEFAULT '', saved_as TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'queued', progress INTEGER NOT NULL DEFAULT 0, started_at TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, logs TEXT NOT NULL DEFAULT '', failure_reason TEXT NOT NULL DEFAULT '', cancel_requested INTEGER NOT NULL DEFAULT 0
	); CREATE INDEX IF NOT EXISTS idx_wav2lip_task_agent_created ON wav2lip_task(agent_id, id DESC); CREATE INDEX IF NOT EXISTS idx_wav2lip_task_queue ON wav2lip_task(status, cancel_requested, id);`)
	if err != nil {
		return err
	}
	rows, err := db.Query(`PRAGMA table_info(wav2lip_task)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	hasFailureReason := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == "failure_reason" {
			hasFailureReason = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasFailureReason {
		_, err = db.Exec(`ALTER TABLE wav2lip_task ADD COLUMN failure_reason TEXT NOT NULL DEFAULT ''`)
	}
	return err
}
func startWav2LipTaskManager(ctx context.Context, cfg *Config) {
	if cronDB == nil {
		log.Printf("[wav2lip] task queue unavailable: database is not ready")
		return
	}
	if err := ensureWav2LipTaskSchema(cronDB); err != nil {
		log.Printf("[wav2lip] ensure task schema failed: %v", err)
		return
	}
	m := &wav2lipTaskManager{cfg: cfg, db: cronDB, wake: make(chan struct{}, 1)}
	if err := m.recover(); err != nil {
		log.Printf("[wav2lip] recover interrupted tasks failed: %v", err)
	}
	wav2lipTasks = m
	go m.run(ctx)
}
func (m *wav2lipTaskManager) recover() error {
	now := wav2lipTimestamp()
	if _, err := m.db.Exec(`UPDATE wav2lip_task SET status=?,progress=0,started_at='',updated_at=?,logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ? WHERE status=? AND cancel_requested=0`, wav2lipTaskQueued, now, wav2lipTaskLogLimit-2048, wav2lipTaskLogLimit/2, "\n[服务重启] 上次执行被中断，任务已恢复为排队中。\n", wav2lipTaskRunning); err != nil {
		return err
	}
	_, err := m.db.Exec(`UPDATE wav2lip_task SET status=?,updated_at=? WHERE status=? AND cancel_requested=1`, wav2lipTaskCancelled, now, wav2lipTaskRunning)
	return err
}
func (m *wav2lipTaskManager) signal() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *wav2lipTaskManager) run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			m.cancelRunning()
			return
		}
		slotHeld, err := reserveModelTaskSlot(ctx, m.cfg, m.db, "wav2lip_task")
		if err != nil {
			if errors.Is(err, context.Canceled) {
				m.cancelRunning()
				return
			}
			log.Printf("[wav2lip] reserve shared task slot failed: %v", err)
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if !slotHeld {
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-m.wake:
			}
			continue
		}
		task, err := m.claim()
		if err != nil {
			releaseModelTaskSlot(m.cfg)
			log.Printf("[wav2lip] claim task failed: %v", err)
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-time.After(time.Second):
			}
			continue
		}
		if task == nil {
			releaseModelTaskSlot(m.cfg)
			select {
			case <-ctx.Done():
				m.cancelRunning()
				return
			case <-m.wake:
			}
			continue
		}
		m.execute(ctx, *task)
		releaseModelTaskSlot(m.cfg)
	}
}
func (m *wav2lipTaskManager) claim() (*wav2lipTask, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	task, err := scanWav2LipTask(tx.QueryRow(`SELECT id,agent_id,video_path,audio_path,output_path,saved_as,status,progress,started_at,created_at,updated_at,logs,failure_reason,cancel_requested FROM wav2lip_task WHERE status=? AND cancel_requested=0 ORDER BY id LIMIT 1`, wav2lipTaskQueued))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if err != nil {
		return nil, err
	}
	now := wav2lipTimestamp()
	result, err := tx.Exec(`UPDATE wav2lip_task SET status=?,started_at=?,updated_at=?,progress=0 WHERE id=? AND status=? AND cancel_requested=0`, wav2lipTaskRunning, now, now, task.ID, wav2lipTaskQueued)
	if err != nil {
		return nil, err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return nil, tx.Commit()
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	task.Status, task.StartedAt, task.UpdatedAt, task.Progress = wav2lipTaskRunning, now, now, 0
	return &task, nil
}
func (m *wav2lipTaskManager) execute(parent context.Context, task wav2lipTask) {
	ctx, cancel := context.WithCancel(parent)
	m.mu.Lock()
	m.runningID, m.cancel = task.ID, cancel
	m.mu.Unlock()
	defer func() {
		cancel()
		m.mu.Lock()
		if m.runningID == task.ID {
			m.runningID, m.cancel = 0, nil
		}
		m.mu.Unlock()
	}()
	m.appendLog(task.ID, "开始使用 Wav2Lip 为人物视频对口型。优先使用 GPU，失败会自动回退 CPU。")
	m.appendLog(task.ID, "任务目的：根据输入音频驱动人物视频口型。")
	if err := m.extract(ctx, task); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || m.cancelled(task.ID) {
			m.finish(task.ID, wav2lipTaskCancelled, 0, "任务已取消。")
		} else {
			m.finish(task.ID, wav2lipTaskFailed, 0, "任务失败，具体原因："+err.Error())
		}
		return
	}
	m.finish(task.ID, wav2lipTaskCompleted, 100, "人物视频对口型完成。")
}
func isWav2LipVideo(path string) bool {
	mime, ok := mediaPreviewMIMEType(path)
	return ok && strings.HasPrefix(mime, "video/")
}

// File names are useful for choosing the browser picker, but they are not a
// trustworthy media type boundary. ffprobe confirms that a client cannot put
// an arbitrary renamed file into the Wav2Lip queue.
func validateWav2LipVideo(ctx context.Context, path string) error {
	if !isWav2LipVideo(path) {
		return errors.New("仅支持视频文件")
	}
	ffprobePath, err := videoTrimLookPathFn("ffprobe")
	if err != nil {
		return errors.New("未检测到 FFprobe")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	output, err := videoTrimCommandContextFn(probeCtx, ffprobePath, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_type", "-of", "csv=p=0", path).CombinedOutput()
	if err != nil || probeCtx.Err() != nil || strings.TrimSpace(string(output)) != "video" {
		return errors.New("仅支持包含视频流的视频文件")
	}
	return nil
}

func isWav2LipAudio(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".wav", ".mp3", ".m4a", ".aac", ".flac", ".ogg":
		return true
	}
	return false
}

func validateWav2LipAudio(ctx context.Context, path string) error {
	if !isWav2LipAudio(path) {
		return errors.New("仅支持 WAV、MP3、M4A、AAC、FLAC 或 OGG 音频文件")
	}
	ffprobePath, err := videoTrimLookPathFn("ffprobe")
	if err != nil {
		return errors.New("未检测到 FFprobe")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	output, err := videoTrimCommandContextFn(probeCtx, ffprobePath, "-v", "error", "-select_streams", "a:0", "-show_entries", "stream=codec_type", "-of", "csv=p=0", path).CombinedOutput()
	if err != nil || probeCtx.Err() != nil || strings.TrimSpace(string(output)) != "audio" {
		return errors.New("仅支持包含音频流的音频文件")
	}
	return nil
}
func (m *wav2lipTaskManager) extract(ctx context.Context, task wav2lipTask) error {
	status, ffmpeg := checkFFmpegDependency()
	if status != http.StatusOK || !ffmpeg.Available {
		return errors.New("FFmpeg 或 FFprobe 未安装")
	}
	video, _, status, msg := resolveMediaPreviewFile(m.cfg, task.AgentID, task.VideoPath)
	if status != 0 {
		return errors.New(msg)
	}
	if err := validateWav2LipVideo(ctx, video); err != nil {
		return err
	}
	audio, _, status, msg := resolveMediaPreviewFile(m.cfg, task.AgentID, task.AudioPath)
	if status != 0 {
		return errors.New(msg)
	}
	if err := validateWav2LipAudio(ctx, audio); err != nil {
		return err
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, task.AgentID)
	if err != nil {
		return errors.New("Agent 工作目录不存在")
	}
	runtimeValue, ok := wav2lipRuntimeFor(time.Hour, workspace)
	if !ok {
		return errors.New("未检测到可用的 Wav2Lip")
	}
	m.appendLog(task.ID, taskKnownPythonEnvironment(runtimeValue.Python, runtimeValue.Script))
	target, err := resolveCaseInsensitiveUnderRoot(workspace, filepath.FromSlash(task.OutputPath))
	if err != nil || !ensureWritablePathWithinRoot(workspace, target) {
		return errors.New("无法创建视频输出路径")
	}
	if target, err = m.ensureAvailableOutputPath(&task, workspace, target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return errors.New("无法创建视频输出目录")
	}
	m.appendLog(task.ID, "原始视频绝对路径："+video)
	m.appendLog(task.ID, "原始音频绝对路径："+audio)
	m.appendLog(task.ID, "输出视频绝对路径："+target)
	m.appendLog(task.ID, "Wav2Lip 推理脚本："+runtimeValue.Script)
	m.appendLog(task.ID, "Wav2Lip 模型权重："+runtimeValue.Checkpoint)
	part, err := os.CreateTemp(filepath.Dir(target), ".wav2lip-*.mp4")
	if err != nil {
		return err
	}
	temporary := part.Name()
	if err := part.Close(); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	defer os.Remove(temporary)
	devices := wav2lipPreferredDevices()
	var lastErr error
	for index, device := range devices {
		if index > 0 {
			m.appendLog(task.ID, "GPU/MPS 推理失败，已清理临时输出并回退 CPU："+lastErr.Error())
			_ = os.Remove(temporary)
		}
		m.appendLog(task.ID, "正在使用 "+device+" 执行 Wav2Lip 推理。")
		m.appendLog(task.ID, "实际 Wav2Lip 模型参数：--checkpoint_path="+runtimeValue.Checkpoint+" --face="+video+" --audio="+audio+" --outfile="+target+" --device="+device)
		// Wav2Lip's unmodified inference.py builds its final FFmpeg command as
		// a string without quoting --audio or --outfile.  The Agent workspace on
		// macOS commonly contains "Application Support", so pass only paths from
		// our own no-whitespace staging directory to that upstream command.
		// The downloaded Wav2Lip package itself remains read-only.
		staging, stageErr := wav2lipPrepareStaging(video, audio)
		if stageErr != nil {
			return stageErr
		}
		m.appendLog(task.ID, "Wav2Lip 执行参数路径映射：--face="+staging.Video+" --audio="+staging.Audio+" --outfile="+staging.Output)
		err = m.runWav2Lip(ctx, task.ID, runtimeValue, device, staging.Video, staging.Audio, staging.Output)
		if err == nil {
			err = copyWav2LipOutput(staging.Output, temporary)
		}
		_ = os.RemoveAll(staging.Directory)
		if err == nil {
			lastErr = nil
			break
		}
		if errors.Is(err, context.Canceled) {
			return err
		}
		lastErr = err
	}
	if lastErr != nil {
		return lastErr
	}
	info, err := os.Stat(temporary)
	if err != nil || info.Size() == 0 {
		return errors.New("Wav2Lip 未生成有效 MP4 输出")
	}
	for attempt := 0; attempt < 4; attempt++ {
		if err := os.Link(temporary, target); err == nil {
			break
		}
		if _, statErr := os.Lstat(target); statErr == nil {
			target, err = m.ensureAvailableOutputPath(&task, workspace, target)
			if err != nil {
				return err
			}
			continue
		}
		return errors.New("无法保存视频输出文件")
	}
	if info, err = os.Stat(target); err != nil || info.Size() == 0 {
		return errors.New("无法保存有效 MP4 输出")
	}
	m.appendLog(task.ID, "最终输出视频绝对路径："+target)
	m.progress(task.ID, 100)
	return nil
}
func wav2lipPreferredDevices() []string {
	if runtime.GOOS == "darwin" {
		return []string{"mps", "cpu"}
	}
	return []string{"cuda", "cpu"}
}

func wav2lipInvocationArgs(r wav2lipRuntime, video, audio, output string) []string {
	return []string{"-c", wav2lipRuntimeBootstrap, r.Script, "--checkpoint_path", r.Checkpoint, "--face", video, "--audio", audio, "--outfile", output}
}

func wav2lipProbeArgs(script string) []string {
	return []string{"-c", wav2lipRuntimeBootstrap, script, "--help"}
}

// wav2lipPrepareStaging maps inputs and output to a self-owned temporary
// directory before invoking the upstream script. Its internal FFmpeg command
// is a shell string, so these paths must not contain whitespace.
func wav2lipPrepareStaging(video, audio string) (wav2lipStagingFiles, error) {
	directory, err := wav2lipStagingDirectory()
	if err != nil {
		return wav2lipStagingFiles{}, err
	}
	cleanup := func(err error) (wav2lipStagingFiles, error) {
		_ = os.RemoveAll(directory)
		return wav2lipStagingFiles{}, err
	}
	stage := wav2lipStagingFiles{
		Directory: directory,
		Video:     filepath.Join(directory, "input"+strings.ToLower(filepath.Ext(video))),
		Audio:     filepath.Join(directory, "audio"+strings.ToLower(filepath.Ext(audio))),
		Output:    filepath.Join(directory, "result.mp4"),
	}
	if err := os.Symlink(video, stage.Video); err != nil {
		return cleanup(fmt.Errorf("无法准备 Wav2Lip 视频输入: %w", err))
	}
	if err := os.Symlink(audio, stage.Audio); err != nil {
		return cleanup(fmt.Errorf("无法准备 Wav2Lip 音频输入: %w", err))
	}
	return stage, nil
}

func wav2lipStagingDirectory() (string, error) {
	// Prefer /tmp on POSIX. It is independent from the macOS application
	// container path, which includes "Application Support". On Windows, use
	// the configured temporary directory and accept it only when it is safe for
	// the upstream unquoted command.
	roots := []string{""}
	if runtime.GOOS != "windows" {
		roots = append([]string{"/tmp"}, roots...)
	}
	var lastErr error
	for _, root := range roots {
		directory, err := os.MkdirTemp(root, "deepright-wav2lip-")
		if err != nil {
			lastErr = err
			continue
		}
		if !strings.ContainsAny(directory, " \t\r\n") {
			return directory, nil
		}
		_ = os.RemoveAll(directory)
		lastErr = errors.New("临时目录包含空格")
	}
	if lastErr != nil {
		return "", fmt.Errorf("无法创建不含空格的 Wav2Lip 临时工作目录: %w", lastErr)
	}
	return "", errors.New("无法创建不含空格的 Wav2Lip 临时工作目录")
}

// copyWav2LipOutput keeps the final write within the Agent workspace. The
// staging directory may be on another filesystem, so a hard link is not safe
// here; the caller later atomically links this workspace-local temporary file.
func copyWav2LipOutput(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return errors.New("Wav2Lip 未生成有效视频输出")
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("无法暂存 Wav2Lip 视频输出: %w", err)
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return fmt.Errorf("无法暂存 Wav2Lip 视频输出: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("无法暂存 Wav2Lip 视频输出: %w", closeErr)
	}
	return nil
}

func (m *wav2lipTaskManager) runWav2Lip(ctx context.Context, id int64, r wav2lipRuntime, device, video, audio, output string) error {
	cmd := wav2lipCommandContext(ctx, r.Python, wav2lipInvocationArgs(r, video, audio, output)...)
	// Some dependencies launch short-lived shell helpers before the bootstrap
	// script changes directory. Always give the whole child-process tree a
	// durable upstream working directory; never inherit a task staging directory
	// that can be removed after a failed device attempt.
	cmd.Dir = filepath.Dir(r.Script)
	cmd.Env = append(os.Environ(), "DEEPRIGHT_WAV2LIP_DEVICE="+device, "DEEPRIGHT_MODEL_TASK_DOWNLOAD="+strconv.Itoa(modelTaskDownloadWorkers(m.cfg)), "DEEPRIGHT_MODEL_DOWNLOAD_RETRY="+strconv.Itoa(modelTaskDownloadRetries(m.cfg)), "DEEPRIGHT_MODEL_DOWNLOAD_PART_TIMEOUT="+strconv.Itoa(int(resumableModelDownloadPartTimeout/time.Second)))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	var captured strings.Builder
	var capturedMu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); m.capture(id, stdout, &captured, &capturedMu) }()
	go func() { defer wg.Done(); m.capture(id, stderr, &captured, &capturedMu) }()
	wg.Wait()
	if err = cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return context.Canceled
		}
		capturedMu.Lock()
		outputText := captured.String()
		capturedMu.Unlock()
		if reason := wav2lipReadableFailureReason(outputText); reason != "" {
			return errors.New(reason)
		}
		return taskExecutionError("Wav2Lip（"+device+"）", err, outputText)
	}
	info, statErr := os.Stat(output)
	if statErr != nil || info.Size() == 0 {
		return errors.New("Wav2Lip 未生成有效视频输出")
	}
	return nil
}
func wav2lipReadableFailureReason(output string) string {
	value := strings.ToLower(output)
	if strings.Contains(value, "timed out") || strings.Contains(value, "timeout") {
		return "Wav2Lip 人脸检测模型下载在 90 秒内没有进度，已停止。请检查网络或在安装阶段提供已校验的本地 S3FD 模型后重试。"
	}
	if strings.Contains(value, "unexpected eof") {
		return "Wav2Lip 人脸检测模型缓存不完整或已损坏。请删除 ~/.cache/torch/hub/checkpoints/s3fd-619a316812.pth 后重新执行任务，系统会重新下载完整模型。"
	}
	return ""
}

func (m *wav2lipTaskManager) capture(id int64, r io.Reader, captured *strings.Builder, capturedMu *sync.Mutex) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 128*1024)
	scanner.Split(splitRembgOutput)
	for scanner.Scan() {
		line := scanner.Text()
		m.appendLog(id, line)
		capturedMu.Lock()
		if captured.Len() < 128*1024 {
			captured.WriteString(line + "\n")
		}
		capturedMu.Unlock()
	}
	if err := scanner.Err(); err != nil {
		m.appendLog(id, "读取 Wav2Lip 输出失败："+err.Error())
	}
}
func (m *wav2lipTaskManager) progress(id int64, p int) {
	_, _ = m.db.Exec(`UPDATE wav2lip_task SET progress=CASE WHEN progress>? THEN progress ELSE ? END,updated_at=? WHERE id=? AND status=?`, p, p, wav2lipTimestamp(), id, wav2lipTaskRunning)
}
func (m *wav2lipTaskManager) appendLog(id int64, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	if len(message) > 4096 {
		message = message[:4096] + "…"
	}
	now := wav2lipTimestamp()
	_, _ = m.db.Exec(`UPDATE wav2lip_task SET logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ?,updated_at=? WHERE id=?`, wav2lipTaskLogLimit-8192, wav2lipTaskLogLimit/2, "["+now+"] "+message+"\n", now, id)
}
func (m *wav2lipTaskManager) finish(id int64, status string, p int, message string) {
	m.appendLog(id, message)
	failureReason := ""
	if status == wav2lipTaskFailed {
		failureReason = strings.TrimSpace(message)
	}
	_, _ = m.db.Exec(`UPDATE wav2lip_task SET status=?,progress=?,failure_reason=?,updated_at=? WHERE id=?`, status, p, failureReason, wav2lipTimestamp(), id)
	notifyIntegrationSleepConditionChanged(m.cfg)
}
func (m *wav2lipTaskManager) cancelled(id int64) bool {
	var value int
	return m.db.QueryRow(`SELECT cancel_requested FROM wav2lip_task WHERE id=?`, id).Scan(&value) != nil || value != 0
}
func (m *wav2lipTaskManager) cancelRunning() {
	m.mu.Lock()
	cancel := m.cancel
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (m *wav2lipTaskManager) allocate(agentID, workspace, source string, reserved map[string]bool, except int64) (string, string, error) {
	stem := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	stamp := wav2lipNow().Format("20060102_150405")
	for attempt := 0; attempt < 1000; attempt++ {
		name := stem + "_lip_sync.mp4"
		if attempt > 0 {
			name = stem + "_lip_sync_" + stamp
			if attempt > 1 {
				name += "_" + strconv.Itoa(attempt)
			}
			name += ".mp4"
		}
		relative := filepath.Join("videos", name)
		key := filepath.ToSlash(relative)
		if reserved[key] {
			continue
		}
		absolute, err := resolveCaseInsensitiveUnderRoot(workspace, relative)
		if err != nil || !ensureWritablePathWithinRoot(workspace, absolute) {
			return "", "", errors.New("无法创建视频输出路径")
		}
		var count int
		if err := m.db.QueryRow(`SELECT COUNT(*) FROM wav2lip_task WHERE agent_id=? AND output_path=? AND id<>?`, agentID, key, except).Scan(&count); err != nil {
			return "", "", err
		}
		if count != 0 {
			continue
		}
		if _, err := os.Lstat(absolute); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "", err
		}
		reserved[key] = true
		return relative, absolute, nil
	}
	return "", "", errors.New("无法分配视频输出文件名")
}

// ensureAvailableOutputPath preserves the no-overwrite rule if a path appears
// after the task was queued.
func (m *wav2lipTaskManager) ensureAvailableOutputPath(task *wav2lipTask, workspace, target string) (string, error) {
	_, targetErr := os.Lstat(target)
	if errors.Is(targetErr, os.ErrNotExist) {
		return target, nil
	}
	if targetErr != nil {
		return "", errors.New("无法检查视频输出路径")
	}
	relative, absolute, err := m.allocate(task.AgentID, workspace, task.VideoPath, map[string]bool{}, task.ID)
	if err != nil {
		return "", err
	}
	result, err := m.db.Exec(`UPDATE wav2lip_task SET output_path=?,saved_as=?,updated_at=? WHERE id=? AND agent_id=?`, filepath.ToSlash(relative), absolute, wav2lipTimestamp(), task.ID, task.AgentID)
	if err != nil {
		return "", err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return "", errors.New("无法更新视频输出路径")
	}
	task.OutputPath, task.SavedAs = filepath.ToSlash(relative), absolute
	m.appendLog(task.ID, "检测到同名输出，已改用带时间戳的文件名。")
	return absolute, nil
}
func (m *wav2lipTaskManager) create(request wav2lipTaskCreateRequest) ([]wav2lipTask, error) {
	if m == nil || m.cfg == nil || m.db == nil {
		return nil, errors.New("人物视频对口型任务服务未就绪")
	}
	request.AgentID = strings.TrimSpace(request.AgentID)
	if !isValidMediaPreviewAgentID(request.AgentID) || len(request.Tasks) == 0 || len(request.Tasks) > 64 {
		return nil, errors.New("agentId 和 1 至 64 组视频、音频文件不能为空")
	}
	m.createMu.Lock()
	defer m.createMu.Unlock()
	workspace, err := getWorkspaceByAgentID(m.cfg, request.AgentID)
	if err != nil {
		return nil, errors.New("Agent 不存在")
	}
	type source struct{ video, audio, output, saved string }
	sources := []source{}
	seen, reserved := map[string]bool{}, map[string]bool{}
	for _, item := range request.Tasks {
		videoPath, audioPath := normalizeQuotedPathArg(item.VideoPath), normalizeQuotedPathArg(item.AudioPath)
		if videoPath == "" || audioPath == "" {
			return nil, errors.New("每个任务都必须选择一个视频和一个音频文件")
		}
		key := videoPath + "\x00" + audioPath
		if seen[key] {
			continue
		}
		seen[key] = true
		video, _, status, msg := resolveMediaPreviewFile(m.cfg, request.AgentID, videoPath)
		if status != 0 {
			return nil, errors.New(msg)
		}
		if err := validateWav2LipVideo(context.Background(), video); err != nil {
			return nil, err
		}
		audio, _, status, msg := resolveMediaPreviewFile(m.cfg, request.AgentID, audioPath)
		if status != 0 {
			return nil, errors.New(msg)
		}
		if err := validateWav2LipAudio(context.Background(), audio); err != nil {
			return nil, err
		}
		videoRelative, err := filepath.Rel(workspace, video)
		if err != nil || videoRelative == "." || strings.HasPrefix(videoRelative, ".."+string(filepath.Separator)) {
			return nil, errors.New("视频文件必须位于当前 Agent 工作目录")
		}
		audioRelative, err := filepath.Rel(workspace, audio)
		if err != nil || audioRelative == "." || strings.HasPrefix(audioRelative, ".."+string(filepath.Separator)) {
			return nil, errors.New("音频文件必须位于当前 Agent 工作目录")
		}
		output, saved, err := m.allocate(request.AgentID, workspace, videoRelative, reserved, 0)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source{filepath.ToSlash(videoRelative), filepath.ToSlash(audioRelative), filepath.ToSlash(output), saved})
	}
	if len(sources) == 0 {
		return nil, errors.New("未找到可添加的视频文件")
	}
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := wav2lipTimestamp()
	created := make([]wav2lipTask, 0, len(sources))
	for _, source := range sources {
		result, err := tx.Exec(`INSERT INTO wav2lip_task(agent_id,video_path,audio_path,output_path,saved_as,status,progress,created_at,updated_at,logs,cancel_requested) VALUES(?,?,?,?,?,?,?,?,?,?,0)`, request.AgentID, source.video, source.audio, source.output, source.saved, wav2lipTaskQueued, 0, now, now, "")
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		created = append(created, wav2lipTask{ID: id, AgentID: request.AgentID, VideoPath: source.video, AudioPath: source.audio, OutputPath: source.output, SavedAs: source.saved, Status: wav2lipTaskQueued, CreatedAt: now, UpdatedAt: now})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	m.signal()
	return created, nil
}
func validWav2LipStatus(value string) bool {
	switch value {
	case "", wav2lipTaskQueued, wav2lipTaskRunning, wav2lipTaskCompleted, wav2lipTaskCancelled, wav2lipTaskFailed:
		return true
	}
	return false
}
func (m *wav2lipTaskManager) list(agentID, status string, page int, logs bool) (wav2lipTaskPage, error) {
	result := wav2lipTaskPage{Tasks: []wav2lipTask{}, PageSize: 5}
	agentID = strings.TrimSpace(agentID)
	status = strings.TrimSpace(status)
	if !isValidMediaPreviewAgentID(agentID) || !validWav2LipStatus(status) {
		return result, errors.New("任务参数无效")
	}
	if page < 1 {
		page = 1
	}
	where, args := "agent_id=?", []interface{}{agentID}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM wav2lip_task WHERE `+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	last := (result.Total + result.PageSize - 1) / result.PageSize
	if last < 1 {
		last = 1
	}
	if page > last {
		page = last
	}
	result.Page, result.Status = page, status
	column := "''"
	if logs {
		column = "logs"
	}
	args = append(args, result.PageSize, (page-1)*result.PageSize)
	rows, err := m.db.Query(`SELECT id,agent_id,video_path,audio_path,output_path,saved_as,status,progress,started_at,created_at,updated_at,`+column+`,failure_reason,cancel_requested FROM wav2lip_task WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		task, err := scanWav2LipTask(rows)
		if err != nil {
			return result, err
		}
		result.Tasks = append(result.Tasks, task)
	}
	return result, rows.Err()
}
func (m *wav2lipTaskManager) log(agentID string, id int64) (wav2lipTask, error) {
	if id <= 0 || !isValidMediaPreviewAgentID(strings.TrimSpace(agentID)) {
		return wav2lipTask{}, errors.New("任务参数无效")
	}
	return scanWav2LipTask(m.db.QueryRow(`SELECT id,agent_id,video_path,audio_path,output_path,saved_as,status,progress,started_at,created_at,updated_at,logs,failure_reason,cancel_requested FROM wav2lip_task WHERE id=? AND agent_id=?`, id, strings.TrimSpace(agentID)))
}
func (m *wav2lipTaskManager) cancelTask(agentID string, id int64) error {
	now := wav2lipTimestamp()
	result, err := m.db.Exec(`UPDATE wav2lip_task SET cancel_requested=1,status=CASE WHEN status=? THEN ? ELSE status END,updated_at=?,logs=CASE WHEN length(logs)>? THEN substr(logs,-?) ELSE logs END || ? WHERE id=? AND agent_id=? AND status IN (?,?)`, wav2lipTaskQueued, wav2lipTaskCancelled, now, wav2lipTaskLogLimit-4096, wav2lipTaskLogLimit/2, "["+now+"] 已请求取消任务。\n", id, strings.TrimSpace(agentID), wav2lipTaskQueued, wav2lipTaskRunning)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("仅可取消排队中或执行中的任务")
	}
	m.mu.Lock()
	running, cancel := m.runningID == id, m.cancel
	m.mu.Unlock()
	if running && cancel != nil {
		cancel()
	}
	notifyIntegrationSleepConditionChanged(m.cfg)
	return nil
}
func (m *wav2lipTaskManager) restart(agentID string, id int64) error {
	agentID = strings.TrimSpace(agentID)
	var video, audio string
	if err := m.db.QueryRow(`SELECT video_path,audio_path FROM wav2lip_task WHERE id=? AND agent_id=? AND status IN (?, ?)`, id, agentID, wav2lipTaskFailed, wav2lipTaskCancelled).Scan(&video, &audio); err != nil {
		return errors.New("仅可重新开始失败或已取消任务")
	}
	workspace, err := getWorkspaceByAgentID(m.cfg, agentID)
	if err != nil {
		return errors.New("Agent 不存在")
	}
	output, saved, err := m.allocate(agentID, workspace, video, map[string]bool{}, id)
	if err != nil {
		return err
	}
	now := wav2lipTimestamp()
	result, err := m.db.Exec(`UPDATE wav2lip_task SET output_path=?,saved_as=?,status=?,progress=0,started_at='',cancel_requested=0,failure_reason='',updated_at=?,logs='' WHERE id=? AND agent_id=? AND status IN (?, ?)`, filepath.ToSlash(output), saved, wav2lipTaskQueued, now, id, agentID, wav2lipTaskFailed, wav2lipTaskCancelled)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("重新开始任务失败")
	}
	m.signal()
	return nil
}
func (m *wav2lipTaskManager) deleteFailedOrCancelled(agentID string, id int64) error {
	result, err := m.db.Exec(`DELETE FROM wav2lip_task WHERE id=? AND agent_id=? AND status IN (?, ?)`, id, strings.TrimSpace(agentID), wav2lipTaskFailed, wav2lipTaskCancelled)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return errors.New("仅可删除失败或已取消任务")
	}
	return nil
}

type wav2lipScanner interface{ Scan(...interface{}) error }

func scanWav2LipTask(scanner wav2lipScanner) (wav2lipTask, error) {
	var task wav2lipTask
	var cancelled int
	err := scanner.Scan(&task.ID, &task.AgentID, &task.VideoPath, &task.AudioPath, &task.OutputPath, &task.SavedAs, &task.Status, &task.Progress, &task.StartedAt, &task.CreatedAt, &task.UpdatedAt, &task.Logs, &task.FailureReason, &cancelled)
	task.CancelAsked = cancelled != 0
	return task, err
}
func readWav2LipAction(w http.ResponseWriter, r *http.Request) (wav2lipTaskActionRequest, error) {
	var request wav2lipTaskActionRequest
	if r.Method != http.MethodPost {
		return request, errors.New("仅支持 POST 请求")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return request, errors.New("请求内容无效")
	}
	return request, nil
}
func handleWav2LipTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if wav2lipTasks == nil {
			writeWav2LipJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "人物视频对口型任务服务未就绪"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			page := 1
			if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
				value, err := strconv.Atoi(raw)
				if err != nil || value < 1 {
					writeWav2LipJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "页码必须是正整数"})
					return
				}
				page = value
			}
			value, err := wav2lipTasks.list(r.URL.Query().Get("agentId"), r.URL.Query().Get("status"), page, false)
			if err != nil {
				writeWav2LipJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeWav2LipJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": value.Tasks, "total": value.Total, "page": value.Page, "pageSize": value.PageSize, "taskStatus": value.Status})
		case http.MethodPost:
			var request wav2lipTaskCreateRequest
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
				writeWav2LipJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "请求内容无效"})
				return
			}
			value, err := wav2lipTasks.create(request)
			if err != nil {
				writeWav2LipJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
				return
			}
			writeWav2LipJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "tasks": value})
		default:
			w.Header().Set("Allow", "GET, POST")
			writeWav2LipJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 或 POST 请求"})
		}
	}
}
func wav2lipActionHandler(action func(*wav2lipTaskManager, wav2lipTaskActionRequest) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeWav2LipJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 POST 请求"})
			return
		}
		if wav2lipTasks == nil {
			writeWav2LipJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "人物视频对口型任务服务未就绪"})
			return
		}
		request, err := readWav2LipAction(w, r)
		if err == nil {
			err = action(wav2lipTasks, request)
		}
		if err != nil {
			writeWav2LipJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": err.Error()})
			return
		}
		writeWav2LipJSON(w, http.StatusOK, map[string]interface{}{"status": 0})
	}
}
func handleWav2LipTaskCancel() http.HandlerFunc {
	return wav2lipActionHandler(func(m *wav2lipTaskManager, r wav2lipTaskActionRequest) error { return m.cancelTask(r.AgentID, r.ID) })
}
func handleWav2LipTaskRestart() http.HandlerFunc {
	return wav2lipActionHandler(func(m *wav2lipTaskManager, r wav2lipTaskActionRequest) error { return m.restart(r.AgentID, r.ID) })
}
func handleWav2LipTaskDelete() http.HandlerFunc {
	return wav2lipActionHandler(func(m *wav2lipTaskManager, r wav2lipTaskActionRequest) error {
		return m.deleteFailedOrCancelled(r.AgentID, r.ID)
	})
}
func handleWav2LipTaskLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeWav2LipJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"status": 1, "content": "仅支持 GET 请求"})
			return
		}
		if wav2lipTasks == nil {
			writeWav2LipJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"status": 1, "content": "人物视频对口型任务服务未就绪"})
			return
		}
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err == nil {
			task, lookupErr := wav2lipTasks.log(r.URL.Query().Get("agentId"), id)
			if lookupErr == nil {
				writeWav2LipJSON(w, http.StatusOK, map[string]interface{}{"status": 0, "task": task})
				return
			}
		}
		writeWav2LipJSON(w, http.StatusBadRequest, map[string]interface{}{"status": 1, "content": "任务参数无效"})
	}
}
