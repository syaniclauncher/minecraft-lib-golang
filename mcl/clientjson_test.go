package mcl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMergeInheritedKeepsChildLibraries(t *testing.T) {
	parent := map[string]any{
		"id": "1.20.1",
		"libraries": []any{
			map[string]any{"name": "org.ow2.asm:asm:9.7.1"},
			map[string]any{"name": "com.google.guava:guava:32.0.0"},
		},
	}
	child := map[string]any{
		"id":           "1.20.1-forge",
		"inheritsFrom": "1.20.1",
		"libraries": []any{
			map[string]any{"name": "org.ow2.asm:asm:9.6"},
		},
	}

	merged := mergeInherited(parent, child)

	libraries, ok := merged["libraries"].([]any)
	if !ok {
		t.Fatalf("libraries is %T, want []any", merged["libraries"])
	}
	if len(libraries) != 2 {
		t.Fatalf("got %d libraries, want 2: %v", len(libraries), libraries)
	}
	if got := libraries[0].(map[string]any)["name"]; got != "org.ow2.asm:asm:9.6" {
		t.Errorf("first library is %v, want the one of the child", got)
	}
	if got := libraries[1].(map[string]any)["name"]; got != "com.google.guava:guava:32.0.0" {
		t.Errorf("second library is %v, want the inherited one", got)
	}
}

func TestMergeInheritedConcatenatesLists(t *testing.T) {
	parent := map[string]any{
		"arguments": map[string]any{
			"game": []any{"--parent"},
			"jvm":  []any{"-Dparent"},
		},
	}
	child := map[string]any{
		"arguments": map[string]any{
			"game": []any{"--child"},
			"jvm":  []any{"-Dchild"},
		},
	}

	merged := mergeInherited(parent, child)
	arguments := merged["arguments"].(map[string]any)

	game := arguments["game"].([]any)
	if len(game) != 2 || game[0] != "--parent" || game[1] != "--child" {
		t.Errorf("game arguments are %v, want the inherited ones first", game)
	}
	jvm := arguments["jvm"].([]any)
	if len(jvm) != 2 || jvm[0] != "-Dparent" || jvm[1] != "-Dchild" {
		t.Errorf("jvm arguments are %v, want the inherited ones first", jvm)
	}
}

func TestMergeInheritedKeepsNestedValuesOfTheParent(t *testing.T) {
	parent := map[string]any{
		"javaVersion": map[string]any{"component": "java-runtime-gamma", "majorVersion": 17},
	}
	child := map[string]any{
		"javaVersion": map[string]any{"component": "java-runtime-alpha", "majorVersion": 8},
	}

	merged := mergeInherited(parent, child)
	javaVersion := merged["javaVersion"].(map[string]any)
	if javaVersion["majorVersion"] != 17 {
		t.Errorf("majorVersion is %v, want the inherited 17", javaVersion["majorVersion"])
	}
}

func TestMergeInheritedReplacesSimpleValues(t *testing.T) {
	parent := map[string]any{"id": "1.20.1", "type": "release"}
	child := map[string]any{"id": "1.20.1-forge"}

	merged := mergeInherited(parent, child)
	if merged["id"] != "1.20.1-forge" {
		t.Errorf("id is %v, want the one of the child", merged["id"])
	}
	if merged["type"] != "release" {
		t.Errorf("type is %v, want the inherited one", merged["type"])
	}
}

func TestLoadClientJSONResolvesInheritance(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.json"), `{
		"id": "1.20.1",
		"type": "release",
		"mainClass": "net.minecraft.client.main.Main",
		"libraries": [{"name": "com.google.guava:guava:32.0.0"}]
	}`)
	writeFile(t, filepath.Join(dir, "versions", "1.20.1-forge", "1.20.1-forge.json"), `{
		"id": "1.20.1-forge",
		"inheritsFrom": "1.20.1",
		"mainClass": "cpw.mods.bootstraplauncher.BootstrapLauncher",
		"libraries": [{"name": "net.minecraftforge:forge:1.20.1-47.1.0"}]
	}`)

	data, err := loadClientJSON(dir, "1.20.1-forge")
	if err != nil {
		t.Fatalf("loadClientJSON: %v", err)
	}
	if data.MainClass != "cpw.mods.bootstraplauncher.BootstrapLauncher" {
		t.Errorf("main class is %q, want the one of the child", data.MainClass)
	}
	if len(data.Libraries) != 2 {
		t.Fatalf("got %d libraries, want 2", len(data.Libraries))
	}
	if data.Libraries[1].Name != "com.google.guava:guava:32.0.0" {
		t.Errorf("the inherited library is missing: %v", data.Libraries)
	}
}

func TestLibraryPath(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "org.lwjgl:lwjgl:3.3.1",
			want: []string{"libraries", "org", "lwjgl", "lwjgl", "3.3.1", "lwjgl-3.3.1.jar"},
		},
		{
			name: "org.lwjgl:lwjgl:3.3.1:natives-linux",
			want: []string{"libraries", "org", "lwjgl", "lwjgl", "3.3.1", "lwjgl-3.3.1-natives-linux.jar"},
		},
		{
			name: "com.example:artifact:1.0@zip",
			want: []string{"libraries", "com", "example", "artifact", "1.0", "artifact-1.0.zip"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := libraryPath(test.name, "mc")
			want := filepath.Join(append([]string{"mc"}, test.want...)...)
			if got != want {
				t.Errorf("libraryPath() = %q, want %q", got, want)
			}
		})
	}
}

func TestLibraryPathWithTooFewParts(t *testing.T) {
	if got := libraryPath("invalid", "mc"); got != "" {
		t.Errorf("libraryPath() = %q, want an empty string", got)
	}
}

func TestArgumentUnmarshal(t *testing.T) {
	var args Arguments
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

	versions, err := InstalledVersions(dir)
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
	versions, err := InstalledVersions(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("InstalledVersions: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("got %d versions, want none", len(versions))
	}
}

func TestIsMinecraftInstalled(t *testing.T) {
	dir := t.TempDir()
	if IsMinecraftInstalled(dir) {
		t.Error("an empty directory is not an installation")
	}
	for _, sub := range []string{"versions", "libraries", "assets"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if !IsMinecraftInstalled(dir) {
		t.Error("the directory should be an installation")
	}
}
