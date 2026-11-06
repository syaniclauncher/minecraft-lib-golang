package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"

	"github.com/syaniclauncher/minecraft-launcher-lib-go/mcl"
)

// versions holds the version lists that the window shows.
type versions struct {
	// releases and snapshots are the versions Mojang offers.
	releases  []string
	snapshots []string
	// local are the versions that are already installed.
	local []string

	// modLoaders holds the mod loaders by their id.
	modLoaders map[string]mcl.ModLoader
	// minecraft holds the Minecraft versions every mod loader supports.
	minecraft map[string][]string
	// loader holds the versions of Fabric and Quilt, which are the same for
	// every Minecraft version.
	loader map[string][]string
	// loaderByMinecraft holds the versions of Forge and NeoForge for every
	// Minecraft version.
	loaderByMinecraft map[string]map[string][]string
}

// loadVersions collects the versions of the game and of the mod loaders. It
// talks to several servers, so it takes a while.
func loadVersions(ctx context.Context, minecraftDirectory string) (*versions, error) {
	list, err := mcl.VersionList(ctx)
	if err != nil {
		return nil, err
	}
	data := &versions{
		modLoaders:        map[string]mcl.ModLoader{},
		minecraft:         map[string][]string{},
		loader:            map[string][]string{},
		loaderByMinecraft: map[string]map[string][]string{},
	}
	for _, version := range list {
		switch version.Type {
		case "release":
			data.releases = append(data.releases, version.ID)
		case "snapshot":
			data.snapshots = append(data.snapshots, version.ID)
		}
	}

	installed, err := mcl.InstalledVersions(minecraftDirectory)
	if err != nil {
		return nil, err
	}
	for _, version := range installed {
		data.local = append(data.local, version.ID)
	}

	for _, id := range mcl.ListModLoaders() {
		loader, err := mcl.GetModLoader(id)
		if err != nil {
			return nil, err
		}
		data.modLoaders[id] = loader

		// Fabric and Quilt only support the stable Minecraft versions.
		stable := id == "fabric" || id == "quilt"
		minecraftVersions, err := loader.MinecraftVersions(ctx, stable)
		if err != nil {
			return nil, err
		}
		data.minecraft[id] = minecraftVersions

		if stable {
			// The versions of Fabric and Quilt do not depend on the
			// Minecraft version.
			loaderVersions, err := loader.LoaderVersions(ctx, "", false)
			if err != nil {
				return nil, err
			}
			data.loader[id] = loaderVersions
			continue
		}

		// The versions of Forge and NeoForge depend on it.
		byMinecraft := map[string][]string{}
		for _, minecraftVersion := range minecraftVersions {
			loaderVersions, err := loader.LoaderVersions(ctx, minecraftVersion, false)
			if err != nil {
				return nil, err
			}
			byMinecraft[minecraftVersion] = loaderVersions
		}
		data.loaderByMinecraft[id] = byMinecraft
	}
	return data, nil
}

// installAndLaunch installs the chosen version and starts the game.
func (l *launcher) installAndLaunch(username, loaderName, minecraftVersion, loaderVersion string) error {
	ctx := context.Background()
	version := minecraftVersion

	if id := loaderIDs[loaderName]; id != "" {
		loader := l.data.modLoaders[id]
		version = loader.InstalledVersion(minecraftVersion, loaderVersion)
		if !l.isInstalled(version) {
			installed, err := loader.Install(ctx, minecraftVersion, l.minecraftDirectory, mcl.InstallOptions{
				LoaderVersion: loaderVersion,
				Progress:      l.newProgress(),
			})
			if err != nil {
				return err
			}
			version = installed
		}
	} else if loaderName != "Custom" && !l.isInstalled(minecraftVersion) {
		if err := mcl.Install(ctx, minecraftVersion, l.minecraftDirectory, l.newProgress()); err != nil {
			return err
		}
	}

	command, err := mcl.Command(version, l.minecraftDirectory, mcl.Options{
		Username:      username,
		UUID:          offlineUUID(username),
		GameDirectory: l.gameDirectory,
	})
	if err != nil {
		return err
	}
	fmt.Println("launching:", strings.Join(command, " "))

	// The game writes its saves and its logs into the game directory, so it
	// has to exist when the game starts. It can be gone by then, because
	// somebody deleted it while the launcher was open.
	if err := l.ensureGameDirectory(); err != nil {
		return err
	}

	game := exec.Command(command[0], command[1:]...)
	game.Dir = l.gameDirectory
	game.Stdout = os.Stdout
	game.Stderr = os.Stderr
	if err := game.Start(); err != nil {
		return fmt.Errorf("start the game: %w", err)
	}
	return nil
}

// ensureGameDirectory creates the directory the game runs in.
func (l *launcher) ensureGameDirectory() error {
	return os.MkdirAll(l.gameDirectory, 0o755)
}

// isInstalled reports whether a version is already in the Minecraft
// directory.
func (l *launcher) isInstalled(version string) bool {
	installed, err := mcl.InstalledVersions(l.minecraftDirectory)
	if err != nil {
		return false
	}
	for _, current := range installed {
		if current.ID == version {
			return true
		}
	}
	return false
}

// newProgress returns the callbacks that show the progress of the
// installation.
func (l *launcher) newProgress() *mcl.Progress {
	return &mcl.Progress{
		Status: func(message string) {
			fyne.Do(func() { l.status.SetText(message) })
		},
		Done: func(n int) {
			fyne.Do(func() { l.progressBar.SetValue(float64(n)) })
		},
		Max: func(n int) {
			fyne.Do(func() {
				if n > 0 {
					l.progressBar.Max = float64(n)
				}
				l.progressBar.SetValue(0)
			})
		},
	}
}

// offlineUUID returns the UUID of a player that plays without an account. It
// is the version 3 UUID of the name with the prefix "OfflinePlayer:", the
// same UUID the official launcher uses.
func offlineUUID(username string) string {
	sum := md5.Sum([]byte("OfflinePlayer:" + username))
	sum[6] = (sum[6] & 0x0f) | 0x30 // the version
	sum[8] = (sum[8] & 0x3f) | 0x80 // the variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}
