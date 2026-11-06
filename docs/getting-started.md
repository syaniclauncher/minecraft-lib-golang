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

On Windows the same directory is:

```go
directory := `C:\Users\user\games\minecraft`
```

The library keeps everything inside that directory:

```text
/home/user/games/minecraft
├── assets
│   ├── indexes
│   └── objects
├── libraries
├── runtime
│   └── temurin-21
│       └── linux
│           └── jdk-21.0.3+7
└── versions
    └── 1.21
        ├── 1.21.jar
        ├── 1.21.json
        └── natives
```

On Windows the layout is the same, only the separator and the name of the
runtime folder differ:

```text
C:\Users\user\games\minecraft
├── assets
│   ├── indexes
│   └── objects
├── libraries
├── runtime
│   └── temurin-21
│       └── windows-x64
│           └── jdk-21.0.3+7
└── versions
    └── 1.21
        ├── 1.21.jar
        ├── 1.21.json
        └── natives
```

The `natives` folder holds the native libraries of the game and the `runtime`
folder the Java installations that were downloaded for it. The library builds
every path with `filepath.Join`, so it always uses the separator of the
platform it runs on. Do the same when you build a path of your own:

```go
import (
	"os"
	"path/filepath"
)

home, err := os.UserHomeDir()
if err != nil {
	log.Fatal(err)
}
directory := filepath.Join(home, "games", "minecraft")
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

On Windows the executable of a Java installation is `javaw.exe`, the variant
that runs without a console window:

```go
command, err := mcl.Command("1.21", directory, mcl.Options{
	Username:      "Player",
	UUID:          "00000000-0000-4000-8000-000000000000",
	ExecutablePath: `C:\Program Files\Java\jdk-21\bin\javaw.exe`,
})
```

`DefaultExecutablePath` is used when `ExecutablePath` is empty and the version
has no embedded runtime. `mcl.DefaultJavaExecutable` returns the executable of
the Java installation of the system, so a launcher that has no Java to pick
from can use it as the `DefaultExecutablePath`. On Windows it is a
`javaw.exe`, on Linux and macOS a `java`.

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
save games, so it is usually the game directory. `mcl.Command` creates the
Minecraft directory and the game directory when they do not exist yet, so the
game finds its working directory.
