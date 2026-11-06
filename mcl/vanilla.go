package mcl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// vanillaProfilesFile is the launcher_profiles.json of the official
// launcher.
type vanillaProfilesFile struct {
	Profiles map[string]vanillaProfileEntry `json:"profiles"`
	Settings map[string]any                 `json:"settings"`
	Version  int                            `json:"version"`
}

// vanillaProfileEntry is a single profile of the official launcher.
type vanillaProfileEntry struct {
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	Created       string      `json:"created,omitempty"`
	LastUsed      string      `json:"lastUsed,omitempty"`
	LastVersionID string      `json:"lastVersionId"`
	GameDir       string      `json:"gameDir,omitempty"`
	JavaDir       string      `json:"javaDir,omitempty"`
	JavaArgs      *string     `json:"javaArgs,omitempty"`
	Resolution    *Resolution `json:"resolution,omitempty"`
}

// LoadVanillaLauncherProfiles loads the profiles of the official launcher
// from the given Minecraft directory.
func LoadVanillaLauncherProfiles(minecraftDirectory string) ([]VanillaLauncherProfile, error) {
	data, err := readVanillaProfiles(minecraftDirectory)
	if err != nil {
		return nil, err
	}

	var profiles []VanillaLauncherProfile
	for _, entry := range data.Profiles {
		profile := VanillaLauncherProfile{
			Name:             entry.Name,
			GameDirectory:    entry.GameDir,
			JavaExecutable:   entry.JavaDir,
			CustomResolution: entry.Resolution,
		}

		switch entry.Type {
		case VersionTypeLatestRelease:
			profile.Name = "Latest release"
		case VersionTypeLatestSnapshot:
			profile.Name = "Latest snapshot"
		}

		switch entry.LastVersionID {
		case VersionTypeLatestRelease:
			profile.VersionType = VersionTypeLatestRelease
		case VersionTypeLatestSnapshot:
			profile.VersionType = VersionTypeLatestSnapshot
		default:
			profile.VersionType = VersionTypeCustom
			profile.Version = entry.LastVersionID
		}

		if entry.JavaArgs != nil {
			profile.JavaArguments = strings.Split(*entry.JavaArgs, " ")
		}

		profiles = append(profiles, profile)
	}
	return profiles, nil
}

// VanillaLauncherProfileToOptions converts a profile of the official launcher
// into launch options. The login data still has to be added to them.
func VanillaLauncherProfileToOptions(profile VanillaLauncherProfile) (Options, error) {
	if err := validateVanillaProfile(profile); err != nil {
		return Options{}, err
	}

	var options Options
	if profile.GameDirectory != "" {
		options.GameDirectory = profile.GameDirectory
	}
	if profile.JavaExecutable != "" {
		options.ExecutablePath = profile.JavaExecutable
	}
	if profile.JavaArguments != nil {
		options.JVMArguments = profile.JavaArguments
	}
	if profile.CustomResolution != nil {
		options.CustomResolution = true
		options.ResolutionWidth = fmt.Sprint(profile.CustomResolution.Width)
		options.ResolutionHeight = fmt.Sprint(profile.CustomResolution.Height)
	}
	return options, nil
}

// VanillaLauncherProfileVersion returns the Minecraft version of a profile of
// the official launcher. It resolves the "latest-release" and
// "latest-snapshot" version types.
func VanillaLauncherProfileVersion(ctx context.Context, profile VanillaLauncherProfile) (string, error) {
	if err := validateVanillaProfile(profile); err != nil {
		return "", err
	}

	switch profile.VersionType {
	case VersionTypeLatestRelease, VersionTypeLatestSnapshot:
		latest, err := LatestVersions(ctx)
		if err != nil {
			return "", err
		}
		if profile.VersionType == VersionTypeLatestRelease {
			return latest.Release, nil
		}
		return latest.Snapshot, nil
	default:
		return profile.Version, nil
	}
}

// AddVanillaLauncherProfile adds a profile to the official launcher.
func AddVanillaLauncherProfile(minecraftDirectory string, profile VanillaLauncherProfile) error {
	if err := validateVanillaProfile(profile); err != nil {
		return err
	}

	data, err := readVanillaProfiles(minecraftDirectory)
	if err != nil {
		return err
	}

	entry := vanillaProfileEntry{
		Name:    profile.Name,
		Type:    VersionTypeCustom,
		GameDir: profile.GameDirectory,
		JavaDir: profile.JavaExecutable,
	}
	switch profile.VersionType {
	case VersionTypeLatestRelease:
		entry.LastVersionID = VersionTypeLatestRelease
	case VersionTypeLatestSnapshot:
		entry.LastVersionID = VersionTypeLatestSnapshot
	default:
		entry.LastVersionID = profile.Version
	}
	if profile.JavaArguments != nil {
		javaArgs := strings.Join(profile.JavaArguments, " ")
		entry.JavaArgs = &javaArgs
	}
	if profile.CustomResolution != nil {
		entry.Resolution = profile.CustomResolution
	}

	now := time.Now().Format(time.RFC3339)
	entry.Created = now
	entry.LastUsed = now

	key := randomUUID()
	for {
		if _, exists := data.Profiles[key]; !exists {
			break
		}
		key = randomUUID()
	}
	data.Profiles[key] = entry

	return writeVanillaProfiles(minecraftDirectory, data)
}

// VanillaLauncherProfilesExist reports whether the official launcher profiles
// are present in the given Minecraft directory.
func VanillaLauncherProfilesExist(minecraftDirectory string) bool {
	info, err := os.Stat(filepath.Join(minecraftDirectory, "launcher_profiles.json"))
	return err == nil && !info.IsDir()
}

// CreateEmptyVanillaLauncherProfiles creates a launcher_profiles.json with the
// correct structure in the given Minecraft directory. An existing file is
// overwritten.
func CreateEmptyVanillaLauncherProfiles(minecraftDirectory string) error {
	return writeVanillaProfiles(minecraftDirectory, vanillaProfilesFile{
		Profiles: map[string]vanillaProfileEntry{},
		Settings: map[string]any{},
		Version:  3,
	})
}

// EnsureVanillaLauncherProfiles creates the launcher profiles when they do not
// exist yet. Some installers need them to be present.
func EnsureVanillaLauncherProfiles(minecraftDirectory string) error {
	if VanillaLauncherProfilesExist(minecraftDirectory) {
		return nil
	}
	return CreateEmptyVanillaLauncherProfiles(minecraftDirectory)
}

// validateVanillaProfile reports whether a profile of the official launcher
// is usable.
func validateVanillaProfile(profile VanillaLauncherProfile) error {
	if profile.Name == "" {
		return fmt.Errorf("%w: the profile has no name", ErrInvalidVanillaLauncherProfile)
	}
	switch profile.VersionType {
	case VersionTypeLatestRelease, VersionTypeLatestSnapshot:
		return nil
	case VersionTypeCustom:
		if profile.Version == "" {
			return fmt.Errorf("%w: the profile has no version", ErrInvalidVanillaLauncherProfile)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown version type %q", ErrInvalidVanillaLauncherProfile, profile.VersionType)
	}
}

// readVanillaProfiles reads the launcher_profiles.json of the official
// launcher.
func readVanillaProfiles(minecraftDirectory string) (vanillaProfilesFile, error) {
	var data vanillaProfilesFile
	path := filepath.Join(minecraftDirectory, "launcher_profiles.json")
	if err := readJSONFile(path, &data); err != nil {
		return data, err
	}
	if data.Profiles == nil {
		data.Profiles = map[string]vanillaProfileEntry{}
	}
	return data, nil
}

// writeVanillaProfiles writes the launcher_profiles.json of the official
// launcher.
func writeVanillaProfiles(minecraftDirectory string, data vanillaProfilesFile) error {
	path := filepath.Join(minecraftDirectory, "launcher_profiles.json")
	encoded, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}
