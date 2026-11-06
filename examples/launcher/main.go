// Command launcher is a small desktop launcher for Minecraft.
//
// It shows the versions Mojang offers and the versions of the mod loaders
// Forge, NeoForge, Fabric and Quilt, installs the chosen one and starts the
// game.
//
// The player is not logged in, so the game runs in offline mode with the name
// from the window.
//
// Usage:
//
//	go run ./examples/launcher
//
// The launcher keeps the versions in the "launcher_files" folder and the
// saves of the game in the "game_dir" folder. Both are created next to the
// working directory, the -dir flag puts them somewhere else.
//
// The example is a module of its own, because a window needs a GUI toolkit.
// The library itself has no dependencies.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const appID = "com.syaniclauncher.examples.launcher"

// loaderNames are the entries of the mod loader selector.
var loaderNames = []string{"Vanilla", "Snapshots", "Forge", "NeoForge", "Fabric", "Quilt", "Custom"}

// loaderIDs maps the name of the selector to the id of the mod loader. The
// vanilla game and the snapshots have no mod loader.
var loaderIDs = map[string]string{
	"Forge":    "forge",
	"NeoForge": "neoforge",
	"Fabric":   "fabric",
	"Quilt":    "quilt",
}

// launcher holds the window and the data behind it.
type launcher struct {
	// minecraftDirectory holds the versions, the libraries and the assets.
	minecraftDirectory string
	// gameDirectory is the working directory of the game. It is separate
	// from the Minecraft directory because NeoForge needs it.
	gameDirectory string

	// data holds the versions that are shown in the window. It is filled in
	// the background, so it is nil until the loading is done.
	data *versions

	application         fyne.App
	window              fyne.Window
	username            *widget.Entry
	loaderSelect        *widget.Select
	minecraftSelect     *widget.Select
	loaderVersionSelect *widget.Select
	playButton          *widget.Button
	progressBar         *widget.ProgressBar
	status              *widget.Label
}

func main() {
	directory := flag.String("dir", ".", "the directory the launcher keeps its files in")
	flag.Parse()

	base, err := filepath.Abs(*directory)
	if err != nil {
		log.Fatal(err)
	}
	l := &launcher{
		minecraftDirectory: filepath.Join(base, "launcher_files"),
		gameDirectory:      filepath.Join(base, "game_dir"),
	}
	// The game directory has to exist before the game starts, NeoForge
	// writes its files into it.
	if err := os.MkdirAll(l.minecraftDirectory, 0o755); err != nil {
		log.Fatal(err)
	}
	if err := l.ensureGameDirectory(); err != nil {
		log.Fatal(err)
	}

	l.build()
	l.show()
}

// build fills the window with the widgets.
func (l *launcher) build() {
	l.application = app.NewWithID(appID)
	l.window = l.application.NewWindow("SyanicLauncher Core Tester")
	l.window.Resize(fyne.NewSize(480, 360))

	l.username = widget.NewEntry()
	l.username.SetText("Player")

	l.loaderSelect = widget.NewSelect(loaderNames, func(string) { l.refreshMinecraftVersions() })
	l.loaderSelect.SetSelected("Vanilla")
	l.minecraftSelect = widget.NewSelect(nil, func(string) { l.refreshLoaderVersions() })
	l.loaderVersionSelect = widget.NewSelect(nil, nil)
	l.loaderVersionSelect.Disable()

	l.playButton = widget.NewButton("Install & Play", l.play)
	l.playButton.Disable()

	l.progressBar = widget.NewProgressBar()
	l.status = widget.NewLabel("Loading the versions...")
	l.status.Wrapping = fyne.TextWrapWord

	form := widget.NewForm(
		widget.NewFormItem("Username", l.username),
		widget.NewFormItem("Mod Loader", l.loaderSelect),
		widget.NewFormItem("Minecraft Version", l.minecraftSelect),
		widget.NewFormItem("Loader Version", l.loaderVersionSelect),
	)

	l.window.SetContent(container.NewPadded(container.NewVBox(
		form,
		l.playButton,
		l.progressBar,
		l.status,
	)))
}

// show shows the window and loads the versions in the background.
func (l *launcher) show() {
	l.window.Show()
	go l.load()
	l.application.Run()
}

// load fetches the versions of the game and of the mod loaders. It talks to
// several servers, so it takes a while.
func (l *launcher) load() {
	data, err := loadVersions(context.Background(), l.minecraftDirectory)
	if err != nil {
		fyne.Do(func() {
			l.setStatus("Failed to load the versions: " + err.Error())
		})
		return
	}
	fyne.Do(func() {
		l.data = data
		l.playButton.Enable()
		l.setStatus("Ready")
		l.refreshMinecraftVersions()
	})
}

// refreshMinecraftVersions fills the Minecraft version list with the versions
// that belong to the selected mod loader.
func (l *launcher) refreshMinecraftVersions() {
	if l.data == nil {
		return
	}
	var list []string
	switch l.loaderSelect.Selected {
	case "Vanilla":
		list = l.data.releases
	case "Snapshots":
		list = l.data.snapshots
	case "Custom":
		list = l.data.local
	default:
		list = l.data.minecraft[loaderIDs[l.loaderSelect.Selected]]
	}

	l.minecraftSelect.SetOptions(list)
	if len(list) > 0 {
		l.minecraftSelect.SetSelected(list[0])
	} else {
		l.minecraftSelect.ClearSelected()
	}
	l.refreshLoaderVersions()
}

// refreshLoaderVersions fills the loader version list with the versions of the
// selected mod loader.
func (l *launcher) refreshLoaderVersions() {
	if l.data == nil {
		return
	}
	id := loaderIDs[l.loaderSelect.Selected]
	if id == "" {
		l.loaderVersionSelect.SetOptions(nil)
		l.loaderVersionSelect.ClearSelected()
		l.loaderVersionSelect.Disable()
		return
	}
	l.loaderVersionSelect.Enable()

	// The versions of Fabric and Quilt are the same for every Minecraft
	// version, the versions of Forge and NeoForge depend on it.
	list := l.data.loader[id]
	if byMinecraft, ok := l.data.loaderByMinecraft[id]; ok {
		list = byMinecraft[l.minecraftSelect.Selected]
	}

	l.loaderVersionSelect.SetOptions(list)
	if len(list) > 0 {
		l.loaderVersionSelect.SetSelected(list[0])
	} else {
		l.loaderVersionSelect.ClearSelected()
	}
}

// play installs the selected version and starts the game.
func (l *launcher) play() {
	username := strings.TrimSpace(l.username.Text)
	if username == "" {
		l.setStatus("Please enter a username.")
		return
	}
	if l.minecraftSelect.Selected == "" {
		l.setStatus("Please select a Minecraft version.")
		return
	}
	// The selection is read here, because the window belongs to another
	// thread than the installation.
	loaderName := l.loaderSelect.Selected
	minecraftVersion := l.minecraftSelect.Selected
	loaderVersion := l.loaderVersionSelect.Selected

	l.playButton.Disable()
	l.progressBar.SetValue(0)
	l.setStatus("Starting...")

	go func() {
		err := l.installAndLaunch(username, loaderName, minecraftVersion, loaderVersion)
		fyne.Do(func() {
			l.playButton.Enable()
			if err != nil {
				l.setStatus("Failed: " + err.Error())
				return
			}
			l.setStatus("Minecraft launched!")
		})
	}()
}

// setStatus writes a message under the progress bar.
func (l *launcher) setStatus(message string) {
	l.status.SetText(message)
}
