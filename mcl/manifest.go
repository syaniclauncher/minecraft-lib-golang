package mcl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ManifestURL is Mojang's version manifest. It lists every version that can
// be downloaded.
const ManifestURL = "https://launchermeta.mojang.com/mc/game/version_manifest_v2.json"

// ManifestVersion is one entry of the version manifest.
type ManifestVersion struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	URL             string `json:"url"`
	Time            string `json:"time"`
	ReleaseTime     string `json:"releaseTime"`
	SHA1            string `json:"sha1"`
	ComplianceLevel int    `json:"complianceLevel"`
}

// Manifest is Mojang's version manifest.
type VersionManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []ManifestVersion `json:"versions"`
}

// ClientJSON is the version metadata ("client.json") of a Minecraft version.
type ClientJSON struct {
	// ID is the version identifier.
	ID string `json:"id"`
	// Jar is the name of the jar file, when it differs from ID.
	Jar string `json:"jar,omitempty"`
	// InheritsFrom is the version this one is based on.
	InheritsFrom string `json:"inheritsFrom,omitempty"`
	// MainClass is the entry point of the game.
	MainClass string `json:"mainClass"`
	// Type is "release" or "snapshot".
	Type string `json:"type"`
	// Assets is the name of the asset index.
	Assets string `json:"assets,omitempty"`
	// AssetIndex points at the asset index file.
	AssetIndex *AssetIndex `json:"assetIndex,omitempty"`
	// Downloads holds the downloadable files, keyed by "client", "server",
	// "client_mappings" and "server_mappings".
	Downloads map[string]Download `json:"downloads,omitempty"`
	// Libraries are the libraries that make up the classpath.
	Libraries []Library `json:"libraries"`
	// JavaVersion describes the Java version the game needs.
	JavaVersion *JavaVersion `json:"javaVersion,omitempty"`
	// Logging holds the log4j configuration, keyed by "client".
	Logging map[string]Logging `json:"logging,omitempty"`
	// Arguments holds the game and JVM arguments of newer versions.
	Arguments *Arguments `json:"arguments,omitempty"`
	// MinecraftArguments holds the arguments of versions older than 1.13.
	MinecraftArguments string `json:"minecraftArguments,omitempty"`
	// ReleaseTime is when the version was released, in ISO 8601 format.
	ReleaseTime string `json:"releaseTime,omitempty"`
	// Time is the last modification time, in ISO 8601 format.
	Time string `json:"time,omitempty"`
	// ComplianceLevel is the classification level of the version.
	ComplianceLevel int `json:"complianceLevel,omitempty"`
	// MinimumLauncherVersion is the oldest launcher that supports the
	// version.
	MinimumLauncherVersion int `json:"minimumLauncherVersion,omitempty"`
}

// Download describes a downloadable file.
type Download struct {
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

// AssetIndex points at the index of the assets of a version.
type AssetIndex struct {
	ID        string `json:"id"`
	SHA1      string `json:"sha1"`
	Size      int64  `json:"size"`
	TotalSize int64  `json:"totalSize"`
	URL       string `json:"url"`
}

// JavaVersion describes the Java version a Minecraft version requires.
type JavaVersion struct {
	Component    string `json:"component"`
	MajorVersion int    `json:"majorVersion"`
}

// Logging holds the log4j configuration of a version.
type Logging struct {
	Argument string      `json:"argument"`
	File     LoggingFile `json:"file"`
	Type     string      `json:"type"`
}

// LoggingFile describes the log configuration file.
type LoggingFile struct {
	ID   string `json:"id"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

// Library is one entry of the classpath.
type Library struct {
	// Name is the Maven coordinate, for example
	// "org.lwjgl:lwjgl:3.3.1".
	Name string `json:"name"`
	// Downloads holds the artifact and the native classifiers.
	Downloads *LibraryDownloads `json:"downloads,omitempty"`
	// Extract describes which files of a native jar must not be extracted.
	Extract *Extract `json:"extract,omitempty"`
	// Rules restrict the platforms the library is used on.
	Rules []Rule `json:"rules,omitempty"`
	// Natives maps an operating system to the classifier of the native
	// jar.
	Natives map[string]string `json:"natives,omitempty"`
	// URL overrides the Maven repository the library is downloaded from.
	URL string `json:"url,omitempty"`
}

// LibraryDownloads holds the downloadable files of a library.
type LibraryDownloads struct {
	Artifact    *LibraryArtifact           `json:"artifact,omitempty"`
	Classifiers map[string]LibraryArtifact `json:"classifiers,omitempty"`
}

// LibraryArtifact describes a downloadable library file.
type LibraryArtifact struct {
	Path string `json:"path"`
	URL  string `json:"url"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
}

// Extract lists the files that must not be extracted from a native jar.
type Extract struct {
	Exclude []string `json:"exclude"`
}

// Rule restricts a library or an argument to certain platforms and features.
type Rule struct {
	// Action is "allow" or "disallow".
	Action string `json:"action"`
	// OS restricts the rule to an operating system.
	OS *OSRule `json:"os,omitempty"`
	// Features restricts the rule to launch options.
	Features map[string]bool `json:"features,omitempty"`
}

// OSRule restricts a rule to an operating system, an architecture or an OS
// version.
type OSRule struct {
	// Name is "windows", "osx" or "linux".
	Name string `json:"name,omitempty"`
	// Arch is "x86" or "x86_64".
	Arch string `json:"arch,omitempty"`
	// Version is a regular expression matched against the OS version.
	Version string `json:"version,omitempty"`
}

// Arguments holds the arguments of a version, split into game and JVM
// arguments.
type Arguments struct {
	Game []Argument `json:"game"`
	JVM  []Argument `json:"jvm"`
}

// Argument is a single argument of a version. It is either a plain string or
// a list of strings guarded by rules.
type Argument struct {
	// Value holds the argument strings.
	Value []string
	// Rules restrict when the argument applies.
	Rules []Rule
	// CompatibilityRules restrict when the argument applies.
	CompatibilityRules []Rule
}

// UnmarshalJSON accepts both a plain string and an object with a "value"
// field.
func (a *Argument) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		a.Value = []string{s}
		return nil
	}

	var raw struct {
		Rules              []Rule          `json:"rules"`
		CompatibilityRules []Rule          `json:"compatibilityRules"`
		Value              json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	a.Rules = raw.Rules
	a.CompatibilityRules = raw.CompatibilityRules

	var single string
	if err := json.Unmarshal(raw.Value, &single); err == nil {
		a.Value = []string{single}
		return nil
	}
	return json.Unmarshal(raw.Value, &a.Value)
}

// Manifest fetches Mojang's version manifest. The result is cached for an
// hour, so calling this repeatedly does not hammer the endpoint.
func Manifest(ctx context.Context) (*VersionManifest, error) {
	var m VersionManifest
	if err := cachedJSON(ctx, ManifestURL, &m); err != nil {
		return nil, fmt.Errorf("fetch version manifest: %w", err)
	}
	return &m, nil
}

// LatestVersions returns the latest release and snapshot versions.
func LatestVersions(ctx context.Context) (Latest, error) {
	m, err := Manifest(ctx)
	if err != nil {
		return Latest{}, err
	}
	return Latest{Release: m.Latest.Release, Snapshot: m.Latest.Snapshot}, nil
}

// VersionList returns every version Mojang offers for download.
func VersionList(ctx context.Context) ([]VersionInfo, error) {
	m, err := Manifest(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]VersionInfo, 0, len(m.Versions))
	for _, v := range m.Versions {
		list = append(list, VersionInfo{
			ID:              v.ID,
			Type:            v.Type,
			ReleaseTime:     parseISOTime(v.ReleaseTime),
			ComplianceLevel: v.ComplianceLevel,
		})
	}
	return list, nil
}

// InstalledVersions returns the versions that are installed in the given
// Minecraft directory.
func InstalledVersions(minecraftDirectory string) ([]VersionInfo, error) {
	entries, err := os.ReadDir(filepath.Join(minecraftDirectory, "versions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read versions directory: %w", err)
	}

	var list []VersionInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := readClientJSONFile(minecraftDirectory, entry.Name())
		if err != nil {
			// A directory without a readable client.json is not a
			// version.
			continue
		}
		releaseTime := parseISOTime(data.ReleaseTime)
		if releaseTime.IsZero() {
			releaseTime = time.Unix(0, 0)
		}
		list = append(list, VersionInfo{
			ID:              data.ID,
			Type:            data.Type,
			ReleaseTime:     releaseTime,
			ComplianceLevel: data.ComplianceLevel,
		})
	}
	return list, nil
}

// AvailableVersions returns the installed versions plus every version Mojang
// offers for download.
func AvailableVersions(ctx context.Context, minecraftDirectory string) ([]VersionInfo, error) {
	list, err := VersionList(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(list))
	for _, v := range list {
		seen[v.ID] = true
	}

	installed, err := InstalledVersions(minecraftDirectory)
	if err != nil {
		return nil, err
	}
	for _, v := range installed {
		if !seen[v.ID] {
			list = append(list, v)
		}
	}
	return list, nil
}

// IsVersionValid reports whether the given version is installed or offered
// for download by Mojang.
func IsVersionValid(ctx context.Context, version, minecraftDirectory string) (bool, error) {
	if info, err := os.Stat(filepath.Join(minecraftDirectory, "versions", version)); err == nil && info.IsDir() {
		return true, nil
	}
	list, err := VersionList(ctx)
	if err != nil {
		return false, err
	}
	for _, v := range list {
		if v.ID == version {
			return true, nil
		}
	}
	return false, nil
}

// IsVanillaVersion reports whether the given version is a vanilla version
// offered by Mojang.
func IsVanillaVersion(ctx context.Context, version string) (bool, error) {
	list, err := VersionList(ctx)
	if err != nil {
		return false, err
	}
	for _, v := range list {
		if v.ID == version {
			return true, nil
		}
	}
	return false, nil
}

// IsMinecraftInstalled reports whether the given directory holds a Minecraft
// installation.
func IsMinecraftInstalled(minecraftDirectory string) bool {
	for _, dir := range []string{"versions", "libraries", "assets"} {
		info, err := os.Stat(filepath.Join(minecraftDirectory, dir))
		if err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

func parseISOTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}
