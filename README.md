# minecraft-lib-golang

A Go library for building a custom Minecraft launcher.

It is a port of the Python library [minecraft-launcher-lib](https://github.com/syaniclauncher/minecraft-launcher-lib).
The Python library is the behavioural reference, so the Go library offers the
same features through an idiomatic Go API.

Read the docs [here](https://syaniclauncher.github.io/minecraft-lib-golang/).

## Features

- Install Minecraft versions: libraries, assets, natives, the client jar and a
  matching Java runtime.
- Build the launch command for modern and old versions.
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
go get github.com/syaniclauncher/minecraft-lib-golang
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
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

## Examples

The [examples](examples) folder holds small programs that show how the library
is used.

| Example | What it does |
| --- | --- |
| [launch](examples/launch) | Installs a Minecraft version and starts it |
| [login](examples/login) | Logs in with a Microsoft account |
| [modloader](examples/modloader) | Installs a mod loader and prints the launch command |
| [launcher](examples/launcher) | A desktop launcher with a window |

The launcher is a module of its own, because a window needs a GUI toolkit.
Run it from its own folder:

```bash
cd examples/launcher
go run .
```

## Documentation

The documentation is an [mdBook](https://rust-lang.github.io/mdBook/) that
lives in [docs](docs). 

You can read the docs from [here](https://syaniclauncher.github.io/minecraft-lib-golang/).

## Project documents (used by Claude)

- [ROADMAP.md](ROADMAP.md) — the implementation roadmap.
- [MIGRATION.md](MIGRATION.md) — the feature mapping of the Python library.
- [TODO.md](TODO.md) — open tasks and known limitations.

## License

BSD 2-Clause
