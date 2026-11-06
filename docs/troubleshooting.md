# Troubleshooting

## The game does not start

1. **Look at the error of the process.** `mcl.Command` only builds the
   command, the game itself is started by the caller. The output of the game
   is on its standard output and standard error.
2. **Check the working directory.** Minecraft writes its logs and its save
   games into the working directory of the process. Start the game in the
   game directory.
3. **Check the Java version.** The game needs the Java version that its
   metadata asks for. `mcl.VersionRuntimeInformation` tells you which one it
   is:

   ```go
   information, err := mcl.VersionRuntimeInformation("1.21", minecraftDirectory)
   if err != nil {
       log.Fatal(err)
   }
   if information != nil {
       fmt.Println("the game needs Java", information.JavaMajorVersion)
   }
   ```

   The runtime that was installed next to the Minecraft directory matches it.

## The installation fails with a checksum error

```go
var checksumError *mcl.InvalidChecksumError
if errors.As(err, &checksumError) {
	fmt.Println("the file", checksumError.Path, "does not match the checksum")
	fmt.Println("expected:", checksumError.Expected)
	fmt.Println("got:", checksumError.Actual)
}
```

The download was corrupted. Run the installation again, it downloads the file
again.

## The installation fails with "file outside of the Minecraft directory"

```go
var outsideError *mcl.FileOutsideMinecraftDirectoryError
if errors.As(err, &outsideError) {
	fmt.Println(outsideError.Path, "is not inside", outsideError.MinecraftDirectory)
}
```

A metadata file tried to write outside of the Minecraft directory. That is
never legitimate, so the library refuses it. It happens with a mod loader
whose metadata was tampered with.

## The login fails with "azure app is not permitted"

The Azure app of the launcher does not have the permission to use the
Minecraft API. See the [Microsoft login](microsoft-login.md) guide.

## The login fails with "account does not own Minecraft"

The account was logged in successfully, but it does not own Minecraft: Java
Edition.

## The game starts in demo mode

The token is missing or empty. A real token comes from
`mcl.CompleteLogin` or `mcl.CompleteRefresh`.

## The version is not found

```go
if errors.Is(err, mcl.ErrVersionNotFound) {
	fmt.Println("there is no such version")
}
```

The version is neither installed nor offered by Mojang.

## The mod loader does not support the version

```go
if errors.Is(err, mcl.ErrUnsupportedVersion) {
	fmt.Println("the mod loader does not support this Minecraft version")
}
```

`mcl.ModLoader.MinecraftVersions` lists the versions that are supported.

## The game creates a logs folder in the project directory

Minecraft writes its logs into the working directory of the process. Start
the game in the game directory.

## Nothing happens during the installation

The installation only downloads what is missing. When everything is already
there, it finishes immediately. Use the `Progress` callbacks to show what it
is doing.
