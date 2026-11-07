package mcl_test

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

// jvmPlatform returns the name of the folder the Java runtimes of the current
// platform are installed in.
func jvmPlatform() string {
	switch runtime.GOOS {
	case "windows":
		if strconv.IntSize == 32 {
			return "windows-x86"
		}
		return "windows-x64"
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "mac-os-arm64"
		}
		return "mac-os"
	case "linux":
		if strconv.IntSize == 32 {
			return "linux-i386"
		}
		return "linux"
	default:
		return "gamecore"
	}
}

// installJavaRuntime creates the runtime of a Java version, so that the
// installation does not have to download it.
func installJavaRuntime(t *testing.T, minecraftDirectory string, majorVersion int) {
	t.Helper()
	base := filepath.Join(minecraftDirectory, "runtime", fmt.Sprintf("temurin-%d", majorVersion), jvmPlatform())
	root := "jdk-21.0.1+1"
	writeFile(t, filepath.Join(base, ".temurin"), root)
	writeFile(t, filepath.Join(base, root, "bin", "java"), "")
}

func TestInstall(t *testing.T) {
	assetContent := "asset"
	assetHash := sha1.Sum([]byte(assetContent))
	assetHashHex := hex.EncodeToString(assetHash[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/assets.json":
			w.Write([]byte(`{"objects": {"minecraft/sounds/click": {"hash": "` + assetHashHex + `", "size": 5}}}`))
		case "/resources/" + assetHashHex[:2] + "/" + assetHashHex:
			w.Write([]byte(assetContent))
		default:
			w.Write([]byte("library"))
		}
	}))
	defer server.Close()

	originalResourcesURL := mcl.ResourcesURL
	originalLibrariesURL := mcl.LibrariesURL
	mcl.ResourcesURL = server.URL + "/resources"
	mcl.LibrariesURL = server.URL + "/libraries"
	defer func() {
		mcl.ResourcesURL = originalResourcesURL
		mcl.LibrariesURL = originalLibrariesURL
	}()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.json"), fmt.Sprintf(`{
		"id": "1.20.1",
		"type": "release",
		"assets": "12",
		"assetIndex": {"id": "12", "url": "%s/assets.json"},
		"libraries": [{
			"name": "org.example:example:1.0",
			"downloads": {"artifact": {"path": "org/example/artifact/1.0/artifact-1.0.jar", "url": "%s/artifact"}}
		}]
	}`, server.URL, server.URL))
	installJavaRuntime(t, dir, 8)

	var downloadedBytes, totalBytes atomic.Int64
	progress := &mcl.Progress{
		Download: func(name string, downloaded, total int64) {
			if name == "artifact-1.0.jar" {
				downloadedBytes.Store(downloaded)
				totalBytes.Store(total)
			}
		},
	}
	if err := mcl.Install(context.Background(), "1.20.1", dir, progress); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if downloadedBytes.Load() != 7 || totalBytes.Load() != 7 {
		t.Errorf("download progress for artifact-1.0.jar = [%d %d]; want [7 7]", downloadedBytes.Load(), totalBytes.Load())
	}

	for _, path := range []string{
		filepath.Join("libraries", "org", "example", "artifact", "1.0", "artifact-1.0.jar"),
		filepath.Join("assets", "indexes", "12.json"),
		filepath.Join("assets", "objects", assetHashHex[:2], assetHashHex),
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("the file %s was not written: %v", path, err)
		}
	}
}
