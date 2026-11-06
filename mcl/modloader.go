package mcl

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ModLoader is a mod loader like Forge, Fabric, Quilt or NeoForge.
//
// Use GetModLoader to obtain an instance.
type ModLoader interface {
	// ID returns the identifier of the mod loader, for example "fabric".
	ID() string
	// Name returns the display name of the mod loader.
	Name() string
	// MinecraftVersions returns the Minecraft versions the mod loader
	// supports, newest first.
	MinecraftVersions(ctx context.Context, stableOnly bool) ([]string, error)
	// IsMinecraftVersionSupported reports whether the mod loader supports
	// the given Minecraft version.
	IsMinecraftVersionSupported(ctx context.Context, minecraftVersion string) (bool, error)
	// LoaderVersions returns the versions of the mod loader that exist for
	// the given Minecraft version, newest first.
	LoaderVersions(ctx context.Context, minecraftVersion string, stableOnly bool) ([]string, error)
	// LatestLoaderVersion returns the newest version of the mod loader for
	// the given Minecraft version.
	LatestLoaderVersion(ctx context.Context, minecraftVersion string) (string, error)
	// InstalledVersion returns the version id the mod loader is installed
	// under. It is the version to pass to Command.
	InstalledVersion(minecraftVersion, loaderVersion string) string
	// InstallerURL returns the URL of the installer of the mod loader.
	InstallerURL(ctx context.Context, minecraftVersion, loaderVersion string) (string, error)
	// Install installs the mod loader and returns the version id it was
	// installed under.
	Install(ctx context.Context, minecraftVersion, minecraftDirectory string, opts InstallOptions) (string, error)
}

// InstallOptions configures the installation of a mod loader.
type InstallOptions struct {
	// LoaderVersion is the version of the mod loader. The newest available
	// one is used when it is empty.
	LoaderVersion string
	// Progress receives the installation progress. It may be nil.
	Progress *Progress
	// Java is the Java executable that runs the installer of the mod
	// loader. "java" is used when it is empty.
	Java string
}

// modLoaders holds every mod loader the library knows about.
var modLoaders = []ModLoader{
	Forge,
	NeoForge,
	Fabric,
	Quilt,
}

// GetModLoader returns the mod loader with the given id.
func GetModLoader(id string) (ModLoader, error) {
	for _, loader := range modLoaders {
		if loader.ID() == id {
			return loader, nil
		}
	}
	return nil, fmt.Errorf("mod loader %q not found", id)
}

// ListModLoaders returns the ids of all available mod loaders.
func ListModLoaders() []string {
	ids := make([]string, 0, len(modLoaders))
	for _, loader := range modLoaders {
		ids = append(ids, loader.ID())
	}
	return ids
}

// installModLoader installs the Minecraft version a mod loader needs and
// afterwards the version of the mod loader itself.
//
// installLoader performs the loader specific part of the installation.
func installModLoader(
	ctx context.Context,
	loader ModLoader,
	minecraftVersion, minecraftDirectory string,
	opts InstallOptions,
	installLoader func(ctx context.Context, loaderVersion string) error,
) (string, error) {
	valid, err := IsVersionValid(ctx, minecraftVersion, minecraftDirectory)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", fmt.Errorf("%w: %s", ErrVersionNotFound, minecraftVersion)
	}

	supported, err := loader.IsMinecraftVersionSupported(ctx, minecraftVersion)
	if err != nil {
		return "", err
	}
	if !supported {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedVersion, minecraftVersion)
	}

	loaderVersion := opts.LoaderVersion
	if loaderVersion == "" {
		loaderVersion, err = loader.LatestLoaderVersion(ctx, minecraftVersion)
		if err != nil {
			return "", err
		}
	}

	if err := Install(ctx, minecraftVersion, minecraftDirectory, opts.Progress); err != nil {
		return "", err
	}

	if err := installLoader(ctx, loaderVersion); err != nil {
		return "", err
	}

	installedVersion := loader.InstalledVersion(minecraftVersion, loaderVersion)
	if err := Install(ctx, installedVersion, minecraftDirectory, opts.Progress); err != nil {
		return "", err
	}
	return installedVersion, nil
}

// mavenMetadata is the content of a maven-metadata.xml file.
type mavenMetadata struct {
	// Release is the latest release version.
	Release string
	// Latest is the latest version, which may be a snapshot.
	Latest string
	// Versions lists every version.
	Versions []string
}

// parseMavenMetadata downloads and parses a maven-metadata.xml file. The
// format is simple enough to be parsed without an XML parser.
func parseMavenMetadata(ctx context.Context, url string) (mavenMetadata, error) {
	text, err := cachedText(ctx, url)
	if err != nil {
		return mavenMetadata{}, err
	}
	return mavenMetadata{
		Release:  matchXMLTag(text, "release"),
		Latest:   matchXMLTag(text, "latest"),
		Versions: matchXMLTags(text, "version"),
	}, nil
}

// matchXMLTag returns the content of the first tag with the given name.
func matchXMLTag(text, name string) string {
	re := regexp.MustCompile(`(?s)<` + name + `>(.*?)</` + name + `>`)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	return match[1]
}

// matchXMLTags returns the content of every tag with the given name.
func matchXMLTags(text, name string) []string {
	re := regexp.MustCompile(`(?s)<` + name + `>(.*?)</` + name + `>`)
	var values []string
	for _, match := range re.FindAllStringSubmatch(text, -1) {
		values = append(values, match[1])
	}
	return values
}

// versionChunk is a part of a version string: either a number or a piece of
// text.
type versionChunk struct {
	isNumber bool
	number   int
	text     string
}

// versionNumberPattern matches the numbers of a version.
var versionNumberPattern = regexp.MustCompile(`\d+`)

// versionChunks splits a version into its numbers and text parts.
func versionChunks(version string) []versionChunk {
	var chunks []versionChunk
	last := 0
	for _, location := range versionNumberPattern.FindAllStringIndex(version, -1) {
		if location[0] > last {
			chunks = append(chunks, versionChunk{text: strings.ToLower(version[last:location[0]])})
		}
		number, _ := strconv.Atoi(version[location[0]:location[1]])
		chunks = append(chunks, versionChunk{isNumber: true, number: number})
		last = location[1]
	}
	if last < len(version) {
		chunks = append(chunks, versionChunk{text: strings.ToLower(version[last:])})
	}
	return chunks
}

// compareVersions compares two version strings so that they sort naturally:
// 1.21.11 sorts before 1.21.10.
func compareVersions(a, b string) int {
	chunksA, chunksB := versionChunks(a), versionChunks(b)
	for i := 0; i < len(chunksA) && i < len(chunksB); i++ {
		x, y := chunksA[i], chunksB[i]
		switch {
		case x.isNumber && y.isNumber:
			if x.number != y.number {
				return compareInt(x.number, y.number)
			}
		case x.isNumber != y.isNumber:
			// Numbers sort after text.
			if x.isNumber {
				return 1
			}
			return -1
		default:
			if x.text != y.text {
				return strings.Compare(x.text, y.text)
			}
		}
	}
	return compareInt(len(chunksA), len(chunksB))
}

// compareInt compares two integers.
func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// sortVersionsNewestFirst sorts version strings so that the newest one comes
// first.
func sortVersionsNewestFirst(versions []string) {
	sort.SliceStable(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})
}
