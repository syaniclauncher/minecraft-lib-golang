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

The documentation is an [mdBook](https://rust-lang.github.io/mdBook/) that
lives in [docs](docs). It is published to GitHub Pages by a workflow, so it
can also be read in the browser without building it.

| Guide | Topic |
| --- | --- |
| [Introduction](docs/index.md) | What the library does and how it is put together |
| [Installation](docs/installation.md) | Adding the library to a project |
| [Getting started](docs/getting-started.md) | Install a version and launch the game |
| [Installation progress](docs/installation-progress.md) | Follow the progress of an installation |
| [Launch options](docs/launch-options.md) | Every option of the launch command |
| [Mod loaders](docs/mod-loaders.md) | Forge, NeoForge, Fabric and Quilt |
| [Microsoft login](docs/microsoft-login.md) | Authenticate an account |
| [Modpacks](docs/modpacks.md) | Install a Modrinth modpack |
| [Java installations](docs/java.md) | Find and inspect Java |
| [API reference](docs/api-reference.md) | Every exported function, type and error |
| [Migration guide](docs/migration-guide.md) | Moving from the Python library |
| [Troubleshooting](docs/troubleshooting.md) | When something does not work |

To build the book yourself, install [mdBook](https://rust-lang.github.io/mdBook/)
and run:

```bash
mdbook serve      # a preview on http://localhost:3000
mdbook build      # the result in the book directory
```

The workflow in `.github/workflows/docs.yml` builds the book and publishes it
to GitHub Pages. It needs the pages of the repository to be enabled with the
source "GitHub Actions".

## Project documents

- [ROADMAP.md](ROADMAP.md) — the implementation roadmap.
- [MIGRATION.md](MIGRATION.md) — the feature mapping of the Python library.
- [TODO.md](TODO.md) — open tasks and known limitations.

## License

BSD 2-Clause, like the Python library it is based on.
