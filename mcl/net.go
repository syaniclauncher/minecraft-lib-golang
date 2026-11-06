package mcl

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Version is the version of this library.
const Version = "1.0.0"

// userAgent identifies this library to the servers it talks to.
var userAgent = "minecraft-launcher-lib-go/" + Version

// httpClient performs every network request of the library.
var httpClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConnsPerHost:   32,
		ForceAttemptHTTP2:     true,
	},
}

// cacheTTL is how long a downloaded document is reused.
const cacheTTL = time.Hour

var (
	cacheMu sync.Mutex
	cache   = map[string]cacheEntry{}
)

type cacheEntry struct {
	text string
	at   time.Time
}

// cachedText downloads a document and reuses the result for an hour, so that
// calling the same function repeatedly does not hammer the server.
func cachedText(ctx context.Context, url string) (string, error) {
	cacheMu.Lock()
	entry, ok := cache[url]
	cacheMu.Unlock()
	if ok && time.Since(entry.at) < cacheTTL {
		return entry.text, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("user-agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	text := string(data)
	cacheMu.Lock()
	cache[url] = cacheEntry{text: text, at: time.Now()}
	cacheMu.Unlock()
	return text, nil
}

// cachedJSON downloads a JSON document and decodes it into v.
func cachedJSON(ctx context.Context, url string, v any) error {
	text, err := cachedText(ctx, url)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(text), v); err != nil {
		return fmt.Errorf("parse %s: %w", url, err)
	}
	return nil
}

// postJSON sends a JSON document and decodes the response into v. The
// response is not cached.
func postJSON(ctx context.Context, url string, body any, header http.Header, v any) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = strings.NewReader(string(encoded))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("Content-Type", "application/json")
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s: %s: %s", url, resp.Status, strings.TrimSpace(string(data)))
	}
	if v == nil {
		return nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse response of %s: %w", url, err)
	}
	return nil
}

// downloadFile downloads url into path.
//
// When sha1 is not empty the checksum of the downloaded file is verified. The
// file is written to a temporary file first and only moved into place once it
// is complete. When the file already exists and matches the checksum it is
// left alone.
//
// The returned boolean reports whether the file was actually downloaded.
func downloadFile(ctx context.Context, url, path, sha1sum, minecraftDirectory string, overwrite bool, progress *Progress) (bool, error) {
	if minecraftDirectory != "" {
		if err := checkPathInsideMinecraftDirectory(minecraftDirectory, path); err != nil {
			return false, err
		}
	}

	if !overwrite {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			if sha1sum == "" {
				return false, nil
			}
			got, err := sha1Hash(path)
			if err != nil {
				return false, err
			}
			if got == sha1sum {
				return false, nil
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	progress.setStatus("Downloading " + filepath.Base(path))

	tmpPath := path + ".download"
	var lastErr error
	for attempt := 0; attempt <= 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if err := downloadOnce(ctx, url, tmpPath); err != nil {
			lastErr = err
			if attempt < 3 && isRetryable(err) {
				select {
				case <-ctx.Done():
					return false, ctx.Err()
				case <-time.After(downloadBackoff(attempt)):
				}
				continue
			}
			return false, lastErr
		}

		if err := os.Rename(tmpPath, path); err != nil {
			return false, err
		}
		if sha1sum != "" {
			got, err := sha1Hash(path)
			if err != nil {
				return false, err
			}
			if got != sha1sum {
				return false, &InvalidChecksumError{URL: url, Path: path, Expected: sha1sum, Actual: got}
			}
		}
		return true, nil
	}
	return false, lastErr
}

// downloadOnce downloads url into tmpPath, resuming a partial download when
// the server supports it.
func downloadOnce(ctx context.Context, url, tmpPath string) error {
	var resume int64
	if info, err := os.Stat(tmpPath); err == nil && info.Mode().IsRegular() {
		resume = info.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("user-agent", userAgent)
	if resume > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resume))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		resume = 0
	case http.StatusPartialContent:
		if resume == 0 {
			// The server answered with a partial document even
			// though no range was requested.
			resume = 0
		}
	default:
		return &HTTPError{
			URL:       url,
			Status:    resp.Status,
			Retryable: canRetryStatus(resp.StatusCode),
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if resume > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(tmpPath, flags, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return err
	}
	return file.Close()
}

// canRetryStatus reports whether a failed request is worth repeating.
func canRetryStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooEarly, http.StatusTooManyRequests:
		return true
	}
	return status >= 500 && status < 600
}

// isRetryable reports whether a failed request is worth another try. Network
// errors always are, an error status only when the server says so.
func isRetryable(err error) bool {
	var httpError *HTTPError
	if errors.As(err, &httpError) {
		return httpError.Retryable
	}
	return true
}

// downloadBackoff returns the delay before the n-th retry.
func downloadBackoff(attempt int) time.Duration {
	return time.Duration(250*time.Millisecond) * time.Duration(1<<attempt)
}

// checkPathInsideMinecraftDirectory reports whether path is inside the given
// Minecraft directory. It guards against a malicious metadata file escaping
// into the rest of the filesystem.
func checkPathInsideMinecraftDirectory(minecraftDirectory, path string) error {
	absDir, err := filepath.Abs(minecraftDirectory)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return &FileOutsideMinecraftDirectoryError{Path: absPath, MinecraftDirectory: absDir}
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return &FileOutsideMinecraftDirectoryError{Path: absPath, MinecraftDirectory: absDir}
	}
	return nil
}

// sha1Hash returns the SHA-1 checksum of a file.
func sha1Hash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// runParallel runs fn for every index below n, with at most max concurrent
// calls. It returns the first error that occurred.
func runParallel(max, n int, fn func(i int) error) error {
	if max < 1 {
		max = 1
	}
	sem := make(chan struct{}, max)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := fn(i); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	return firstErr
}

// randomUUID returns a random version 4 UUID in the dashed form Minecraft
// uses.
func randomUUID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

// randomDigits returns a random number of n digits.
func randomDigits(n int) string {
	const digits = "0123456789"
	out := make([]byte, n)
	for i := range out {
		var b [1]byte
		if _, err := rand.Read(b[:]); err != nil {
			out[i] = '0'
			continue
		}
		out[i] = digits[int(b[0])%len(digits)]
	}
	return string(out)
}
