package mcl

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

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
