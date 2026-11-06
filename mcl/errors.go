package mcl

import (
	"errors"
	"strings"
)

// Sentinel errors returned by the library. Use errors.Is to test for them.
var (
	// ErrVersionNotFound is returned when the requested Minecraft version
	// does not exist, neither locally nor in Mojang's version manifest.
	ErrVersionNotFound = errors.New("version not found")

	// ErrUnsupportedVersion is returned when a mod loader does not support
	// the requested Minecraft version.
	ErrUnsupportedVersion = errors.New("unsupported version")

	// ErrInvalidRefreshToken is returned when a stored refresh token is
	// rejected by Microsoft.
	ErrInvalidRefreshToken = errors.New("invalid refresh token")

	// ErrAzureAppNotPermitted is returned when the configured Azure app is
	// not allowed to use the Minecraft API.
	ErrAzureAppNotPermitted = errors.New("azure app is not permitted to use the Minecraft API")

	// ErrAccountNotOwnMinecraft is returned when the authenticated account
	// does not own Minecraft: Java Edition.
	ErrAccountNotOwnMinecraft = errors.New("account does not own Minecraft")

	// ErrPlatformNotSupported is returned when the current platform is not
	// supported by the requested feature.
	ErrPlatformNotSupported = errors.New("platform not supported")

	// ErrInvalidVanillaLauncherProfile is returned when a vanilla launcher
	// profile is missing required fields.
	ErrInvalidVanillaLauncherProfile = errors.New("invalid vanilla launcher profile")
)

// InvalidChecksumError is returned when a downloaded file does not match the
// checksum that the server advertised.
type InvalidChecksumError struct {
	// URL is the source the file was downloaded from.
	URL string
	// Path is the file that was written.
	Path string
	// Expected is the checksum the server advertised.
	Expected string
	// Actual is the checksum of the downloaded file.
	Actual string
}

func (e *InvalidChecksumError) Error() string {
	return "invalid checksum for " + e.Path + ": expected " + e.Expected + ", got " + e.Actual
}

// FileOutsideMinecraftDirectoryError is returned when a file would be written
// outside of the Minecraft directory. It guards against a malicious metadata
// file escaping into the rest of the filesystem.
type FileOutsideMinecraftDirectoryError struct {
	// Path is the offending path.
	Path string
	// MinecraftDirectory is the directory the path had to stay inside.
	MinecraftDirectory string
}

func (e *FileOutsideMinecraftDirectoryError) Error() string {
	return "tried to place " + e.Path + " outside " + e.MinecraftDirectory
}

// HTTPError is returned when a request fails with an error status.
type HTTPError struct {
	// URL is the address that was requested.
	URL string
	// Status is the status the server answered with.
	Status string
	// Retryable reports whether the request is worth another try.
	Retryable bool
}

func (e *HTTPError) Error() string {
	return "GET " + e.URL + ": " + e.Status
}

// ExternalProgramError is returned when an external program (the Forge or
// NeoForge installer) exits with a failure.
type ExternalProgramError struct {
	// Command is the command line that was executed.
	Command []string
	// Output is the combined standard output and standard error.
	Output string
	// Err is the underlying error, if any.
	Err error
}

func (e *ExternalProgramError) Error() string {
	msg := "external program failed: " + strings.Join(e.Command, " ")
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

func (e *ExternalProgramError) Unwrap() error { return e.Err }
