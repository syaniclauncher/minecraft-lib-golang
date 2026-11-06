# Modpacks

The library installs modpacks in the Modrinth format (`.mrpack`).

## Reading a modpack

```go
information, err := mcl.GetMrpackInformation(path)
if err != nil {
	log.Fatal(err)
}

fmt.Println(information.Name)
fmt.Println(information.Summary)
fmt.Println("for Minecraft", information.MinecraftVersion)
fmt.Println("optional files:", information.OptionalFiles)
```

## Installing a modpack

```go
err := mcl.InstallMrpack(ctx, path, minecraftDirectory, "", mcl.MrpackInstallOptions{}, progress)
```

The fourth argument is the directory the modpack is installed into. An empty
string uses the Minecraft directory. The `mcl.MrpackInstallOptions` argument
holds the optional files that should be installed:

```go
err := mcl.InstallMrpack(ctx, path, minecraftDirectory, "", mcl.MrpackInstallOptions{
	OptionalFiles: []string{"mods/optional.jar"},
}, progress)
```

The installation downloads the files of the modpack, extracts the overrides
and installs the Minecraft version and the mod loader the modpack asks for.

## Launching a modpack

The version of the modpack depends on the mod loader it uses:

```go
version, err := mcl.MrpackLaunchVersion(path)
if err != nil {
	log.Fatal(err)
}

command, err := mcl.Command(version, minecraftDirectory, options)
```

| Mod loader | Version id |
| --- | --- |
| Forge | `<minecraft version>-forge-<forge version>` |
| NeoForge | `neoforge-<neoforge version>` |
| Fabric | `fabric-loader-<loader version>-<minecraft version>` |
| Quilt | `quilt-loader-<loader version>-<minecraft version>` |
| none | `<minecraft version>` |

## Installing into another directory

A modpack can be installed into a directory of its own, while the Minecraft
version and the mod loader go into the Minecraft directory:

```go
err := mcl.InstallMrpack(ctx, path, minecraftDirectory, "/home/user/games/modpack", mcl.MrpackInstallOptions{}, progress)
```

The files of the modpack are then in `/home/user/games/modpack` and the
version of the modpack still lives in the Minecraft directory.
