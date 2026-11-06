# Roadmap

The port follows the Python library feature by feature. Every feature is
implemented, tested and documented before the next one is started.

## Status

| Phase | Content | Status |
| --- | --- | --- |
| 1 | Analysis of the Python library, inventory of the features | done |
| 2 | Project setup, Go module, documentation structure | done |
| 3 | Version metadata, version manifest, rules | done |
| 4 | Installer for libraries, assets, natives and the client jar | done |
| 5 | Launch command for modern and legacy versions | done |
| 6 | Java runtime (Temurin) | done |
| 7 | Mod loaders: Forge, NeoForge, Fabric, Quilt | done |
| 8 | Microsoft account login | done |
| 9 | Modrinth modpacks | done |
| 10 | Profiles of the official launcher | done |
| 11 | News, Java discovery, platform helpers | done |
| 12 | Documentation and examples | done |

## Planned

### Command execution helper

The library builds the launch command but leaves the execution to the caller.
A small helper that starts the game, streams its output and waits for it to
exit would be convenient. It is deliberately left out for now: every launcher
has its own idea of how the output should be displayed, and `os/exec` is only
a few lines of code.

### Cancellation of a running installation

`context.Context` already cancels a download. Cancelling an installation that
is already writing files could leave a half written installation behind; the
files are verified and repaired on the next run, so this is acceptable for
now.

### Caching of the version manifest on disk

The manifest is cached in memory for an hour. A cache on disk would let a
launcher work without an internet connection, at the cost of a second source
of truth that has to be invalidated.

## Design principles

- **The standard library first.** The library has no external dependencies.
- **Errors instead of panics.** Every failure is an `error`; the typed ones
  can be inspected with `errors.As`.
- **Small public API.** Only the functions a launcher needs are exported.
- **The Python library is the reference.** Behaviour is ported, not invented.
  Intentional differences are listed in [MIGRATION.md](MIGRATION.md).
- **One feature per commit.** Every commit builds, is tested and passes
  `go vet`.
