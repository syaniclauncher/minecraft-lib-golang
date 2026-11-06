# Getting started

This guide installs a Minecraft version and starts the game.

## 1. Install a version

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/syaniclauncher/minecraft-launcher-lib-go/mcl"
)

func main() {
	ctx := context.Background()
	directory := mcl.DefaultMinecraftDirectory()

	if err := mcl.Install(ctx, "1.21", directory, nil); err != nil {
		log.Fatal(err)
	}

	command, err := mcl.Command("1.21", directory, mcl.Options{
		Username: "Player",
		UUID:     "00000000-0000-4000-8000-000000000000",
		Token:    "",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(command)
	exec.Command(command[0], command[1:]...).Run()
}
```

`mcl.Install` downloads the libraries, the assets, the natives, the client jar
and a matching Java runtime. It verifies an existing installation and only
downloads what is missing or broken, so it is safe to call it on every start.

`mcl.Command` returns the command line as a `[]string`. The first entry is the
Java executable, the rest are the arguments of the game. The example uses an
empty token, which starts the game in demo mode. Use the
[Microsoft login](microsoft-login.md) guide to get a real token.

## 2. Pick the version

Mojang publishes a manifest with every version it offers:

```go
latest, err := mcl.LatestVersions(ctx)
if err != nil {
	log.Fatal(err)
}
fmt.Println("latest release:", latest.Release)
fmt.Println("latest snapshot:", latest.Snapshot)

versions, err := mcl.VersionList(ctx)
if err != nil {
	log.Fatal(err)
}
for _, version := range versions {
	fmt.Println(version.ID, version.Type, version.ReleaseTime)
}
```

The versions that are already installed:

```go
installed, err := mcl.InstalledVersions(directory)
```

And the versions that are installed plus the ones Mojang offers:

```go
available, err := mcl.AvailableVersions(ctx, directory)
```

## 3. Use another Minecraft directory

Every function that works with an installation takes the directory as its
first argument. Use a directory of your own to keep the installation
separate from the one of the official launcher:

```go
directory := "/home/user/games/minecraft"
if err := mcl.Install(ctx, "1.21", directory, nil); err != nil {
	log.Fatal(err)
}
```

## 4. Choose the Java executable

By default the game runs with the Java runtime that was installed next to the
Minecraft directory. That runtime matches the Java version the game needs.

To use another Java installation, set `ExecutablePath`:

```go
command, err := mcl.Command("1.21", directory, mcl.Options{
	Username:      "Player",
	UUID:          "00000000-0000-4000-8000-000000000000",
	ExecutablePath: "/usr/lib/jvm/java-21/bin/java",
})
```

`DefaultExecutablePath` is used when `ExecutablePath` is empty and the version
has no embedded runtime.

## 5. Start the game

`mcl.Command` only builds the command. Starting it is up to the caller, so
the output of the game can be shown however the launcher wants:

```go
cmd := exec.Command(command[0], command[1:]...)
cmd.Dir = directory
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
if err := cmd.Run(); err != nil {
	log.Fatal(err)
}
```

The working directory of the game decides where it writes its logs and its
save games, so it is usually the game directory.
