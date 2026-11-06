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
