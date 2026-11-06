package mcl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.21", "1.21", 0},
		{"1.21.1", "1.21", 1},
		{"1.21", "1.21.1", -1},
		{"1.21.11", "1.21.10", 1},
		{"1.21.10", "1.21.9", 1},
		{"1.20.1", "1.21", -1},
		{"2.0", "1.99.99", 1},
		{"1.7.10", "1.7.9", 1},
	}

	for _, test := range tests {
		got := compareVersions(test.a, test.b)
		if got != test.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", test.a, test.b, got, test.want)
		}
		if reverse := compareVersions(test.b, test.a); reverse != -test.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", test.b, test.a, reverse, -test.want)
		}
	}
}

func TestSortVersionsNewestFirst(t *testing.T) {
	versions := []string{"1.20.1", "1.21.10", "1.21.9", "1.19", "1.21.11"}
	sortVersionsNewestFirst(versions)

	want := []string{"1.21.11", "1.21.10", "1.21.9", "1.20.1", "1.19"}
	for i := range want {
		if versions[i] != want[i] {
			t.Fatalf("the versions are %v, want %v", versions, want)
		}
	}
}

func TestGetModLoader(t *testing.T) {
	for _, id := range ListModLoaders() {
		loader, err := GetModLoader(id)
		if err != nil {
			t.Fatalf("GetModLoader(%q): %v", id, err)
		}
		if loader.ID() != id {
			t.Errorf("the loader %q reports the id %q", id, loader.ID())
		}
		if loader.Name() == "" {
			t.Errorf("the loader %q has no name", id)
		}
	}

	if _, err := GetModLoader("does-not-exist"); err == nil {
		t.Error("GetModLoader should fail for an unknown id")
	}
}

func TestModLoaderInstalledVersions(t *testing.T) {
	tests := []struct {
		loader        ModLoader
		minecraft     string
		loaderVersion string
		want          string
	}{
		{Fabric, "1.20.1", "0.15.0", "fabric-loader-0.15.0-1.20.1"},
		{Quilt, "1.20.1", "0.20.0", "quilt-loader-0.20.0-1.20.1"},
		{Forge, "1.20.1", "47.1.0", "1.20.1-forge-47.1.0"},
		{NeoForge, "1.20.1", "47.1.0", "neoforge-47.1.0"},
	}

	for _, test := range tests {
		got := test.loader.InstalledVersion(test.minecraft, test.loaderVersion)
		if got != test.want {
			t.Errorf("%s is installed under %q, want %q", test.loader.ID(), got, test.want)
		}
	}
}

func TestNormalizeMinecraftVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"21.1", "1.21.1"},
		{"20.4", "1.20.4"},
		{"21.1.172", "1.21.1.172"},
		{"25.0", "25"},
		{"26.1", "26.1"},
	}

	for _, test := range tests {
		if got := normalizeMinecraftVersion(test.input); got != test.want {
			t.Errorf("normalizeMinecraftVersion(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestParseMavenMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <groupId>net.minecraftforge</groupId>
  <versioning>
    <release>1.20.1-47.1.0</release>
    <latest>1.20.1-47.1.0</latest>
    <versions>
      <version>1.7.10-10.13.4.1614-1.7.10</version>
      <version>1.20.1-47.0.0</version>
      <version>1.20.1-47.1.0</version>
    </versions>
  </versioning>
</metadata>`))
	}))
	defer server.Close()

	metadata, err := parseMavenMetadata(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("parseMavenMetadata: %v", err)
	}
	if metadata.Release != "1.20.1-47.1.0" {
		t.Errorf("the release is %q", metadata.Release)
	}
	if metadata.Latest != "1.20.1-47.1.0" {
		t.Errorf("the latest version is %q", metadata.Latest)
	}
	if len(metadata.Versions) != 3 {
		t.Fatalf("got %d versions, want 3", len(metadata.Versions))
	}
}

func TestForgeMinecraftVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<metadata>
  <versioning>
    <versions>
      <version>1.7.10-10.13.4.1614-1.7.10</version>
      <version>1.7.10_pre1-10.13.4.1614-1.7.10</version>
      <version>1.12.2-14.23.5.2859</version>
      <version>1.20.1-47.1.0</version>
      <version>1.20.1-47.0.0</version>
    </versions>
  </versioning>
</metadata>`))
	}))
	defer server.Close()

	loader := forgeModLoader{mavenURL: server.URL}

	versions, err := loader.MinecraftVersions(context.Background(), false)
	if err != nil {
		t.Fatalf("MinecraftVersions: %v", err)
	}
	want := []string{"1.20.1", "1.12.2", "1.7.10"}
	if len(versions) != len(want) {
		t.Fatalf("got %v, want %v", versions, want)
	}
	for i := range want {
		if versions[i] != want[i] {
			t.Fatalf("got %v, want %v", versions, want)
		}
	}

	loaders, err := loader.LoaderVersions(context.Background(), "1.20.1", false)
	if err != nil {
		t.Fatalf("LoaderVersions: %v", err)
	}
	if len(loaders) != 2 || loaders[0] != "47.1.0" || loaders[1] != "47.0.0" {
		t.Errorf("got %v, want the newest forge version first", loaders)
	}

	latest, err := loader.LatestLoaderVersion(context.Background(), "1.20.1")
	if err != nil {
		t.Fatalf("LatestLoaderVersion: %v", err)
	}
	if latest != "47.1.0" {
		t.Errorf("the latest version is %q, want 47.1.0", latest)
	}
}

func TestFabricMinecraftVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/game":
			w.Write([]byte(`[{"version": "1.20.1", "stable": true}, {"version": "1.21", "stable": false}]`))
		case "/loader":
			w.Write([]byte(`[{"version": "0.15.0", "stable": true}, {"version": "0.16.0-beta.1", "stable": false}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	loader := fabricQuiltModLoader{
		id:         "fabric",
		name:       "Fabric",
		mavenURL:   server.URL + "/maven",
		gameURL:    server.URL + "/game",
		loaderURL:  server.URL + "/loader",
		loaderName: "fabric",
	}

	versions, err := loader.MinecraftVersions(context.Background(), false)
	if err != nil {
		t.Fatalf("MinecraftVersions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("got %v, want both versions", versions)
	}

	stable, err := loader.MinecraftVersions(context.Background(), true)
	if err != nil {
		t.Fatalf("MinecraftVersions: %v", err)
	}
	if len(stable) != 1 || stable[0] != "1.20.1" {
		t.Errorf("got %v, want only the stable version", stable)
	}

	loaders, err := loader.LoaderVersions(context.Background(), "1.20.1", false)
	if err != nil {
		t.Fatalf("LoaderVersions: %v", err)
	}
	if len(loaders) != 2 {
		t.Fatalf("got %v, want both loader versions", loaders)
	}

	stableLoaders, err := loader.LoaderVersions(context.Background(), "1.20.1", true)
	if err != nil {
		t.Fatalf("LoaderVersions: %v", err)
	}
	if len(stableLoaders) != 1 || stableLoaders[0] != "0.15.0" {
		t.Errorf("got %v, want only the stable loader version", stableLoaders)
	}

	supported, err := loader.IsMinecraftVersionSupported(context.Background(), "1.20.1")
	if err != nil {
		t.Fatalf("IsMinecraftVersionSupported: %v", err)
	}
	if !supported {
		t.Error("1.20.1 should be supported")
	}

	supported, err = loader.IsMinecraftVersionSupported(context.Background(), "1.8.9")
	if err != nil {
		t.Fatalf("IsMinecraftVersionSupported: %v", err)
	}
	if supported {
		t.Error("1.8.9 should not be supported")
	}
}

func TestFabricProfileInstallation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id": "fabric-loader-0.15.0-1.20.1", "mainClass": "net.fabricmc.loader.impl.launch.knot.KnotClient", "libraries": []}`))
	}))
	defer server.Close()

	loader := fabricQuiltModLoader{
		id:         "fabric",
		name:       "Fabric",
		mavenURL:   server.URL + "/maven",
		gameURL:    server.URL + "/game",
		loaderURL:  server.URL + "/loader",
		loaderName: "fabric",
	}
	dir := t.TempDir()

	if err := loader.installProfile(context.Background(), "1.20.1", "0.15.0", dir, InstallOptions{}); err != nil {
		t.Fatalf("installProfile: %v", err)
	}

	versionPath := filepath.Join(dir, "versions", "fabric-loader-0.15.0-1.20.1", "fabric-loader-0.15.0-1.20.1.json")
	data, err := readClientJSONFile(dir, "fabric-loader-0.15.0-1.20.1")
	if err != nil {
		t.Fatalf("readClientJSONFile: %v", err)
	}
	if data.ID != "fabric-loader-0.15.0-1.20.1" {
		t.Errorf("the id is %q", data.ID)
	}
	if _, err := os.Stat(versionPath); err != nil {
		t.Errorf("the version file was not written: %v", err)
	}
}
