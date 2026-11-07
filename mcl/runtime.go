package mcl

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// AdoptiumURL is the Adoptium API the Java runtimes are downloaded from.
const AdoptiumURL = "https://api.adoptium.net/v3/assets/latest/%d/hotspot"

// JVMManifestURL is Mojang's list of the Java runtimes that ship with the
// game.
const JVMManifestURL = "https://launchermeta.mojang.com/v1/products/java-runtime/2ec0cc96c44e5a76b9c8b7c39df7210883d12871/all.json"

// JVMRuntimeInformation describes a Java runtime of Mojang.
type JVMRuntime struct {
	// Name is the name of the runtime, for example "Java 17".
	Name string
	// Released is when the runtime was released.
	Released time.Time
}

// VersionRuntimeInformation describes the Java runtime a Minecraft version
// uses.
type VersionRuntime struct {
	// Name is the name of the runtime component.
	Name string
	// JavaMajorVersion is the Java major version the game needs.
	JavaMajorVersion int
}

// jvmManifest is Mojang's list of Java runtimes.
type jvmManifest map[string]map[string][]struct {
	Manifest struct {
		URL string `json:"url"`
	} `json:"manifest"`
	Version struct {
		Name     string `json:"name"`
		Released string `json:"released"`
	} `json:"version"`
}

// InstallJVMRuntime installs a Temurin (Eclipse Adoptium) JRE for the given
// Java major version into the Minecraft directory.
//
// The runtime is keyed by its Java major version, so 8, 17 or 21. An already
// installed runtime is kept.
func InstallJVMRuntime(ctx context.Context, majorVersion int, minecraftDirectory string, progress *Progress) error {
	if executable := TemurinExecutablePath(majorVersion, minecraftDirectory); executable != "" {
		return nil
	}

	osName, architecture := adoptiumPlatform()
	progress.setStatus(fmt.Sprintf("Fetching Temurin JRE %d", majorVersion))

	url := fmt.Sprintf(AdoptiumURL, majorVersion)
	text, err := cachedText(ctx, url+"?"+adoptiumQuery(osName, architecture))
	if err != nil {
		return err
	}
	var assets []struct {
		Binary struct {
			Package struct {
				Link     string `json:"link"`
				Name     string `json:"name"`
				Checksum string `json:"checksum"`
			} `json:"package"`
		} `json:"binary"`
	}
	if err := json.Unmarshal([]byte(text), &assets); err != nil {
		return fmt.Errorf("parse %s: %w", url, err)
	}
	if len(assets) == 0 {
		return fmt.Errorf("%w: temurin-%d", ErrVersionNotFound, majorVersion)
	}
	pkg := assets[0].Binary.Package

	basePath := filepath.Join(minecraftDirectory, "runtime", fmt.Sprintf("temurin-%d", majorVersion), jvmPlatformString())
	archivePath := filepath.Join(basePath, "_temurin.zip")
	if !strings.HasSuffix(pkg.Name, ".zip") {
		archivePath = filepath.Join(basePath, "_temurin.tar.gz")
	}
	if err := checkPathInsideMinecraftDirectory(minecraftDirectory, archivePath); err != nil {
		return err
	}

	// The archive is always fetched again, the checksum below makes sure
	// that a broken download cannot slip through.
	if _, err := downloadFile(ctx, pkg.Link, archivePath, "", minecraftDirectory, true, progress, fmt.Sprintf("Temurin Java %d", majorVersion)); err != nil {
		return err
	}
	checksum, err := sha256Hash(archivePath)
	if err != nil {
		return err
	}
	if checksum != pkg.Checksum {
		os.Remove(archivePath)
		return &InvalidChecksumError{URL: pkg.Link, Path: archivePath, Expected: pkg.Checksum, Actual: checksum}
	}

	progress.setStatus(fmt.Sprintf("Extracting Temurin JRE %d", majorVersion))
	root, err := extractRuntimeArchive(archivePath, basePath, minecraftDirectory, strings.HasSuffix(pkg.Name, ".zip"))
	if err != nil {
		return err
	}
	if err := os.Remove(archivePath); err != nil {
		return err
	}

	markerPath := filepath.Join(basePath, ".temurin")
	if err := checkPathInsideMinecraftDirectory(minecraftDirectory, markerPath); err != nil {
		return err
	}
	return os.WriteFile(markerPath, []byte(root), 0o644)
}

// TemurinExecutablePath returns the path of the java executable of a Temurin
// runtime that was installed with InstallJVMRuntime. It returns an empty
// string when no runtime is installed.
func TemurinExecutablePath(majorVersion int, minecraftDirectory string) string {
	basePath := filepath.Join(minecraftDirectory, "runtime", fmt.Sprintf("temurin-%d", majorVersion), jvmPlatformString())
	data, err := os.ReadFile(filepath.Join(basePath, ".temurin"))
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(string(data))

	candidates := []string{
		filepath.Join(basePath, root, "bin", "java"),
		filepath.Join(basePath, root, "Contents", "Home", "bin", "java"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		if info, err := os.Stat(candidate + ".exe"); err == nil && !info.IsDir() {
			return candidate + ".exe"
		}
	}
	return ""
}

// JVMRuntimes returns the names of the Java runtimes Mojang offers for the
// current platform.
func JVMRuntimes(ctx context.Context) ([]string, error) {
	manifest, err := jvmManifestFor(ctx)
	if err != nil {
		return nil, err
	}
	var runtimes []string
	for name := range manifest[jvmPlatformString()] {
		runtimes = append(runtimes, name)
	}
	return runtimes, nil
}

// InstalledJVMRuntimes returns the names of the Java runtimes that are
// installed in the given Minecraft directory.
func InstalledJVMRuntimes(minecraftDirectory string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(minecraftDirectory, "runtime"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var runtimes []string
	for _, entry := range entries {
		runtimes = append(runtimes, entry.Name())
	}
	return runtimes, nil
}

// JVMRuntimeInformation returns information about a Java runtime of Mojang.
func JVMRuntimeInformation(ctx context.Context, jvmVersion string) (JVMRuntime, error) {
	manifest, err := jvmManifestFor(ctx)
	if err != nil {
		return JVMRuntime{}, err
	}
	versions, ok := manifest[jvmPlatformString()][jvmVersion]
	if !ok {
		return JVMRuntime{}, fmt.Errorf("%w: %s", ErrVersionNotFound, jvmVersion)
	}
	if len(versions) == 0 {
		return JVMRuntime{}, fmt.Errorf("%w: %s", ErrPlatformNotSupported, jvmVersion)
	}
	return JVMRuntime{
		Name:     versions[0].Version.Name,
		Released: parseISOTime(versions[0].Version.Released),
	}, nil
}

// VersionRuntimeInformation returns the Java runtime a Minecraft version
// uses. It returns nil when the version does not need a specific runtime.
func VersionRuntimeInformation(version, minecraftDirectory string) (*VersionRuntime, error) {
	data, err := VersionMetadata(context.Background(), version, minecraftDirectory)
	if err != nil {
		return nil, err
	}
	if data.JavaVersion == nil {
		return nil, nil
	}
	return &VersionRuntime{
		Name:             data.JavaVersion.Component,
		JavaMajorVersion: data.JavaVersion.MajorVersion,
	}, nil
}

func jvmManifestFor(ctx context.Context) (jvmManifest, error) {
	var manifest jvmManifest
	if err := cachedJSON(ctx, JVMManifestURL, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

// jvmPlatformString returns the name Mojang uses for the current platform.
func jvmPlatformString() string {
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

// adoptiumPlatform returns the operating system and architecture the
// Adoptium API knows this machine by.
func adoptiumPlatform() (osName, architecture string) {
	switch runtime.GOOS {
	case "windows":
		osName = "windows"
	case "darwin":
		osName = "mac"
	default:
		osName = "linux"
	}

	switch runtime.GOARCH {
	case "amd64", "x86_64":
		architecture = "x64"
	case "arm64", "aarch64":
		architecture = "aarch64"
	case "386", "i386", "i686":
		architecture = "x86"
	case "arm":
		architecture = "arm"
	default:
		architecture = "x64"
	}
	return osName, architecture
}

// adoptiumQuery builds the query string of an Adoptium request.
func adoptiumQuery(osName, architecture string) string {
	return "os=" + osName + "&architecture=" + architecture + "&image_type=jre"
}

// extractRuntimeArchive unpacks a Temurin archive and returns the name of
// its single top level directory.
func extractRuntimeArchive(archivePath, dest, minecraftDirectory string, isZip bool) (string, error) {
	tops := map[string]bool{}

	if isZip {
		reader, err := zip.OpenReader(archivePath)
		if err != nil {
			return "", err
		}
		defer reader.Close()

		for _, file := range reader.File {
			if err := checkPathInsideMinecraftDirectory(minecraftDirectory, filepath.Join(dest, filepath.FromSlash(file.Name))); err != nil {
				return "", err
			}
			if top := strings.SplitN(file.Name, "/", 2)[0]; top != "" {
				tops[top] = true
			}
		}
		for _, file := range reader.File {
			if err := extractZipFile(file, dest); err != nil {
				return "", err
			}
		}
	} else {
		file, err := os.Open(archivePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		gz, err := gzip.NewReader(file)
		if err != nil {
			return "", err
		}
		defer gz.Close()

		reader := tar.NewReader(gz)
		for {
			header, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			target := filepath.Join(dest, filepath.FromSlash(header.Name))
			if err := checkPathInsideMinecraftDirectory(minecraftDirectory, target); err != nil {
				return "", err
			}
			if top := strings.SplitN(header.Name, "/", 2)[0]; top != "" {
				tops[top] = true
			}
			if err := writeTarEntry(header, reader, target); err != nil {
				return "", err
			}
		}
	}

	if len(tops) != 1 {
		return "", fmt.Errorf("unexpected Temurin archive layout: %v", tops)
	}
	for top := range tops {
		return top, nil
	}
	return "", fmt.Errorf("unexpected Temurin archive layout: %v", tops)
}

// writeTarEntry writes a single entry of a tar archive to disk.
func writeTarEntry(header *tar.Header, reader *tar.Reader, target string) error {
	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(target, 0o755)
	case tar.TypeSymlink:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		os.Remove(target)
		return os.Symlink(header.Linkname, target)
	case tar.TypeReg:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		mode := header.FileInfo().Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, reader); err != nil {
			return err
		}
		return out.Close()
	default:
		return nil
	}
}

// sha256Hash returns the SHA-256 checksum of a file.
func sha256Hash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
