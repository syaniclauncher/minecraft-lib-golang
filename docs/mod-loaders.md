# Mod loaders

The library supports Forge, NeoForge, Fabric and Quilt. They share one
interface, so the code that installs a mod loader does not have to know which
one it is.

```go
loader, err := mcl.GetModLoader("fabric")
if err != nil {
	log.Fatal(err)
}
```

`mcl.ListModLoaders` returns the ids of all of them: `forge`, `neoforge`,
`fabric` and `quilt`.

## The interface

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

## Installing a mod loader

```go
ctx := context.Background()
directory := mcl.DefaultMinecraftDirectory()

loader, err := mcl.GetModLoader("fabric")
if err != nil {
	log.Fatal(err)
}

installed, err := loader.Install(ctx, "1.20.1", directory, mcl.InstallOptions{
	Progress: progress,
})
if err != nil {
	log.Fatal(err)
}

fmt.Println("the mod loader was installed as", installed)
```

The installation takes care of the whole chain: the Minecraft version is
installed, the mod loader is installed and the version of the mod loader is
installed on top of it. The returned version id is the one that has to be
passed to `mcl.Command`:

```go
command, err := mcl.Command(installed, directory, options)
```

## Choosing a version

Without a version the newest one is used:

```go
version, err := loader.LatestLoaderVersion(ctx, "1.20.1")
```

To pick a version yourself, list them and pass the one you want:

```go
versions, err := loader.LoaderVersions(ctx, "1.20.1", true)
if err != nil {
	log.Fatal(err)
}

installed, err := loader.Install(ctx, "1.20.1", directory, mcl.InstallOptions{
	LoaderVersion: versions[0],
	Progress:      progress,
})
```

`stableOnly` filters the list. Not every mod loader uses it the same way: for
Fabric and Quilt a version is stable when the meta API says so and when its
name does not contain `beta`; for NeoForge the versions with `beta` in their
name are left out.

## The Minecraft versions of a mod loader

```go
versions, err := loader.MinecraftVersions(ctx, false)
if err != nil {
	log.Fatal(err)
}

for _, version := range versions {
	fmt.Println(version)
}
```

## The Java of the installer

Forge and NeoForge run their installers with Java. By default the `java` of
the `PATH` is used. Set `Java` to use another one:

```go
installed, err := loader.Install(ctx, "1.20.1", directory, mcl.InstallOptions{
	Java: "/usr/lib/jvm/java-17/bin/java",
})
```

On Windows the Java executable of the installer is a Windows path:

```go
installed, err := loader.Install(ctx, "1.20.1", directory, mcl.InstallOptions{
	Java: `C:\Program Files\Java\jdk-17\bin\java.exe`,
})
```

## How the version id is built

| Mod loader | Version id |
| --- | --- |
| Forge | `<minecraft version>-forge-<loader version>` |
| NeoForge | `neoforge-<loader version>` |
| Fabric | `fabric-loader-<loader version>-<minecraft version>` |
| Quilt | `quilt-loader-<loader version>-<minecraft version>` |

The same id is used by `mcl.MrpackLaunchVersion` for the modpack format.
