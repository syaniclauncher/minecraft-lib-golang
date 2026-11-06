// Command launch installs a Minecraft version and starts it.
//
// Usage:
//
//	go run ./examples/launch [version]
//
// The version defaults to the latest release. The game is started with the
// test options, so it runs in demo mode.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

func main() {
	ctx := context.Background()
	directory := mcl.DefaultMinecraftDirectory()

	version := "latest release"
	if len(os.Args) > 1 {
		version = os.Args[1]
	}
	if version == "latest release" {
		latest, err := mcl.LatestVersions(ctx)
		if err != nil {
			log.Fatal(err)
		}
		version = latest.Release
	}

	progress := &mcl.Progress{
		Status: func(message string) { fmt.Println(message) },
	}
	if err := mcl.Install(ctx, version, directory, progress); err != nil {
		log.Fatal(err)
	}

	command, err := mcl.Command(version, directory, mcl.TestOptions())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("starting:", command)

	game := exec.Command(command[0], command[1:]...)
	game.Dir = directory
	game.Stdout = os.Stdout
	game.Stderr = os.Stderr
	if err := game.Run(); err != nil {
		log.Fatal(err)
	}
}
