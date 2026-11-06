# Documentation

`minecraft-launcher-lib-go` is a Go library for building a custom Minecraft
launcher. It is a port of the Python library
[minecraft-launcher-lib](https://github.com/syaniclauncher/minecraft-launcher-lib)
and offers the same features through an idiomatic Go API.

## Guides

| Guide | What it covers |
| --- | --- |
| [Installation](installation.md) | Adding the library to a project |
| [Getting started](getting-started.md) | Installing a version and launching the game |
| [Installation progress](installation-progress.md) | Showing the progress of an installation |
| [Launch options](launch-options.md) | Every option of the launch command |
| [Mod loaders](mod-loaders.md) | Forge, NeoForge, Fabric and Quilt |
| [Microsoft login](microsoft-login.md) | Authenticating an account |
| [Modpacks](modpacks.md) | Installing a Modrinth modpack |
| [Java installations](java.md) | Finding and inspecting Java |
| [Migration guide](migration-guide.md) | Moving from the Python library |
| [Troubleshooting](troubleshooting.md) | When something does not work |

## The shape of the API

Everything lives in one package, `mcl`:

```go
import "github.com/syaniclauncher/minecraft-launcher-lib-go/mcl"
```

The package is split by topic, not by sub package, because the parts build on
each other: the installer needs the metadata, the launch command needs the
installer and the mod loaders need both.

| Area | Functions |
| --- | --- |
| Versions | `Manifest`, `LatestVersions`, `VersionList`, `InstalledVersions`, `AvailableVersions`, `VersionMetadata` |
| Installing | `Install`, `InstallJVMRuntime`, `ExtractNatives` |
| Launching | `Command`, `Options`, `Progress` |
| Mod loaders | `GetModLoader`, `ListModLoaders`, `ModLoader`, `Forge`, `NeoForge`, `Fabric`, `Quilt` |
| Accounts | `LoginURL`, `SecureLoginData`, `CompleteLogin`, `CompleteRefresh`, `Account` |
| Modpacks | `InstallMrpack`, `GetMrpackInformation`, `MrpackLaunchVersion` |
| Official launcher | `LoadVanillaLauncherProfiles`, `AddVanillaLauncherProfile`, … |
| Java | `JavaInformation`, `FindSystemJavaVersions`, `DefaultJavaExecutable` |
| News | `GetMinecraftNews`, `GetJavaPatchNotes` |

## Conventions

- **Errors, not panics.** Every failure is an `error`. The errors that carry
  data are typed, so `errors.As` can be used on them.
- **Contexts for the network.** Every function that talks to a server takes a
  `context.Context` as its first argument.
- **Nil is allowed.** A nil `*mcl.Progress` discards every update.
- **Empty means unset.** The optional fields of `mcl.Options` are strings;
  an empty string means "not set".
