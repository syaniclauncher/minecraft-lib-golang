# Installation progress

A long installation should show what it is doing. `mcl.Progress` collects the
updates of the installer.

```go
progress := &mcl.Progress{
	Status: func(message string) {
		log.Println(message)
	},
	Max: func(total int) {
		bar.SetMax(total)
	},
	Done: func(done int) {
		bar.Set(done)
	},
}

if err := mcl.Install(ctx, "1.21", directory, progress); err != nil {
	log.Fatal(err)
}
```

All three fields are optional. A field that is not set is simply not called.
A `nil` `*mcl.Progress` is valid as well and discards every update, so it can
be passed straight through:

```go
if err := mcl.Install(ctx, "1.21", directory, nil); err != nil {
	log.Fatal(err)
}
```

## What is reported

| Callback | When it is called |
| --- | --- |
| `Status` | With a message like "Downloading Libraries" or "Installing java runtime" |
| `Max` | With the number of steps of the current phase |
| `Done` | With the number of finished steps of the current phase |

`Max` and `Done` are called per phase, so a progress bar has to be reset
whenever `Max` is called with a new total.

## The mod loaders

The mod loaders take the same progress:

```go
installed, err := loader.Install(ctx, "1.20.1", directory, mcl.InstallOptions{
	LoaderVersion: "0.15.0",
	Progress:      progress,
})
```

## Cancelling

Every function that talks to the network takes a context. Cancelling the
context stops the installation:

```go
ctx, cancel := context.WithCancel(context.Background())
go func() {
	time.Sleep(30 * time.Second)
	cancel()
}()

err := mcl.Install(ctx, "1.21", directory, progress)
if errors.Is(err, context.Canceled) {
	log.Println("the installation was cancelled")
}
```

A cancelled installation leaves the files it already wrote in place. They are
verified and repaired on the next run.
