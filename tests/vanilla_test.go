package mcl_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
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

	profiles, err := mcl.LoadVanillaLauncherProfiles(dir)
	if err != nil {
		t.Fatalf("LoadVanillaLauncherProfiles: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2", len(profiles))
	}

	byName := map[string]mcl.VanillaLauncherProfile{}
	for _, profile := range profiles {
		byName[profile.Name] = profile
	}

	release, ok := byName["Latest release"]
	if !ok {
		t.Fatalf("the profiles are %v", profiles)
	}
	if release.VersionType != mcl.VersionTypeLatestRelease {
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
	if modded.VersionType != mcl.VersionTypeCustom || modded.Version != "1.20.1-forge-47.1.0" {
		t.Errorf("the version is %q of the type %q", modded.Version, modded.VersionType)
	}
}

func TestVanillaLauncherProfileToOptions(t *testing.T) {
	profile := mcl.VanillaLauncherProfile{
		Name:             "Latest release",
		VersionType:      mcl.VersionTypeLatestRelease,
		GameDirectory:    "/games/minecraft",
		JavaExecutable:   "/usr/lib/jvm/java-21",
		JavaArguments:    []string{"-Xmx2G"},
		CustomResolution: &mcl.Resolution{Width: 1920, Height: 1080},
	}

	options, err := mcl.VanillaLauncherProfileToOptions(profile)
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
	if _, err := mcl.VanillaLauncherProfileToOptions(mcl.VanillaLauncherProfile{Name: "x"}); err == nil {
		t.Error("a profile without a version type should be refused")
	}
	if _, err := mcl.VanillaLauncherProfileToOptions(mcl.VanillaLauncherProfile{Name: "x", VersionType: mcl.VersionTypeCustom}); err == nil {
		t.Error("a custom profile without a version should be refused")
	}
	if _, err := mcl.VanillaLauncherProfileToOptions(mcl.VanillaLauncherProfile{VersionType: mcl.VersionTypeCustom, Version: "1.20.1"}); err == nil {
		t.Error("a profile without a name should be refused")
	}
}

func TestVanillaLauncherProfileVersion(t *testing.T) {
	version, err := mcl.VanillaLauncherProfileVersion(context.Background(), mcl.VanillaLauncherProfile{
		Name:        "Modded",
		Version:     "1.20.1-forge-47.1.0",
		VersionType: mcl.VersionTypeCustom,
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

	profile := mcl.VanillaLauncherProfile{
		Name:          "New",
		Version:       "1.20.1",
		VersionType:   mcl.VersionTypeCustom,
		GameDirectory: "/games/other",
		JavaArguments: []string{"-Xmx4G"},
	}
	if err := mcl.AddVanillaLauncherProfile(dir, profile); err != nil {
		t.Fatalf("AddVanillaLauncherProfile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "launcher_profiles.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file map[string]any
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse the profiles: %v", err)
	}
	entries := file["profiles"].(map[string]any)
	if len(entries) != 3 {
		t.Fatalf("got %d profiles, want %d", len(entries), 3)
	}

	var added map[string]any
	for _, raw := range entries {
		entry := raw.(map[string]any)
		if entry["name"] == "New" {
			added = entry
		}
	}
	if added == nil {
		t.Fatal("the new profile is missing")
	}
	if added["lastVersionId"] != "1.20.1" {
		t.Errorf("the version is %q", added["lastVersionId"])
	}
	if added["gameDir"] != "/games/other" {
		t.Errorf("the game directory is %q", added["gameDir"])
	}
	if added["javaArgs"] != "-Xmx4G" {
		t.Errorf("the java arguments are %v", added["javaArgs"])
	}
}

func TestEnsureVanillaLauncherProfiles(t *testing.T) {
	dir := t.TempDir()

	if mcl.VanillaLauncherProfilesExist(dir) {
		t.Error("the profiles should not exist yet")
	}
	if err := mcl.EnsureVanillaLauncherProfiles(dir); err != nil {
		t.Fatalf("EnsureVanillaLauncherProfiles: %v", err)
	}
	if !mcl.VanillaLauncherProfilesExist(dir) {
		t.Error("the profiles should exist now")
	}

	profiles, err := mcl.LoadVanillaLauncherProfiles(dir)
	if err != nil {
		t.Fatalf("LoadVanillaLauncherProfiles: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("got %d profiles, want none", len(profiles))
	}

	// An existing file is not overwritten.
	writeFile(t, filepath.Join(dir, "launcher_profiles.json"), testLauncherProfiles)
	if err := mcl.EnsureVanillaLauncherProfiles(dir); err != nil {
		t.Fatalf("EnsureVanillaLauncherProfiles: %v", err)
	}
	profiles, err = mcl.LoadVanillaLauncherProfiles(dir)
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

	profiles, err := mcl.LoadVanillaLauncherProfiles(dir)
	if err != nil {
		t.Fatalf("LoadVanillaLauncherProfiles: %v", err)
	}
	if len(profiles) != 2 || profiles[0].Name != "First" || profiles[1].Name != "Second" {
		t.Fatalf("the profiles are %v, want them in the order of the file", profiles)
	}

	if err := mcl.AddVanillaLauncherProfile(dir, mcl.VanillaLauncherProfile{
		Name:        "Third",
		Version:     "1.21",
		VersionType: mcl.VersionTypeCustom,
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
		t.Fatalf("got %d profiles, want %d", len(entries), 3)
	}
	first := entries["first"].(map[string]any)
	if first["icon"] != "ProfileName.png" {
		t.Errorf("the icon of the first profile is %v, want it to be kept", first["icon"])
	}
	if raw["settings"].(map[string]any)["language"] != "en_us" {
		t.Errorf("the settings are %v, want them to be kept", raw["settings"])
	}

	profiles, err = mcl.LoadVanillaLauncherProfiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 3 {
		t.Fatalf("got %d profiles, want %d", len(profiles), 3)
	}
	if profiles[0].Name != "First" || profiles[2].Name != "Third" {
		t.Errorf("the profiles are %v, want them in the order of the file", profiles)
	}
}
