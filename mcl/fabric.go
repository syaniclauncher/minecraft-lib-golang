package mcl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fabricQuiltModLoader implements the mod loaders that are built the same
// way: Fabric and Quilt.
type fabricQuiltModLoader struct {
	id         string
	name       string
	mavenURL   string
	gameURL    string
	loaderURL  string
	loaderName string
}

// Fabric is the Fabric mod loader.
var Fabric ModLoader = fabricQuiltModLoader{
	id:         "fabric",
	name:       "Fabric",
	mavenURL:   "https://maven.fabricmc.net/net/fabricmc/fabric-installer",
	gameURL:    "https://meta.fabricmc.net/v2/versions/game",
	loaderURL:  "https://meta.fabricmc.net/v2/versions/loader",
	loaderName: "fabric",
}

// Quilt is the Quilt mod loader.
var Quilt ModLoader = fabricQuiltModLoader{
	id:         "quilt",
	name:       "Quilt",
	mavenURL:   "https://maven.quiltmc.org/repository/release/org/quiltmc/quilt-installer",
	gameURL:    "https://meta.quiltmc.org/v3/versions/game",
	loaderURL:  "https://meta.quiltmc.org/v3/versions/loader",
	loaderName: "quilt",
}

// metaVersion is an entry of the version lists of the meta API of Fabric and
// Quilt.
type metaVersion struct {
	Version string `json:"version"`
	Stable  *bool  `json:"stable"`
}

// isStable reports whether the version is a stable one. Versions without the
// field are stable.
func (v metaVersion) isStable() bool {
	return v.Stable == nil || *v.Stable
}

func (l fabricQuiltModLoader) ID() string   { return l.id }
func (l fabricQuiltModLoader) Name() string { return l.name }

// MinecraftVersions returns the Minecraft versions the loader supports.
func (l fabricQuiltModLoader) MinecraftVersions(ctx context.Context, stableOnly bool) ([]string, error) {
	var entries []metaVersion
	if err := cachedJSON(ctx, l.gameURL, &entries); err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if !entry.isStable() && stableOnly {
			continue
		}
		versions = append(versions, entry.Version)
	}
	return versions, nil
}

// LoaderVersions returns the versions of the loader for a Minecraft version.
func (l fabricQuiltModLoader) LoaderVersions(ctx context.Context, minecraftVersion string, stableOnly bool) ([]string, error) {
	var entries []metaVersion
	if err := cachedJSON(ctx, l.loaderURL, &entries); err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if (!entry.isStable() || strings.Contains(entry.Version, "beta")) && stableOnly {
			continue
		}
		versions = append(versions, entry.Version)
	}
	return versions, nil
}

// IsMinecraftVersionSupported reports whether the loader supports the given
// Minecraft version.
func (l fabricQuiltModLoader) IsMinecraftVersionSupported(ctx context.Context, minecraftVersion string) (bool, error) {
	versions, err := l.MinecraftVersions(ctx, false)
	if err != nil {
		return false, err
	}
	for _, version := range versions {
		if version == minecraftVersion {
			return true, nil
		}
	}
	return false, nil
}

// LatestLoaderVersion returns the newest version of the loader.
func (l fabricQuiltModLoader) LatestLoaderVersion(ctx context.Context, minecraftVersion string) (string, error) {
	versions, err := l.LoaderVersions(ctx, minecraftVersion, true)
	if err != nil {
		return "", err
	}
	if len(versions) > 0 {
		return versions[0], nil
	}
	versions, err = l.LoaderVersions(ctx, minecraftVersion, false)
	if err != nil {
		return "", err
	}
	if len(versions) > 0 {
		return versions[0], nil
	}
	return "", fmt.Errorf("%w: %s", ErrUnsupportedVersion, minecraftVersion)
}

// InstalledVersion returns the version id the loader is installed under.
func (l fabricQuiltModLoader) InstalledVersion(minecraftVersion, loaderVersion string) string {
	return fmt.Sprintf("%s-loader-%s-%s", l.loaderName, loaderVersion, minecraftVersion)
}

// InstallerURL returns the URL of the installer of the loader.
func (l fabricQuiltModLoader) InstallerURL(ctx context.Context, minecraftVersion, loaderVersion string) (string, error) {
	metadata, err := parseMavenMetadata(ctx, l.mavenURL+"/maven-metadata.xml")
	if err != nil {
		return "", err
	}
	installerVersion := metadata.Latest
	return fmt.Sprintf("%s/%s/%s-installer-%s.jar", l.mavenURL, installerVersion, l.loaderName, installerVersion), nil
}

// Install installs the loader.
func (l fabricQuiltModLoader) Install(ctx context.Context, minecraftVersion, minecraftDirectory string, opts InstallOptions) (string, error) {
	return installModLoader(ctx, l, minecraftVersion, minecraftDirectory, opts,
		func(ctx context.Context, loaderVersion string) error {
			return l.installProfile(ctx, minecraftVersion, loaderVersion, minecraftDirectory, opts)
		})
}

// installProfile writes the version profile of the loader. The meta API of
// the loader serves the same document the installer would write.
func (l fabricQuiltModLoader) installProfile(ctx context.Context, minecraftVersion, loaderVersion, minecraftDirectory string, opts InstallOptions) error {
	opts.Progress.setStatus("Installing " + l.name)

	profileURL := fmt.Sprintf("%s/%s/%s/profile/json", l.loaderURL, minecraftVersion, loaderVersion)
	var profile map[string]any
	if err := cachedJSON(ctx, profileURL, &profile); err != nil {
		return err
	}

	installedVersion := l.InstalledVersion(minecraftVersion, loaderVersion)
	profile["id"] = installedVersion

	versionPath := filepath.Join(minecraftDirectory, "versions", installedVersion, installedVersion+".json")
	if err := checkPathInsideMinecraftDirectory(minecraftDirectory, versionPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(versionPath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(profile, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(versionPath, data, 0o644)
}
