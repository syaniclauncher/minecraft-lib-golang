// Command modloader installs a mod loader and prints the command that starts
// the game.
//
// Usage:
//
//	go run ./examples/modloader <forge|neoforge|fabric|quilt> <minecraft version>
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: modloader <forge|neoforge|fabric|quilt> <minecraft version>")
	}
	ctx := context.Background()
	directory := mcl.DefaultMinecraftDirectory()

	loader, err := mcl.GetModLoader(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	progress := &mcl.Progress{
		Status: func(message string) { fmt.Println(message) },
	}
	installed, err := loader.Install(ctx, os.Args[2], directory, mcl.InstallOptions{
		Progress: progress,
	})
	if err != nil {
		log.Fatal(err)
	}

	command, err := mcl.Command(installed, directory, mcl.TestOptions())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("the mod loader was installed as", installed)
	fmt.Println("start the game with:", command)
}
