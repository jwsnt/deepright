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
	Client             *http.Client
	URL                string
	BackupURL          string
	PartPath           string
	ExpectedSize       int64
	BackupExpectedSize int64
	SourceID           string
	BackupSourceID     string
	Revision           string
	BackupRevision     string
	ArtifactPath       string
	BackupArtifactPath string
	PrepareBackup      func(context.Context) (modelDownloadSourceMetadata, error)
	IdleTimeout        time.Duration
	PartTimeout        time.Duration
	Workers            int
	Retries            int
	ChunkSize          int64
	Progress           func(copied, total int64)
	PartProgress       func(worker, part, parts int, copied, total int64)
	Retry              func(worker, part, parts, retry, retries int, err error)
}

type modelDownloadResult struct {
	URL           string
	Bytes         int64
	UsedRanges    bool
	UsedBackup    bool
	UsedPlainHTTP bool
}

type resumableModelDownloadState struct {
	URL          string `json:"url"`
	SourceID     string `json:"source_id,omitempty"`
	Revision     string `json:"revision,omitempty"`
	ArtifactPath string `json:"artifact_path,omitempty"`
	Size         int64  `json:"size"`
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	Chunks       []bool `json:"chunks"`
}

// modelDownloadSourceMetadata is the source-owned manifest identity for one
// artifact. It must never be inherited from the primary source after a
// fallback. ExpectedSize of zero means the source did not publish a size.
type modelDownloadSourceMetadata struct {
	SourceID     string
	Revision     string
	ArtifactPath string
	ExpectedSize int64
}

// modelDownloadSource identifies the exact representation observed during the
// range probe. A saved partial file is reusable only for that representation.
type modelDownloadSource struct {
	URL          string
	SourceID     string
	Revision     string
	ArtifactPath string
	Size         int64
	Ranges       bool
	ETag         string
	LastModified string
}

func (source modelDownloadSource) ifRange() string {
	if etag := strings.TrimSpace(source.ETag); etag != "" {
		return etag
	}
	return strings.TrimSpace(source.LastModified)
}

// modelDownloadRangeNotSatisfiableError is returned only for a valid HTTP 416
// response. Its Size is the server's current representation length from
// Content-Range: bytes */<size>.
type modelDownloadRangeNotSatisfiableError struct {
	Size int64
}

func (err *modelDownloadRangeNotSatisfiableError) Error() string {
	return fmt.Sprintf("下载源拒绝续传范围（HTTP 416，当前大小 %d）", err.Size)
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

func modelDownloadPrimaryMetadata(cfg resumableModelDownloadConfig) modelDownloadSourceMetadata {
	return modelDownloadSourceMetadata{
		SourceID:     firstModelDownloadSourceID(cfg.SourceID, cfg.URL),
		Revision:     strings.TrimSpace(cfg.Revision),
		ArtifactPath: strings.TrimSpace(cfg.ArtifactPath),
		ExpectedSize: cfg.ExpectedSize,
	}
}

func modelDownloadBackupMetadata(cfg resumableModelDownloadConfig) modelDownloadSourceMetadata {
	return modelDownloadSourceMetadata{
		SourceID:     firstModelDownloadSourceID(cfg.BackupSourceID, cfg.BackupURL),
		Revision:     strings.TrimSpace(cfg.BackupRevision),
		ArtifactPath: strings.TrimSpace(cfg.BackupArtifactPath),
		ExpectedSize: cfg.BackupExpectedSize,
	}
}

func firstModelDownloadSourceID(sourceID, fallback string) string {
	if sourceID = strings.TrimSpace(sourceID); sourceID != "" {
		return sourceID
	}
	return strings.TrimSpace(fallback)
}

func applyModelDownloadSourceMetadata(cfg resumableModelDownloadConfig, metadata modelDownloadSourceMetadata) resumableModelDownloadConfig {
	cfg.SourceID = firstModelDownloadSourceID(metadata.SourceID, cfg.URL)
	cfg.Revision = strings.TrimSpace(metadata.Revision)
	cfg.ArtifactPath = strings.TrimSpace(metadata.ArtifactPath)
	cfg.ExpectedSize = metadata.ExpectedSize
	return cfg
}

func downloadModelFromSource(ctx context.Context, cfg resumableModelDownloadConfig, before func(bool)) (int64, bool, error) {
	// A 416 for a data range means that the object changed after the probe, or
	// that the local resume point is stale. Discard only the partial artifact,
	// re-probe, and make one clean attempt. A bounded correction prevents an
	// unstable source from causing an endless restart loop.
	for correction := 0; correction <= 1; correction++ {
		source, err := modelDownloadProbe(ctx, cfg)
		if err != nil {
			return 0, false, err
		}
		total := source.Size
		if cfg.ExpectedSize > 0 && total >= 0 && total != cfg.ExpectedSize {
			return 0, source.Ranges, fmt.Errorf("模型大小与预期不一致（源站 %d，预期 %d）", total, cfg.ExpectedSize)
		}
		if cfg.ExpectedSize > 0 {
			total = cfg.ExpectedSize
		}
		if before != nil && correction == 0 {
			before(source.Ranges && total > 0)
		}

		var (
			bytes      int64
			usedRanges bool
		)
		if source.Ranges && total > 0 {
			bytes, usedRanges, err = downloadModelRanges(ctx, cfg, source, total)
		} else {
			bytes, usedRanges, err = downloadModelSequentially(ctx, cfg, source, total)
		}
		if err == nil {
			return bytes, usedRanges, nil
		}
		var unsatisfied *modelDownloadRangeNotSatisfiableError
		if !errors.As(err, &unsatisfied) || correction == 1 || ctx.Err() != nil {
			return bytes, usedRanges, err
		}
		if err := discardModelDownloadPartial(cfg.PartPath); err != nil {
			return bytes, usedRanges, err
		}
	}
	return 0, false, errors.New("模型下载状态重置后未完成")
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
		url      string
		backup   bool
		metadata modelDownloadSourceMetadata
	}{{url: primary, metadata: modelDownloadPrimaryMetadata(cfg)}}
	if backup != "" && backup != primary {
		sources = append(sources, struct {
			url      string
			backup   bool
			metadata modelDownloadSourceMetadata
		}{url: backup, backup: true, metadata: modelDownloadBackupMetadata(cfg)})
	}
	var lastErr error
	for index, source := range sources {
		attempt := cfg
		attempt.URL = source.url
		attempt.BackupURL = ""
		metadata := source.metadata
		if source.backup && cfg.PrepareBackup != nil {
			prepared, prepareErr := cfg.PrepareBackup(ctx)
			if prepareErr != nil {
				lastErr = fmt.Errorf("无法读取备用下载源元数据: %w", prepareErr)
				reportModelDownload(report, lastErr.Error())
				continue
			}
			metadata = prepared
		}
		attempt = applyModelDownloadSourceMetadata(attempt, metadata)
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
			fallbackSource, probeErr := modelDownloadProbe(ctx, attempt)
			if probeErr != nil {
				fallbackSource = modelDownloadSource{}
			}
			// This is intentionally an ordinary-transfer fallback. It still saves
			// validators in the manifest, but starts with no Range request.
			fallbackSource.Ranges = false
			fallbackTotal := fallbackSource.Size
			if attempt.ExpectedSize > 0 {
				fallbackTotal = attempt.ExpectedSize
			}
			bytes, _, err = downloadModelSequentially(ctx, attempt, fallbackSource, fallbackTotal)
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

func modelDownloadProbe(ctx context.Context, cfg resumableModelDownloadConfig) (modelDownloadSource, error) {
	probeCtx, cancel := context.WithTimeout(ctx, cfg.PartTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return modelDownloadSource{}, fmt.Errorf("无法创建模型下载请求: %w", err)
	}
	request.Header.Set("Range", "bytes=0-0")
	response, err := cfg.Client.Do(request)
	if err != nil {
		if ctx.Err() == nil && errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
			return modelDownloadSource{}, fmt.Errorf("检查下载源是否支持 HTTP Range 超过总时限 %s", cfg.PartTimeout)
		}
		return modelDownloadSource{}, fmt.Errorf("请求模型下载源失败: %w", err)
	}
	defer response.Body.Close()
	source := modelDownloadSource{
		URL:          strings.TrimSpace(cfg.URL),
		SourceID:     firstModelDownloadSourceID(cfg.SourceID, cfg.URL),
		Revision:     strings.TrimSpace(cfg.Revision),
		ArtifactPath: strings.TrimSpace(cfg.ArtifactPath),
		ETag:         strings.TrimSpace(response.Header.Get("ETag")),
		LastModified: strings.TrimSpace(response.Header.Get("Last-Modified")),
	}
	if response.StatusCode == http.StatusPartialContent {
		total, ok := modelDownloadContentRangeSize(response.Header.Get("Content-Range"))
		if !ok || total <= 0 {
			return modelDownloadSource{}, errors.New("模型下载源返回了无效的 Content-Range")
		}
		source.Size, source.Ranges = total, true
		return source, nil
	}
	if response.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		total, ok := modelDownloadUnsatisfiedRangeSize(response.Header.Get("Content-Range"))
		if !ok || total != 0 {
			return modelDownloadSource{}, fmt.Errorf("模型下载源探测 Range 返回 HTTP 416：%s", strings.TrimSpace(response.Header.Get("Content-Range")))
		}
		source.Size, source.Ranges = 0, true
		return source, nil
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return modelDownloadSource{}, fmt.Errorf("模型下载源返回 HTTP %d", response.StatusCode)
	}
	source.Size = response.ContentLength
	return source, nil
}

func modelDownloadContentRangeSize(value string) (int64, bool) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 || parts[1] == "*" {
		return 0, false
	}
	size, err := strconv.ParseInt(parts[1], 10, 64)
	return size, err == nil
}

func modelDownloadUnsatisfiedRangeSize(value string) (int64, bool) {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bytes") || !strings.HasPrefix(parts[1], "*/") {
		return 0, false
	}
	size, err := strconv.ParseInt(strings.TrimPrefix(parts[1], "*/"), 10, 64)
	return size, err == nil && size >= 0
}

func downloadModelRanges(ctx context.Context, cfg resumableModelDownloadConfig, source modelDownloadSource, total int64) (int64, bool, error) {
	chunks := int((total + cfg.ChunkSize - 1) / cfg.ChunkSize)
	if chunks <= 1 {
		return downloadModelSequentially(ctx, cfg, source, total)
	}
	statePath := cfg.PartPath + ".json"
	state, err := prepareModelDownloadState(cfg.PartPath, source, total, chunks, true)
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
				if err := downloadModelRange(workCtx, cfg, source, part, start, end, func(copied, total int64) {
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

func prepareModelDownloadState(partPath string, source modelDownloadSource, size int64, chunks int, fixedLength bool) (resumableModelDownloadState, error) {
	statePath := partPath + ".json"
	state, reusable, err := loadModelDownloadState(statePath, source, size, chunks)
	if err != nil {
		return state, err
	}
	info, statErr := os.Stat(partPath)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return state, fmt.Errorf("无法读取模型临时文件: %w", statErr)
	}
	partExists := statErr == nil && !info.IsDir()
	invalidPart := !reusable && partExists
	if reusable && !partExists {
		invalidPart = true
	}
	if reusable && partExists && fixedLength && info.Size() != size {
		invalidPart = true
	}
	if reusable && partExists && !fixedLength && size > 0 && info.Size() > size {
		invalidPart = true
	}
	if !invalidPart {
		return state, nil
	}
	if err := discardModelDownloadPartial(partPath); err != nil {
		return state, err
	}
	state = resumableModelDownloadState{
		URL:          strings.TrimSpace(source.URL),
		SourceID:     strings.TrimSpace(source.SourceID),
		Revision:     strings.TrimSpace(source.Revision),
		ArtifactPath: strings.TrimSpace(source.ArtifactPath),
		Size:         size,
		ETag:         strings.TrimSpace(source.ETag),
		LastModified: strings.TrimSpace(source.LastModified),
		Chunks:       make([]bool, chunks),
	}
	if err := saveModelDownloadState(statePath, state); err != nil {
		return state, err
	}
	return state, nil
}

func loadModelDownloadState(path string, source modelDownloadSource, size int64, chunks int) (resumableModelDownloadState, bool, error) {
	state := resumableModelDownloadState{
		URL:          strings.TrimSpace(source.URL),
		SourceID:     strings.TrimSpace(source.SourceID),
		Revision:     strings.TrimSpace(source.Revision),
		ArtifactPath: strings.TrimSpace(source.ArtifactPath),
		Size:         size,
		ETag:         strings.TrimSpace(source.ETag),
		LastModified: strings.TrimSpace(source.LastModified),
		Chunks:       make([]bool, chunks),
	}
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := saveModelDownloadState(path, state); err != nil {
			return state, false, err
		}
		return state, false, nil
	}
	if err != nil {
		return state, false, fmt.Errorf("无法读取模型断点续传状态: %w", err)
	}
	var stored resumableModelDownloadState
	decodeErr := json.Unmarshal(content, &stored)
	storedSourceID := strings.TrimSpace(stored.SourceID)
	if storedSourceID == "" {
		// Manifests written before source_id was added were already bound to
		// the exact URL. Preserve that safe resume path while requiring all
		// newer source identities to match explicitly.
		storedSourceID = strings.TrimSpace(stored.URL)
	}
	if decodeErr != nil || stored.URL != strings.TrimSpace(source.URL) || storedSourceID != strings.TrimSpace(source.SourceID) || stored.Revision != strings.TrimSpace(source.Revision) || stored.ArtifactPath != strings.TrimSpace(source.ArtifactPath) || stored.Size != size || stored.ETag != strings.TrimSpace(source.ETag) || stored.LastModified != strings.TrimSpace(source.LastModified) || len(stored.Chunks) != chunks {
		if err := saveModelDownloadState(path, state); err != nil {
			return state, false, err
		}
		return state, false, nil
	}
	return stored, true, nil
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

func downloadModelRange(ctx context.Context, cfg resumableModelDownloadConfig, source modelDownloadSource, output *os.File, start, end int64, progress func(copied, total int64), reportRetry func(retry, retries int, err error)) error {
	var lastErr error
	for retry := 0; retry <= cfg.Retries; retry++ {
		if retry > 0 && reportRetry != nil {
			reportRetry(retry, cfg.Retries, lastErr)
		}
		err := downloadModelRangeAttempt(ctx, cfg, source, output, start, end, progress)
		if err == nil {
			return nil
		}
		lastErr = err
		var unsatisfied *modelDownloadRangeNotSatisfiableError
		if errors.As(err, &unsatisfied) {
			return err
		}
		if ctx.Err() != nil || retry == cfg.Retries {
			return err
		}
	}
	return lastErr
}

func downloadModelRangeAttempt(ctx context.Context, cfg resumableModelDownloadConfig, source modelDownloadSource, output *os.File, start, end int64, progress func(copied, total int64)) error {
	downloadCtx, cancel := context.WithTimeout(ctx, cfg.PartTimeout)
	defer cancel()
	idle := time.AfterFunc(cfg.IdleTimeout, cancel)
	defer idle.Stop()
	request, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	if validator := source.ifRange(); validator != "" {
		request.Header.Set("If-Range", validator)
	}
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
	if response.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		if total, ok := modelDownloadUnsatisfiedRangeSize(response.Header.Get("Content-Range")); ok {
			return &modelDownloadRangeNotSatisfiableError{Size: total}
		}
		return errors.New("下载源返回 HTTP 416 但未提供有效 Content-Range")
	}
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

func downloadModelSequentially(ctx context.Context, cfg resumableModelDownloadConfig, source modelDownloadSource, total int64) (int64, bool, error) {
	if _, err := prepareModelDownloadState(cfg.PartPath, source, total, 0, false); err != nil {
		return 0, false, err
	}
	var copied int64
	var lastErr error
	for retry := 0; retry <= cfg.Retries; retry++ {
		if retry > 0 && cfg.Retry != nil {
			cfg.Retry(0, 0, 0, retry, cfg.Retries, lastErr)
		}
		copied, _, lastErr = downloadModelSequentialAttempt(ctx, cfg, source, total)
		if lastErr == nil {
			return copied, false, nil
		}
		var unsatisfied *modelDownloadRangeNotSatisfiableError
		if errors.As(lastErr, &unsatisfied) {
			// A request for bytes=N- returning "bytes */N" means the local
			// partial file already reaches the remote EOF. The caller still
			// validates the final digest before publication.
			if copied == unsatisfied.Size && (total <= 0 || copied == total) {
				return copied, false, nil
			}
			return copied, false, lastErr
		}
		if ctx.Err() != nil || retry == cfg.Retries {
			return copied, false, lastErr
		}
	}
	return copied, false, lastErr
}

func downloadModelSequentialAttempt(ctx context.Context, cfg resumableModelDownloadConfig, source modelDownloadSource, total int64) (int64, bool, error) {
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
		if validator := source.ifRange(); validator != "" {
			request.Header.Set("If-Range", validator)
		}
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
	if response.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		if size, ok := modelDownloadUnsatisfiedRangeSize(response.Header.Get("Content-Range")); ok {
			return offset, false, &modelDownloadRangeNotSatisfiableError{Size: size}
		}
		return offset, false, errors.New("下载源返回 HTTP 416 但未提供有效 Content-Range")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return offset, false, fmt.Errorf("模型下载源返回 HTTP %d", response.StatusCode)
	}
	if offset > 0 && response.StatusCode == http.StatusOK {
		if source.Ranges && source.ifRange() != "" {
			// A Range-capable source answered 200 to an If-Range request: the
			// representation changed. Re-probe before accepting any bytes.
			return offset, false, &modelDownloadRangeNotSatisfiableError{Size: -1}
		}
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

func discardModelDownloadPartial(partPath string) error {
	if err := os.Remove(partPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("无法清理失效的模型临时文件: %w", err)
	}
	removeModelDownloadState(partPath)
	return nil
}
