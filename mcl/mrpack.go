package mcl

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// mrpackIndex is the modrinth.index.json of a modpack.
type mrpackIndex struct {
	FormatVersion int    `json:"formatVersion"`
	Game          string `json:"game"`
	VersionID     string `json:"versionId"`
	Name          string `json:"name"`
	Summary       string `json:"summary"`
	Dependencies  struct {
		Minecraft    string `json:"minecraft"`
		Forge        string `json:"forge"`
		NeoForge     string `json:"neoforge"`
		FabricLoader string `json:"fabric-loader"`
		QuiltLoader  string `json:"quilt-loader"`
	} `json:"dependencies"`
	Files []mrpackFile `json:"files"`
}

// mrpackFile is a single file of a modpack.
type mrpackFile struct {
	Path   string `json:"path"`
	Hashes struct {
		SHA1   string `json:"sha1"`
		SHA512 string `json:"sha512"`
	} `json:"hashes"`
	Env *struct {
		Client string `json:"client"`
		Server string `json:"server"`
	} `json:"env"`
	Downloads []string `json:"downloads"`
	FileSize  int64    `json:"fileSize"`
}

// GetMrpackInformation returns information about the modpack in the given
// .mrpack file.
func GetMrpackInformation(path string) (MrpackInformation, error) {
	index, err := readMrpackIndex(path)
	if err != nil {
		return MrpackInformation{}, err
	}

	information := MrpackInformation{
		Name:             index.Name,
		Summary:          index.Summary,
		VersionID:        index.VersionID,
		FormatVersion:    index.FormatVersion,
		MinecraftVersion: index.Dependencies.Minecraft,
	}
	for _, file := range index.Files {
		if file.Env != nil && file.Env.Client == "optional" {
			information.OptionalFiles = append(information.OptionalFiles, file.Path)
		}
	}
	return information, nil
}

// MrpackLaunchVersion returns the version that has to be passed to Command to
// launch the modpack.
func MrpackLaunchVersion(path string) (string, error) {
	index, err := readMrpackIndex(path)
	if err != nil {
		return "", err
	}

	switch {
	case index.Dependencies.Forge != "":
		return index.Dependencies.Minecraft + "-forge-" + index.Dependencies.Forge, nil
	case index.Dependencies.NeoForge != "":
		return "neoforge-" + index.Dependencies.NeoForge, nil
	case index.Dependencies.FabricLoader != "":
		return "fabric-loader-" + index.Dependencies.FabricLoader + "-" + index.Dependencies.Minecraft, nil
	case index.Dependencies.QuiltLoader != "":
		return "quilt-loader-" + index.Dependencies.QuiltLoader + "-" + index.Dependencies.Minecraft, nil
	default:
		return index.Dependencies.Minecraft, nil
	}
}

// InstallMrpack installs the modpack in the given .mrpack file.
//
// The modpack is installed into the Minecraft directory, or into
// modpackDirectory when it is not empty.
func InstallMrpack(
	ctx context.Context,
	path, minecraftDirectory, modpackDirectory string,
	opts MrpackInstallOptions,
	progress *Progress,
) error {
	absoluteMinecraftDirectory, err := filepath.Abs(minecraftDirectory)
	if err != nil {
		return err
	}
	if modpackDirectory == "" {
		modpackDirectory = absoluteMinecraftDirectory
	}
	if modpackDirectory, err = filepath.Abs(modpackDirectory); err != nil {
		return err
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer reader.Close()

	index, err := readMrpackIndexFromZip(reader)
	if err != nil {
		return err
	}

	progress.setStatus("Downloading mrpack files")
	files := filterMrpackFiles(index.Files, opts)
	progress.setMax(len(files))

	if err := runParallel(downloadWorkers(), len(files), func(i int) error {
		file := files[i]
		target := filepath.Join(modpackDirectory, filepath.FromSlash(file.Path))
		if err := checkPathInsideMinecraftDirectory(modpackDirectory, target); err != nil {
			return err
		}
		if len(file.Downloads) == 0 {
			return nil
		}
		_, err := downloadFile(ctx, file.Downloads[0], target, file.Hashes.SHA1, modpackDirectory, false, progress)
		return err
	}); err != nil {
		return err
	}

	progress.setStatus("Extracting overrides")
	for _, file := range reader.File {
		var name string
		switch {
		case strings.HasPrefix(file.Name, "client-overrides/"):
			name = strings.TrimPrefix(file.Name, "client-overrides/")
		case strings.HasPrefix(file.Name, "overrides/"):
			name = strings.TrimPrefix(file.Name, "overrides/")
		default:
			continue
		}
		if file.FileInfo().IsDir() {
			continue
		}

		target := filepath.Join(modpackDirectory, filepath.FromSlash(name))
		if err := checkPathInsideMinecraftDirectory(modpackDirectory, target); err != nil {
			return err
		}
		progress.setStatus("Extract " + file.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := writeZipFile(file, target); err != nil {
			return err
		}
	}

	if opts.SkipDependenciesInstall {
		return nil
	}

	minecraftVersion := index.Dependencies.Minecraft
	progress.setStatus("Installing Minecraft " + minecraftVersion)
	if err := Install(ctx, minecraftVersion, absoluteMinecraftDirectory, progress); err != nil {
		return err
	}

	loaders := []struct {
		id      string
		version string
	}{
		{"forge", index.Dependencies.Forge},
		{"neoforge", index.Dependencies.NeoForge},
		{"fabric", index.Dependencies.FabricLoader},
		{"quilt", index.Dependencies.QuiltLoader},
	}
	for _, loader := range loaders {
		if loader.version == "" {
			continue
		}
		progress.setStatus(fmt.Sprintf("Installing %s %s for Minecraft %s", loader.id, loader.version, minecraftVersion))

		modLoader, err := GetModLoader(loader.id)
		if err != nil {
			return err
		}
		_, err = modLoader.Install(ctx, minecraftVersion, absoluteMinecraftDirectory, InstallOptions{
			LoaderVersion: loader.version,
			Progress:      progress,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// filterMrpackFiles returns the files of a modpack that have to be installed.
func filterMrpackFiles(files []mrpackFile, opts MrpackInstallOptions) []mrpackFile {
	var filtered []mrpackFile
	for _, file := range files {
		if file.Env == nil {
			filtered = append(filtered, file)
			continue
		}
		switch file.Env.Client {
		case "required":
			filtered = append(filtered, file)
		case "optional":
			if containsString(opts.OptionalFiles, file.Path) {
				filtered = append(filtered, file)
			}
		}
	}
	return filtered
}

// readMrpackIndex reads the index of the modpack in the given file.
func readMrpackIndex(path string) (mrpackIndex, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return mrpackIndex{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer reader.Close()
	return readMrpackIndexFromZip(reader)
}

// readMrpackIndexFromZip reads the index of a modpack from an open archive.
func readMrpackIndexFromZip(reader *zip.ReadCloser) (mrpackIndex, error) {
	var index mrpackIndex
	for _, file := range reader.File {
		if file.Name != "modrinth.index.json" {
			continue
		}
		data, err := readZipFile(file)
		if err != nil {
			return index, err
		}
		if err := json.Unmarshal(data, &index); err != nil {
			return index, fmt.Errorf("parse modrinth.index.json: %w", err)
		}
		return index, nil
	}
	return index, fmt.Errorf("the modpack does not contain a modrinth.index.json")
}
