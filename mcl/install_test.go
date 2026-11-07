package mcl

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallWithOptionsSkipsChecks(t *testing.T) {
	dir := t.TempDir()
	versionPath := filepath.Join(dir, "versions", "1.20.1", "1.20.1.json")
	if err := os.MkdirAll(filepath.Dir(versionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(versionPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallWithOptions(context.Background(), "1.20.1", dir, MinecraftInstallOptions{
		SkipChecks: true,
	}); err != nil {
		t.Fatalf("InstallWithOptions: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "versions", "1.20.1", ".incomplete")); !os.IsNotExist(err) {
		t.Errorf("incomplete marker exists after skipping checks: %v", err)
	}
}

func TestWithIncomplete(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "versions", "1.20.1", ".incomplete")
	failure := errors.New("install failed")

	err := withIncomplete(dir, "1.20.1", func() error {
		if _, err := os.Stat(marker); err != nil {
			t.Fatalf("marker was not created before install: %v", err)
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("withIncomplete error = %v, want %v", err, failure)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("marker was removed after failed install: %v", err)
	}

	if err := withIncomplete(dir, "1.20.1", func() error { return nil }); err != nil {
		t.Fatalf("withIncomplete successful install: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("marker still exists after successful install: %v", err)
	}
}
