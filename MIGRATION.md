# Migration status

The Python library [minecraft-launcher-lib](https://github.com/syaniclauncher/minecraft-launcher-lib)
is the behavioural reference. This document maps every module of the Python
library to its Go counterpart and records the progress of the port.

Legend: **done** — implemented and tested, **partial** — implemented with a
documented difference, **dropped** — intentionally not ported.

## Module overview

| Python module | Go | Status | Notes |
| --- | --- | --- | --- |
| `minecraft_launcher_lib.utils` | `mcl` | done | Functions moved to the package root |
| `minecraft_launcher_lib.install` | `mcl.Install` | done | |
| `minecraft_launcher_lib.command` | `mcl.Command` | done | |
| `minecraft_launcher_lib.natives` | `mcl.ExtractNatives` | done | |
| `minecraft_launcher_lib.runtime` | `mcl.InstallJVMRuntime`, `mcl.TemurinExecutablePath`, … | done | The Mojang runtime is not ported, see below |
| `minecraft_launcher_lib.java_utils` | `mcl.JavaInformation`, `mcl.FindSystemJavaVersions`, … | done | |
| `minecraft_launcher_lib.mod_loader` | `mcl.ModLoader` | done | One interface for all four mod loaders |
| `minecraft_launcher_lib.forge` | `mcl.Forge` | done | The deprecated module is replaced by `mod_loader` |
| `minecraft_launcher_lib.fabric` | `mcl.Fabric` | done | The deprecated module is replaced by `mod_loader` |
| `minecraft_launcher_lib.quilt` | `mcl.Quilt` | done | The deprecated module is replaced by `mod_loader` |
| `minecraft_launcher_lib.vanilla_launcher` | `mcl.LoadVanillaLauncherProfiles`, … | done | |
| `minecraft_launcher_lib.mrpack` | `mcl.InstallMrpack`, … | done | |
| `minecraft_launcher_lib.microsoft_account` | `mcl.CompleteLogin`, `mcl.CompleteRefresh`, … | done | |
| `minecraft_launcher_lib.news` | `mcl.GetMinecraftNews`, `mcl.GetJavaPatchNotes` | done | |
| `minecraft_launcher_lib.exceptions` | `mcl.ErrVersionNotFound`, `mcl.InvalidChecksumError`, … | done | |
| `minecraft_launcher_lib.types` | `mcl.Options`, `mcl.VersionInfo`, … | done | |

## Function mapping

### utils

| Python | Go |
| --- | --- |
| `get_minecraft_directory()` | `mcl.DefaultMinecraftDirectory()` |
| `get_latest_version()` | `mcl.LatestVersions(ctx)` |
| `get_version_list()` | `mcl.VersionList(ctx)` |
| `get_installed_versions(dir)` | `mcl.InstalledVersions(dir)` |
| `get_available_versions(dir)` | `mcl.AvailableVersions(ctx, dir)` |
| `get_java_executable()` | `mcl.DefaultJavaExecutable()` |
| `get_library_version()` | `mcl.Version` |
| `generate_test_options()` | `mcl.TestOptions()` |
| `is_version_valid(version, dir)` | `mcl.IsVersionValid(ctx, version, dir)` |
| `is_vanilla_version(version)` | `mcl.IsVanillaVersion(ctx, version)` |
| `is_platform_supported()` | `mcl.IsPlatformSupported()` |
| `is_minecraft_installed(dir)` | `mcl.IsMinecraftInstalled(dir)` |

### install

| Python | Go |
| --- | --- |
| `install.install_minecraft_version(version, dir, callback)` | `mcl.Install(ctx, version, dir, progress)` |
| `install.install_libraries(...)` | internal |
| `install.install_assets(...)` | internal |

### command

| Python | Go |
| --- | --- |
| `command.get_minecraft_command(version, dir, options)` | `mcl.Command(version, dir, options)` |

### natives

| Python | Go |
| --- | --- |
| `natives.extract_natives(version, dir, path)` | `mcl.ExtractNatives(version, dir, path)` |

### runtime

| Python | Go |
| --- | --- |
| `runtime.install_jvm_runtime_temurin(major, dir, callback)` | `mcl.InstallJVMRuntime(ctx, major, dir, progress)` |
| `runtime.get_executable_path_temurin(major, dir)` | `mcl.TemurinExecutablePath(major, dir)` |
| `runtime.get_jvm_runtimes()` | `mcl.JVMRuntimes(ctx)` |
| `runtime.get_installed_jvm_runtimes(dir)` | `mcl.InstalledJVMRuntimes(dir)` |
| `runtime.get_jvm_runtime_information(version)` | `mcl.JVMRuntimeInformation(ctx, version)` |
| `runtime.get_version_runtime_information(version, dir)` | `mcl.VersionRuntimeInformation(version, dir)` |
| `runtime.install_jvm_runtime(...)` | not ported | The function is commented out in the Python library as well |
| `runtime.get_executable_path(...)` | not ported | Same |

### java_utils

| Python | Go |
| --- | --- |
| `java_utils.get_java_information(path)` | `mcl.JavaInformation(path)` |
| `java_utils.find_system_java_versions(dirs)` | `mcl.FindSystemJavaVersions(dirs)` |
| `java_utils.find_system_java_versions_information(dirs)` | `mcl.FindSystemJavaVersionsInformation(dirs)` |

### mod_loader

| Python | Go |
| --- | --- |
| `mod_loader.get_mod_loader(id)` | `mcl.GetModLoader(id)` |
| `mod_loader.list_mod_loader()` | `mcl.ListModLoaders()` |
| `ModLoader.get_id()` | `ModLoader.ID()` |
| `ModLoader.get_name()` | `ModLoader.Name()` |
| `ModLoader.get_minecraft_versions(stable_only)` | `ModLoader.MinecraftVersions(ctx, stableOnly)` |
| `ModLoader.is_minecraft_version_supported(version)` | `ModLoader.IsMinecraftVersionSupported(ctx, version)` |
| `ModLoader.get_loader_versions(version, stable_only)` | `ModLoader.LoaderVersions(ctx, version, stableOnly)` |
| `ModLoader.get_latest_loader_version(version)` | `ModLoader.LatestLoaderVersion(ctx, version)` |
| `ModLoader.get_installed_version(version, loader_version)` | `ModLoader.InstalledVersion(version, loaderVersion)` |
| `ModLoader.install(...)` | `ModLoader.Install(ctx, version, dir, opts)` |

The `get_installer_url` method of the Python base class is public in Go as
`ModLoader.InstallerURL(ctx, minecraftVersion, loaderVersion)`.

### vanilla_launcher

| Python | Go |
| --- | --- |
| `vanilla_launcher.load_vanilla_launcher_profiles(dir)` | `mcl.LoadVanillaLauncherProfiles(dir)` |
| `vanilla_launcher.vanilla_launcher_profile_to_minecraft_options(profile)` | `mcl.VanillaLauncherProfileToOptions(profile)` |
| `vanilla_launcher.get_vanilla_launcher_profile_version(profile)` | `mcl.VanillaLauncherProfileVersion(ctx, profile)` |
| `vanilla_launcher.add_vanilla_launcher_profile(dir, profile)` | `mcl.AddVanillaLauncherProfile(dir, profile)` |
| `vanilla_launcher.do_vanilla_launcher_profiles_exists(dir)` | `mcl.VanillaLauncherProfilesExist(dir)` |
| `vanilla_launcher.create_empty_vanilla_launcher_profiles_file(dir)` | `mcl.CreateEmptyVanillaLauncherProfiles(dir)` |
| `vanilla_launcher.ensure_vanilla_launcher_profiles_exists(dir)` | `mcl.EnsureVanillaLauncherProfiles(dir)` |

### mrpack

| Python | Go |
| --- | --- |
| `mrpack.get_mrpack_information(path)` | `mcl.GetMrpackInformation(path)` |
| `mrpack.install_mrpack(path, dir, modpack_dir, callback, options)` | `mcl.InstallMrpack(ctx, path, dir, modpackDir, options, progress)` |
| `mrpack.get_mrpack_launch_version(path)` | `mcl.MrpackLaunchVersion(path)` |

### microsoft_account

| Python | Go |
| --- | --- |
| `microsoft_account.get_login_url(client_id, redirect_uri)` | `mcl.LoginURL(clientID, redirectURI)` |
| `microsoft_account.get_secure_login_data(client_id, redirect_uri, state)` | `mcl.SecureLoginData(clientID, redirectURI, state)` |
| `microsoft_account.generate_state()` | `mcl.GenerateState()` |
| `microsoft_account.url_contains_auth_code(url)` | `mcl.URLContainsAuthCode(url)` |
| `microsoft_account.get_auth_code_from_url(url)` | `mcl.AuthCodeFromURL(url)` |
| `microsoft_account.parse_auth_code_url(url, state)` | `mcl.ParseAuthCodeURL(url, state)` |
| `microsoft_account.complete_login(...)` | `mcl.CompleteLogin(ctx, ...)` |
| `microsoft_account.complete_refresh(...)` | `mcl.CompleteRefresh(ctx, ...)` |
| `microsoft_account.get_authorization_token(...)` | internal |
| `microsoft_account.refresh_authorization_token(...)` | internal |
| `microsoft_account.authenticate_with_xbl(...)` | internal |
| `microsoft_account.authenticate_with_xsts(...)` | internal |
| `microsoft_account.authenticate_with_minecraft(...)` | internal |
| `microsoft_account.get_store_information(...)` | not ported | Only useful for a store browser |
| `microsoft_account.get_profile(...)` | internal |

### news

| Python | Go |
| --- | --- |
| `news.get_minecraft_news()` | `mcl.GetMinecraftNews(ctx)` |
| `news.get_java_patch_notes()` | `mcl.GetJavaPatchNotes(ctx)` |

### exceptions

| Python | Go |
| --- | --- |
| `VersionNotFound` | `mcl.ErrVersionNotFound` |
| `UnsupportedVersion` | `mcl.ErrUnsupportedVersion` |
| `InvalidRefreshToken` | `mcl.ErrInvalidRefreshToken` |
| `InvalidVanillaLauncherProfile` | `mcl.ErrInvalidVanillaLauncherProfile` |
| `SecurityError` | `mcl.FileOutsideMinecraftDirectoryError`, `mcl.InvalidChecksumError` |
| `FileOutsideMinecraftDirectory` | `mcl.FileOutsideMinecraftDirectoryError` |
| `InvalidChecksum` | `mcl.InvalidChecksumError` |
| `AzureAppNotPermitted` | `mcl.ErrAzureAppNotPermitted` |
| `PlatformNotSupported` | `mcl.ErrPlatformNotSupported` |
| `AccountNotOwnMinecraft` | `mcl.ErrAccountNotOwnMinecraft` |
| `ExternalProgramError` | `mcl.ExternalProgramError` |

## Intentional differences

1. **Errors instead of exceptions.** Every failure is reported as an `error`.
   The errors of the Python library that carry data are typed structs, so
   `errors.As` can be used to inspect them:

   ```go
   var checksumError *mcl.InvalidChecksumError
   if errors.As(err, &checksumError) {
       fmt.Println(checksumError.Expected, checksumError.Actual)
   }
   ```

2. **Contexts.** Every function that talks to the network takes a
   `context.Context` as its first argument, so the caller can cancel a
   download. The functions that only read the local file system do not.

3. **No callback dictionary.** The Python library passes a dictionary of
   callbacks around. The Go library uses a `*mcl.Progress` struct whose
   fields are optional functions. A nil `*mcl.Progress` is valid and discards
   every update.

4. **Structs instead of dictionaries.** `minecraft_launcher_lib.types`
   describes the launch options as a `TypedDict`. In Go they are a struct
   (`mcl.Options`), which makes typos a compile error. An empty string means
   "not set".

5. **The Mojang Java runtime is not ported.** The Python library keeps the
   old Mojang runtime installer as a commented out block because it was
   replaced by Temurin. Only the Temurin functions exist in Go.

6. **The Java runtime is not downloaded again.** `InstallJVMRuntime` returns
   immediately when a usable runtime is already installed. The Python library
   downloads it on every call.

7. **The jar of an inherited version is copied the other way round.** The
   Python library copies the jar of the child version over the jar of the
   parent when the child has none, which cannot work because the source file
   does not exist. The Go library copies the jar of the parent version to the
   child, which is what the Forge installer of the same library does.

8. **No LZMA decoder.** The Python library decompresses `.lzma` files of the
   old Mojang runtime. Since that runtime is not ported, the Go library does
   not need an LZMA decoder and stays free of external dependencies. The
   `client.lzma` file of a Forge installer is only extracted and handed to the
   Forge processors, exactly like in the Python library.

9. **The Forge processors are executed with a context.** A cancelled context
   stops the installation.

10. **Unknown operating system names in rules are ignored.** The Python
    library only knows `windows`, `osx` and `linux` and ignores everything
    else, the Go library behaves the same way.

11. **The operating system version is empty on macOS.** The Python library
    returns the version of the kernel, but the rules of Minecraft never use
    it on macOS, so the Go library returns an empty string. On Windows the
    version is read from the registry, on Linux from `/proc`.

12. **The optional files of a modpack are a list of paths**, the same as in
    the Python library.

13. **`${clientid}` and `${auth_xuid}` are options.** The Python library reads
    them from the options dictionary, even though they are not part of its
    type. The Go library has `Options.ClientID` and `Options.AuthXUID` for
    them.

## Deprecated Python modules that are not ported

The Python library still ships the modules `forge`, `fabric` and `quilt`,
which were replaced by `mod_loader` and are marked as deprecated. They are not
ported; `mcl.Forge`, `mcl.Fabric` and `mcl.Quilt` replace them.
