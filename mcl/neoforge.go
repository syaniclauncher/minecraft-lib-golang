package mcl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// NeoForgeMavenURL is the Maven repository of NeoForge.
const NeoForgeMavenURL = "https://maven.neoforged.net/releases/net/neoforged/neoforge"

// NeoForgeAPIURL is the API that lists the released versions of NeoForge.
const NeoForgeAPIURL = "https://maven.neoforged.net/api/maven/versions/releases/net/neoforged/neoforge"

// NeoForge is the NeoForge mod loader.
var NeoForge ModLoader = neoForgeModLoader{}

// neoForgeModLoader implements the NeoForge mod loader.
type neoForgeModLoader struct{}

// neoForgeVersionPattern matches the Minecraft part of a NeoForge version.
var neoForgeVersionPattern = regexp.MustCompile(`^\d+\.\d+`)

func (neoForgeModLoader) ID() string   { return "neoforge" }
func (neoForgeModLoader) Name() string { return "NeoForge" }

// normalizeMinecraftVersion turns the version part of a NeoForge version into
// a normal Minecraft version.
func normalizeMinecraftVersion(minecraftVersion string) string {
	minecraftVersion = strings.TrimSuffix(minecraftVersion, ".0")

	// Old versions are 1.x, for example NeoForge 21.1 belongs to Minecraft
	// 1.21.1. Minecraft 1.21.11 was the last of that scheme, newer
	// versions are year based and already complete.
	major, err := strconv.Atoi(strings.Split(minecraftVersion, ".")[0])
	if err != nil {
		return minecraftVersion
	}
	if major >= 25 {
		return minecraftVersion
	}
	return "1." + minecraftVersion
}

// neoforgeVersions returns every released version of NeoForge.
func (l neoForgeModLoader) neoforgeVersions(ctx context.Context) ([]string, error) {
	var response struct {
		Versions []string `json:"versions"`
	}
	if err := cachedJSON(ctx, NeoForgeAPIURL, &response); err != nil {
		return nil, err
	}
	return response.Versions, nil
}

// MinecraftVersions returns the Minecraft versions NeoForge supports, newest
// first.
func (l neoForgeModLoader) MinecraftVersions(ctx context.Context, stableOnly bool) ([]string, error) {
	versions, err := l.neoforgeVersions(ctx)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var minecraftVersions []string
	for _, version := range versions {
		if strings.HasPrefix(version, "0.") {
			continue
		}
		match := neoForgeVersionPattern.FindString(version)
		if match == "" {
			continue
		}
		minecraftVersion := normalizeMinecraftVersion(match)
		if seen[minecraftVersion] {
			continue
		}
		seen[minecraftVersion] = true
		minecraftVersions = append(minecraftVersions, minecraftVersion)
	}

	sortVersionsNewestFirst(minecraftVersions)
	return minecraftVersions, nil
}

// LoaderVersions returns the versions of NeoForge for a Minecraft version,
// newest first.
func (l neoForgeModLoader) LoaderVersions(ctx context.Context, minecraftVersion string, stableOnly bool) ([]string, error) {
	versions, err := l.neoforgeVersions(ctx)
	if err != nil {
		return nil, err
	}

	var loaderVersions []string
	for _, version := range versions {
		if strings.Contains(version, "beta") && stableOnly {
			continue
		}
		match := neoForgeVersionPattern.FindString(version)
		if match == "" {
			continue
		}
		if normalizeMinecraftVersion(match) == minecraftVersion {
			loaderVersions = append(loaderVersions, version)
		}
	}

	sortVersionsNewestFirst(loaderVersions)
	return loaderVersions, nil
}

// IsMinecraftVersionSupported reports whether NeoForge supports the given
// Minecraft version.
func (l neoForgeModLoader) IsMinecraftVersionSupported(ctx context.Context, minecraftVersion string) (bool, error) {
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

// LatestLoaderVersion returns the newest version of NeoForge for a Minecraft
// version.
func (l neoForgeModLoader) LatestLoaderVersion(ctx context.Context, minecraftVersion string) (string, error) {
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

// InstalledVersion returns the version id NeoForge is installed under.
func (l neoForgeModLoader) InstalledVersion(minecraftVersion, loaderVersion string) string {
	return "neoforge-" + loaderVersion
}

// InstallerURL returns the URL of the installer of NeoForge.
func (l neoForgeModLoader) InstallerURL(ctx context.Context, minecraftVersion, loaderVersion string) (string, error) {
	return fmt.Sprintf("%s/%s/neoforge-%s-installer.jar", NeoForgeMavenURL, loaderVersion, loaderVersion), nil
}

// Install installs NeoForge.
func (l neoForgeModLoader) Install(ctx context.Context, minecraftVersion, minecraftDirectory string, opts InstallOptions) (string, error) {
	return installModLoader(ctx, l, minecraftVersion, minecraftDirectory, opts,
		func(ctx context.Context, loaderVersion string) error {
			return l.installNeoForge(ctx, minecraftVersion, loaderVersion, minecraftDirectory, opts)
		})
}

// installNeoForge downloads the NeoForge installer and runs it.
func (l neoForgeModLoader) installNeoForge(ctx context.Context, minecraftVersion, loaderVersion, minecraftDirectory string, opts InstallOptions) error {
	installerURL, err := l.InstallerURL(ctx, minecraftVersion, loaderVersion)
	if err != nil {
		return err
	}

	// Some installers need the launcher profiles to be present.
	if err := EnsureVanillaLauncherProfiles(minecraftDirectory); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "minecraft-launcher-lib-go-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	installerPath := filepath.Join(tempDir, "neoforge-installer.jar")
	if _, err := downloadFile(ctx, installerURL, installerPath, "", "", true, opts.Progress); err != nil {
		return err
	}

	opts.Progress.setStatus("Running installer")
	command := []string{javaExecutable(opts), "-jar", installerPath, "--install-client", minecraftDirectory}
	return runCommand(ctx, command, tempDir)
}
