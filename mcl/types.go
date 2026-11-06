package mcl

import "time"

// Options holds everything the launch command needs to know about the player
// and the launcher's preferences. Username, UUID and Token are the only
// fields that are usually required; the rest have sensible defaults.
//
// An empty string means "not set" for the optional string fields.
type Options struct {
	// Username is the in-game name of the player.
	Username string
	// UUID is the (dashed or undashed) UUID of the player.
	UUID string
	// Token is the Minecraft access token.
	Token string

	// ExecutablePath is the Java executable to run. It takes precedence
	// over DefaultExecutablePath and over the embedded runtime.
	ExecutablePath string
	// DefaultExecutablePath is the Java executable to run when
	// ExecutablePath is empty and the version has no embedded runtime.
	DefaultExecutablePath string
	// JVMArguments are prepended to the arguments of the Java process.
	JVMArguments []string

	// LauncherName is reported to the game as ${launcher_name}.
	LauncherName string
	// LauncherVersion is reported to the game as ${launcher_version}.
	LauncherVersion string

	// GameDirectory is the working directory of the game. Defaults to the
	// Minecraft directory.
	GameDirectory string
	// NativesDirectory holds the extracted native libraries. Defaults to
	// <minecraft directory>/versions/<version>/natives.
	NativesDirectory string

	// Demo runs the game in demo mode.
	Demo bool
	// CustomResolution enables the --width/--height arguments.
	CustomResolution bool
	// ResolutionWidth is the width used when CustomResolution is set.
	ResolutionWidth string
	// ResolutionHeight is the height used when CustomResolution is set.
	ResolutionHeight string

	// Server is the address the game connects to right after starting.
	Server string
	// Port is the port of Server. Only used when Server is set.
	Port string

	// EnableLoggingConfig passes the log4j configuration to the game.
	EnableLoggingConfig bool
	// DisableMultiplayer passes --disableMultiplayer to the game.
	DisableMultiplayer bool
	// DisableChat passes --disableChat to the game.
	DisableChat bool

	// QuickPlayPath is the save the game quick-plays into.
	QuickPlayPath string
	// QuickPlaySingleplayer is the save the game quick-plays into.
	QuickPlaySingleplayer string
	// QuickPlayMultiplayer is the server the game quick-plays into.
	QuickPlayMultiplayer string
	// QuickPlayRealms is the realm the game quick-plays into.
	QuickPlayRealms string
}

// VersionInfo describes a Minecraft version.
type VersionInfo struct {
	// ID is the version identifier, for example "1.21".
	ID string
	// Type is "release" or "snapshot".
	Type string
	// ReleaseTime is when the version was released.
	ReleaseTime time.Time
	// ComplianceLevel is the classification level of the version.
	ComplianceLevel int
}

// Latest holds the latest release and snapshot versions.
type Latest struct {
	// Release is the latest stable version.
	Release string
	// Snapshot is the latest snapshot version.
	Snapshot string
}

// JavaInformation describes a Java installation.
type JavaInformation struct {
	// Path is the installation directory.
	Path string
	// Name is the base name of the installation directory.
	Name string
	// Version is the Java version, for example "17.0.9".
	Version string
	// JavaPath is the path of the java executable.
	JavaPath string
	// JavawPath is the path of the javaw executable. Empty on non-Windows
	// platforms.
	JavawPath string
	// Is64Bit reports whether the installation is 64 bit.
	Is64Bit bool
	// OpenJDK reports whether the installation is an OpenJDK build.
	OpenJDK bool
}

// Resolution is a window size in pixels.
type Resolution struct {
	Width  int
	Height int
}

// VanillaLauncherProfile describes a profile of the official Minecraft
// launcher.
type VanillaLauncherProfile struct {
	// Name is the display name of the profile.
	Name string
	// Version is the Minecraft version. Only set when VersionType is
	// "custom".
	Version string
	// VersionType is "latest-release", "latest-snapshot" or "custom".
	VersionType string
	// GameDirectory is the directory the game runs in.
	GameDirectory string
	// JavaExecutable is the Java installation used by the profile.
	JavaExecutable string
	// JavaArguments are the extra JVM arguments of the profile.
	JavaArguments []string
	// CustomResolution is the window size of the profile, if any.
	CustomResolution *Resolution
}

// Version type constants used by VanillaLauncherProfile.VersionType.
const (
	VersionTypeLatestRelease  = "latest-release"
	VersionTypeLatestSnapshot = "latest-snapshot"
	VersionTypeCustom         = "custom"
)

// MrpackInformation describes a Modrinth modpack (.mrpack file).
type MrpackInformation struct {
	// Name is the name of the modpack.
	Name string
	// Summary is the short description of the modpack.
	Summary string
	// VersionID is the version of the modpack.
	VersionID string
	// FormatVersion is the version of the .mrpack format.
	FormatVersion int
	// MinecraftVersion is the Minecraft version the modpack targets.
	MinecraftVersion string
	// OptionalFiles lists the paths of the optional files of the modpack.
	OptionalFiles []string
}

// MrpackInstallOptions configures an .mrpack installation.
type MrpackInstallOptions struct {
	// OptionalFiles are the paths of the optional files to install.
	OptionalFiles []string
	// SkipDependenciesInstall skips the installation of Minecraft and the
	// mod loader. Only useful for testing.
	SkipDependenciesInstall bool
}

// Progress receives status and progress updates while something is being
// installed. All fields are optional; a nil *Progress is valid and discards
// every update.
type Progress struct {
	// Status is called with a human readable status message.
	Status func(msg string)
	// Done is called with the number of finished steps.
	Done func(n int)
	// Max is called with the total number of steps.
	Max func(n int)
}

func (p *Progress) setStatus(msg string) {
	if p != nil && p.Status != nil {
		p.Status(msg)
	}
}

func (p *Progress) setDone(n int) {
	if p != nil && p.Done != nil {
		p.Done(n)
	}
}

func (p *Progress) setMax(n int) {
	if p != nil && p.Max != nil {
		p.Max(n)
	}
}

// TestOptions returns launch options with a random player name and UUID.
// They are meant for development and testing only.
func TestOptions() Options {
	return Options{
		Username: "Player" + randomDigits(3),
		UUID:     randomUUID(),
		Token:    "",
	}
}
