# API reference

Everything lives in one package:

```go
import "github.com/syaniclauncher/minecraft-lib-golang/mcl"
```

The complete documentation of every exported symbol is on
[pkg.go.dev](https://pkg.go.dev/github.com/syaniclauncher/minecraft-lib-golang/mcl).
This page is the map of the package.

## Versions

```go
func Manifest(ctx context.Context) (*VersionManifest, error)
func LatestVersions(ctx context.Context) (Latest, error)
func VersionList(ctx context.Context) ([]VersionInfo, error)
func InstalledVersions(minecraftDirectory string) ([]VersionInfo, error)
func AvailableVersions(ctx context.Context, minecraftDirectory string) ([]VersionInfo, error)
func VersionMetadata(ctx context.Context, version, minecraftDirectory string) (*ClientJSON, error)
func IsVersionValid(ctx context.Context, version, minecraftDirectory string) (bool, error)
func IsVanillaVersion(ctx context.Context, version string) (bool, error)
func IsMinecraftInstalled(minecraftDirectory string) bool
```

## Installing

```go
func Install(ctx context.Context, version, minecraftDirectory string, progress *Progress) error
func InstallJVMRuntime(ctx context.Context, majorVersion int, minecraftDirectory string, progress *Progress) error
func ExtractNatives(version, minecraftDirectory, extractPath string) error
func TemurinExecutablePath(majorVersion int, minecraftDirectory string) string
func JVMRuntimes(ctx context.Context) ([]string, error)
func InstalledJVMRuntimes(minecraftDirectory string) ([]string, error)
func JVMRuntimeInformation(ctx context.Context, jvmVersion string) (JVMRuntime, error)
func VersionRuntimeInformation(version, minecraftDirectory string) (*VersionRuntime, error)
```

## Launching

```go
func Command(version, minecraftDirectory string, opts Options) ([]string, error)
func OfflineUUID(username string) string
func TestOptions() Options
```

`Options` holds the name, the UUID and the token of the player and everything
else the launcher configures. See [Launch options](launch-options.md).
`OfflineUUID` returns the UUID of a player that plays without an account;
`Command` uses it when the options hold no UUID.

## Mod loaders

```go
func GetModLoader(id string) (ModLoader, error)
func ListModLoaders() []string
```

`ModLoader` is the interface of a mod loader:

```go
type ModLoader interface {
	ID() string
	Name() string
	MinecraftVersions(ctx context.Context, stableOnly bool) ([]string, error)
	IsMinecraftVersionSupported(ctx context.Context, minecraftVersion string) (bool, error)
	LoaderVersions(ctx context.Context, minecraftVersion string, stableOnly bool) ([]string, error)
	LatestLoaderVersion(ctx context.Context, minecraftVersion string) (string, error)
	InstalledVersion(minecraftVersion, loaderVersion string) string
	InstallerURL(ctx context.Context, minecraftVersion, loaderVersion string) (string, error)
	Install(ctx context.Context, minecraftVersion, minecraftDirectory string, opts InstallOptions) (string, error)
}
```

The four implementations are `mcl.Forge`, `mcl.NeoForge`, `mcl.Fabric` and
`mcl.Quilt`. `InstallOptions` holds the version of the loader, the progress
and the Java executable of the installer.

## Accounts

```go
func LoginURL(clientID, redirectURI string) string
func SecureLoginData(clientID, redirectURI, state string) LoginData
func GenerateState() string
func URLContainsAuthCode(rawURL string) bool
func AuthCodeFromURL(rawURL string) (string, bool)
func ParseAuthCodeURL(rawURL, state string) (string, error)
func CompleteLogin(ctx context.Context, clientID, clientSecret, redirectURI, authCode, codeVerifier string) (*Account, error)
func CompleteRefresh(ctx context.Context, clientID, clientSecret, refreshToken string) (*Account, error)
```

See [Microsoft login](microsoft-login.md).

## Modpacks

```go
func GetMrpackInformation(path string) (MrpackInformation, error)
func InstallMrpack(ctx context.Context, path, minecraftDirectory, modpackDirectory string, opts MrpackInstallOptions, progress *Progress) error
func MrpackLaunchVersion(path string) (string, error)
```

## The official launcher

```go
func LoadVanillaLauncherProfiles(minecraftDirectory string) ([]VanillaLauncherProfile, error)
func VanillaLauncherProfileToOptions(profile VanillaLauncherProfile) (Options, error)
func VanillaLauncherProfileVersion(ctx context.Context, profile VanillaLauncherProfile) (string, error)
func AddVanillaLauncherProfile(minecraftDirectory string, profile VanillaLauncherProfile) error
func VanillaLauncherProfilesExist(minecraftDirectory string) bool
func CreateEmptyVanillaLauncherProfiles(minecraftDirectory string) error
func EnsureVanillaLauncherProfiles(minecraftDirectory string) error
```

## Java

```go
func DefaultJavaExecutable() string
func JavaInformation(path string) (JavaInstallation, error)
func FindSystemJavaVersions(additionalDirectories []string) []string
func FindSystemJavaVersionsInformation(additionalDirectories []string) ([]JavaInstallation, error)
```

## News

```go
func GetMinecraftNews(ctx context.Context) (*News, error)
func GetJavaPatchNotes(ctx context.Context) (*JavaPatchNotes, error)
```

## Directories

```go
func DefaultMinecraftDirectory() string
func IsPlatformSupported() bool
```

## The progress

```go
type Progress struct {
	Status func(msg string)
	Done   func(n int)
	Max    func(n int)
}
```

Every field is optional and a nil `*Progress` is valid. See
[Installation progress](installation-progress.md).

## The errors

The errors that only say what went wrong are sentinels, so `errors.Is` finds
them:

| Error | Meaning |
| --- | --- |
| `ErrVersionNotFound` | The version is neither installed nor offered by Mojang. |
| `ErrUnsupportedVersion` | The mod loader does not support the version. |
| `ErrInvalidRefreshToken` | The refresh token was rejected. |
| `ErrAzureAppNotPermitted` | The Azure app may not use the Minecraft API. |
| `ErrAccountNotOwnMinecraft` | The account does not own Minecraft: Java Edition. |
| `ErrPlatformNotSupported` | The platform is not supported by the feature. |
| `ErrInvalidVanillaLauncherProfile` | The profile of the official launcher is incomplete. |

The errors that carry data are types, so `errors.As` finds them:

```go
var checksumError *mcl.InvalidChecksumError
var outsideError *mcl.FileOutsideMinecraftDirectoryError
var httpError *mcl.HTTPError
var programError *mcl.ExternalProgramError
```

| Type | Fields |
| --- | --- |
| `InvalidChecksumError` | `URL`, `Path`, `Expected`, `Actual` |
| `FileOutsideMinecraftDirectoryError` | `Path`, `MinecraftDirectory` |
| `HTTPError` | `URL`, `Status`, `Retryable` |
| `ExternalProgramError` | `Command`, `Output`, `Err` |

## The addresses

The library talks to these servers. They are exported so a caller can see
where the data comes from.

| Constant | Server |
| --- | --- |
| `ManifestURL` | The version manifest of Mojang |
| `ResourcesURL` | The assets of the game |
| `LibrariesURL` | The default Maven repository of the libraries |
| `AdoptiumURL` | The Java runtimes |
| `JVMManifestURL` | The Java runtimes of Mojang |
| `ForgeMavenURL` | The versions of Forge |
| `NeoForgeMavenURL` | The versions of NeoForge |
| `NeoForgeAPIURL` | The released versions of NeoForge |
| `MSAAuthURL`, `MSATokenURL`, `MSARefreshURL` | The login of Microsoft |
| `NewsURL`, `JavaPatchNotesURL` | The launcher content of Mojang |
