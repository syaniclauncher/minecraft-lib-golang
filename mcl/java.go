package mcl

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// javaVersionPattern matches the version in the output of "java
// -showversion".
var javaVersionPattern = regexp.MustCompile(`version "([0-9._]+)"`)

// JavaInformation returns information about the Java installation in the given
// directory.
//
// The path has to be the installation directory, so for
// /usr/lib/jvm/java-19 it is that directory and not the java executable.
//
// The java executable is run to determine the details, which is a security
// risk when the path is not trusted.
func JavaInformation(path string) (JavaInstallation, error) {
	javaPath := filepath.Join(path, "bin", "java")
	if runtime.GOOS == "windows" {
		javaPath = filepath.Join(path, "bin", "java.exe")
	}
	if info, err := os.Stat(javaPath); err != nil || info.IsDir() {
		return JavaInstallation{}, fmt.Errorf("%s was not found", javaPath)
	}

	cmd := exec.Command(javaPath, "-showversion")
	var output strings.Builder
	cmd.Stderr = &output
	cmd.Stdout = io.Discard
	// "java -showversion" without arguments exits with an error, but it
	// still prints the version.
	if err := cmd.Run(); err != nil && output.Len() == 0 {
		return JavaInstallation{}, err
	}

	text := output.String()
	version := ""
	if match := javaVersionPattern.FindStringSubmatch(text); match != nil {
		version = match[1]
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}

	information := JavaInstallation{
		Path:     path,
		Name:     filepath.Base(path),
		Version:  version,
		Is64Bit:  strings.Contains(text, "64-Bit"),
		OpenJDK:  strings.HasPrefix(text, "openjdk"),
		JavaPath: filepath.Join(absolute, "bin", "java"),
	}
	if runtime.GOOS == "windows" {
		information.JavaPath = filepath.Join(absolute, "bin", "java.exe")
		information.JavawPath = filepath.Join(absolute, "bin", "javaw.exe")
	}
	return information, nil
}

// FindSystemJavaVersions returns the Java installations that are found on the
// system. It can be used to let the user pick a Java version.
//
// macOS is not supported.
func FindSystemJavaVersions(additionalDirectories []string) []string {
	var directories []string
	switch runtime.GOOS {
	case "windows":
		directories = append(directories, `C:\Program Files (x86)\Java`, `C:\Program Files\Java`)
	case "linux":
		directories = append(directories, "/usr/lib/jvm", "/usr/lib/sdk")
	}
	directories = append(directories, additionalDirectories...)

	var installations []string
	for _, directory := range directories {
		installations = append(installations, searchJavaDirectory(directory)...)
	}
	return installations
}

// FindSystemJavaVersionsInformation returns the Java installations of the
// system together with their information.
//
// macOS is not supported.
func FindSystemJavaVersionsInformation(additionalDirectories []string) ([]JavaInstallation, error) {
	var information []JavaInstallation
	for _, installation := range FindSystemJavaVersions(additionalDirectories) {
		current, err := JavaInformation(installation)
		if err != nil {
			continue
		}
		information = append(information, current)
	}
	return information, nil
}

// searchJavaDirectory returns the Java installations inside a directory.
func searchJavaDirectory(directory string) []string {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}

	var installations []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		current := filepath.Join(directory, entry.Name())
		for _, name := range []string{"java", "java.exe"} {
			if info, err := os.Stat(filepath.Join(current, "bin", name)); err == nil && !info.IsDir() {
				installations = append(installations, current)
				break
			}
		}
	}
	return installations
}
