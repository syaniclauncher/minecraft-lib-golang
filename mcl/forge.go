package mcl

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ForgeMavenURL is the Maven repository of Forge.
const ForgeMavenURL = "https://maven.minecraftforge.net/net/minecraftforge/forge"

// Forge is the Forge mod loader.
var Forge ModLoader = forgeModLoader{mavenURL: ForgeMavenURL}

// forgeModLoader implements the Forge mod loader.
type forgeModLoader struct {
	// mavenURL is the Maven repository the versions of Forge come from.
	mavenURL string
}

// forgeInstallProfile is the install_profile.json of a Forge installer.
type forgeInstallProfile struct {
	Minecraft string `json:"minecraft"`
	Version   string `json:"version"`
	Data      map[string]struct {
		Client string `json:"client"`
		Server string `json:"server"`
	} `json:"data"`
	Processors []forgeProcessor `json:"processors"`
	Libraries  []Library        `json:"libraries"`
	Install    struct {
		Minecraft string `json:"minecraft"`
		Version   string `json:"version"`
	} `json:"install"`
	VersionInfo json.RawMessage `json:"versionInfo"`
}

// forgeProcessor is a single processor of the install profile of Forge.
type forgeProcessor struct {
	Jar       string   `json:"jar"`
	Classpath []string `json:"classpath"`
	Args      []string `json:"args"`
	Sides     []string `json:"sides"`
}

func (forgeModLoader) ID() string   { return "forge" }
func (forgeModLoader) Name() string { return "Forge" }

// MinecraftVersions returns the Minecraft versions Forge supports, newest
// first. Only Minecraft 1.7.10 and newer are supported.
func (l forgeModLoader) MinecraftVersions(ctx context.Context, stableOnly bool) ([]string, error) {
	metadata, err := parseMavenMetadata(ctx, l.mavenURL+"/maven-metadata.xml")
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var versions []string
	for _, version := range metadata.Versions {
		minecraftVersion, _, found := strings.Cut(version, "-")
		if !found {
			continue
		}
		// Pre releases are not supported.
		if strings.Contains(minecraftVersion, "_pre") {
			continue
		}
		if compareVersions(minecraftVersion, "1.7.10") < 0 {
			continue
		}
		if seen[minecraftVersion] {
			continue
		}
		seen[minecraftVersion] = true
		versions = append(versions, minecraftVersion)
	}

	// The maven metadata is not sorted, so the newest version comes first.
	sortVersionsNewestFirst(versions)
	return versions, nil
}

// LoaderVersions returns the versions of Forge for a Minecraft version, newest
// first.
func (l forgeModLoader) LoaderVersions(ctx context.Context, minecraftVersion string, stableOnly bool) ([]string, error) {
	metadata, err := parseMavenMetadata(ctx, l.mavenURL+"/maven-metadata.xml")
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, version := range metadata.Versions {
		current, forgeVersion, found := strings.Cut(version, "-")
		if found && current == minecraftVersion {
			versions = append(versions, forgeVersion)
		}
	}

	sortVersionsNewestFirst(versions)
	return versions, nil
}

// IsMinecraftVersionSupported reports whether Forge supports the given
// Minecraft version.
func (l forgeModLoader) IsMinecraftVersionSupported(ctx context.Context, minecraftVersion string) (bool, error) {
	versions, err := l.MinecraftVersions(ctx, false)
	if err != nil {
		return false, err
	}
	for _, version := range versions {
		if version == minecraftVersion {
			return true, nil
		}
	}
	return false, nil
}

// LatestLoaderVersion returns the newest version of Forge for a Minecraft
// version.
func (l forgeModLoader) LatestLoaderVersion(ctx context.Context, minecraftVersion string) (string, error) {
	versions, err := l.LoaderVersions(ctx, minecraftVersion, true)
	if err != nil {
		return "", err
	}
	if len(versions) > 0 {
		return versions[0], nil
	}
	versions, err = l.LoaderVersions(ctx, minecraftVersion, false)
	if err != nil {
		return "", err
	}
	if len(versions) > 0 {
		return versions[0], nil
	}
	return "", fmt.Errorf("%w: %s", ErrUnsupportedVersion, minecraftVersion)
}

// InstalledVersion returns the version id Forge is installed under.
func (l forgeModLoader) InstalledVersion(minecraftVersion, loaderVersion string) string {
	return fmt.Sprintf("%s-forge-%s", minecraftVersion, loaderVersion)
}

// InstallerURL returns the URL of the installer of Forge.
func (l forgeModLoader) InstallerURL(ctx context.Context, minecraftVersion, loaderVersion string) (string, error) {
	forgeVersion := minecraftVersion + "-" + loaderVersion
	return fmt.Sprintf("%s/%s/forge-%s-installer.jar", l.mavenURL, forgeVersion, forgeVersion), nil
}

// Install installs Forge.
func (l forgeModLoader) Install(ctx context.Context, minecraftVersion, minecraftDirectory string, opts InstallOptions) (string, error) {
	return installModLoader(ctx, l, minecraftVersion, minecraftDirectory, opts,
		func(ctx context.Context, loaderVersion string) error {
			return l.installForge(ctx, minecraftVersion, loaderVersion, minecraftDirectory, opts)
		})
}

// installForge downloads the Forge installer and runs it.
func (l forgeModLoader) installForge(ctx context.Context, minecraftVersion, loaderVersion, minecraftDirectory string, opts InstallOptions) error {
	installerURL, err := l.InstallerURL(ctx, minecraftVersion, loaderVersion)
	if err != nil {
		return err
	}
	forgeVersion := minecraftVersion + "-" + loaderVersion
	forgeVersionID := l.InstalledVersion(minecraftVersion, loaderVersion)

	tempDir, err := os.MkdirTemp("", "minecraft-launcher-lib-go-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	installerPath := filepath.Join(tempDir, "installer.jar")
	if _, err := downloadFile(ctx, installerURL, installerPath, "", "", true, opts.Progress); err != nil {
		return err
	}

	reader, err := zip.OpenReader(installerPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	profile, err := readForgeInstallProfile(reader)
	if err != nil {
		return err
	}
	if profile.Minecraft != "" {
		minecraftVersion = profile.Minecraft
	} else if profile.Install.Minecraft != "" {
		minecraftVersion = profile.Install.Minecraft
	}

	if len(profile.Libraries) > 0 {
		if err := installLibraries(ctx, minecraftVersion, profile.Libraries, minecraftDirectory, opts.Progress); err != nil {
			return err
		}
	}

	versionDir := filepath.Join(minecraftDirectory, "versions", forgeVersionID)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		return err
	}
	if err := writeForgeVersionJSON(reader, profile, versionDir, forgeVersionID); err != nil {
		return err
	}

	forgeLibPath := filepath.Join(minecraftDirectory, "libraries", "net", "minecraftforge", "forge", forgeVersion)
	extractors := []struct{ in, out string }{
		{
			fmt.Sprintf("maven/net/minecraftforge/forge/%s/forge-%s-universal.jar", forgeVersion, forgeVersion),
			fmt.Sprintf("forge-%s-universal.jar", forgeVersion),
		},
		{
			fmt.Sprintf("forge-%s-universal.jar", forgeVersion),
			fmt.Sprintf("forge-%s.jar", forgeVersion),
		},
		{
			fmt.Sprintf("maven/net/minecraftforge/forge/%s/forge-%s.jar", forgeVersion, forgeVersion),
			fmt.Sprintf("forge-%s.jar", forgeVersion),
		},
	}
	for _, extractor := range extractors {
		if err := extractFileFromZip(reader, extractor.in, filepath.Join(forgeLibPath, extractor.out), minecraftDirectory); err != nil {
			// Not every Forge version ships every file.
			continue
		}
	}

	lzmaPath := filepath.Join(tempDir, "client.lzma")
	_ = extractFileFromZip(reader, "data/client.lzma", lzmaPath, minecraftDirectory)

	if err := Install(ctx, forgeVersionID, minecraftDirectory, opts.Progress); err != nil {
		return err
	}

	if len(profile.Processors) > 0 {
		if err := runForgeProcessors(ctx, profile, minecraftDirectory, tempDir, lzmaPath, installerPath, opts); err != nil {
			return err
		}
	}

	// Forge does not always ship a client jar, in that case the one of the
	// base version is used.
	forgeJar := filepath.Join(minecraftDirectory, "versions", forgeVersionID, forgeVersionID+".jar")
	if _, err := os.Stat(forgeJar); os.IsNotExist(err) {
		baseJar := filepath.Join(minecraftDirectory, "versions", minecraftVersion, minecraftVersion+".jar")
		if err := copyFile(baseJar, forgeJar); err != nil {
			return err
		}
	}
	return nil
}

// readForgeInstallProfile reads the install_profile.json of a Forge
// installer.
func readForgeInstallProfile(reader *zip.ReadCloser) (forgeInstallProfile, error) {
	var profile forgeInstallProfile
	for _, file := range reader.File {
		if file.Name != "install_profile.json" {
			continue
		}
		data, err := readZipFile(file)
		if err != nil {
			return profile, err
		}
		if err := json.Unmarshal(data, &profile); err != nil {
			return profile, fmt.Errorf("parse install_profile.json: %w", err)
		}
		return profile, nil
	}
	return profile, fmt.Errorf("the installer does not contain an install_profile.json")
}

// writeForgeVersionJSON writes the version metadata of the Forge version.
func writeForgeVersionJSON(reader *zip.ReadCloser, profile forgeInstallProfile, versionDir, forgeVersionID string) error {
	var data []byte
	for _, file := range reader.File {
		if file.Name == "version.json" {
			content, err := readZipFile(file)
			if err != nil {
				return err
			}
			data = content
			break
		}
	}
	if data == nil && len(profile.VersionInfo) > 0 {
		data = profile.VersionInfo
	}
	if data == nil {
		return fmt.Errorf("the installer does not contain a version.json")
	}

	var version map[string]any
	if err := json.Unmarshal(data, &version); err != nil {
		return fmt.Errorf("parse version.json: %w", err)
	}
	version["id"] = forgeVersionID

	encoded, err := json.MarshalIndent(version, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(versionDir, forgeVersionID+".json"), encoded, 0o644)
}

// runForgeProcessors runs the processors of the install profile of Forge.
func runForgeProcessors(
	ctx context.Context,
	profile forgeInstallProfile,
	minecraftDirectory, tempDir, lzmaPath, installerPath string,
	opts InstallOptions,
) error {
	arguments := map[string]string{
		"{MINECRAFT_JAR}": filepath.Join(minecraftDirectory, "versions", profile.Minecraft, profile.Minecraft+".jar"),
		"{INSTALLER}":     installerPath,
		"{BINPATCH}":      lzmaPath,
		"{ROOT}":          filepath.Join(tempDir, "root"),
		"{SIDE}":          "client",
	}
	for key, value := range profile.Data {
		client := value.Client
		if strings.HasPrefix(client, "[") && strings.HasSuffix(client, "]") {
			arguments["{"+key+"}"] = libraryPath(client[1:len(client)-1], minecraftDirectory)
			continue
		}
		arguments["{"+key+"}"] = client
	}

	rootPath := filepath.Join(tempDir, "root")
	if err := os.MkdirAll(rootPath, 0o755); err != nil {
		return err
	}

	opts.Progress.setMax(len(profile.Processors))
	for count, processor := range profile.Processors {
		if len(processor.Sides) > 0 && !containsString(processor.Sides, "client") {
			continue
		}

		opts.Progress.setStatus("Running processor " + processor.Jar)

		var classpath strings.Builder
		for _, entry := range processor.Classpath {
			classpath.WriteString(libraryPath(entry, minecraftDirectory))
			classpath.WriteString(classpathSeparator())
		}
		processorJar := libraryPath(processor.Jar, minecraftDirectory)
		classpath.WriteString(processorJar)

		mainClass, err := jarMainClass(processorJar)
		if err != nil {
			return err
		}

		command := []string{javaExecutable(opts), "-cp", classpath.String(), mainClass}
		for _, arg := range processor.Args {
			value, ok := arguments[arg]
			if !ok {
				value = arg
			}
			if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
				command = append(command, libraryPath(value[1:len(value)-1], minecraftDirectory))
				continue
			}
			command = append(command, value)
		}
		for key, value := range arguments {
			for i := range command {
				command[i] = strings.ReplaceAll(command[i], key, value)
			}
		}

		if err := runCommand(ctx, command, rootPath); err != nil {
			return err
		}
		opts.Progress.setDone(count)
	}
	return nil
}

// runCommand runs an external program and reports its output when it fails.
func runCommand(ctx context.Context, command []string, workingDirectory string) error {
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = workingDirectory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ExternalProgramError{
			Command: command,
			Output:  string(output),
			Err:     err,
		}
	}
	return nil
}

// javaExecutable returns the Java executable to use for an installer.
func javaExecutable(opts InstallOptions) string {
	if opts.Java != "" {
		return opts.Java
	}
	return "java"
}

// containsString reports whether the list contains the value.
func containsString(list []string, value string) bool {
	for _, entry := range list {
		if entry == value {
			return true
		}
	}
	return false
}

// extractFileFromZip writes a single file of a zip archive to disk.
func extractFileFromZip(reader *zip.ReadCloser, name, target, minecraftDirectory string) error {
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		if err := checkPathInsideMinecraftDirectory(minecraftDirectory, target); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := readZipFile(file)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}
	return fmt.Errorf("%s is not in the archive", name)
}

// readZipFile returns the content of a file of a zip archive.
func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// jarMainClass returns the main class of a jar file.
func jarMainClass(path string) (string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != "META-INF/MANIFEST.MF" {
			continue
		}
		data, err := readZipFile(file)
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(string(data), "\n") {
			key, value, found := strings.Cut(strings.TrimRight(line, "\r"), ":")
			if !found {
				continue
			}
			if key == "Main-Class" {
				return strings.TrimSpace(value), nil
			}
		}
	}
	return "", fmt.Errorf("%s does not contain a main class", path)
}
