package main

import (
	"bytes"
	"context"
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
