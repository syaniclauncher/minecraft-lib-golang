package mcl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const testLauncherProfiles = `{
    "profiles": {
        "a0b1c2": {
            "name": "Latest release",
            "type": "latest-release",
            "created": "2024-01-01T00:00:00+00:00",
            "lastUsed": "2024-01-02T00:00:00+00:00",
            "lastVersionId": "latest-release",
            "gameDir": "/games/minecraft",
            "javaDir": "/usr/lib/jvm/java-21",
            "javaArgs": "-Xmx2G -Xms512M",
            "resolution": {"height": 1080, "width": 1920}
        },
        "d3e4f5": {
            "name": "Modded",
            "type": "custom",
            "created": "2024-01-01T00:00:00+00:00",
            "lastUsed": "2024-01-02T00:00:00+00:00",
            "lastVersionId": "1.20.1-forge-47.1.0"
        }
    },
    "settings": {},
    "version": 3
}`

func TestLoadVanillaLauncherProfiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "launcher_profiles.json"), testLauncherProfiles)

	profiles, err := LoadVanillaLauncherProfiles(dir)
	if err != nil {
		t.Fatalf("LoadVanillaLauncherProfiles: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2", len(profiles))
	}

	byName := map[string]VanillaLauncherProfile{}
	for _, profile := range profiles {
		byName[profile.Name] = profile
	}

	release, ok := byName["Latest release"]
	if !ok {
		t.Fatalf("the profiles are %v", profiles)
	}
	if release.VersionType != VersionTypeLatestRelease {
		t.Errorf("the version type is %q", release.VersionType)
	}
	if release.GameDirectory != "/games/minecraft" {
		t.Errorf("the game directory is %q", release.GameDirectory)
	}
	if release.JavaExecutable != "/usr/lib/jvm/java-21" {
		t.Errorf("the java executable is %q", release.JavaExecutable)
	}
	if len(release.JavaArguments) != 2 || release.JavaArguments[0] != "-Xmx2G" {
		t.Errorf("the java arguments are %v", release.JavaArguments)
	}
	if release.CustomResolution == nil || release.CustomResolution.Width != 1920 || release.CustomResolution.Height != 1080 {
		t.Errorf("the resolution is %v", release.CustomResolution)
	}

	modded, ok := byName["Modded"]
	if !ok {
		t.Fatalf("the profiles are %v", profiles)
	}
	if modded.VersionType != VersionTypeCustom || modded.Version != "1.20.1-forge-47.1.0" {
		t.Errorf("the version is %q of the type %q", modded.Version, modded.VersionType)
	}
}

func TestVanillaLauncherProfileToOptions(t *testing.T) {
	profile := VanillaLauncherProfile{
		Name:             "Latest release",
		VersionType:      VersionTypeLatestRelease,
		GameDirectory:    "/games/minecraft",
		JavaExecutable:   "/usr/lib/jvm/java-21",
		JavaArguments:    []string{"-Xmx2G"},
		CustomResolution: &Resolution{Width: 1920, Height: 1080},
	}

	options, err := VanillaLauncherProfileToOptions(profile)
	if err != nil {
		t.Fatalf("VanillaLauncherProfileToOptions: %v", err)
	}
	if options.GameDirectory != "/games/minecraft" {
		t.Errorf("the game directory is %q", options.GameDirectory)
	}
	if options.ExecutablePath != "/usr/lib/jvm/java-21" {
		t.Errorf("the java executable is %q", options.ExecutablePath)
	}
	if len(options.JVMArguments) != 1 {
		t.Errorf("the jvm arguments are %v", options.JVMArguments)
	}
	if !options.CustomResolution || options.ResolutionWidth != "1920" || options.ResolutionHeight != "1080" {
		t.Errorf("the resolution is %v", options.CustomResolution)
	}
}

func TestVanillaLauncherProfileToOptionsRejectsInvalidProfiles(t *testing.T) {
	if _, err := VanillaLauncherProfileToOptions(VanillaLauncherProfile{Name: "x"}); err == nil {
		t.Error("a profile without a version type should be refused")
	}
	if _, err := VanillaLauncherProfileToOptions(VanillaLauncherProfile{Name: "x", VersionType: VersionTypeCustom}); err == nil {
		t.Error("a custom profile without a version should be refused")
	}
	if _, err := VanillaLauncherProfileToOptions(VanillaLauncherProfile{VersionType: VersionTypeCustom, Version: "1.20.1"}); err == nil {
		t.Error("a profile without a name should be refused")
	}
}

func TestVanillaLauncherProfileVersion(t *testing.T) {
	version, err := VanillaLauncherProfileVersion(context.Background(), VanillaLauncherProfile{
		Name:        "Modded",
		Version:     "1.20.1-forge-47.1.0",
		VersionType: VersionTypeCustom,
	})
	if err != nil {
		t.Fatalf("VanillaLauncherProfileVersion: %v", err)
	}
	if version != "1.20.1-forge-47.1.0" {
		t.Errorf("the version is %q", version)
	}
}

func TestAddVanillaLauncherProfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "launcher_profiles.json"), testLauncherProfiles)

	profile := VanillaLauncherProfile{
		Name:          "New",
		Version:       "1.20.1",
		VersionType:   VersionTypeCustom,
		GameDirectory: "/games/other",
		JavaArguments: []string{"-Xmx4G"},
	}
	if err := AddVanillaLauncherProfile(dir, profile); err != nil {
		t.Fatalf("AddVanillaLauncherProfile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "launcher_profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file vanillaProfilesFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse the profiles: %v", err)
	}
	if len(file.Profiles.keys) != 3 {
		t.Fatalf("got %d profiles, want 3", len(file.Profiles.keys))
	}

	var added *vanillaProfileEntry
	for _, raw := range file.Profiles.values {
		encoded, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		var entry vanillaProfileEntry
		if err := json.Unmarshal(encoded, &entry); err != nil {
			t.Fatal(err)
		}
		if entry.Name == "New" {
			entry := entry
			added = &entry
		}
	}
	if added == nil {
		t.Fatal("the new profile is missing")
	}
	if added.LastVersionID != "1.20.1" {
		t.Errorf("the version is %q", added.LastVersionID)
	}
	if added.GameDir != "/games/other" {
		t.Errorf("the game directory is %q", added.GameDir)
	}
	if added.JavaArgs == nil || *added.JavaArgs != "-Xmx4G" {
		t.Errorf("the java arguments are %v", added.JavaArgs)
	}
}

func TestEnsureVanillaLauncherProfiles(t *testing.T) {
	dir := t.TempDir()

	if VanillaLauncherProfilesExist(dir) {
		t.Error("the profiles should not exist yet")
	}
	if err := EnsureVanillaLauncherProfiles(dir); err != nil {
		t.Fatalf("EnsureVanillaLauncherProfiles: %v", err)
	}
	if !VanillaLauncherProfilesExist(dir) {
		t.Error("the profiles should exist now")
	}

	profiles, err := LoadVanillaLauncherProfiles(dir)
	if err != nil {
		t.Fatalf("LoadVanillaLauncherProfiles: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("got %d profiles, want none", len(profiles))
	}

	// An existing file is not overwritten.
	writeFile(t, filepath.Join(dir, "launcher_profiles.json"), testLauncherProfiles)
	if err := EnsureVanillaLauncherProfiles(dir); err != nil {
		t.Fatalf("EnsureVanillaLauncherProfiles: %v", err)
	}
	profiles, err = LoadVanillaLauncherProfiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 {
		t.Errorf("got %d profiles, want 2", len(profiles))
	}
}

func TestAddVanillaLauncherProfileKeepsTheOtherProfiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "launcher_profiles.json"), `{
    "profiles": {
        "first": {
            "name": "First",
            "type": "custom",
            "created": "2024-01-01T00:00:00+00:00",
            "lastUsed": "2024-01-02T00:00:00+00:00",
            "lastVersionId": "1.20.1",
            "icon": "ProfileName.png"
        },
        "second": {
            "name": "Second",
            "type": "custom",
            "created": "2024-01-01T00:00:00+00:00",
            "lastUsed": "2024-01-02T00:00:00+00:00",
            "lastVersionId": "1.19.4"
        }
    },
    "settings": {"language": "en_us"},
    "version": 3
}`)

	profiles, err := LoadVanillaLauncherProfiles(dir)
	if err != nil {
		t.Fatalf("LoadVanillaLauncherProfiles: %v", err)
	}
	if len(profiles) != 2 || profiles[0].Name != "First" || profiles[1].Name != "Second" {
		t.Fatalf("the profiles are %v, want them in the order of the file", profiles)
	}

	if err := AddVanillaLauncherProfile(dir, VanillaLauncherProfile{
		Name:        "Third",
		Version:     "1.21",
		VersionType: VersionTypeCustom,
	}); err != nil {
		t.Fatalf("AddVanillaLauncherProfile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "launcher_profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	entries := raw["profiles"].(map[string]any)
	if len(entries) != 3 {
		t.Fatalf("got %d profiles, want 3", len(entries))
	}
	first := entries["first"].(map[string]any)
	if first["icon"] != "ProfileName.png" {
		t.Errorf("the icon of the first profile is %v, want it to be kept", first["icon"])
	}
	if raw["settings"].(map[string]any)["language"] != "en_us" {
		t.Errorf("the settings are %v, want them to be kept", raw["settings"])
	}

	profiles, err = LoadVanillaLauncherProfiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 3 {
		t.Fatalf("got %d profiles, want 3", len(profiles))
	}
	if profiles[0].Name != "First" || profiles[2].Name != "Third" {
		t.Errorf("the profiles are %v, want them in the order of the file", profiles)
	}
}
