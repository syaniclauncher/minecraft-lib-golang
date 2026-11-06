package mcl

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	content := "hello minecraft"
	sum := sha1.Sum([]byte(content))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(content))
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "file.txt")

	downloaded, err := downloadFile(context.Background(), server.URL, target, hex.EncodeToString(sum[:]), dir, false, nil)
	if err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	if !downloaded {
		t.Error("the file should have been downloaded")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("the file contains %q, want %q", data, content)
	}
}

func TestDownloadFileSkipsExistingFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the server should not be called")
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	writeFile(t, target, "content")

	downloaded, err := downloadFile(context.Background(), server.URL, target, "", dir, false, nil)
	if err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	if downloaded {
		t.Error("the file should have been skipped")
	}
}

func TestDownloadFileWithWrongChecksum(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")

	_, err := downloadFile(context.Background(), server.URL, target, strings.Repeat("0", 40), dir, false, nil)
	if err == nil {
		t.Fatal("downloadFile should fail on a wrong checksum")
	}
	var checksumError *InvalidChecksumError
	if !errors.As(err, &checksumError) {
		t.Fatalf("unexpected error: %v", err)
	}
	if checksumError.Expected != strings.Repeat("0", 40) {
		t.Errorf("the expected checksum is %q", checksumError.Expected)
	}
}

func TestDownloadFileOutsideMinecraftDirectory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the server should not be called")
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "..", "outside.txt")

	_, err := downloadFile(context.Background(), server.URL, target, "", dir, false, nil)
	if err == nil {
		t.Fatal("downloadFile should refuse to write outside of the directory")
	}
	var outsideError *FileOutsideMinecraftDirectoryError
	if !errors.As(err, &outsideError) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadFileReportsStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer server.Close()

	var status string
	progress := &Progress{Status: func(msg string) { status = msg }}

	dir := t.TempDir()
	if _, err := downloadFile(context.Background(), server.URL, filepath.Join(dir, "file.txt"), "", dir, false, progress); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	if !strings.Contains(status, "file.txt") {
		t.Errorf("the status is %q, want the name of the file", status)
	}
}

func TestDownloadFileReportsLabel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer server.Close()

	var status string
	progress := &Progress{Status: func(msg string) { status = msg }}

	dir := t.TempDir()
	if _, err := downloadFile(context.Background(), server.URL, filepath.Join(dir, "abc123"), "", dir, false, progress, "Assets"); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	if status != "Downloading Assets abc123" {
		t.Errorf("the status is %q, want %q", status, "Downloading Assets abc123")
	}
}

func TestCheckPathInsideMinecraftDirectory(t *testing.T) {
	dir := t.TempDir()

	if err := checkPathInsideMinecraftDirectory(dir, filepath.Join(dir, "file.txt")); err != nil {
		t.Errorf("a file inside the directory should be allowed: %v", err)
	}
	if err := checkPathInsideMinecraftDirectory(dir, filepath.Join(dir, "a", "b", "file.txt")); err != nil {
		t.Errorf("a nested file inside the directory should be allowed: %v", err)
	}
	if err := checkPathInsideMinecraftDirectory(dir, filepath.Join(dir, "..", "file.txt")); err == nil {
		t.Error("a file outside of the directory should be refused")
	}
	if err := checkPathInsideMinecraftDirectory(dir, dir+"other"); err == nil {
		t.Error("a sibling directory should be refused")
	}
}

func TestRunParallel(t *testing.T) {
	var count atomic.Int64
	err := runParallel(4, 20, func(i int) error {
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("runParallel: %v", err)
	}
	if count.Load() != 20 {
		t.Errorf("the function ran %d times, want 20", count.Load())
	}
}

func TestRunParallelReportsTheFirstError(t *testing.T) {
	err := runParallel(2, 10, func(i int) error {
		if i == 3 {
			return os.ErrInvalid
		}
		return nil
	})
	if err == nil {
		t.Fatal("runParallel should return the error")
	}
}

func TestDownloadFileFailsOnErrorStatus(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dir := t.TempDir()
	_, err := downloadFile(context.Background(), server.URL, filepath.Join(dir, "file.txt"), "", dir, false, nil)
	if err == nil {
		t.Fatal("downloadFile should fail on a 404")
	}
	var httpError *HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("unexpected error: %v", err)
	}
	if httpError.Retryable {
		t.Error("a 404 should not be retried")
	}
	if requests != 1 {
		t.Errorf("the server was called %d times, want 1", requests)
	}
}

func TestDownloadFileRetriesServerErrors(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("content"))
	}))
	defer server.Close()

	dir := t.TempDir()
	downloaded, err := downloadFile(context.Background(), server.URL, filepath.Join(dir, "file.txt"), "", dir, false, nil)
	if err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	if !downloaded {
		t.Error("the file should have been downloaded")
	}
	if requests != 3 {
		t.Errorf("the server was called %d times, want 3", requests)
	}
}

func TestDownloadFileResumes(t *testing.T) {
	content := "the whole content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "bytes=4-" {
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte(content[4:]))
			return
		}
		w.Write([]byte(content))
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	writeFile(t, target+".download", content[:4])

	if _, err := downloadFile(context.Background(), server.URL, target, "", dir, false, nil); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("the file contains %q, want %q", data, content)
	}
}

func TestDownloadFileHonoursTheContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := t.TempDir()
	_, err := downloadFile(ctx, server.URL, filepath.Join(dir, "file.txt"), "", dir, false, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the error is %v, want a cancelled context", err)
	}
}
