package mcl_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

// testMinecraftDirectory creates a Minecraft directory with a single modern
// version and returns its path.
func testMinecraftDirectory(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.json"), `{
		"id": "1.20.1",
		"type": "release",
		"assets": "12",
		"mainClass": "net.minecraft.client.main.Main",
		"javaVersion": {"component": "java-runtime-gamma", "majorVersion": 17},
		"libraries": [
			{"name": "com.google.guava:guava:32.0.0"},
			{"name": "org.lwjgl:lwjgl:3.3.1", "downloads": {"artifact": {"path": "org/lwjgl/lwjgl/3.3.1/lwjgl-3.3.1.jar"}}}
		],
		"arguments": {
			"game": [
				"--username", "${auth_player_name}",
				"--version", "${version_name}",
				"--gameDir", "${game_directory}",
				"--assetsDir", "${assets_root}",
				"--uuid", "${auth_uuid}",
				{"rules": [{"action": "allow", "features": {"has_custom_resolution": true}}], "value": ["--width", "${resolution_width}", "--height", "${resolution_height}"]},
				{"rules": [{"action": "allow", "features": {"is_demo_user": true}}], "value": ["--demo"]}
			],
			"jvm": ["-Djava.library.path=${natives_directory}", "-cp", "${classpath}"]
		}
	}`)
	return dir
}

func TestCommand(t *testing.T) {
	dir := testMinecraftDirectory(t)

	command, err := mcl.Command("1.20.1", dir, mcl.Options{
		Username: "Player",
		UUID:     "1234",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}

	joined := strings.Join(command, " ")

	if command[0] != "java" {
		t.Errorf("the command starts with %q, want the java executable", command[0])
	}
	if !strings.Contains(joined, "net.minecraft.client.main.Main") {
		t.Errorf("the main class is missing: %v", command)
	}
	if !strings.Contains(joined, "--username Player") {
		t.Errorf("the player name is missing: %v", command)
	}
	if !strings.Contains(joined, "--version 1.20.1") {
		t.Errorf("the version is missing: %v", command)
	}
	if !strings.Contains(joined, "--uuid 1234") {
		t.Errorf("the uuid is missing: %v", command)
	}
	if !strings.Contains(joined, "--assetsDir "+filepath.Join(dir, "assets")) {
		t.Errorf("the assets directory is missing: %v", command)
	}
	if !strings.Contains(joined, "-Djava.library.path="+filepath.Join(dir, "versions", "1.20.1", "natives")) {
		t.Errorf("the natives directory is missing: %v", command)
	}

	// The classpath holds every allowed library and the jar of the
	// version.
	classpath := ""
	for i, arg := range command {
		if arg == "-cp" && i+1 < len(command) {
			classpath = command[i+1]
		}
	}
	if !strings.Contains(classpath, filepath.Join("libraries", "com", "google", "guava", "guava", "32.0.0", "guava-32.0.0.jar")) {
		t.Errorf("the guava library is missing from the classpath: %s", classpath)
	}
	if !strings.Contains(classpath, filepath.Join("libraries", "org", "lwjgl", "lwjgl", "3.3.1", "lwjgl-3.3.1.jar")) {
		t.Errorf("the lwjgl library is missing from the classpath: %s", classpath)
	}
	if !strings.HasSuffix(classpath, filepath.Join("versions", "1.20.1", "1.20.1.jar")) {
		t.Errorf("the jar of the version is missing from the classpath: %s", classpath)
	}
}

func TestOfflineUUID(t *testing.T) {
	// The UUIDs are the ones the official launcher uses for a player that
	// plays without an account.
	tests := map[string]string{
		"Player":  "a01e3843-e521-3998-958a-f459800e4d11",
		"xsyanic": "6b7db67b-146b-3989-a0a3-1e6d51641e36",
		"Notch":   "b50ad385-829d-3141-a216-7e7d7539ba7f",
	}
	for username, want := range tests {
		if got := mcl.OfflineUUID(username); got != want {
			t.Errorf("OfflineUUID(%q) = %q, want %q", username, got, want)
		}
	}
}

func TestCommandGeneratesUUID(t *testing.T) {
	dir := testMinecraftDirectory(t)

	command, err := mcl.Command("1.20.1", dir, mcl.Options{Username: "Player"})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	if !strings.Contains(strings.Join(command, " "), "--uuid "+mcl.OfflineUUID("Player")) {
		t.Errorf("the UUID is missing: %v", command)
	}

	// A UUID that is given is used as it is.
	command, err = mcl.Command("1.20.1", dir, mcl.Options{Username: "Player", UUID: "1234"})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	if !strings.Contains(strings.Join(command, " "), "--uuid 1234") {
		t.Errorf("the UUID is missing: %v", command)
	}
}

func TestCommandCreatesDirectories(t *testing.T) {
	dir := testMinecraftDirectory(t)
	gameDir := filepath.Join(t.TempDir(), "game_dir")

	if _, err := mcl.Command("1.20.1", dir, mcl.Options{Username: "Player", GameDirectory: gameDir}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	info, err := os.Stat(gameDir)
	if err != nil {
		t.Fatalf("the game directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("the game directory is not a directory")
	}
}

func TestCommandCreatesMinecraftDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "minecraft")
	writeFile(t, filepath.Join(dir, "versions", "1.20.1", "1.20.1.json"), `{
		"id": "1.20.1",
		"type": "release",
		"mainClass": "net.minecraft.client.main.Main"
	}`)

	if _, err := mcl.Command("1.20.1", dir, mcl.Options{Username: "Player"}); err != nil {
		t.Fatalf("Command: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("the Minecraft directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("the Minecraft directory is not a directory")
	}
}

func TestCommandWithFeatureRules(t *testing.T) {
	dir := testMinecraftDirectory(t)

	command, err := mcl.Command("1.20.1", dir, mcl.Options{
		Username:         "Player",
		CustomResolution: true,
		ResolutionWidth:  "1920",
		ResolutionHeight: "1080",
		Demo:             true,
	})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	joined := strings.Join(command, " ")
	if !strings.Contains(joined, "--width 1920") {
		t.Errorf("the width is missing: %v", command)
	}
	if !strings.Contains(joined, "--height 1080") {
		t.Errorf("the height is missing: %v", command)
	}
	if !strings.Contains(joined, "--demo") {
		t.Errorf("the demo flag is missing: %v", command)
	}
}

func TestCommandWithoutFeatureRules(t *testing.T) {
	dir := testMinecraftDirectory(t)

	command, err := mcl.Command("1.20.1", dir, mcl.Options{Username: "Player"})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	joined := strings.Join(command, " ")
	if strings.Contains(joined, "--width") {
		t.Errorf("the width should be missing: %v", command)
	}
	if strings.Contains(joined, "--demo") {
		t.Errorf("the demo flag should be missing: %v", command)
	}
}

func TestCommandWithServerAndSwitches(t *testing.T) {
	dir := testMinecraftDirectory(t)

	command, err := mcl.Command("1.20.1", dir, mcl.Options{
		Username:           "Player",
		Server:             "example.com",
		Port:               "25565",
		DisableMultiplayer: true,
		DisableChat:        true,
	})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	joined := strings.Join(command, " ")
	for _, want := range []string{"--server example.com", "--port 25565", "--disableMultiplayer", "--disableChat"} {
		if !strings.Contains(joined, want) {
			t.Errorf("%q is missing: %v", want, command)
		}
	}
}

func TestCommandWithExplicitJava(t *testing.T) {
	dir := testMinecraftDirectory(t)

	command, err := mcl.Command("1.20.1", dir, mcl.Options{
		Username:       "Player",
		ExecutablePath: "/usr/bin/java",
		JVMArguments:   []string{"-Xmx2G"},
	})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	if command[0] != "/usr/bin/java" {
		t.Errorf("the command starts with %q", command[0])
	}
	if command[1] != "-Xmx2G" {
		t.Errorf("the jvm arguments are missing: %v", command)
	}
}

func TestCommandOfUnknownVersion(t *testing.T) {
	_, err := mcl.Command("1.2.3", t.TempDir(), mcl.Options{})
	if err == nil {
		t.Fatal("Command should fail for an unknown version")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCommandOfLegacyVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "versions", "1.12.2", "1.12.2.json"), `{
		"id": "1.12.2",
		"type": "release",
		"assets": "1.12",
		"mainClass": "net.minecraft.client.main.Main",
		"minecraftArguments": "--username ${auth_player_name} --version ${version_name}"
	}`)

	command, err := mcl.Command("1.12.2", dir, mcl.Options{Username: "Player", CustomResolution: true})
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	joined := strings.Join(command, " ")
	if !strings.Contains(joined, "--username Player") {
		t.Errorf("the player name is missing: %v", command)
	}
	if !strings.Contains(joined, "--width 854") {
		t.Errorf("the width is missing: %v", command)
	}
	if !strings.Contains(joined, "-cp ") {
		t.Errorf("the classpath is missing: %v", command)
	}
}
