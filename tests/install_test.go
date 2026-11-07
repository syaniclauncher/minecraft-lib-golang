package mcl_test

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

// writeNativeJar creates a jar with the given files.
func writeNativeJar(t *testing.T, path string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	archive, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	for name, content := range files {
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
}

// nativesVersion returns the client.json of a version with a library that has
// natives for every platform.
func nativesVersion(native string) string {
	return fmt.Sprintf(`{
		"id": "1.20.1",
		"type": "release",
		"libraries": [{
			"name": "org.example:example:1.0",
			"natives": {"windows": %[1]q, "osx": %[1]q, "linux": %[1]q},
			"extract": {"exclude": ["META-INF/"]}
		}]
	}`, native)
}

// nativeJarPath returns the path of the native jar of the library of
// nativesVersion.
func nativeJarPath(minecraftDirectory, native string) string {
	return filepath.Join(minecraftDirectory, "libraries", "org", "example", "example", "1.0", "example-1.0-"+native+".jar")
}

func TestExtractNatives(t *testing.T) {
	dir := t.TempDir()
	native := "natives-example"
	writeNativeJar(t, nativeJarPath(dir, native), map[string]string{
		"libexample.so":        "library",
		"META-INF/MANIFEST.MF": "manifest",
	})
	writeFile(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.json"), nativesVersion(native))

	target := filepath.Join(dir, "natives")
	if err := mcl.ExtractNatives("1.20.1", dir, target); err != nil {
		t.Fatalf("ExtractNatives: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "libexample.so")); err != nil {
		t.Errorf("the native library was not extracted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "META-INF", "MANIFEST.MF")); !os.IsNotExist(err) {
		t.Error("the excluded file should not be extracted")
	}
}

func TestExtractNativesRefusesToEscape(t *testing.T) {
	dir := t.TempDir()
	native := "natives-example"
	writeNativeJar(t, nativeJarPath(dir, native), map[string]string{
		"../escaped.txt": "evil",
	})
	writeFile(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.json"), nativesVersion(native))

	target := filepath.Join(dir, "natives")
	if err := mcl.ExtractNatives("1.20.1", dir, target); err == nil {
		t.Error("the extraction should refuse to write outside of the directory")
	}
	if _, err := os.Stat(filepath.Join(dir, "escaped.txt")); !os.IsNotExist(err) {
		t.Error("the file should not have been written")
	}
}

func TestExtractNativesOfUnknownVersion(t *testing.T) {
	if err := mcl.ExtractNatives("1.2.3", t.TempDir(), t.TempDir()); err == nil {
		t.Error("ExtractNatives should fail for an unknown version")
	}
}
