package mcl

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// writeMrpack creates a .mrpack file with the given index and returns its
// path.
func writeMrpack(t *testing.T, dir, index string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, "pack.mrpack")

	archive, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()

	writer := zip.NewWriter(archive)
	entry, err := writer.Create("modrinth.index.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(index)); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

const testMrpackIndex = `{
    "formatVersion": 1,
    "game": "minecraft",
    "versionId": "1.0.0",
    "name": "Test Pack",
    "summary": "A pack for the tests",
    "dependencies": {
        "minecraft": "1.20.1",
        "fabric-loader": "0.15.0"
    },
    "files": [
        {
            "path": "mods/required.jar",
            "hashes": {"sha1": "0000000000000000000000000000000000000000"},
            "env": {"client": "required", "server": "required"},
            "downloads": ["https://example.com/required.jar"]
        },
        {
            "path": "mods/optional.jar",
            "hashes": {"sha1": "0000000000000000000000000000000000000000"},
            "env": {"client": "optional", "server": "optional"},
            "downloads": ["https://example.com/optional.jar"]
        }
    ]
}`

func TestGetMrpackInformation(t *testing.T) {
	dir := t.TempDir()
	path := writeMrpack(t, dir, testMrpackIndex, nil)

	information, err := GetMrpackInformation(path)
	if err != nil {
		t.Fatalf("GetMrpackInformation: %v", err)
	}
	if information.Name != "Test Pack" {
		t.Errorf("the name is %q", information.Name)
	}
	if information.Summary != "A pack for the tests" {
		t.Errorf("the summary is %q", information.Summary)
	}
	if information.VersionID != "1.0.0" {
		t.Errorf("the version is %q", information.VersionID)
	}
	if information.FormatVersion != 1 {
		t.Errorf("the format version is %d", information.FormatVersion)
	}
	if information.MinecraftVersion != "1.20.1" {
		t.Errorf("the minecraft version is %q", information.MinecraftVersion)
	}
	if len(information.OptionalFiles) != 1 || information.OptionalFiles[0] != "mods/optional.jar" {
		t.Errorf("the optional files are %v", information.OptionalFiles)
	}
}

func TestMrpackLaunchVersion(t *testing.T) {
	tests := []struct {
		dependencies string
		want         string
	}{
		{`"minecraft": "1.20.1", "fabric-loader": "0.15.0"`, "fabric-loader-0.15.0-1.20.1"},
		{`"minecraft": "1.20.1", "quilt-loader": "0.20.0"`, "quilt-loader-0.20.0-1.20.1"},
		{`"minecraft": "1.20.1", "forge": "47.1.0"`, "1.20.1-forge-47.1.0"},
		{`"minecraft": "1.20.1", "neoforge": "47.1.0"`, "neoforge-47.1.0"},
		{`"minecraft": "1.20.1"`, "1.20.1"},
	}

	for _, test := range tests {
		dir := t.TempDir()
		index := `{"formatVersion": 1, "versionId": "1.0.0", "name": "Pack", "dependencies": {` + test.dependencies + `}, "files": []}`
		path := writeMrpack(t, dir, index, nil)

		version, err := MrpackLaunchVersion(path)
		if err != nil {
			t.Fatalf("MrpackLaunchVersion: %v", err)
		}
		if version != test.want {
			t.Errorf("got %q, want %q", version, test.want)
		}
	}
}

func TestFilterMrpackFiles(t *testing.T) {
	index, err := readMrpackIndex(writeMrpack(t, t.TempDir(), testMrpackIndex, nil))
	if err != nil {
		t.Fatal(err)
	}

	files := filterMrpackFiles(index.Files, MrpackInstallOptions{})
	if len(files) != 1 || files[0].Path != "mods/required.jar" {
		t.Errorf("got %v, want only the required file", files)
	}

	files = filterMrpackFiles(index.Files, MrpackInstallOptions{
		OptionalFiles: []string{"mods/optional.jar"},
	})
	if len(files) != 2 {
		t.Errorf("got %v, want both files", files)
	}
}

func TestInstallMrpackExtractsOverrides(t *testing.T) {
	dir := t.TempDir()
	index := `{"formatVersion": 1, "versionId": "1.0.0", "name": "Pack", "dependencies": {"minecraft": "1.20.1"}, "files": []}`
	path := writeMrpack(t, dir, index, map[string]string{
		"overrides/config/test.txt":    "content",
		"client-overrides/options.txt": "options",
	})

	minecraftDirectory := t.TempDir()
	if err := InstallMrpack(context.Background(), path, minecraftDirectory, "", MrpackInstallOptions{
		SkipDependenciesInstall: true,
	}, nil); err != nil {
		t.Fatalf("InstallMrpack: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(minecraftDirectory, "config", "test.txt"))
	if err != nil {
		t.Fatalf("the override was not extracted: %v", err)
	}
	if string(data) != "content" {
		t.Errorf("the override contains %q", data)
	}

	data, err = os.ReadFile(filepath.Join(minecraftDirectory, "options.txt"))
	if err != nil {
		t.Fatalf("the client override was not extracted: %v", err)
	}
	if string(data) != "options" {
		t.Errorf("the client override contains %q", data)
	}
}
