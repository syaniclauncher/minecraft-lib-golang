package mcl_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

func TestVersionMetadataResolvesInheritance(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.json"), `{
		"id": "1.20.1",
		"type": "release",
		"mainClass": "net.minecraft.client.main.Main",
		"libraries": [
			{"name": "org.ow2.asm:asm:9.7.1"},
			{"name": "com.google.guava:guava:32.0.0"}
		]
	}`)
	writeFile(t, filepath.Join(dir, "versions", "1.20.1-forge", "1.20.1-forge.json"), `{
		"id": "1.20.1-forge",
		"inheritsFrom": "1.20.1",
		"mainClass": "cpw.mods.bootstraplauncher.BootstrapLauncher",
		"libraries": [{"name": "org.ow2.asm:asm:9.6"}]
	}`)

	data, err := mcl.VersionMetadata(context.Background(), "1.20.1-forge", dir)
	if err != nil {
		t.Fatalf("VersionMetadata: %v", err)
	}
	if data.MainClass != "cpw.mods.bootstraplauncher.BootstrapLauncher" {
		t.Errorf("the main class is %q, want the one of the child", data.MainClass)
	}
	if len(data.Libraries) != 2 {
		t.Fatalf("got %d libraries, want 2", len(data.Libraries))
	}
	if data.Libraries[0].Name != "org.ow2.asm:asm:9.6" {
		t.Errorf("the library of the child is missing: %v", data.Libraries)
	}
	if data.Libraries[1].Name != "com.google.guava:guava:32.0.0" {
		t.Errorf("the inherited library is missing: %v", data.Libraries)
	}
}

func TestArgumentUnmarshal(t *testing.T) {
	var args mcl.Arguments
	input := `{
		"game": ["--demo", {"rules": [{"action": "allow"}], "value": ["--width", "854"]}],
		"jvm": ["-Xmx1G"]
	}`
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(args.Game) != 2 {
		t.Fatalf("got %d game arguments, want 2", len(args.Game))
	}
	if !reflect.DeepEqual(args.Game[0].Value, []string{"--demo"}) {
		t.Errorf("first argument is %v", args.Game[0].Value)
	}
	if !reflect.DeepEqual(args.Game[1].Value, []string{"--width", "854"}) {
		t.Errorf("second argument is %v", args.Game[1].Value)
	}
	if len(args.Game[1].Rules) != 1 {
		t.Errorf("the rules of the second argument are missing")
	}
	if len(args.JVM) != 1 || args.JVM[0].Value[0] != "-Xmx1G" {
		t.Errorf("the jvm arguments are %v", args.JVM)
	}
}

func TestInstalledVersions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.json"), `{
		"id": "1.20.1",
		"type": "release",
		"releaseTime": "2023-06-12T13:25:51+00:00"
	}`)
	// A directory without a client.json is not a version.
	if err := os.MkdirAll(filepath.Join(dir, "versions", "broken"), 0o755); err != nil {
		t.Fatal(err)
	}

	versions, err := mcl.InstalledVersions(dir)
	if err != nil {
		t.Fatalf("InstalledVersions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("got %d versions, want 1", len(versions))
	}
	if versions[0].ID != "1.20.1" || versions[0].Type != "release" {
		t.Errorf("got %+v, want the 1.20.1 release", versions[0])
	}
	if versions[0].ReleaseTime.Year() != 2023 {
		t.Errorf("the release time is %v", versions[0].ReleaseTime)
	}
}

func TestInstalledVersionsWithoutDirectory(t *testing.T) {
	versions, err := mcl.InstalledVersions(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("InstalledVersions: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("got %d versions, want none", len(versions))
	}
}

func TestIsMinecraftInstalled(t *testing.T) {
	dir := t.TempDir()
	if mcl.IsMinecraftInstalled(dir) {
		t.Error("an empty directory is not an installation")
	}
	for _, sub := range []string{"versions", "libraries", "assets"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if !mcl.IsMinecraftInstalled(dir) {
		t.Error("the directory should be an installation")
	}
}
