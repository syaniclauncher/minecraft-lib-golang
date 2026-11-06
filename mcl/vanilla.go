package mcl

import (
	"bytes"
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
	Profiles vanillaProfiles `json:"profiles"`
	Settings map[string]any  `json:"settings"`
	Version  int             `json:"version"`
}

// vanillaProfiles holds the profiles of the official launcher.
//
// It keeps the order of the file and the fields the library does not know,
// so that adding a profile does not change the rest of the file.
type vanillaProfiles struct {
	keys   []string
	values []map[string]any
}

// UnmarshalJSON reads the profiles in the order of the file.
func (p *vanillaProfiles) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return fmt.Errorf("the profiles are not a JSON object")
	}

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("the name of a profile is not a string")
		}
		var value map[string]any
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		p.keys = append(p.keys, key)
		p.values = append(p.values, value)
	}
	return nil
}

// MarshalJSON writes the profiles in the order they were read.
func (p vanillaProfiles) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteByte('{')
	for i, key := range p.keys {
		if i > 0 {
			buffer.WriteByte(',')
		}
		name, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buffer.Write(name)
		buffer.WriteByte(':')
		value, err := json.Marshal(p.values[i])
		if err != nil {
			return nil, err
		}
		buffer.Write(value)
	}
	buffer.WriteByte('}')
	return buffer.Bytes(), nil
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
	for _, raw := range data.Profiles.values {
		encoded, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		var entry vanillaProfileEntry
		if err := json.Unmarshal(encoded, &entry); err != nil {
			return nil, err
		}

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
//
// The profiles that are already there keep their fields and their order.
func AddVanillaLauncherProfile(minecraftDirectory string, profile VanillaLauncherProfile) error {
	if err := validateVanillaProfile(profile); err != nil {
		return err
	}

	data, err := readVanillaProfiles(minecraftDirectory)
	if err != nil {
		return err
	}

	lastVersionID := profile.Version
	switch profile.VersionType {
	case VersionTypeLatestRelease:
		lastVersionID = VersionTypeLatestRelease
	case VersionTypeLatestSnapshot:
		lastVersionID = VersionTypeLatestSnapshot
	}

	now := time.Now().Format(time.RFC3339)
	entry := map[string]any{
		"name":          profile.Name,
		"type":          VersionTypeCustom,
		"created":       now,
		"lastUsed":      now,
		"lastVersionId": lastVersionID,
	}
	if profile.GameDirectory != "" {
		entry["gameDir"] = profile.GameDirectory
	}
	if profile.JavaExecutable != "" {
		entry["javaDir"] = profile.JavaExecutable
	}
	if profile.JavaArguments != nil {
		entry["javaArgs"] = strings.Join(profile.JavaArguments, " ")
	}
	if profile.CustomResolution != nil {
		entry["resolution"] = map[string]any{
			"height": profile.CustomResolution.Height,
			"width":  profile.CustomResolution.Width,
		}
	}

	key := randomUUID()
	for {
		if !data.Profiles.has(key) {
			break
		}
		key = randomUUID()
	}
	data.Profiles.add(key, entry)

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

// has reports whether a profile with the given name exists.
func (p *vanillaProfiles) has(key string) bool {
	for _, existing := range p.keys {
		if existing == key {
			return true
		}
	}
	return false
}

// add appends a profile.
func (p *vanillaProfiles) add(key string, value map[string]any) {
	p.keys = append(p.keys, key)
	p.values = append(p.values, value)
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
