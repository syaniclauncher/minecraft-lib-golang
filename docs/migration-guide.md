# Migration guide

This guide shows how to move a launcher from the Python library to the Go
library. The complete mapping of every function is in
[MIGRATION.md](../MIGRATION.md).

## The import

```python
import minecraft_launcher_lib
```

```go
import "github.com/syaniclauncher/minecraft-lib-golang/mcl"
```

Everything lives in the package `mcl`, so there is a single import.

## Exceptions become errors

The Python library raises exceptions, the Go library returns errors.

```python
try:
    minecraft_launcher_lib.install.install_minecraft_version("1.21", directory)
except minecraft_launcher_lib.exceptions.VersionNotFound:
    print("there is no such version")
```

```go
if err := mcl.Install(ctx, "1.21", directory, nil); errors.Is(err, mcl.ErrVersionNotFound) {
	fmt.Println("there is no such version")
}
```

The errors that carry data are typed:

```python
except minecraft_launcher_lib.exceptions.InvalidChecksum as error:
    print(error.expected_checksum, error.actual_checksum)
```

```go
var checksumError *mcl.InvalidChecksumError
if errors.As(err, &checksumError) {
	fmt.Println(checksumError.Expected, checksumError.Actual)
}
```

## The callback dictionary becomes a struct

The Python library passes a dictionary of callbacks around:

```python
callback = {
    "setStatus": print,
    "setProgress": lambda done: print(done),
    "setMax": lambda total: print(total),
}
minecraft_launcher_lib.install.install_minecraft_version("1.21", directory, callback)
```

The Go library uses a struct with optional functions:

```go
progress := &mcl.Progress{
	Status:   func(message string) { fmt.Println(message) },
	Done:     func(done int) { fmt.Println(done) },
	Max:      func(total int) { fmt.Println(total) },
}
if err := mcl.Install(ctx, "1.21", directory, progress); err != nil {
	log.Fatal(err)
}
```

## The options dictionary becomes a struct

```python
options = {
    "username": "Player",
    "uuid": "00000000-0000-4000-8000-000000000000",
    "token": "",
    "executablePath": "/usr/bin/java",
    "jvmArguments": ["-Xmx2G"],
    "gameDirectory": "/home/user/.minecraft",
}
command = minecraft_launcher_lib.command.get_minecraft_command("1.21", directory, options)
```

```go
options := mcl.Options{
	Username:      "Player",
	UUID:          "00000000-0000-4000-8000-000000000000",
	Token:         "",
	ExecutablePath: "/usr/bin/java",
	JVMArguments:  []string{"-Xmx2G"},
	GameDirectory: "/home/user/.minecraft",
}
command, err := mcl.Command("1.21", directory, options)
```

The paths are the ones of the platform the launcher runs on. On Windows the
same options look like this, with the backslashes escaped in the JSON:

```python
options = {
    "username": "Player",
    "uuid": "00000000-0000-4000-8000-000000000000",
    "token": "",
    "executablePath": "C:\\Program Files\\Java\\jdk-21\\bin\\java.exe",
    "jvmArguments": ["-Xmx2G"],
    "gameDirectory": "C:\\Users\\user\\AppData\\Roaming\\.minecraft",
}
```

```go
options := mcl.Options{
	Username:      "Player",
	UUID:          "00000000-0000-4000-8000-000000000000",
	Token:         "",
	ExecutablePath: `C:\Program Files\Java\jdk-21\bin\java.exe`,
	JVMArguments:  []string{"-Xmx2G"},
	GameDirectory: `C:\Users\user\AppData\Roaming\.minecraft`,
}
```

The game directory of the official launcher is in the roaming application
data of the user on Windows, in `~/.minecraft` on Linux and in
`~/Library/Application Support/minecraft` on macOS.
`mcl.DefaultMinecraftDirectory` returns the right one for the platform.

The names changed from `snake_case` to `PascalCase`, which is the Go
convention for the fields of a struct.

## Network functions take a context

Every function that talks to a server takes a `context.Context` as its first
argument:

```python
versions = minecraft_launcher_lib.utils.get_version_list()
```

```go
versions, err := mcl.VersionList(ctx)
```

The context makes it possible to cancel a request and to set a deadline.

## The mod loaders

The Python library has a module per mod loader and a `mod_loader` module that
unifies them. The Go library only has the unified one:

```python
loader = minecraft_launcher_lib.mod_loader.get_mod_loader("fabric")
loader.install("1.20.1", directory)
```

```go
loader, err := mcl.GetModLoader("fabric")
if err != nil {
	log.Fatal(err)
}
installed, err := loader.Install(ctx, "1.20.1", directory, mcl.InstallOptions{})
```

The returned version id is the one that has to be used to launch the game.
The Python library computes it with `get_installed_version`, in Go it comes
back from `Install` and is also available as `InstalledVersion`.

## The Microsoft login

```python
url, state, code_verifier = minecraft_launcher_lib.microsoft_account.get_secure_login_data(client_id, redirect_uri)
auth_code = minecraft_launcher_lib.microsoft_account.parse_auth_code_url(url_from_user, state)
profile = minecraft_launcher_lib.microsoft_account.complete_login(client_id, None, redirect_uri, auth_code, code_verifier)
```

```go
data := mcl.SecureLoginData(clientID, redirectURI, "")
// the user opens data.URL and your launcher receives the answer
code, err := mcl.ParseAuthCodeURL(urlFromUser, data.State)
if err != nil {
	log.Fatal(err)
}
account, err := mcl.CompleteLogin(ctx, clientID, "", redirectURI, code, data.CodeVerifier)
if err != nil {
	log.Fatal(err)
}
```

## The launch command

The Python library returns a list that is passed to `subprocess.run`:

```python
subprocess.run(command, cwd=directory)
```

```go
cmd := exec.Command(command[0], command[1:]...)
cmd.Dir = directory
cmd.Run()
```

## What is different

- The functions that talk to the network take a context.
- The options are a struct, so a typo is a compile error.
- The deprecated modules `forge`, `fabric` and `quilt` do not exist; use
  `mcl.Forge`, `mcl.Fabric` and `mcl.Quilt`.
- The Mojang Java runtime is not supported, only the Temurin runtime.
- `mcl.InstallJVMRuntime` keeps a runtime that is already installed instead of
  downloading it again.
