package mcl

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// OfflineUUID returns the UUID of a player that plays without an account. It
// is the version 3 UUID of the name with the prefix "OfflinePlayer:", the
// same UUID the official launcher uses.
//
// mcl.Command uses it when the options hold no UUID.
func OfflineUUID(username string) string {
	sum := md5.Sum([]byte("OfflinePlayer:" + username))
	sum[6] = (sum[6] & 0x0f) | 0x30 // the version
	sum[8] = (sum[8] & 0x3f) | 0x80 // the variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

// Command returns the command line that launches the given Minecraft version.
//
// The returned slice can be handed to os/exec as is. Username and Token
// should be filled with the values of the signed in account, the other
// options are optional. An empty UUID is generated from the name.
//
// The Minecraft directory and the game directory are created when they do not
// exist yet, because the game writes its saves and its logs into the game
// directory.
//
// See the documentation of Options for a description of every field.
func Command(version, minecraftDirectory string, opts Options) ([]string, error) {
	// The game directory has to exist when the game starts.
	gameDirectory := defaultString(opts.GameDirectory, minecraftDirectory)
	if err := os.MkdirAll(minecraftDirectory, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(gameDirectory, 0o755); err != nil {
		return nil, err
	}

	versionDir := filepath.Join(minecraftDirectory, "versions", version)
	if info, err := os.Stat(versionDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrVersionNotFound, version)
	}

	data, err := loadClientJSON(minecraftDirectory, version)
	if err != nil {
		return nil, err
	}

	if opts.NativesDirectory == "" {
		opts.NativesDirectory = filepath.Join(minecraftDirectory, "versions", data.ID, "natives")
	}
	if opts.UUID == "" {
		opts.UUID = OfflineUUID(opts.Username)
	}
	classpath := buildClasspath(data, minecraftDirectory)

	var command []string
	switch {
	case opts.ExecutablePath != "":
		command = append(command, opts.ExecutablePath)
	case opts.DefaultExecutablePath != "":
		command = append(command, opts.DefaultExecutablePath)
	default:
		majorVersion := 8
		if data.JavaVersion != nil {
			majorVersion = data.JavaVersion.MajorVersion
		}
		executable := TemurinExecutablePath(majorVersion, minecraftDirectory)
		if executable == "" {
			executable = "java"
		}
		command = append(command, executable)
	}
	command = append(command, opts.JVMArguments...)

	// Newer versions bring their own JVM arguments, older ones only know
	// the classpath.
	if data.Arguments != nil && data.Arguments.JVM != nil {
		command = append(command, replaceArguments(data.Arguments.JVM, data, minecraftDirectory, opts, classpath)...)
	} else {
		command = append(command, "-Djava.library.path="+opts.NativesDirectory, "-cp", classpath)
	}

	if opts.EnableLoggingConfig {
		if client, ok := data.Logging["client"]; ok && len(data.Logging) > 0 {
			logPath := filepath.Join(minecraftDirectory, "assets", "log_configs", client.File.ID)
			command = append(command, strings.ReplaceAll(client.Argument, "${path}", logPath))
		}
	}

	command = append(command, data.MainClass)

	if data.MinecraftArguments != "" {
		command = append(command, legacyArguments(data, minecraftDirectory, opts, classpath)...)
	} else if data.Arguments != nil {
		command = append(command, replaceArguments(data.Arguments.Game, data, minecraftDirectory, opts, classpath)...)
	}

	if opts.Server != "" {
		command = append(command, "--server", opts.Server)
		if opts.Port != "" {
			command = append(command, "--port", opts.Port)
		}
	}
	if opts.DisableMultiplayer {
		command = append(command, "--disableMultiplayer")
	}
	if opts.DisableChat {
		command = append(command, "--disableChat")
	}

	return command, nil
}

// buildClasspath returns the classpath of a version.
func buildClasspath(data *ClientJSON, minecraftDirectory string) string {
	separator := classpathSeparator()

	var parts []string
	for _, lib := range data.Libraries {
		if len(lib.Rules) > 0 && !ruleListAllows(lib.Rules, Options{}) {
			continue
		}
		path := libraryPath(lib.Name, minecraftDirectory)
		if path == "" {
			// The name of the library is not a valid Maven
			// coordinate, so there is no file for it.
			continue
		}
		parts = append(parts, path)

		native := nativeClassifier(lib)
		if native == "" {
			continue
		}
		if lib.Downloads != nil {
			if classifier, ok := lib.Downloads.Classifiers[native]; ok && classifier.Path != "" {
				parts = append(parts, filepath.Join(minecraftDirectory, "libraries", filepath.FromSlash(classifier.Path)))
				continue
			}
		}
		parts = append(parts, libraryPath(lib.Name+"-"+native, minecraftDirectory))
	}

	jarName := data.ID
	if data.Jar != "" {
		jarName = data.Jar
	}
	parts = append(parts, filepath.Join(minecraftDirectory, "versions", jarName, jarName+".jar"))

	return strings.Join(parts, separator)
}

// legacyArguments returns the arguments of a version older than 1.13, which
// stores them in a single string.
func legacyArguments(data *ClientJSON, minecraftDirectory string, opts Options, classpath string) []string {
	var args []string
	for _, value := range strings.Split(data.MinecraftArguments, " ") {
		args = append(args, replaceArgument(value, data, minecraftDirectory, opts, classpath))
	}

	// The resolution and the demo mode are not part of the argument string
	// of old versions.
	if opts.CustomResolution {
		args = append(args, "--width", defaultString(opts.ResolutionWidth, "854"))
		args = append(args, "--height", defaultString(opts.ResolutionHeight, "480"))
	}
	if opts.Demo {
		args = append(args, "--demo")
	}
	return args
}

// replaceArguments returns the arguments of a version with every placeholder
// replaced by its value. Arguments whose rules do not match are dropped.
func replaceArguments(args []Argument, data *ClientJSON, minecraftDirectory string, opts Options, classpath string) []string {
	var out []string
	for _, arg := range args {
		if len(arg.CompatibilityRules) > 0 && !ruleListAllows(arg.CompatibilityRules, opts) {
			continue
		}
		if len(arg.Rules) > 0 && !ruleListAllows(arg.Rules, opts) {
			continue
		}
		for _, value := range arg.Value {
			out = append(out, replaceArgument(value, data, minecraftDirectory, opts, classpath))
		}
	}
	return out
}

// replaceArgument replaces every placeholder of a single argument.
func replaceArgument(value string, data *ClientJSON, minecraftDirectory string, opts Options, classpath string) string {
	replacements := []struct{ placeholder, value string }{
		{"${natives_directory}", opts.NativesDirectory},
		{"${launcher_name}", defaultString(opts.LauncherName, "minecraft-launcher-lib")},
		{"${launcher_version}", defaultString(opts.LauncherVersion, Version)},
		{"${classpath}", classpath},
		{"${auth_player_name}", defaultString(opts.Username, "DefaultPlayer")},
		{"${version_name}", data.ID},
		{"${game_directory}", defaultString(opts.GameDirectory, minecraftDirectory)},
		{"${assets_root}", filepath.Join(minecraftDirectory, "assets")},
		{"${assets_index_name}", defaultString(data.Assets, data.ID)},
		{"${auth_uuid}", defaultString(opts.UUID, "{uuid}")},
		{"${user_type}", "msa"},
		{"${version_type}", data.Type},
		{"${user_properties}", "{}"},
		{"${resolution_width}", defaultString(opts.ResolutionWidth, "854")},
		{"${resolution_height}", defaultString(opts.ResolutionHeight, "480")},
		{"${game_assets}", filepath.Join(minecraftDirectory, "assets", "virtual", "legacy")},
		{"${auth_session}", defaultString(opts.Token, "0")}, //--accessToken or --token
		{"${auth_access_token}", defaultString(opts.Token, "0")},
		{"${library_directory}", filepath.Join(minecraftDirectory, "libraries")},
		{"${classpath_separator}", classpathSeparator()},
		{"${quickPlayPath}", defaultString(opts.QuickPlayPath, "{quickPlayPath}")},
		{"${quickPlaySingleplayer}", defaultString(opts.QuickPlaySingleplayer, "{quickPlaySingleplayer}")},
		{"${quickPlayMultiplayer}", defaultString(opts.QuickPlayMultiplayer, "{quickPlayMultiplayer}")},
		{"${quickPlayRealms}", defaultString(opts.QuickPlayRealms, "{quickPlayRealms}")},
		{"${clientid}", defaultString(opts.ClientID, "0")},
		{"${auth_xuid}", defaultString(opts.AuthXUID, "0")},
	}
	for _, replacement := range replacements {
		value = strings.ReplaceAll(value, replacement.placeholder, replacement.value)
	}
	return value
}

// classpathSeparator returns the separator of the classpath on the current
// platform.
func classpathSeparator() string {
	if runtime.GOOS == "windows" {
		return ";"
	}
	return ":"
}

// defaultString returns value, or fallback when value is empty.
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
