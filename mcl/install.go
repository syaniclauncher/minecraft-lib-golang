package mcl

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// ResourcesURL is the server that hosts the assets of the game.
var ResourcesURL = "https://resources.download.minecraft.net"

// LibrariesURL is the default Maven repository the libraries are downloaded
// from.
var LibrariesURL = "https://libraries.minecraft.net"

// Install installs a Minecraft version into the given directory.
//
// It verifies and repairs an existing installation, so only missing or
// corrupted files are downloaded. The version must either be installed
// already or be offered by Mojang.
//
// An .incomplete marker is kept in the version directory until installation
// succeeds, so interrupted installations are not reported as installed.
//
// The progress argument may be nil.
func Install(ctx context.Context, version, minecraftDirectory string, progress *Progress) error {
	versionPath := filepath.Join(minecraftDirectory, "versions", version, version+".json")
	if _, err := os.Stat(versionPath); err == nil {
		return withIncomplete(minecraftDirectory, version, func() error {
			return installVersion(ctx, version, minecraftDirectory, "", "", progress)
		})
	}

	manifest, err := Manifest(ctx)
	if err != nil {
		return err
	}
	for _, v := range manifest.Versions {
		if v.ID == version {
			return withIncomplete(minecraftDirectory, version, func() error {
				return installVersion(ctx, version, minecraftDirectory, v.URL, v.SHA1, progress)
			})
		}
	}
	return fmt.Errorf("%w: %s", ErrVersionNotFound, version)
}

func withIncomplete(minecraftDirectory, version string, install func() error) error {
	markerPath := filepath.Join(minecraftDirectory, "versions", version, ".incomplete")
	if err := checkPathInsideMinecraftDirectory(minecraftDirectory, markerPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(markerPath, nil, 0o644); err != nil {
		return err
	}
	if err := install(); err != nil {
		return err
	}
	return os.Remove(markerPath)
}

// installVersion installs a single version. When url is not empty the
// version metadata is downloaded first.
func installVersion(ctx context.Context, versionID, minecraftDirectory, url, sha1sum string, progress *Progress) error {
	if url != "" {
		metaPath := filepath.Join(minecraftDirectory, "versions", versionID, versionID+".json")
		if _, err := downloadFile(ctx, url, metaPath, sha1sum, minecraftDirectory, false, progress); err != nil {
			return err
		}
	}

	raw, err := readRawClientJSON(minecraftDirectory, versionID)
	if err != nil {
		return err
	}
	if parent, ok := raw["inheritsFrom"].(string); ok && parent != "" {
		// The base version has to be present before the inheritance
		// can be resolved.
		if err := Install(ctx, parent, minecraftDirectory, progress); err != nil && !errors.Is(err, ErrVersionNotFound) {
			return err
		}
	}

	data, err := loadClientJSON(minecraftDirectory, versionID)
	if err != nil {
		return err
	}

	if err := installLibraries(ctx, data.ID, data.Libraries, minecraftDirectory, progress); err != nil {
		return err
	}
	if err := installAssets(ctx, data, minecraftDirectory, progress); err != nil {
		return err
	}

	if client, ok := data.Logging["client"]; ok && client.File.URL != "" {
		logPath := filepath.Join(minecraftDirectory, "assets", "log_configs", client.File.ID)
		if _, err := downloadFile(ctx, client.File.URL, logPath, client.File.SHA1, minecraftDirectory, false, progress); err != nil {
			return err
		}
	}

	jarPath := filepath.Join(minecraftDirectory, "versions", data.ID, data.ID+".jar")
	if client, ok := data.Downloads["client"]; ok && client.URL != "" {
		if _, err := downloadFile(ctx, client.URL, jarPath, client.SHA1, minecraftDirectory, false, progress); err != nil {
			return err
		}
	}

	// Old Forge versions ship no jar of their own and use the one of the
	// base version.
	if _, err := os.Stat(jarPath); os.IsNotExist(err) && data.InheritsFrom != "" {
		inherited := filepath.Join(minecraftDirectory, "versions", data.InheritsFrom, data.InheritsFrom+".jar")
		if err := checkPathInsideMinecraftDirectory(minecraftDirectory, inherited); err != nil {
			return err
		}
		if err := copyFile(inherited, jarPath); err != nil {
			return err
		}
	}

	majorVersion := 8
	if data.JavaVersion != nil {
		majorVersion = data.JavaVersion.MajorVersion
	}
	progress.setStatus("Installing java runtime")
	if err := InstallJVMRuntime(ctx, majorVersion, minecraftDirectory, progress); err != nil {
		return err
	}

	progress.setStatus("Installation complete")
	return nil
}

// installLibraries downloads every library of a version.
func installLibraries(ctx context.Context, versionID string, libraries []Library, minecraftDirectory string, progress *Progress) error {
	progress.setStatus("Downloading Libraries")
	progress.setMax(len(libraries))

	var mu sync.Mutex
	done := 0
	count := func() {
		mu.Lock()
		done++
		mu.Unlock()
		progress.setDone(done)
	}

	return runParallel(downloadWorkers(), len(libraries), func(i int) error {
		lib := libraries[i]
		if len(lib.Rules) > 0 && !ruleListAllows(lib.Rules, Options{}) {
			count()
			return nil
		}
		if err := installLibrary(ctx, versionID, lib, minecraftDirectory, progress); err != nil {
			return err
		}
		count()
		return nil
	})
}

// installLibrary downloads a single library and extracts its natives.
func installLibrary(ctx context.Context, versionID string, lib Library, minecraftDirectory string, progress *Progress) error {
	groupID, name, version, ok := splitLibraryName(lib.Name)
	if !ok {
		return nil
	}

	currentPath := filepath.Join(minecraftDirectory, "libraries")
	downloadURL := strings.TrimSuffix(lib.URL, "/")
	if downloadURL == "" {
		downloadURL = LibrariesURL
	}
	for _, part := range strings.Split(groupID, ".") {
		currentPath = filepath.Join(currentPath, part)
		downloadURL += "/" + part
	}

	fileExtension := "jar"
	if parts := strings.Split(version, "@"); len(parts) == 2 {
		version, fileExtension = parts[0], parts[1]
	}

	jarFileName := name + "-" + version + "." + fileExtension
	downloadURL += "/" + name + "/" + version + "/" + jarFileName
	currentPath = filepath.Join(currentPath, name, version)

	native := nativeClassifier(lib)

	// The library itself is best effort: some repositories do not have it
	// and the artifact download below is the authoritative one.
	_, _ = downloadFile(ctx, downloadURL, filepath.Join(currentPath, jarFileName), "", minecraftDirectory, false, progress)

	if lib.Downloads == nil {
		if lib.Extract != nil && native != "" {
			nativeFileName := name + "-" + version + "-" + native + ".jar"
			return extractNativesFile(
				filepath.Join(currentPath, nativeFileName),
				filepath.Join(minecraftDirectory, "versions", versionID, "natives"),
				lib.Extract)
		}
		return nil
	}

	if artifact := lib.Downloads.Artifact; artifact != nil && artifact.URL != "" && artifact.Path != "" {
		target := filepath.Join(minecraftDirectory, "libraries", filepath.FromSlash(artifact.Path))
		if _, err := downloadFile(ctx, artifact.URL, target, artifact.SHA1, minecraftDirectory, false, progress); err != nil {
			var httpError *HTTPError
			if !errors.As(err, &httpError) {
				return err
			}
			// Some libraries are not available at every address.
			// The game may still work without them.
		}
	}

	if native != "" {
		classifier, ok := lib.Downloads.Classifiers[native]
		if !ok {
			return nil
		}
		nativeFileName := name + "-" + version + "-" + native + ".jar"
		nativePath := filepath.Join(currentPath, nativeFileName)
		if _, err := downloadFile(ctx, classifier.URL, nativePath, classifier.SHA1, minecraftDirectory, false, progress); err != nil {
			return err
		}
		return extractNativesFile(
			nativePath,
			filepath.Join(minecraftDirectory, "versions", versionID, "natives"),
			lib.Extract)
	}
	return nil
}

// assetsIndex is the index of the assets of a version.
type assetsIndex struct {
	Objects map[string]struct {
		Hash string `json:"hash"`
		Size int64  `json:"size"`
	} `json:"objects"`
}

// installAssets downloads the assets of a version.
func installAssets(ctx context.Context, data *ClientJSON, minecraftDirectory string, progress *Progress) error {
	// Old versions do not have assets.
	if data.AssetIndex == nil {
		return nil
	}

	progress.setStatus("Downloading Assets")
	indexPath := filepath.Join(minecraftDirectory, "assets", "indexes", data.Assets+".json")
	if _, err := downloadFile(ctx, data.AssetIndex.URL, indexPath, data.AssetIndex.SHA1, minecraftDirectory, false, progress, "Assets"); err != nil {
		return err
	}

	var index assetsIndex
	if err := readJSONFile(indexPath, &index); err != nil {
		return err
	}

	seen := map[string]bool{}
	var hashes []string
	for _, object := range index.Objects {
		// The hash is the name of the file and the first two
		// characters are the directory it lives in.
		if len(object.Hash) < 2 || seen[object.Hash] {
			continue
		}
		seen[object.Hash] = true
		hashes = append(hashes, object.Hash)
	}

	progress.setMax(len(hashes))
	var mu sync.Mutex
	done := 0
	return runParallel(downloadWorkers(), len(hashes), func(i int) error {
		hash := hashes[i]
		target := filepath.Join(minecraftDirectory, "assets", "objects", hash[:2], hash)
		if _, err := downloadFile(ctx, ResourcesURL+"/"+hash[:2]+"/"+hash, target, hash, minecraftDirectory, false, progress, "Assets"); err != nil {
			return err
		}
		mu.Lock()
		done++
		mu.Unlock()
		progress.setDone(done)
		return nil
	})
}

// ExtractNatives extracts the native libraries of an installed version into
// the given directory. The natives are also extracted while installing, so
// this is only needed for versions that were installed by other software.
func ExtractNatives(version, minecraftDirectory, extractPath string) error {
	if _, err := os.Stat(filepath.Join(minecraftDirectory, "versions", version, version+".json")); err != nil {
		return fmt.Errorf("%w: %s", ErrVersionNotFound, version)
	}

	data, err := loadClientJSON(minecraftDirectory, version)
	if err != nil {
		return err
	}

	for _, lib := range data.Libraries {
		if len(lib.Rules) > 0 && !ruleListAllows(lib.Rules, Options{}) {
			continue
		}
		native := nativeClassifier(lib)
		if native == "" {
			continue
		}
		libPath := libraryPath(lib.Name, minecraftDirectory)
		extension := filepath.Ext(libPath)
		nativePath := strings.TrimSuffix(libPath, extension) + "-" + native + extension
		if err := extractNativesFile(nativePath, extractPath, lib.Extract); err != nil {
			return err
		}
	}
	return nil
}

// extractNativesFile unpacks the native libraries of a jar into a directory.
func extractNativesFile(jarPath, extractPath string, extract *Extract) error {
	if err := os.MkdirAll(extractPath, 0o755); err != nil {
		return err
	}

	reader, err := zip.OpenReader(jarPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if extract != nil {
			excluded := false
			for _, prefix := range extract.Exclude {
				if strings.HasPrefix(file.Name, prefix) {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}
		if err := extractZipFile(file, extractPath); err != nil {
			return err
		}
	}
	return nil
}

// extractZipFile writes a single file of a zip archive into a directory. It
// refuses to write outside of that directory.
func extractZipFile(file *zip.File, extractPath string) error {
	target := filepath.Join(extractPath, filepath.FromSlash(file.Name))
	if err := checkPathInsideMinecraftDirectory(extractPath, target); err != nil {
		return err
	}
	return writeZipFile(file, target)
}

// writeZipFile writes a single file of a zip archive to an exact path.
func writeZipFile(file *zip.File, target string) error {
	if file.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	source, err := file.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	mode := file.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}
	destination, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}
	return destination.Close()
}

// nativeClassifier returns the classifier of the native jar of a library for
// the current platform, for example "natives-windows". It returns an empty
// string when the library has no natives.
func nativeClassifier(lib Library) string {
	if len(lib.Natives) == 0 {
		return ""
	}
	arch := "64"
	if strconv.IntSize == 32 {
		arch = "32"
	}

	key := "linux"
	switch runtime.GOOS {
	case "windows":
		key = "windows"
	case "darwin":
		key = "osx"
	}

	value, ok := lib.Natives[key]
	if !ok {
		return ""
	}
	return strings.ReplaceAll(value, "${arch}", arch)
}

// libraryPath returns the path of a library inside the libraries directory.
func libraryPath(name, minecraftDirectory string) string {
	suffix := "jar"
	if index := strings.Index(name, "@"); index >= 0 {
		name, suffix = name[:index], name[index+1:]
	}

	parts := strings.Split(name, ":")
	if len(parts) < 3 {
		return ""
	}
	groupID, libName, version := parts[0], parts[1], parts[2]

	libPath := filepath.Join(minecraftDirectory, "libraries")
	for _, part := range strings.Split(groupID, ".") {
		libPath = filepath.Join(libPath, part)
	}

	fileName := fmt.Sprintf("%s-%s", libName, version)
	for _, extra := range parts[3:] {
		fileName += "-" + extra
	}
	fileName += "." + suffix

	return filepath.Join(libPath, libName, version, fileName)
}

// splitLibraryName splits a Maven coordinate into its group, artifact and
// version.
func splitLibraryName(name string) (groupID, artifact, version string, ok bool) {
	parts := strings.Split(name, ":")
	if len(parts) < 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// downloadWorkers returns the number of concurrent downloads.
func downloadWorkers() int {
	workers := runtime.NumCPU() + 4
	if workers > 32 {
		workers = 32
	}
	if workers < 1 {
		workers = 1
	}
	return workers
}

// readJSONFile reads and decodes a JSON file.
func readJSONFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// copyFile copies a file, creating the target directory when needed.
func copyFile(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
