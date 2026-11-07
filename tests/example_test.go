package mcl_test

import (
	"context"
	"fmt"
	"log"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

// This example installs a Minecraft version and builds the command that
// starts it.
func Example() {
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

	fmt.Println(command[0])
}

// This example installs a mod loader.
func Example_modLoader() {
	ctx := context.Background()
	directory := mcl.DefaultMinecraftDirectory()

	loader, err := mcl.GetModLoader("fabric")
	if err != nil {
		log.Fatal(err)
	}

	installed, err := loader.Install(ctx, "1.20.1", directory, mcl.InstallOptions{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(installed)
}

// This example prints the versions that are installed in the Minecraft
// directory.
func Example_installedVersions() {
	versions, err := mcl.InstalledVersions(mcl.DefaultMinecraftDirectory())
	if err != nil {
		log.Fatal(err)
	}

	for _, version := range versions {
		fmt.Println(version.ID, version.Type)
	}
}
