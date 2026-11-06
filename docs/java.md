# Java installations

Minecraft needs Java. The library installs a matching Temurin runtime next to
the Minecraft directory, so most launchers never have to care about Java. The
functions below are for the cases where the user should pick a Java version.

## Finding the Java installations of the system

```go
installations := mcl.FindSystemJavaVersions(nil)
for _, installation := range installations {
	fmt.Println(installation)
}
```

The argument is a list of extra directories to search. `nil` searches only the
usual ones: `C:\Program Files (x86)\Java` and `C:\Program Files\Java` on
Windows, `/usr/lib/jvm` and `/usr/lib/sdk` on Linux. macOS is not searched.

With the details of every installation:

```go
information, err := mcl.FindSystemJavaVersionsInformation([]string{"/opt/java"})
if err != nil {
	log.Fatal(err)
}

for _, current := range information {
	fmt.Println(current.Name, current.Version, current.JavaPath)
}
```

## The information of a single installation

```go
information, err := mcl.JavaInformation("/usr/lib/jvm/java-21-openjdk")
if err != nil {
	log.Fatal(err)
}

fmt.Println(information.Name)    // java-21-openjdk
fmt.Println(information.Version) // 21.0.3
fmt.Println(information.Is64Bit)
fmt.Println(information.OpenJDK)
fmt.Println(information.JavaPath) // /usr/lib/jvm/java-21-openjdk/bin/java
```

The path has to be the installation directory, not the executable.

The function runs the java executable to find out the version, which is a
security risk when the path is not trusted.

## The default Java executable

```go
fmt.Println(mcl.DefaultJavaExecutable())
```

The function looks at `JAVA_HOME` first, then at the usual places of the
platform, and falls back to `java` from the `PATH`.

## The runtime of the Minecraft directory

The runtime that was installed for the game:

```go
path := mcl.TemurinExecutablePath(17, minecraftDirectory)
if path == "" {
	fmt.Println("no runtime is installed")
}
```

The Java versions that Mojang offers:

```go
runtimes, err := mcl.JVMRuntimes(ctx)
```

And the runtimes that are installed in the Minecraft directory:

```go
installed, err := mcl.InstalledJVMRuntimes(minecraftDirectory)
```

## Installing a runtime by hand

```go
if err := mcl.InstallJVMRuntime(ctx, 21, minecraftDirectory, progress); err != nil {
	log.Fatal(err)
}
```

The number is the Java major version: 8, 17 or 21. A runtime that is already
installed is kept, so the function is cheap to call.
