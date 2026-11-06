# minecraft-launcher-lib-go

A Go library for building a custom Minecraft launcher.

It is a port of the Python library [minecraft-launcher-lib](https://github.com/syaniclauncher/minecraft-launcher-lib).
The Python library is the behavioural reference, so the Go library offers the
same features through an idiomatic Go API.

## Features

- Install Minecraft versions: libraries, assets, natives, the client jar and a
  matching Java runtime.
- Build the launch command for modern and legacy versions.
- Install the mod loaders Forge, NeoForge, Fabric and Quilt.
- Install Modrinth modpacks (`.mrpack`).
- Log in with a Microsoft account, with PKCE and token refresh.
- Read and write the profiles of the official launcher.
- Find the Java installations of the system.
- Read the news and the patch notes of Minecraft.

The library has **no external dependencies**; everything is built on the Go
standard library.

## Installation

```bash
go get github.com/syaniclauncher/minecraft-launcher-lib-go
```

## Quick start

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

`mcl.Options` without a token starts the game in demo mode. Use the
[`microsoft-login`](docs/microsoft-login.md) guide to obtain a real token.

## Documentation

The documentation lives in [docs](docs):

| Guide | Topic |
| --- | --- |
| [Getting started](docs/getting-started.md) | Install a version and launch the game |
| [Installation progress](docs/installation-progress.md) | Follow the progress of an installation |
| [Launch options](docs/launch-options.md) | Every option of the launch command |
| [Mod loaders](docs/mod-loaders.md) | Forge, NeoForge, Fabric and Quilt |
| [Microsoft login](docs/microsoft-login.md) | Authenticate an account |
| [Modpacks](docs/modpacks.md) | Install a Modrinth modpack |
| [Migration guide](docs/migration-guide.md) | Moving from the Python library |
| [Troubleshooting](docs/troubleshooting.md) | When something does not work |

## Project documents

- [ROADMAP.md](ROADMAP.md) — the implementation roadmap.
- [MIGRATION.md](MIGRATION.md) — the feature mapping of the Python library.
- [TODO.md](TODO.md) — open tasks and known limitations.

## License

BSD 2-Clause, like the Python library it is based on.
