package mcl

import (
	"archive/zip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSplitLibraryName(t *testing.T) {
	group, artifact, version, ok := splitLibraryName("org.ow2.asm:asm:9.7.1")
	if !ok {
		t.Fatal("the name should be valid")
	}
	if group != "org.ow2.asm" || artifact != "asm" || version != "9.7.1" {
		t.Errorf("got %s %s %s", group, artifact, version)
	}

	if _, _, _, ok := splitLibraryName("org.ow2.asm:asm"); ok {
		t.Error("a name without a version should be refused")
	}
}

func TestNativeClassifier(t *testing.T) {
	lib := Library{Natives: map[string]string{
		"windows": "natives-windows-${arch}",
		"linux":   "natives-linux",
		"osx":     "natives-macos",
	}}

	classifier := nativeClassifier(lib)
	if classifier == "" {
		t.Fatal("the library should have natives on this platform")
	}

	switch currentOSName() {
	case "windows":
		if classifier != "natives-windows-64" && classifier != "natives-windows-32" {
			t.Errorf("the classifier is %q", classifier)
		}
	case "osx":
		if classifier != "natives-macos" {
			t.Errorf("the classifier is %q", classifier)
		}
	default:
		if classifier != "natives-linux" {
			t.Errorf("the classifier is %q", classifier)
		}
	}

	if nativeClassifier(Library{}) != "" {
		t.Error("a library without natives should not have a classifier")
	}
	if nativeClassifier(Library{Natives: map[string]string{"plan9": "natives"}}) != "" {
		t.Error("a library without natives for this platform should not have a classifier")
	}
}

func TestInstallLibrary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("library"))
	}))
	defer server.Close()

	dir := t.TempDir()
	lib := Library{
		Name: "org.example:example:1.0",
		URL:  server.URL,
		Downloads: &LibraryDownloads{
			Artifact: &LibraryArtifact{
				Path: "org/example/other/1.0/other-1.0.jar",
				URL:  server.URL + "/artifact",
				SHA1: "",
			},
		},
	}

	if err := installLibrary(context.Background(), "1.20.1", lib, dir, nil); err != nil {
		t.Fatalf("installLibrary: %v", err)
	}

	for _, path := range []string{
		filepath.Join("libraries", "org", "example", "example", "1.0", "example-1.0.jar"),
		filepath.Join("libraries", "org", "example", "other", "1.0", "other-1.0.jar"),
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("the file %s was not written: %v", path, err)
		}
	}
}

func TestExtractNativesFile(t *testing.T) {
	dir := t.TempDir()
	jarPath := filepath.Join(dir, "natives.jar")

	archive, err := os.Create(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	for name, content := range map[string]string{
		"libexample.so":        "library",
		"META-INF/MANIFEST.MF": "manifest",
	} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "natives")
	extract := &Extract{Exclude: []string{"META-INF/"}}
	if err := extractNativesFile(jarPath, target, extract); err != nil {
		t.Fatalf("extractNativesFile: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "libexample.so")); err != nil {
		t.Errorf("the native library was not extracted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "META-INF", "MANIFEST.MF")); !os.IsNotExist(err) {
		t.Error("the excluded file should not be extracted")
	}
}

func TestExtractNativesFileRefusesToEscape(t *testing.T) {
	dir := t.TempDir()
	jarPath := filepath.Join(dir, "evil.jar")

	archive, err := os.Create(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	entry, err := writer.Create("../escaped.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("evil")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "natives")
	if err := extractNativesFile(jarPath, target, nil); err == nil {
		t.Error("the extraction should refuse to write outside of the directory")
	}
	if _, err := os.Stat(filepath.Join(dir, "escaped.txt")); !os.IsNotExist(err) {
		t.Error("the file should not have been written")
	}
}

func TestJarMainClass(t *testing.T) {
	dir := t.TempDir()
	jarPath := filepath.Join(dir, "library.jar")

	archive, err := os.Create(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	entry, err := writer.Create("META-INF/MANIFEST.MF")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("Manifest-Version: 1.0\nMain-Class: net.example.Main\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	mainClass, err := jarMainClass(jarPath)
	if err != nil {
		t.Fatalf("jarMainClass: %v", err)
	}
	if mainClass != "net.example.Main" {
		t.Errorf("the main class is %q", mainClass)
	}
}
