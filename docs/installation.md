# Installation

The library is a normal Go module. It has no external dependencies, so
`go.mod` stays clean.

```bash
go get github.com/syaniclauncher/minecraft-lib-golang
```

## Requirements

- Go 1.24 or newer.
- Windows, macOS or Linux. Use `mcl.IsPlatformSupported` to check.

## The import

```go
import "github.com/syaniclauncher/minecraft-lib-golang/mcl"
```

Everything is in the package `mcl`, so a single import is enough.

## Verifying the installation

```go
package main

import (
	"fmt"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

func main() {
	fmt.Println("the Minecraft directory is", mcl.DefaultMinecraftDirectory())
	fmt.Println("the Java executable is", mcl.DefaultJavaExecutable())
}
```

Run it with:

```bash
go run .
```

## Skipping verification and repair

`Install` verifies an existing installation and downloads missing or corrupted
files. To skip that work for an already-installed version, use
`InstallWithOptions`:

```go
err := mcl.InstallWithOptions(ctx, "1.21", directory, mcl.MinecraftInstallOptions{
	SkipChecks: true,
})
```

`SkipChecks` only applies when the version metadata is already present; fresh
installations are still installed. Use it only when you know the version's
runtime, assets and libraries are complete, since missing files will not be
repaired.
