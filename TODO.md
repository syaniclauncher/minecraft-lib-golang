# Known limitations and open tasks

## Limitations

- **Only Minecraft: Java Edition.** The Bedrock Edition is not supported, the
  same as in the Python library.
- **Windows, macOS and Linux.** These are the platforms Mojang supports.
  `mcl.IsPlatformSupported` reports whether the current platform is one of
  them.
- **The Mojang Java runtime is not installed.** Only the Temurin runtime of
  the Adoptium API is supported. The Python library keeps the Mojang runtime
  installer as a commented out block; it is not ported.
- **The store information of an account is not exposed.** The Python library
  has a function for it, it is only useful for a store browser and was left
  out.
- **The Java version of a system installation is detected by running it.**
  `mcl.JavaInformation` executes the java executable, which is a security
  risk when the path is not trusted. The Python library does the same.
- **macOS is not searched for Java installations.**
  `mcl.FindSystemJavaVersions` only looks at the usual directories of Windows
  and Linux.

## Open tasks

- [ ] Example programs under `examples/` that use the library.
- [ ] A test that runs the installer against the real Mojang servers. It
      needs network access and is therefore not part of `go test ./...`.
- [ ] Benchmark the download of a version against the Python implementation.
- [ ] Check the installation of a mod loader against the real Forge, NeoForge,
      Fabric and Quilt servers.
- [ ] Release tags and a changelog.

## Deliberate simplifications

- **The version manifest is cached in memory for an hour.** The Python
  library caches every response it fetches, the Go library only caches the
  documents that are fetched repeatedly: the version manifest, the Maven
  metadata and the version lists of the mod loaders.
- **The rule evaluation ignores unknown operating system names.** Only
  `windows`, `osx` and `linux` are known, everything else is ignored. The
  Python library behaves the same way.
- **The path guard uses `filepath.Rel` instead of a string prefix.** The
  Python library compares the strings, which lets a sibling directory with a
  common prefix pass. The Go library does it correctly.
