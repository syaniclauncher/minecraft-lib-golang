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
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

const appID = "com.syaniclauncher.examples.launcher"

var BuildNumber string = "dev"

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
	// baseDirectory is the directory the launcher keeps its files in when no
	// other folder is chosen.
	baseDirectory string
	// minecraftDirectory holds the versions, the libraries and the assets.
	minecraftDirectory string
	// gameDirectory is the working directory of the game. It is separate
	// from the Minecraft directory because NeoForge needs it.
	gameDirectory string

	// data holds the versions that are shown in the window. It is filled in
	// the background, so it is nil until the loading is done.
	data *versions

	// game is the running game. It is nil when no game runs. Only the
	// thread of the window touches it.
	game          *exec.Cmd
	installCancel context.CancelFunc

	application         fyne.App
	window              fyne.Window
	username            *widget.Entry
	uuid                *widget.Entry
	token               *widget.Entry
	loaderFolder        *widget.Entry
	minecraftFolder     *widget.Entry
	javaPath            *widget.Entry
	minMemory           *widget.Entry
	maxMemory           *widget.Entry
	jvmArguments        *widget.Entry
	loaderSelect        *widget.Select
	minecraftSelect     *widget.Select
	loaderVersionSelect *widget.Select
	playButton          *widget.Button
	// form holds the widgets of the window. The form greys the label of a
	// field that is disabled, so it has to be refreshed when a field is
	// enabled again.
	form        *widget.Form
	progressBar *widget.ProgressBar
	status      *widget.Label
	// consoleBox holds the console with a theme of its own.
	consoleBox *container.ThemeOverride
	// console shows the output of the game.
	console *widget.Entry
	// consoleText is the text of the console. It is kept here, because the
	// console only shows the last lines of it.
	consoleText string
	// clearButton empties the console.
	clearButton *widget.Button
	// consoleButton shows and hides the console.
	consoleButton *widget.Button
}

func main() {
	directory := flag.String("dir", ".", "the directory the launcher keeps its files in")
	flag.Parse()

	base, err := filepath.Abs(*directory)
	if err != nil {
		log.Fatal(err)
	}
	l := &launcher{
		baseDirectory:      base,
		minecraftDirectory: filepath.Join(base, "launcher_files"),
		gameDirectory:      filepath.Join(base, "game_dir"),
	}
	// The launcher keeps its files in the Minecraft directory. The game
	// directory is created by mcl.Command, because the game writes its
	// saves and its logs into it.
	if err := os.MkdirAll(l.minecraftDirectory, 0o755); err != nil {
		log.Fatal(err)
	}

	l.build()
	l.show()
}

// build fills the window with the widgets.
func (l *launcher) build() {
	l.application = app.NewWithID(appID)
	l.window = l.application.NewWindow("SyanicLauncher Core Tester " + BuildNumber)
	l.window.Resize(fyne.NewSize(640, 480))

	l.username = widget.NewEntry()
	l.username.SetText("Player")

	// The UUID and the token are optional. The game runs offline without
	// them.
	l.uuid = widget.NewEntry()
	l.uuid.SetPlaceHolder("(optional)")
	l.token = widget.NewEntry()
	l.token.SetPlaceHolder("(optional)")

	// The folders are optional too, the launcher keeps its files next to
	// the working directory when they are empty.
	l.loaderFolder = widget.NewEntry()
	l.loaderFolder.SetPlaceHolder("/launcher_files")
	l.loaderFolder.OnChanged = func(string) { l.foldersChanged() }
	l.minecraftFolder = widget.NewEntry()
	l.minecraftFolder.SetPlaceHolder("/game_dir")
	l.minecraftFolder.OnChanged = func(string) { l.foldersChanged() }

	// The Java of the launcher is used when no other one is given.
	l.javaPath = widget.NewEntry()
	l.javaPath.SetPlaceHolder("java to be used by game")

	// The memory of the game in megabytes. The game decides when the fields
	// are empty.
	l.minMemory = widget.NewEntry()
	l.minMemory.SetPlaceHolder("512")
	l.maxMemory = widget.NewEntry()
	l.maxMemory.SetPlaceHolder("2048")

	l.jvmArguments = widget.NewEntry()
	l.jvmArguments.SetPlaceHolder("put custom args here")

	l.loaderSelect = widget.NewSelect(loaderNames, func(string) { l.refreshMinecraftVersions() })
	l.loaderSelect.SetSelected("Vanilla")
	l.minecraftSelect = widget.NewSelect(nil, func(string) { l.refreshLoaderVersions() })
	// The versions are loaded in the background.
	l.minecraftSelect.PlaceHolder = "Loading.."
	l.loaderVersionSelect = widget.NewSelect(nil, nil)
	l.loaderVersionSelect.PlaceHolder = "Loading.."
	l.loaderVersionSelect.Disable()

	l.playButton = widget.NewButton("Launch Game", l.toggleGame)
	l.playButton.Disable()

	l.progressBar = widget.NewProgressBar()
	// The bar is only shown while the game is launching.
	l.progressBar.Hide()

	l.status = widget.NewLabel("Loading the versions...")
	l.status.Wrapping = fyne.TextWrapWord

	// The console shows the output of the game. It is not editable, but it
	// can be scrolled.
	l.console = widget.NewMultiLineEntry()
	l.console.TextStyle = fyne.TextStyle{Monospace: true}
	l.console.Wrapping = fyne.TextWrapOff
	l.console.Scroll = fyne.ScrollVerticalOnly
	l.console.SetMinRowsVisible(6)
	l.console.Disable()

	l.clearButton = widget.NewButton("Clear", l.clearConsole)
	l.consoleButton = widget.NewButton("Show console", l.toggleConsole)

	// The text of the console is a bit bigger than the text of the window.
	l.consoleBox = container.NewThemeOverride(l.console, consoleTheme{Theme: l.application.Settings().Theme()})
	// The console is only shown when the user asks for it.
	l.consoleBox.Hide()

	l.form = widget.NewForm(
		widget.NewFormItem("Username", l.username),
		widget.NewFormItem("", widget.NewAccordion(
			widget.NewAccordionItem("Advance User Options", container.New(layout.NewFormLayout(),
				widget.NewLabel("UUID"), l.uuid,
				widget.NewLabel("Token"), l.token,
			)),
		)),
		widget.NewFormItem("Mod Loader", l.loaderSelect),
		widget.NewFormItem("Minecraft Version", l.minecraftSelect),
		widget.NewFormItem("Loader Version", l.loaderVersionSelect),
		widget.NewFormItem("Instance Folder", l.folderRow(l.minecraftFolder)),
		widget.NewFormItem("", widget.NewAccordion(
			widget.NewAccordionItem("Advance Game Options", container.New(layout.NewFormLayout(),
				widget.NewLabel("Launcher Folder"), l.folderRow(l.loaderFolder),
				widget.NewLabel("JRE Location"), container.NewVBox(
					l.pathRow(l.javaPath, l.selectJava),
					hint("Reccomended to keep the JRE location empty, the core will automatically download the appropriate Java from Temurin."),
				),
				widget.NewLabel("Minimum Memory (MB)"), l.minMemory,
				widget.NewLabel("Maximum Memory (MB)"), l.maxMemory,
				widget.NewLabel("JVM Arguments"), l.jvmArguments,
			)),
		)),
	)

	// The console fills the space that is left over.
	l.window.SetContent(container.NewPadded(container.NewBorder(
		container.NewVBox(
			l.form,
			l.playButton,
			l.progressBar,
			l.status,
			container.NewHBox(l.clearButton, l.consoleButton),
		),
		nil, nil, nil,
		l.consoleBox,
	)))
}

// consoleTheme makes the text of the console a bit bigger.
type consoleTheme struct {
	fyne.Theme
}

// Size returns the size of the text of the console. The other sizes are the
// ones of the theme of the window.
func (t consoleTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return 10
	}
	return t.Theme.Size(name)
}

// folderRow returns the field of a folder with the buttons that choose and
// reset it.
func (l *launcher) folderRow(field *widget.Entry) fyne.CanvasObject {
	return l.pathRow(field, l.selectFolder)
}

// pathRow returns the field with the buttons that choose and reset it. The
// choose button calls the given function with the field.
func (l *launcher) pathRow(field *widget.Entry, choose func(*widget.Entry)) fyne.CanvasObject {
	buttons := container.NewHBox(
		widget.NewButton("Select", func() { choose(field) }),
		widget.NewButton("Reset", func() { field.SetText("") }),
	)
	// The field fills the space that the buttons leave over.
	return container.NewBorder(nil, nil, nil, buttons, field)
}

// selectFolder lets the user choose the folder of the entry.
func (l *launcher) selectFolder(field *widget.Entry) {
	dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		field.SetText(uri.Path())
	}, l.window).Show()
}

// selectJava lets the user choose the folder of a Java installation. The java
// executable inside it runs the game.
func (l *launcher) selectJava(field *widget.Entry) {
	dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		field.SetText(javaExecutable(uri.Path()))
	}, l.window).Show()
}

// javaExecutable returns the path of the java executable of a Java folder.
func javaExecutable(folder string) string {
	name := "java"
	if runtime.GOOS == "windows" {
		// The windowed executable does not open a console window.
		name = "javaw.exe"
	}
	return filepath.Join(folder, "bin", name)
}

// hint returns a small grey line that explains a field.
func hint(text string) *widget.RichText {
	line := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{
			ColorName: theme.ColorNamePlaceHolder,
			SizeName:  theme.SizeNameCaptionText,
			Inline:    true,
		},
		Text: text,
	})
	line.Wrapping = fyne.TextWrapWord
	return line
}

// javaArguments returns the arguments of the Java process: the ones the user
// wrote and the memory the game may use.
//
// The memory is given in megabytes. The game decides when it is empty.
func (l *launcher) javaArguments() ([]string, error) {
	arguments := strings.Fields(l.jvmArguments.Text)

	for _, memory := range []struct {
		name  string
		field *widget.Entry
		flag  string
	}{
		{"minimum", l.minMemory, "-Xms"},
		{"maximum", l.maxMemory, "-Xmx"},
	} {
		text := strings.TrimSpace(memory.field.Text)
		if text == "" {
			continue
		}
		size, err := strconv.Atoi(text)
		if err != nil || size <= 0 {
			return nil, fmt.Errorf("The %s memory must be a number of megabytes.", memory.name)
		}
		// The lower case m is the megabyte of the Java virtual machine.
		arguments = append(arguments, memory.flag+strconv.Itoa(size)+"m")
	}
	return arguments, nil
}

// foldersChanged uses the folders of the window. Only the versions that are
// installed depend on them, the other ones are the same everywhere.
func (l *launcher) foldersChanged() {
	l.minecraftDirectory = l.folder(l.loaderFolder.Text, "launcher_files")
	l.gameDirectory = l.folder(l.minecraftFolder.Text, "game_dir")

	if l.data == nil {
		return
	}
	installed, err := mcl.InstalledVersions(l.minecraftDirectory)
	if err != nil {
		return
	}
	l.data.local = nil
	for _, version := range installed {
		l.data.local = append(l.data.local, version.ID)
	}
	if l.loaderSelect.Selected == "Custom" {
		l.refreshMinecraftVersions()
	}
}

// folder returns the folder of the entry. The default folder of the launcher
// is used when the entry is empty.
func (l *launcher) folder(path, defaultName string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return filepath.Join(l.baseDirectory, defaultName)
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(l.baseDirectory, path)
}

// clearConsole removes the lines from the console.
func (l *launcher) clearConsole() {
	l.consoleText = ""
	l.console.SetText("")
}

// toggleConsole shows and hides the console.
func (l *launcher) toggleConsole() {
	if l.consoleBox.Visible() {
		l.consoleBox.Hide()
		l.consoleButton.SetText("Show console")
		return
	}
	l.consoleBox.Show()
	l.consoleButton.SetText("Hide console")
}

// appendConsole adds the lines to the console and scrolls to the newest one.
func (l *launcher) appendConsole(lines []string) {
	fyne.Do(func() {
		l.consoleText += strings.Join(lines, "\n") + "\n"
		l.console.SetText(l.consoleText)
		// The cursor is on the last line, so the console scrolls down to it.
		l.console.CursorRow = strings.Count(l.consoleText, "\n") - 1
		l.console.CursorColumn = 0
		l.console.Refresh()
	})
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
		// There is no version to choose.
		l.minecraftSelect.PlaceHolder = "Not Available"
		l.minecraftSelect.Refresh()
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
		// The vanilla game and the snapshots have no mod loader.
		l.loaderVersionSelect.PlaceHolder = "Not Available"
		l.loaderVersionSelect.Refresh()
		// The form greys the label of a disabled field.
		l.form.Refresh()
		return
	}
	l.loaderVersionSelect.Enable()
	// The label of the field is grey while it is disabled.
	l.form.Refresh()

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
		// The loader has no version for this Minecraft version.
		l.loaderVersionSelect.PlaceHolder = "Not Available"
		l.loaderVersionSelect.Refresh()
	}
}

// toggleGame starts the game, or stops it when it already runs.
func (l *launcher) toggleGame() {
	if l.game != nil {
		// The game is stopped. The button is turned back when the game is
		// really gone.
		l.game.Process.Kill()
		return
	}
	if l.installCancel != nil {
		l.installCancel()
		l.playButton.SetText("Cancelling...")
		l.playButton.Disable()
		return
	}
	l.play()
}

// showCancelButton turns the button into the button that cancels installation.
func (l *launcher) showCancelButton() {
	l.playButton.SetText("Cancel Installation")
	l.playButton.Importance = widget.DangerImportance
	l.playButton.Refresh()
}

// showKillButton turns the button into the button that stops the running
// game.
func (l *launcher) showKillButton(game *exec.Cmd) {
	l.game = game

	l.playButton.SetText("Kill Game")
	// The danger theme paints the button red.
	l.playButton.Importance = widget.DangerImportance
	l.playButton.Refresh()
}

// showLaunchButton turns the button back into the button that starts the
// game.
func (l *launcher) showLaunchButton() {
	l.game = nil

	l.playButton.SetText("Launch Game")
	l.playButton.Importance = widget.MediumImportance
	l.playButton.Refresh()
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
	arguments, err := l.javaArguments()
	if err != nil {
		l.setStatus(err.Error())
		return
	}
	// The selection is read here, because the window belongs to another
	// thread than the installation.
	loaderName := l.loaderSelect.Selected
	minecraftVersion := l.minecraftSelect.Selected
	loaderVersion := l.loaderVersionSelect.Selected

	ctx, cancel := context.WithCancel(context.Background())
	l.installCancel = cancel
	l.showCancelButton()
	// The bar is only shown while the game is launching.
	l.progressBar.Show()
	l.progressBar.SetValue(0)
	l.setStatus("Starting...")
	// Every launch starts with an empty console.
	l.clearConsole()

	go func() {
		game, err := l.installAndLaunch(ctx, username, loaderName, minecraftVersion, loaderVersion, arguments)
		fyne.Do(func() {
			cancel()
			l.installCancel = nil
			l.playButton.Enable()
			l.progressBar.Hide()
			if err != nil {
				l.showLaunchButton()
				if errors.Is(err, context.Canceled) {
					l.setStatus("Installation cancelled")
				} else {
					l.setStatus("Failed: " + err.Error())
				}
				return
			}
			l.setStatus("Minecraft launched! (The graphics engine may take time to load.)")
			l.showKillButton(game)
		})
		if game == nil {
			return
		}
		// The status is cleared and the button starts the next game when the
		// game closes.
		game.Wait()
		fyne.Do(func() {
			l.setStatus("")
			l.showLaunchButton()
		})
	}()
}

// setStatus writes a message under the progress bar.
func (l *launcher) setStatus(message string) {
	l.status.SetText(message)
}
