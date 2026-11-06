package mcl

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DefaultMinecraftDirectory returns the default path of the .minecraft
// directory of the current user.
func DefaultMinecraftDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, ".minecraft")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "minecraft")
	default:
		return filepath.Join(home, ".minecraft")
	}
}

// DefaultJavaExecutable returns the path of the Java executable of the
// system. It falls back to "java" when no installation is found.
func DefaultJavaExecutable() string {
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		if runtime.GOOS == "windows" {
			return filepath.Join(javaHome, "bin", "javaw.exe")
		}
		return filepath.Join(javaHome, "bin", "java")
	}

	switch runtime.GOOS {
	case "windows":
		oracle := `C:\Program Files (x86)\Common Files\Oracle\Java\javapath\javaw.exe`
		if _, err := os.Stat(oracle); err == nil {
			return oracle
		}
		if path, err := exec.LookPath("javaw"); err == nil {
			return path
		}
		return "javaw"
	case "linux":
		if target, err := os.Readlink("/etc/alternatives/java"); err == nil {
			return target
		}
		if target, err := os.Readlink("/usr/lib/jvm/default-runtime"); err == nil {
			return filepath.Join("/usr/lib/jvm", target, "bin", "java")
		}
	}

	if path, err := exec.LookPath("java"); err == nil {
		return path
	}
	return "java"
}

// IsPlatformSupported reports whether the current platform is supported by
// the library.
func IsPlatformSupported() bool {
	switch runtime.GOOS {
	case "windows", "darwin", "linux":
		return true
	default:
		return false
	}
}
