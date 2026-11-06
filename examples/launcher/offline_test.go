package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The UUIDs are the ones the official launcher uses for a player that plays
// without an account. They were computed with the same formula the Python
// library uses.
func TestOfflineUUID(t *testing.T) {
	tests := map[string]string{
		"Player":  "a01e3843-e521-3998-958a-f459800e4d11",
		"xsyanic": "6b7db67b-146b-3989-a0a3-1e6d51641e36",
		"Notch":   "b50ad385-829d-3141-a216-7e7d7539ba7f",
	}
	for username, want := range tests {
		if got := offlineUUID(username); got != want {
			t.Errorf("offlineUUID(%q) = %q, want %q", username, got, want)
		}
	}
}

// The game writes its saves into the game directory, so it has to exist when
// the game starts. It can be gone by then, because somebody deleted it while
// the launcher was open.
func TestEnsureGameDirectory(t *testing.T) {
	l := &launcher{gameDirectory: filepath.Join(t.TempDir(), "game_dir")}

	if err := l.ensureGameDirectory(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(l.gameDirectory)
	if err != nil {
		t.Fatalf("the game directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("the game directory is not a directory")
	}

	// Calling it again is fine, the directory is already there.
	if err := l.ensureGameDirectory(); err != nil {
		t.Fatal(err)
	}
}
