package mcl_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

// writeFile creates a file with the given content, including its parent
// directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// currentOS returns the name the version metadata uses for the current
// operating system.
func currentOS() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "osx"
	default:
		return "linux"
	}
}

// otherOS returns the name of an operating system that is not the one the
// test runs on.
func otherOS() string {
	if currentOS() == "linux" {
		return "windows"
	}
	return "linux"
}

func TestCommandAppliesTheRulesOfTheLibraries(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.json"), fmt.Sprintf(`{
		"id": "1.20.1",
		"type": "release",
		"assets": "12",
		"mainClass": "net.minecraft.client.main.Main",
		"javaVersion": {"component": "java-runtime-gamma", "majorVersion": 17},
		"libraries": [
			{"name": "org.example:alpha:1.0", "rules": [{"action": "allow", "os": {"name": %q}}]},
			{"name": "org.example:beta:1.0", "rules": [{"action": "disallow", "os": {"name": %q}}]},
			{"name": "org.example:gamma:1.0", "rules": [{"action": "allow", "os": {"name": %q}}]},
			{"name": "org.example:delta:1.0", "rules": [{"action": "disallow", "os": {"name": %q}}]}
		]
	}`, currentOS(), currentOS(), otherOS(), otherOS()))

	command, err := mcl.Command("1.20.1", dir, mcl.Options{Username: "Player"})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}

	classpath := ""
	for i, arg := range command {
		if arg == "-cp" && i+1 < len(command) {
			classpath = command[i+1]
		}
	}

	if !strings.Contains(classpath, filepath.Join("org", "example", "alpha", "1.0", "alpha-1.0.jar")) {
		t.Errorf("the allowed library is missing from the classpath: %s", classpath)
	}
	if strings.Contains(classpath, filepath.Join("org", "example", "beta", "1.0", "beta-1.0.jar")) {
		t.Errorf("the disallowed library is in the classpath: %s", classpath)
	}
	if strings.Contains(classpath, filepath.Join("org", "example", "gamma", "1.0", "gamma-1.0.jar")) {
		t.Errorf("the library of another platform is in the classpath: %s", classpath)
	}
	if !strings.Contains(classpath, filepath.Join("org", "example", "delta", "1.0", "delta-1.0.jar")) {
		t.Errorf("the library of another platform is missing from the classpath: %s", classpath)
	}
}
