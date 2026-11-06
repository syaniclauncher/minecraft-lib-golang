package main

import (
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
