package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"fyne.io/fyne/v2"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
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
//
// The arguments are the ones of the Java process. It returns the running
// game, so that the caller can wait for it to close.
func (l *launcher) installAndLaunch(ctx context.Context, username, loaderName, minecraftVersion, loaderVersion string, arguments []string) (*exec.Cmd, error) {
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
				return nil, err
			}
			version = installed
		}
	} else if loaderName != "Custom" && !l.isInstalled(minecraftVersion) {
		if err := mcl.Install(ctx, minecraftVersion, l.minecraftDirectory, l.newProgress()); err != nil {
			return nil, err
		}
	}

	// The UUID and the token are optional. The game runs offline when there
	// is no token.
	uuid := strings.TrimSpace(l.uuid.Text)
	if uuid == "" {
		uuid = mcl.OfflineUUID(username)
	}
	command, err := mcl.Command(version, l.minecraftDirectory, mcl.Options{
		Username:       username,
		UUID:           uuid,
		Token:          strings.TrimSpace(l.token.Text),
		ExecutablePath: strings.TrimSpace(l.javaPath.Text),
		JVMArguments:   arguments,
		GameDirectory:  l.gameDirectory,
	})
	if err != nil {
		return nil, err
	}
	fmt.Println("launching:", strings.Join(command, " "))

	// The output of the game goes to the console of the window.
	output := &consoleWriter{launcher: l}
	game := exec.Command(command[0], command[1:]...)
	game.Dir = l.gameDirectory
	game.Stdout = output
	game.Stderr = output
	if err := game.Start(); err != nil {
		return nil, fmt.Errorf("start the game: %w", err)
	}
	return game, nil
}

// consoleWriter sends the output of the game to the console of the window.
//
// The output is split into lines, the last line is kept until it is
// complete.
type consoleWriter struct {
	launcher *launcher
	mutex    sync.Mutex
	buffer   []byte
}

// Write adds the lines of the output to the console.
func (w *consoleWriter) Write(data []byte) (int, error) {
	w.mutex.Lock()
	w.buffer = append(w.buffer, data...)
	var lines []string
	for {
		index := bytes.IndexByte(w.buffer, '\n')
		if index < 0 {
			break
		}
		lines = append(lines, string(w.buffer[:index]))
		w.buffer = w.buffer[index+1:]
	}
	w.mutex.Unlock()

	if len(lines) > 0 {
		w.launcher.appendConsole(lines)
	}
	return len(data), nil
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
		Download: func(name string, downloaded, total int64) {
			message := fmt.Sprintf("Downloading %s (%.1f MB", name, float64(downloaded)/(1<<20))
			if total > 0 {
				message += fmt.Sprintf(" / %.1f MB", float64(total)/(1<<20))
			}
			message += ")"
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
