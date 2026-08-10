package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestModelDownloadFallbackUsesBackupRangesAfterPlainPrimaryFails(t *testing.T) {
	payload := []byte("0123456789abcdefghijklmnopqrstuv")
	var mu sync.Mutex
	var backupRanges []string
	var partProgress []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/primary":
			if request.Header.Get("Range") != "" {
				// Deliberately ignore Range so the downloader selects ordinary HTTP.
				writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
				_, _ = writer.Write(payload)
				return
			}
			writer.WriteHeader(http.StatusBadGateway)
		case "/backup":
			raw := request.Header.Get("Range")
			if raw == "" {
				t.Fatal("backup must use byte ranges")
			}
			start, end := modelDownloadTestRange(t, raw)
			if end >= len(payload) {
				t.Fatalf("range end %d beyond payload", end)
			}
			mu.Lock()
			backupRanges = append(backupRanges, raw)
			mu.Unlock()
			writer.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
			writer.Header().Set("Content-Length", strconv.Itoa(end-start+1))
			writer.WriteHeader(http.StatusPartialContent)
			_, _ = writer.Write(payload[start : end+1])
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	var reports []string
	part := t.TempDir() + "/model.part"
	result, err := downloadModelWithFallback(context.Background(), resumableModelDownloadConfig{
		Client:       server.Client(),
		URL:          server.URL + "/primary",
		BackupURL:    server.URL + "/backup",
		PartPath:     part,
		ExpectedSize: int64(len(payload)),
		Workers:      3,
		ChunkSize:    8,
		PartProgress: func(worker, part, parts int, copied, total int64) {
			mu.Lock()
			partProgress = append(partProgress, fmt.Sprintf("%d/%d/%d/%d", worker, part, parts, copied*100/total))
			mu.Unlock()
		},
	}, func(message string) { reports = append(reports, message) })
	if err != nil {
		t.Fatalf("downloadModelWithFallback() error = %v", err)
	}
	if !result.UsedBackup || !result.UsedRanges || result.URL != server.URL+"/backup" {
		t.Fatalf("result = %#v, want backup Range download", result)
	}
	actual, err := os.ReadFile(part)
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("downloaded bytes = %q, %v", actual, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(backupRanges) < 2 {
		t.Fatalf("backup range requests = %v, want probe plus multiple parts", backupRanges)
	}
	if len(partProgress) < 4 {
		t.Fatalf("part progress = %v, want every download part to report progress", partProgress)
	}
	joined := strings.Join(reports, "\n")
	if !strings.Contains(joined, "不支持 HTTP Range，回退到普通下载") || !strings.Contains(joined, "主下载源普通下载失败") || !strings.Contains(joined, "备用下载源支持 HTTP Range") {
		t.Fatalf("fallback logs = %q", joined)
	}
}

func modelDownloadTestRange(t *testing.T, value string) (int, int) {
	t.Helper()
	if !strings.HasPrefix(value, "bytes=") {
		t.Fatalf("invalid Range %q", value)
	}
	values := strings.Split(strings.TrimPrefix(value, "bytes="), "-")
	if len(values) != 2 {
		t.Fatalf("invalid Range %q", value)
	}
	start, startErr := strconv.Atoi(values[0])
	end, endErr := strconv.Atoi(values[1])
	if startErr != nil || endErr != nil || start < 0 || end < start {
		t.Fatalf("invalid Range %q", value)
	}
	return start, end
}

func TestModelRangeDownloadStopsAtPartTimeout(t *testing.T) {
	payload := []byte("0123456789abcdef")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		start, end := modelDownloadTestRange(t, request.Header.Get("Range"))
		writer.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		if start == 0 && end == 0 {
			writer.WriteHeader(http.StatusPartialContent)
			_, _ = writer.Write(payload[:1])
			return
		}
		select {
		case <-request.Context().Done():
			return
		case <-time.After(time.Second):
		}
		writer.WriteHeader(http.StatusPartialContent)
		_, _ = writer.Write(payload[start : end+1])
	}))
	defer server.Close()
	_, _, err := downloadModelWithResume(context.Background(), resumableModelDownloadConfig{
		Client:       server.Client(),
		URL:          server.URL,
		PartPath:     t.TempDir() + "/model.part",
		ExpectedSize: int64(len(payload)),
		Workers:      2,
		ChunkSize:    8,
		IdleTimeout:  time.Second,
		PartTimeout:  20 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "总时限") {
		t.Fatalf("timeout error = %v, want part total timeout", err)
	}
}

func TestModelRangeDownloadRetriesEachPart(t *testing.T) {
	payload := []byte("0123456789abcdef")
	var mu sync.Mutex
	attempts := map[string]int{}
	var retries []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		start, end := modelDownloadTestRange(t, request.Header.Get("Range"))
		key := fmt.Sprintf("%d-%d", start, end)
		mu.Lock()
		attempts[key]++
		attempt := attempts[key]
		mu.Unlock()
		if start != 0 || end != 0 {
			if attempt == 1 {
				writer.WriteHeader(http.StatusBadGateway)
				return
			}
		}
		writer.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		writer.Header().Set("Content-Length", strconv.Itoa(end-start+1))
		writer.WriteHeader(http.StatusPartialContent)
		_, _ = writer.Write(payload[start : end+1])
	}))
	defer server.Close()
	part := t.TempDir() + "/model.part"
	_, usedRanges, err := downloadModelWithResume(context.Background(), resumableModelDownloadConfig{
		Client:       server.Client(),
		URL:          server.URL,
		PartPath:     part,
		ExpectedSize: int64(len(payload)),
		Workers:      2,
		Retries:      1,
		ChunkSize:    8,
		Retry: func(worker, part, parts, retry, retriesCount int, err error) {
			mu.Lock()
			retries = append(retries, fmt.Sprintf("%d/%d/%d/%d", worker, part, retry, retriesCount))
			mu.Unlock()
		},
	})
	if err != nil || !usedRanges {
		t.Fatalf("downloadModelWithResume() = %t, %v", usedRanges, err)
	}
	actual, err := os.ReadFile(part)
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("downloaded bytes = %q, %v", actual, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(retries) != 2 {
		t.Fatalf("part retries = %v, want one retry for each of two parts", retries)
	}
}

func TestModelSequentialRetryReportsResetProgressWhenRangeIsIgnored(t *testing.T) {
	payload := []byte("0123456789abcdefghij")
	var transfers int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Range") == "bytes=0-0" {
			writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			_, _ = writer.Write(payload)
			return
		}
		transfers++
		writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		if transfers == 1 {
			// Advertise the full object but close early, forcing a retry after
			// some bytes have reached the local part file.
			_, _ = writer.Write(payload[:8])
			return
		}
		// Deliberately ignore the retry's Range request. The downloader must
		// truncate its local partial file and report that it restarted at 0%.
		_, _ = writer.Write(payload)
	}))
	defer server.Close()

	var retryOffsets []int64
	part := t.TempDir() + "/model.part"
	got, usedRanges, err := downloadModelWithResume(context.Background(), resumableModelDownloadConfig{
		Client:       server.Client(),
		URL:          server.URL,
		PartPath:     part,
		ExpectedSize: int64(len(payload)),
		Retries:      1,
		RetryProgress: func(copied, total int64) {
			if total != int64(len(payload)) {
				t.Errorf("retry total = %d, want %d", total, len(payload))
			}
			retryOffsets = append(retryOffsets, copied)
		},
	})
	if err != nil || usedRanges || got != int64(len(payload)) {
		t.Fatalf("downloadModelWithResume() = (%d, %t, %v)", got, usedRanges, err)
	}
	if len(retryOffsets) != 1 || retryOffsets[0] != 0 {
		t.Fatalf("retry offsets = %v, want [0] after ignored Range", retryOffsets)
	}
	actual, readErr := os.ReadFile(part)
	if readErr != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("part = %q, %v", actual, readErr)
	}
}

func TestModelDownloadProbeStopsAtPartTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
			return
		case <-time.After(time.Second):
		}
		writer.WriteHeader(http.StatusPartialContent)
	}))
	defer server.Close()
	_, _, err := downloadModelWithResume(context.Background(), resumableModelDownloadConfig{
		Client:      server.Client(),
		URL:         server.URL,
		PartPath:    t.TempDir() + "/model.part",
		PartTimeout: 20 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "检查下载源是否支持 HTTP Range 超过总时限") {
		t.Fatalf("probe timeout error = %v", err)
	}
}

func TestModelSequentialResumeTreats416AtEOFAsComplete(t *testing.T) {
	payload := []byte("already-complete")
	var eofRanges int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("ETag", `"v1"`)
		switch request.Header.Get("Range") {
		case "bytes=0-0":
			writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			_, _ = writer.Write(payload)
		case fmt.Sprintf("bytes=%d-", len(payload)):
			if got := request.Header.Get("If-Range"); got != `"v1"` {
				t.Fatalf("If-Range = %q, want ETag", got)
			}
			eofRanges++
			writer.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(payload)))
			writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		default:
			t.Fatalf("unexpected Range %q", request.Header.Get("Range"))
		}
	}))
	defer server.Close()

	part := t.TempDir() + "/model.part"
	if err := os.WriteFile(part, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveModelDownloadState(part+".json", resumableModelDownloadState{URL: server.URL, Size: int64(len(payload)), ETag: `"v1"`, Chunks: []bool{}}); err != nil {
		t.Fatal(err)
	}
	got, usedRanges, err := downloadModelWithResume(context.Background(), resumableModelDownloadConfig{
		Client:       server.Client(),
		URL:          server.URL,
		PartPath:     part,
		ExpectedSize: int64(len(payload)),
	})
	if err != nil || usedRanges || got != int64(len(payload)) {
		t.Fatalf("downloadModelWithResume() = (%d, %t, %v)", got, usedRanges, err)
	}
	actual, err := os.ReadFile(part)
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("part = %q, %v", actual, err)
	}
	if eofRanges != 1 {
		t.Fatalf("416 EOF requests = %d, want 1", eofRanges)
	}
}

func TestModelSequentialResumeResetsAfter416ForChangedObject(t *testing.T) {
	oldPayload := []byte("0123456789ab")
	newPayload := []byte("new-data")
	var probes int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Header.Get("Range") {
		case "bytes=0-0":
			probes++
			if probes == 1 {
				writer.Header().Set("ETag", `"old"`)
				writer.Header().Set("Content-Length", strconv.Itoa(len(oldPayload)))
				_, _ = writer.Write(oldPayload)
				return
			}
			writer.Header().Set("ETag", `"new"`)
			writer.Header().Set("Content-Length", strconv.Itoa(len(newPayload)))
			_, _ = writer.Write(newPayload)
		case fmt.Sprintf("bytes=%d-", len(oldPayload)):
			writer.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(newPayload)))
			writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		case "":
			if probes != 2 {
				t.Fatalf("ordinary transfer started before re-probe, probes=%d", probes)
			}
			writer.Header().Set("ETag", `"new"`)
			writer.Header().Set("Content-Length", strconv.Itoa(len(newPayload)))
			_, _ = writer.Write(newPayload)
		default:
			t.Fatalf("unexpected Range %q", request.Header.Get("Range"))
		}
	}))
	defer server.Close()

	part := t.TempDir() + "/model.part"
	if err := os.WriteFile(part, oldPayload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveModelDownloadState(part+".json", resumableModelDownloadState{URL: server.URL, Size: int64(len(oldPayload)), ETag: `"old"`, Chunks: []bool{}}); err != nil {
		t.Fatal(err)
	}
	got, _, err := downloadModelWithResume(context.Background(), resumableModelDownloadConfig{Client: server.Client(), URL: server.URL, PartPath: part})
	if err != nil || got != int64(len(newPayload)) {
		t.Fatalf("downloadModelWithResume() = (%d, %v)", got, err)
	}
	actual, err := os.ReadFile(part)
	if err != nil || !bytes.Equal(actual, newPayload) {
		t.Fatalf("part = %q, %v", actual, err)
	}
	if probes != 2 {
		t.Fatalf("probe count = %d, want 2", probes)
	}
}

func TestModelRangeResumeReprobesAfter416(t *testing.T) {
	oldPayload := []byte("0123456789abcdef")
	newPayload := []byte("new-data")
	var mu sync.Mutex
	probes := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw := request.Header.Get("Range")
		mu.Lock()
		defer mu.Unlock()
		if raw == "bytes=0-0" {
			probes++
			if probes == 1 {
				writer.Header().Set("ETag", `"old"`)
				writer.Header().Set("Content-Range", fmt.Sprintf("bytes 0-0/%d", len(oldPayload)))
				writer.WriteHeader(http.StatusPartialContent)
				_, _ = writer.Write(oldPayload[:1])
				return
			}
			writer.Header().Set("ETag", `"new"`)
			writer.Header().Set("Content-Range", fmt.Sprintf("bytes 0-0/%d", len(newPayload)))
			writer.WriteHeader(http.StatusPartialContent)
			_, _ = writer.Write(newPayload[:1])
			return
		}
		if probes == 1 {
			if got := request.Header.Get("If-Range"); got != `"old"` {
				t.Fatalf("If-Range = %q, want old ETag", got)
			}
			writer.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(newPayload)))
			writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		start, end := modelDownloadTestRange(t, raw)
		if got := request.Header.Get("If-Range"); got != `"new"` {
			t.Fatalf("If-Range = %q, want new ETag", got)
		}
		if end >= len(newPayload) {
			t.Fatalf("range %q exceeds changed payload", raw)
		}
		writer.Header().Set("ETag", `"new"`)
		writer.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(newPayload)))
		writer.WriteHeader(http.StatusPartialContent)
		_, _ = writer.Write(newPayload[start : end+1])
	}))
	defer server.Close()

	part := t.TempDir() + "/model.part"
	got, usedRanges, err := downloadModelWithResume(context.Background(), resumableModelDownloadConfig{
		Client:    server.Client(),
		URL:       server.URL,
		PartPath:  part,
		Workers:   1,
		ChunkSize: 4,
	})
	if err != nil || !usedRanges || got != int64(len(newPayload)) {
		t.Fatalf("downloadModelWithResume() = (%d, %t, %v)", got, usedRanges, err)
	}
	actual, err := os.ReadFile(part)
	if err != nil || !bytes.Equal(actual, newPayload) {
		t.Fatalf("part = %q, %v", actual, err)
	}
	if probes != 2 {
		t.Fatalf("probe count = %d, want 2", probes)
	}
}

func TestModelDownloadFallbackAfter416WithoutContentRange(t *testing.T) {
	payload := []byte("backup-file")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/primary":
			writer.Header().Set("ETag", `"primary"`)
			switch request.Header.Get("Range") {
			case "bytes=0-0":
				writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
				_, _ = writer.Write(payload)
			case "bytes=4-":
				writer.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			default:
				t.Fatalf("unexpected primary Range %q", request.Header.Get("Range"))
			}
		case "/backup":
			writer.Header().Set("ETag", `"backup"`)
			writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			_, _ = writer.Write(payload)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	part := t.TempDir() + "/model.part"
	if err := os.WriteFile(part, []byte("part"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveModelDownloadState(part+".json", resumableModelDownloadState{URL: server.URL + "/primary", Size: int64(len(payload)), ETag: `"primary"`, Chunks: []bool{}}); err != nil {
		t.Fatal(err)
	}
	result, err := downloadModelWithFallback(context.Background(), resumableModelDownloadConfig{
		Client:    server.Client(),
		URL:       server.URL + "/primary",
		BackupURL: server.URL + "/backup",
		PartPath:  part,
	}, nil)
	if err != nil || !result.UsedBackup || !result.UsedPlainHTTP {
		t.Fatalf("downloadModelWithFallback() = (%#v, %v)", result, err)
	}
	actual, err := os.ReadFile(part)
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("part = %q, %v", actual, err)
	}
}

func TestModelResumeRejectsStateWithChangedETag(t *testing.T) {
	payload := []byte("fresh-content")
	var resumedRange bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("ETag", `"new"`)
		switch request.Header.Get("Range") {
		case "bytes=0-0":
			writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			_, _ = writer.Write(payload)
		case "":
			writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			_, _ = writer.Write(payload)
		default:
			resumedRange = true
			writer.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	part := t.TempDir() + "/model.part"
	if err := os.WriteFile(part, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveModelDownloadState(part+".json", resumableModelDownloadState{URL: server.URL, Size: int64(len(payload)), ETag: `"old"`, Chunks: []bool{}}); err != nil {
		t.Fatal(err)
	}
	got, _, err := downloadModelWithResume(context.Background(), resumableModelDownloadConfig{Client: server.Client(), URL: server.URL, PartPath: part, ExpectedSize: int64(len(payload))})
	if err != nil || got != int64(len(payload)) {
		t.Fatalf("downloadModelWithResume() = (%d, %v)", got, err)
	}
	actual, err := os.ReadFile(part)
	if err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("part = %q, %v", actual, err)
	}
	if resumedRange {
		t.Fatal("changed ETag must discard stale partial file before the transfer")
	}
}

func TestModelDownloadFallbackUsesBackupSourceMetadata(t *testing.T) {
	primary := []byte("primary-metadata-is-longer")
	backup := []byte("mirror")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/primary":
			if request.Header.Get("Range") == "bytes=0-0" {
				writer.Header().Set("Content-Length", strconv.Itoa(len(primary)))
				_, _ = writer.Write(primary)
				return
			}
			writer.WriteHeader(http.StatusBadGateway)
		case "/backup":
			if request.Header.Get("Range") == "bytes=0-0" {
				writer.Header().Set("Content-Length", strconv.Itoa(len(backup)))
				_, _ = writer.Write(backup)
				return
			}
			if request.Header.Get("Range") != "" {
				t.Fatalf("backup ordinary transfer must not inherit primary offset: %q", request.Header.Get("Range"))
			}
			writer.Header().Set("Content-Length", strconv.Itoa(len(backup)))
			_, _ = writer.Write(backup)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	part := t.TempDir() + "/model.part"
	result, err := downloadModelWithFallback(context.Background(), resumableModelDownloadConfig{
		Client:             server.Client(),
		URL:                server.URL + "/primary",
		BackupURL:          server.URL + "/backup",
		PartPath:           part,
		ExpectedSize:       int64(len(primary)),
		SourceID:           "official:test",
		ArtifactPath:       "weights.bin",
		BackupSourceID:     "mirror:test",
		BackupArtifactPath: "weights.bin",
		PrepareBackup: func(context.Context) (modelDownloadSourceMetadata, error) {
			return modelDownloadSourceMetadata{SourceID: "mirror:test", Revision: "main", ArtifactPath: "weights.bin", ExpectedSize: int64(len(backup))}, nil
		},
	}, nil)
	if err != nil || !result.UsedBackup || result.Bytes != int64(len(backup)) {
		t.Fatalf("downloadModelWithFallback() = (%#v, %v)", result, err)
	}
	actual, readErr := os.ReadFile(part)
	if readErr != nil || !bytes.Equal(actual, backup) {
		t.Fatalf("backup part = %q, %v", actual, readErr)
	}
	stateContent, readErr := os.ReadFile(part + ".json")
	if readErr != nil {
		t.Fatal(readErr)
	}
	var state resumableModelDownloadState
	if err := json.Unmarshal(stateContent, &state); err != nil {
		t.Fatal(err)
	}
	if state.SourceID != "mirror:test" || state.Revision != "main" || state.ArtifactPath != "weights.bin" || state.Size != int64(len(backup)) {
		t.Fatalf("backup state = %#v", state)
	}
}
