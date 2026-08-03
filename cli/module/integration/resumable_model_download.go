package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// resumableModelDownloadConfig describes an immutable model artifact. The
// downloader retains its part file and small completion manifest until the
// caller has checked the artifact digest and atomically publishes it.
type resumableModelDownloadConfig struct {
	Client       *http.Client
	URL          string
	BackupURL    string
	PartPath     string
	ExpectedSize int64
	IdleTimeout  time.Duration
	PartTimeout  time.Duration
	Workers      int
	Retries      int
	ChunkSize    int64
	Progress     func(copied, total int64)
	PartProgress func(worker, part, parts int, copied, total int64)
	Retry        func(worker, part, parts, retry, retries int, err error)
}

type modelDownloadResult struct {
	URL           string
	Bytes         int64
	UsedRanges    bool
	UsedBackup    bool
	UsedPlainHTTP bool
}

type resumableModelDownloadState struct {
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	Chunks []bool `json:"chunks"`
}

const (
	resumableModelDownloadWorkers     = 4
	resumableModelDownloadChunkSize   = 4 * 1024 * 1024
	resumableModelDownloadPartTimeout = 15 * time.Minute
)

// downloadModelWithResume uses HTTP byte ranges to fetch independent parts in
// parallel. Completed parts are recorded durably, so cancellation and a later
// retry continue from the last verified part rather than starting over. A
// source that does not offer byte ranges still downloads correctly, but cannot
// provide concurrent resume semantics and is intentionally reported to logs.
func downloadModelWithResume(ctx context.Context, cfg resumableModelDownloadConfig) (int64, bool, error) {
	prepared, err := prepareModelDownloadConfig(cfg)
	if err != nil {
		return 0, false, err
	}
	return downloadModelFromSource(ctx, prepared, nil)
}

func prepareModelDownloadConfig(cfg resumableModelDownloadConfig) (resumableModelDownloadConfig, error) {
	if cfg.Client == nil || strings.TrimSpace(cfg.URL) == "" || strings.TrimSpace(cfg.PartPath) == "" {
		return cfg, errors.New("模型下载参数无效")
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = 90 * time.Second
	}
	if cfg.PartTimeout <= 0 {
		cfg.PartTimeout = resumableModelDownloadPartTimeout
	}
	if cfg.Workers <= 0 {
		cfg.Workers = resumableModelDownloadWorkers
	}
	if cfg.Retries < 0 {
		return cfg, errors.New("模型下载重试次数不能小于 0")
	}
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = resumableModelDownloadChunkSize
	}
	if err := os.MkdirAll(filepath.Dir(cfg.PartPath), 0o755); err != nil {
		return cfg, fmt.Errorf("无法创建模型临时目录: %w", err)
	}
	return cfg, nil
}

func downloadModelFromSource(ctx context.Context, cfg resumableModelDownloadConfig, before func(bool)) (int64, bool, error) {
	total, ranges, err := modelDownloadProbe(ctx, cfg)
	if err != nil {
		return 0, false, err
	}
	if cfg.ExpectedSize > 0 && total > 0 && total != cfg.ExpectedSize {
		return 0, ranges, fmt.Errorf("模型大小与预期不一致（源站 %d，预期 %d）", total, cfg.ExpectedSize)
	}
	if cfg.ExpectedSize > 0 {
		total = cfg.ExpectedSize
	}
	if before != nil {
		before(ranges && total > 0)
	}
	if ranges && total > 0 {
		return downloadModelRanges(ctx, cfg, total)
	}
	return downloadModelSequentially(ctx, cfg, total)
}

// downloadModelWithFallback implements the runtime-model source order:
// resumable primary download when Range is available, ordinary primary
// download otherwise, then the same two choices for a backup source. The
// callback is deliberately supplied by each task manager so every transition
// appears in its persisted task log.
func downloadModelWithFallback(ctx context.Context, cfg resumableModelDownloadConfig, report func(string)) (modelDownloadResult, error) {
	prepared, err := prepareModelDownloadConfig(cfg)
	if err != nil {
		return modelDownloadResult{}, err
	}
	cfg = prepared
	primary := strings.TrimSpace(cfg.URL)
	backup := strings.TrimSpace(cfg.BackupURL)
	if primary == "" {
		return modelDownloadResult{}, errors.New("模型主下载源为空")
	}
	sources := []struct {
		url    string
		backup bool
	}{{url: primary}}
	if backup != "" && backup != primary {
		sources = append(sources, struct {
			url    string
			backup bool
		}{url: backup, backup: true})
	}
	var lastErr error
	for index, source := range sources {
		attempt := cfg
		attempt.URL = source.url
		attempt.BackupURL = ""
		if source.backup {
			reportModelDownload(report, "主下载源普通下载失败，正在检查备用下载源是否支持断点续传："+source.url)
		} else {
			reportModelDownload(report, "正在检查下载源是否支持 HTTP Range 断点续传："+source.url)
		}
		bytes, ranges, err := downloadModelFromSource(ctx, attempt, func(available bool) {
			if available {
				if source.backup {
					reportModelDownload(report, fmt.Sprintf("备用下载源支持 HTTP Range，使用 %d 路分段断点续传。", attempt.Workers))
				} else {
					reportModelDownload(report, fmt.Sprintf("下载源支持 HTTP Range，使用 %d 路分段断点续传。", attempt.Workers))
				}
				return
			}
			if source.backup {
				reportModelDownload(report, "备用下载源不支持 HTTP Range，回退到普通下载。")
			} else {
				reportModelDownload(report, "下载源不支持 HTTP Range，回退到普通下载。")
			}
		})
		if err == nil {
			return modelDownloadResult{URL: source.url, Bytes: bytes, UsedRanges: ranges, UsedBackup: source.backup, UsedPlainHTTP: !ranges}, nil
		}
		if ctx.Err() != nil {
			return modelDownloadResult{}, err
		}
		if ranges {
			reportModelDownload(report, "多线程断点续传失败："+err.Error()+"；回退到当前下载源的普通下载。")
			_ = os.Remove(attempt.PartPath)
			removeModelDownloadState(attempt.PartPath)
			bytes, _, err = downloadModelSequentially(ctx, attempt, attempt.ExpectedSize)
			if err == nil {
				return modelDownloadResult{URL: source.url, Bytes: bytes, UsedBackup: source.backup, UsedPlainHTTP: true}, nil
			}
			if ctx.Err() != nil {
				return modelDownloadResult{}, err
			}
			ranges = false
		}
		lastErr = err
		if source.backup {
			reportModelDownload(report, "备用下载源普通下载失败："+err.Error())
			break
		}
		reportModelDownload(report, "主下载源普通下载失败："+err.Error()+"；将尝试备用下载源。")
		// A partial ordinary download is not a completed byte-range map. The
		// backup source must start its own validated range map.
		_ = os.Remove(attempt.PartPath)
		removeModelDownloadState(attempt.PartPath)
		if index == len(sources)-1 {
			break
		}
	}
	if lastErr == nil {
		lastErr = errors.New("全部模型下载源均不可用")
	}
	return modelDownloadResult{}, lastErr
}

func reportModelDownload(report func(string), message string) {
	if report != nil {
		report(message)
	}
}

func modelDownloadRetryMessage(worker, part, parts, retry, retries int, err error) string {
	if worker > 0 {
		return fmt.Sprintf("下载线程 %d，分段 %d/%d 第 %d 次重试（最多 %d 次）：%v", worker, part, parts, retry, retries, err)
	}
	return fmt.Sprintf("普通下载第 %d 次重试（最多 %d 次）：%v", retry, retries, err)
}

func modelDownloadProbe(ctx context.Context, cfg resumableModelDownloadConfig) (int64, bool, error) {
	probeCtx, cancel := context.WithTimeout(ctx, cfg.PartTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return 0, false, fmt.Errorf("无法创建模型下载请求: %w", err)
	}
	request.Header.Set("Range", "bytes=0-0")
	response, err := cfg.Client.Do(request)
	if err != nil {
		if ctx.Err() == nil && errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
			return 0, false, fmt.Errorf("检查下载源是否支持 HTTP Range 超过总时限 %s", cfg.PartTimeout)
		}
		return 0, false, fmt.Errorf("请求模型下载源失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusPartialContent {
		total, ok := modelDownloadContentRangeSize(response.Header.Get("Content-Range"))
		if !ok || total <= 0 {
			return 0, false, errors.New("模型下载源返回了无效的 Content-Range")
		}
		return total, true, nil
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return 0, false, fmt.Errorf("模型下载源返回 HTTP %d", response.StatusCode)
	}
	return response.ContentLength, false, nil
}

func modelDownloadContentRangeSize(value string) (int64, bool) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 || parts[1] == "*" {
		return 0, false
	}
	size, err := strconv.ParseInt(parts[1], 10, 64)
	return size, err == nil
}

func downloadModelRanges(ctx context.Context, cfg resumableModelDownloadConfig, total int64) (int64, bool, error) {
	chunks := int((total + cfg.ChunkSize - 1) / cfg.ChunkSize)
	if chunks <= 1 {
		return downloadModelSequentially(ctx, cfg, total)
	}
	statePath := cfg.PartPath + ".json"
	state, err := loadModelDownloadState(statePath, cfg.URL, total, chunks)
	if err != nil {
		return 0, true, err
	}
	part, err := os.OpenFile(cfg.PartPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return 0, true, fmt.Errorf("无法打开模型临时文件: %w", err)
	}
	if err := part.Truncate(total); err != nil {
		_ = part.Close()
		return 0, true, fmt.Errorf("无法初始化模型临时文件: %w", err)
	}
	defer part.Close()

	var copied int64
	for index, complete := range state.Chunks {
		if complete {
			copied += modelDownloadChunkSize(total, cfg.ChunkSize, index)
		}
	}
	if cfg.Progress != nil {
		cfg.Progress(copied, total)
	}
	if copied == total {
		return copied, true, nil
	}
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	var stateMu sync.Mutex
	var progressMu sync.Mutex
	var firstErr error
	var workers sync.WaitGroup
	workerCount := cfg.Workers
	if workerCount > chunks {
		workerCount = chunks
	}
	for worker := 0; worker < workerCount; worker++ {
		workerID := worker + 1
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				stateMu.Lock()
				failed := firstErr != nil
				complete := state.Chunks[index]
				stateMu.Unlock()
				if failed || complete {
					continue
				}
				start := int64(index) * cfg.ChunkSize
				end := start + modelDownloadChunkSize(total, cfg.ChunkSize, index) - 1
				if err := downloadModelRange(workCtx, cfg, part, start, end, func(copied, total int64) {
					if cfg.PartProgress == nil {
						return
					}
					progressMu.Lock()
					cfg.PartProgress(workerID, index+1, chunks, copied, total)
					progressMu.Unlock()
				}, func(retry, retries int, err error) {
					if cfg.Retry != nil {
						cfg.Retry(workerID, index+1, chunks, retry, retries, err)
					}
				}); err != nil {
					stateMu.Lock()
					if firstErr == nil {
						firstErr = err
						cancel()
					}
					stateMu.Unlock()
					continue
				}
				stateMu.Lock()
				if firstErr == nil {
					state.Chunks[index] = true
					copied += end - start + 1
					if err := saveModelDownloadState(statePath, state); err != nil {
						firstErr = fmt.Errorf("无法保存模型断点续传状态: %w", err)
						cancel()
					}
					if cfg.Progress != nil {
						cfg.Progress(copied, total)
					}
				}
				stateMu.Unlock()
			}
		}()
	}
	for index, complete := range state.Chunks {
		if !complete && workCtx.Err() == nil {
			jobs <- index
		}
	}
	close(jobs)
	workers.Wait()
	if firstErr != nil {
		return copied, true, firstErr
	}
	if err := ctx.Err(); err != nil {
		return copied, true, err
	}
	if copied != total {
		return copied, true, errors.New("模型分段下载未完成")
	}
	return copied, true, nil
}

func modelDownloadChunkSize(total, size int64, index int) int64 {
	start := int64(index) * size
	if remaining := total - start; remaining < size {
		return remaining
	}
	return size
}

func loadModelDownloadState(path, url string, size int64, chunks int) (resumableModelDownloadState, error) {
	state := resumableModelDownloadState{URL: url, Size: size, Chunks: make([]bool, chunks)}
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := saveModelDownloadState(path, state); err != nil {
			return state, err
		}
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("无法读取模型断点续传状态: %w", err)
	}
	var stored resumableModelDownloadState
	if err := json.Unmarshal(content, &stored); err != nil || stored.URL != url || stored.Size != size || len(stored.Chunks) != chunks {
		_ = os.Remove(path)
		_ = os.Remove(strings.TrimSuffix(path, ".json"))
		if err := saveModelDownloadState(path, state); err != nil {
			return state, err
		}
		return state, nil
	}
	return stored, nil
}

func saveModelDownloadState(path string, state resumableModelDownloadState) error {
	content, err := json.Marshal(state)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".model-download-state-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func downloadModelRange(ctx context.Context, cfg resumableModelDownloadConfig, output *os.File, start, end int64, progress func(copied, total int64), reportRetry func(retry, retries int, err error)) error {
	var lastErr error
	for retry := 0; retry <= cfg.Retries; retry++ {
		if retry > 0 && reportRetry != nil {
			reportRetry(retry, cfg.Retries, lastErr)
		}
		err := downloadModelRangeAttempt(ctx, cfg, output, start, end, progress)
		if err == nil {
			return nil
		}
		lastErr = err
		if ctx.Err() != nil || retry == cfg.Retries {
			return err
		}
	}
	return lastErr
}

func downloadModelRangeAttempt(ctx context.Context, cfg resumableModelDownloadConfig, output *os.File, start, end int64, progress func(copied, total int64)) error {
	downloadCtx, cancel := context.WithTimeout(ctx, cfg.PartTimeout)
	defer cancel()
	idle := time.AfterFunc(cfg.IdleTimeout, cancel)
	defer idle.Stop()
	request, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	response, err := cfg.Client.Do(request)
	if err != nil {
		if ctx.Err() == nil && errors.Is(downloadCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("下载分段超过总时限 %s", cfg.PartTimeout)
		}
		if ctx.Err() == nil && downloadCtx.Err() != nil {
			return fmt.Errorf("下载超过 %s 没有进度", cfg.IdleTimeout)
		}
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("下载源不支持分段请求（HTTP %d）", response.StatusCode)
	}
	if rangeStart, rangeEnd, ok := modelDownloadContentRange(response.Header.Get("Content-Range")); !ok || rangeStart != start || rangeEnd != end {
		return errors.New("下载源返回的分段范围无效")
	}
	buffer := make([]byte, 256*1024)
	position := start
	total := end - start + 1
	nextProgress := int64(10)
	for {
		count, readErr := response.Body.Read(buffer)
		if count > 0 {
			if !idle.Stop() && downloadCtx.Err() != nil && ctx.Err() == nil {
				return fmt.Errorf("下载超过 %s 没有进度", cfg.IdleTimeout)
			}
			idle.Reset(cfg.IdleTimeout)
			if _, err := output.WriteAt(buffer[:count], position); err != nil {
				return fmt.Errorf("写入模型分段失败: %w", err)
			}
			position += int64(count)
			if progress != nil {
				copied := position - start
				for copied*100 >= total*nextProgress && nextProgress <= 100 {
					progress(copied, total)
					nextProgress += 10
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			if ctx.Err() == nil && errors.Is(downloadCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("下载分段超过总时限 %s", cfg.PartTimeout)
			}
			if ctx.Err() == nil && downloadCtx.Err() != nil {
				return fmt.Errorf("下载超过 %s 没有进度", cfg.IdleTimeout)
			}
			return readErr
		}
	}
	if position != end+1 {
		return fmt.Errorf("模型分段大小不正确（得到 %d，预期 %d）", position-start, total)
	}
	if progress != nil && nextProgress <= 100 {
		progress(total, total)
	}
	return nil
}

// modelDownloadContentRange returns the inclusive byte bounds. It is separate
// from modelDownloadContentRangeSize because a probe only needs the total.
func modelDownloadContentRange(value string) (int64, int64, bool) {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bytes") {
		return 0, 0, false
	}
	bounds := strings.Split(parts[1], "/")
	if len(bounds) != 2 {
		return 0, 0, false
	}
	rangeParts := strings.Split(bounds[0], "-")
	if len(rangeParts) != 2 {
		return 0, 0, false
	}
	start, startErr := strconv.ParseInt(rangeParts[0], 10, 64)
	end, endErr := strconv.ParseInt(rangeParts[1], 10, 64)
	return start, end, startErr == nil && endErr == nil && start >= 0 && end >= start
}

func downloadModelSequentially(ctx context.Context, cfg resumableModelDownloadConfig, total int64) (int64, bool, error) {
	var copied int64
	var lastErr error
	for retry := 0; retry <= cfg.Retries; retry++ {
		if retry > 0 && cfg.Retry != nil {
			cfg.Retry(0, 0, 0, retry, cfg.Retries, lastErr)
		}
		copied, _, lastErr = downloadModelSequentialAttempt(ctx, cfg, total)
		if lastErr == nil {
			return copied, false, nil
		}
		if ctx.Err() != nil || retry == cfg.Retries {
			return copied, false, lastErr
		}
	}
	return copied, false, lastErr
}

func downloadModelSequentialAttempt(ctx context.Context, cfg resumableModelDownloadConfig, total int64) (int64, bool, error) {
	part, err := os.OpenFile(cfg.PartPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return 0, false, fmt.Errorf("无法打开模型临时文件: %w", err)
	}
	defer part.Close()
	info, err := part.Stat()
	if err != nil {
		return 0, false, err
	}
	offset := info.Size()
	if total > 0 && offset > total {
		if err := part.Truncate(0); err != nil {
			return 0, false, err
		}
		offset = 0
	}
	downloadCtx, cancel := context.WithTimeout(ctx, cfg.PartTimeout)
	defer cancel()
	idle := time.AfterFunc(cfg.IdleTimeout, cancel)
	defer idle.Stop()
	request, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return offset, false, err
	}
	if offset > 0 {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	response, err := cfg.Client.Do(request)
	if err != nil {
		if ctx.Err() == nil && errors.Is(downloadCtx.Err(), context.DeadlineExceeded) {
			return offset, false, fmt.Errorf("下载超过总时限 %s", cfg.PartTimeout)
		}
		if ctx.Err() == nil && downloadCtx.Err() != nil {
			return offset, false, fmt.Errorf("下载超过 %s 没有进度", cfg.IdleTimeout)
		}
		return offset, false, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return offset, false, fmt.Errorf("模型下载源返回 HTTP %d", response.StatusCode)
	}
	if offset > 0 && response.StatusCode == http.StatusOK {
		if err := part.Truncate(0); err != nil {
			return 0, false, err
		}
		offset = 0
	}
	if _, err := part.Seek(offset, io.SeekStart); err != nil {
		return offset, false, err
	}
	if total <= 0 && response.ContentLength >= 0 {
		total = offset + response.ContentLength
	}
	if cfg.Progress != nil {
		cfg.Progress(offset, total)
	}
	buffer := make([]byte, 256*1024)
	for {
		count, readErr := response.Body.Read(buffer)
		if count > 0 {
			if !idle.Stop() && downloadCtx.Err() != nil && ctx.Err() == nil {
				return offset, false, fmt.Errorf("下载超过 %s 没有进度", cfg.IdleTimeout)
			}
			idle.Reset(cfg.IdleTimeout)
			if _, err := part.Write(buffer[:count]); err != nil {
				return offset, false, fmt.Errorf("保存模型失败: %w", err)
			}
			offset += int64(count)
			if cfg.Progress != nil {
				cfg.Progress(offset, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			if ctx.Err() == nil && errors.Is(downloadCtx.Err(), context.DeadlineExceeded) {
				return offset, false, fmt.Errorf("下载超过总时限 %s", cfg.PartTimeout)
			}
			if ctx.Err() == nil && downloadCtx.Err() != nil {
				return offset, false, fmt.Errorf("下载超过 %s 没有进度", cfg.IdleTimeout)
			}
			return offset, false, readErr
		}
	}
	if total > 0 && offset != total {
		return offset, false, fmt.Errorf("模型下载大小不正确（得到 %d，预期 %d）", offset, total)
	}
	return offset, false, nil
}

func removeModelDownloadState(partPath string) {
	_ = os.Remove(partPath + ".json")
}
