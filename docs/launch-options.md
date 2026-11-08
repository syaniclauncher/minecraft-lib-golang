# Launch options

`mcl.Options` holds everything the launch command needs to know. Only the
name, the UUID and the token belong to the player, everything else configures
the launcher.

```go
command, err := mcl.Command(version, minecraftDirectory, mcl.Options{
	Username: "Player",
	UUID:     "00000000-0000-4000-8000-000000000000",
	Token:    "the access token",
})
```

An empty string means "not set". The library uses its own default for every
option that is not set.

## The player

| Field | Meaning |
| --- | --- |
| `Username` | The name of the player in the game. |
| `UUID` | The UUID of the player. An empty UUID is generated from the name, see `mcl.OfflineUUID`. |
| `Token` | The Minecraft access token. An empty token starts the game in demo mode. |

`mcl.TestOptions` returns options with a random name and a random UUID. They
are meant for development only.

`mcl.OfflineUUID` returns the UUID of a player that plays without an account:

```go
uuid := mcl.OfflineUUID("Player")
```

It is the same UUID the official launcher uses for such a player i.e. an MD5-based Version 3 UUID (RFC 4122 variant) of `OfflinePlayer:<username>`, used by Minecraft for offline-mode player identification.

## The Java executable

| Field | Meaning |
| --- | --- |
| `ExecutablePath` | The Java executable to run. It wins over everything else. |
| `DefaultExecutablePath` | The Java executable to run when `ExecutablePath` is empty and the version has no embedded runtime. |
| `JVMArguments` | Java arguments that are put before the game's arguments. Each slice element is passed as one argument, for example `[]string{"-Xmx2G"}`. |

Pass one argument per slice element; do not combine arguments into a
space-separated string or add shell quotes. Spaces in an argument are kept as
is, so a `-javaagent` option whose JAR path contains spaces should be one
element:

```go
JVMArguments: []string{
	"-Xms1G",
	"-Xmx4G",
	"-Dfile.encoding=UTF-8",
	"-XX:+UseG1GC",
	`-javaagent:C:\path with spaces\agent.jar=option`,
},
```

Each flag and its value stays in a single slice element (for example,
`-Dname=value`). The raw Go string literal keeps Windows path backslashes
unchanged. Replace the example agent path and option as needed.

Without both paths the game runs with the Temurin runtime that was installed
next to the Minecraft directory. That runtime matches the Java version the
game needs, so it is the recommended default.

## The directories

| Field | Meaning |
| --- | --- |
| `GameDirectory` | The directory the game runs in. Defaults to the Minecraft directory. |
| `NativesDirectory` | The directory with the extracted native libraries. Defaults to `<minecraft directory>/versions/<version>/natives`. |

The library joins the parts of those directories with `filepath.Join`, so the
separator is the one of the platform the launcher runs on. The natives
directory of a version is:

```text
/home/user/.minecraft/versions/1.21/natives
C:\Users\user\AppData\Roaming\.minecraft\versions\1.21\natives
```

`mcl.Command` creates the Minecraft directory and the game directory when they
do not exist yet. The game writes its saves and its logs into the game
directory, so it has to be there when the game starts.

## The window

| Field | Meaning |
| --- | --- |
| `CustomResolution` | Use `ResolutionWidth` and `ResolutionHeight` instead of the size of the window. |
| `ResolutionWidth` | The width of the window. Defaults to `854`. |
| `ResolutionHeight` | The height of the window. Defaults to `480`. |

The resolution is only applied to versions older than 1.13. Newer versions
carry the arguments in their metadata and use them when `CustomResolution` is
set.

## The server

| Field | Meaning |
| --- | --- |
| `Server` | The address the game connects to right after it starts. |
| `Port` | The port of the server. Only used when `Server` is set. |

## The switches

| Field | Meaning |
| --- | --- |
| `Demo` | Start the game in demo mode. |
| `DisableMultiplayer` | Pass `--disableMultiplayer` to the game. |
| `DisableChat` | Pass `--disableChat` to the game. |
| `EnableLoggingConfig` | Pass the log4j configuration of the version to the game. |
| `LauncherName` | The name the game reports. Defaults to `minecraft-launcher-lib`. |
| `LauncherVersion` | The version the game reports. Defaults to the version of this library. |

## Quick play

| Field | Meaning |
| --- | --- |
| `QuickPlayPath` | The save the game quick plays into. |
| `QuickPlaySingleplayer` | The save the game quick plays into. |
| `QuickPlayMultiplayer` | The server the game quick plays into. |
| `QuickPlayRealms` | The realm the game quick plays into. |

Quick play needs a Minecraft version that supports it, which is every version
from 1.20 onwards.

## The profiles of the official launcher

The profiles of the official launcher can be turned into options:

```go
profiles, err := mcl.LoadVanillaLauncherProfiles(minecraftDirectory)
if err != nil {
	log.Fatal(err)
}

for _, profile := range profiles {
	options, err := mcl.VanillaLauncherProfileToOptions(profile)
	if err != nil {
		log.Fatal(err)
	}
	options.Username = account.Name
	options.UUID = account.ID
	options.Token = account.AccessToken

	version, err := mcl.VanillaLauncherProfileVersion(ctx, profile)
	if err != nil {
		log.Fatal(err)
	}
	command, err := mcl.Command(version, minecraftDirectory, options)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(command)
}
```
