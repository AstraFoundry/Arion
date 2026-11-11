# Building and distributing

`mygo build` turns a project into apps people install: an app bundle and a
disk image on macOS, an executable and an installer on Windows, an
executable, a desktop entry, an archive with its install script, a Debian
package and a PKGBUILD for Arch Linux on Linux.

```sh
bun run build        # mygo build, in a project with a web frontend
go tool mygo build   # in a project of native UI
```

It:

1. with a web frontend, writes the TypeScript client, so the frontend
   builds against the Go code, and runs `buildCommand` from the
   configuration, which builds the frontend into `frontendDist`;
2. compiles the app, with the frontend embedded when there is one, for
   production: without the web inspector, and with the name, identifier
   and version of the configuration linked in;
3. packages it for each platform in `build/<os>-<arch>/` (`out` in
   the configuration);
4. signs what it can, and with [updates](updates.md) configured, writes the
   signed update archives, and delta updates from the published versions.

## Platforms

`-platform` lists `GOOS/GOARCH` targets, by default the machine's own:

```sh
mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64
```

`darwin/universal` combines arm64 and amd64 in one app. MyGo needs no cgo,
so any machine compiles for every platform. Some steps need the tools of a
platform, and are skipped with a note elsewhere: macOS apps are signed and
put in disk images on macOS only, and Windows installers are made on other
systems only where [NSIS](#the-installer) is installed.

The CLI defaults to `CGO_ENABLED=0`. If an app's dependencies need cgo,
set `CGO_ENABLED=1` in the environment for `mygo dev`, `mygo build` and
`mygo generate`. Those builds need a C compiler for the target platform;
when cross-compiling, set `CC` to a suitable cross compiler.

| Platform | In `build/<os>-<arch>/` |
|---|---|
| macOS | `My App.app`, signed, and `my-app-0.1.0-arm64.dmg` (`my-app-0.1.0.dmg` for `darwin/universal`), with [updates](updates.md) also `install.sh` |
| Windows | `My App.exe`, the files of the app, and the installer `my-app-setup-0.1.0-amd64.exe`, with [updates](updates.md) also [`install.ps1`](#install-scripts-for-macos-and-windows) |
| Linux | `my-app`, `my-app.desktop`, `my-app.png`, `my-app.xml` for the file types the app defines, the files of the app, their archive `my-app-0.1.0-linux-amd64.tar.gz` with [`install.sh`](#the-install-script), the Debian package `my-app_0.1.0_amd64.deb`, and a [`PKGBUILD`](#arch-linux) with the icons it installs, `my-app-256x256.png` and `my-app-512x512.png` |

Other flags: `-debug` keeps development features such as the inspector,
`-skip-dmg`, `-skip-notarize` and `-skip-installer` (the Windows
installer) skip those steps, `-sign` overrides the
signing identity of macOS, and `-o` the output directory. See
[the CLI](cli.md#mygo-build).

## Name, icon and version

The configuration describes the app:

```ts
export default defineConfig({
  name: "My App",
  identifier: "com.example.myapp",
  version: "1.2.0",
  copyright: "© 2026 Example Inc.",
});
```

- `name` is what users see: the app bundle, the executable, menus, the
  installer.
- `identifier` is a reverse DNS name unique to the app. Systems key
  preferences, permissions and registrations on it: keep it once released.
- `version` is the app's version, which `App.Version()` returns and
  [updates](updates.md) compare.

The icon is `resources/icon.png`, or the `icon` of the configuration: a square PNG,
ideally 1024×1024. `mygo build` makes it the `.icns` of macOS, the icon
resource of the Windows executable and the icon of the Linux desktop entry.
Beside it, the same name with `-dev` (`resources/icon-dev.png`) is the icon
of the development app that `mygo dev` builds, when it exists, to tell it
from the app in the Dock or the taskbar; it does not ship with the app.

On macOS, `macos.icon` adds an asset catalog, which macOS prefers to the
`.icns`: an `Assets.car` compiled with `actool`, whose app icon is named
`AppIcon` (or what `macos.infoPlist` sets as `CFBundleIconName`), or an
Icon Composer `.icon`, the layered icon of macOS 26, which `mygo build`
compiles with `actool` from Xcode 26 or later, with the `.icns` that
`actool` renders from it in place of the PNG's. Building elsewhere, or when
the compilation fails, skips the `.icon` with a message, and the app keeps
the PNG's `.icns`. `mygo dev` shows the `.icns`.

`macos.icon` can also be an `.icns` of your own, which the app gets in place
of the one rendered from the PNG: for an icon drawn per size, such as a
closer crop for 16 and 32 points, made with `iconutil` from an `.iconset`.
`mygo dev` still shows the PNG's.

```ts
export default defineConfig({
  macos: { icon: "AppIcon.icon" }, // outside resources, which ship as they are
});
```

For builds without Xcode, such as CI on Linux, commit the `Assets.car`
that `actool` compiles from the `.icon` and make it `macos.icon`. The
`.icon` has to be named `AppIcon.icon`: `actool` names the icon after the
file, and compiles nothing when `--app-icon` differs.

```sh
mkdir -p icon
xcrun actool AppIcon.icon --compile icon --platform macosx --target-device mac \
  --minimum-deployment-target 12.0 --app-icon AppIcon \
  --output-partial-info-plist icon/partial.plist
```

This writes `icon/Assets.car`, for `macos: { icon: "icon/Assets.car" }`.

## Resources

Files the app reads at run time, such as a database seed, a helper binary
or the images of a tray icon, ship with it. Everything in the `resources`
directory of the project is copied into the app, but Go sources and what
[`resourcesExclude`](#leaving-files-out) leaves out, as are the files and
directories listed in `resources` in the configuration, under their base names:

```ts
export default defineConfig({
  resources: ["third_party/licenses", "bin/helper"],
});
```

At run time `App.Path(mygo.PathResources)` is where they are:
`Contents/Resources` in a macOS app, the executable's directory elsewhere,
and the project's `resources` directory under `go run` and `go test`:

```go
dir, _ := mygo.App.Path(mygo.PathResources)
seed := filepath.Join(dir, "seed.db")
```

`mygo dev` copies them into the development app too, and rebuilds it when
they change. A dotfile inside a resource directory ships: packages read
such names, as dsh's provider catalog reads
`dist/providers/data/.manifest.json`. Hidden directories (`.git`, `.cache`)
are left out, and so are `.DS_Store`, `Thumbs.db` and `desktop.ini`, and
the top-level entries of `resources` whose names start with a dot, which
belong to the repository rather than the app. A dotfile you would rather
not ship, such as a `.env` beside a resource, is left out with
[`resourcesExclude`](#leaving-files-out).

### Leaving files out

Go source files in `resources` are left out of the apps: builds read
them, such as a package that embeds the images beside it with
`//go:embed`, and apps never do. Other files that only the project needs,
such as design files, are left out with `resourcesExclude`, patterns of
their paths in `resources`:

```ts
export default defineConfig({
  resourcesExclude: ["**/*.psd", "design"],
});
```

Each segment of a pattern, between slashes, matches as in Go's
`path.Match` (`*`, `?`, `[a-z]`), and `**` matches any number of
directories, none included: `**/*.psd` leaves out every `.psd` file,
`*.psd` only those at the top of `resources`. A directory that matches is
left out with what it holds, so `design` and `design/**` both leave out
`resources/design`. Paths in [platform directories](#platform-resources)
start with the directory's name, as in `linux-amd64/bin/server.pdb`, which
`**/*.pdb` matches too. Builds and `mygo dev` leave out the same files.
The files and directories listed in `resources` ship whole, Go sources
included.

### Platform resources

Some resources belong to one platform, such as a program built for one
system and processor. Directories of `resources` named after a platform
hold them, laid out like `resources` itself:

```
resources/
├── data/seed.db                    every app
├── darwin/                         macOS apps
├── darwin-arm64/bin/server         macOS on Apple silicon
├── darwin-amd64/bin/server         macOS on Intel
├── linux-amd64/bin/server          Linux on x86-64
├── linux-arm64/bin/server          Linux on ARM64
└── windows-amd64/bin/server.exe    Windows on x86-64
```

They are named like the targets of `-platform`: a system, `darwin`,
`linux` or `windows`, for all of its apps, or a system and an
architecture, as in `darwin-arm64` or `windows-amd64`, for one. An app
ships the platform directories of its target merged with the other
resources, so the Linux x86-64 app above has `data/seed.db` and its own
`bin/server`, and nothing of the other platforms. A path that two
directories would both install fails the build, and so does a directory
named the way other tools name platforms, such as `darwin-x64` or `macos`,
which would ship to every platform. The build also notes a target without
a directory of its own when another architecture of its system has one,
such as `linux/arm64` next to `linux-amd64`.

A universal macOS app (`darwin/universal`) ships `darwin` and
`darwin-universal`, and combines `darwin-arm64` with `darwin-amd64`:
every file of one needs its counterpart in the other, programs and
libraries become universal binaries, as `lipo` makes them, and other files
must be identical. Put what serves both architectures, such as programs
that already are universal binaries, in `darwin`.

`mygo dev` ships the platform directories of the machine it runs on.
Under `go run` and `go test`, `PathResources` is the `resources`
directory as it is, platform directories included.

### Helper executables

Programs the app runs, such as a server written in another language or a
tool like `ffmpeg`, are resources too: put the build for each platform in
its [platform directory](#platform-resources) and run it from there.

```go
dir, _ := mygo.App.Path(mygo.PathResources)
cmd := exec.Command(filepath.Join(dir, "bin", "server"))
```

The same code runs `bin\server.exe` on Windows, where `exec.Command`
finds the program of an absolute path without its extension.

`mygo build` signs them with the app, so that Gatekeeper and SmartScreen
accept them and notarization passes:

- On macOS, every Mach-O executable and library among the resources, and
  every app, framework or plug-in bundle holding some, is signed from the
  inside out with `signingIdentity`, with the hardened runtime and a
  timestamp for a Developer ID. Signed ad hoc, the default, only code
  without a signature is signed: Apple silicon runs no unsigned code.
- On Windows, with `certificate` or `signCommand`, every executable and
  DLL among the resources that has no signature is signed. Those signed by
  their publishers keep their signatures.

Under the hardened runtime, some programs need entitlements, such as
`com.apple.security.cs.allow-jit` for a JavaScript runtime that compiles
code as it runs. Code keeps the entitlements it was signed with, so the
official builds of such runtimes work as they are, and
`macos.helperEntitlements` gives code entitlements of its own, by its path
in the app's resources, such as `bin/server` for
`resources/darwin-arm64/bin/server`:

```ts
export default defineConfig({
  macos: {
    signingIdentity: "Developer ID Application: Jane Doe (TEAMID)",
    helperEntitlements: { "bin/server": "server.entitlements.plist" },
  },
});
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>com.apple.security.cs.allow-jit</key>
  <true/>
</dict>
</plist>
```

### Native libraries of packages

A Go package that loads a native library with purego, as the
[terminal plugin](plugins/terminal.md) loads libghostty-vt, names its
builds in a `mygo-plugin.json` next to its sources, with a file per
platform, where to download it and its SHA-256:

```json
{
  "libraries": [
    {
      "name": "libexample",
      "files": {
        "darwin-arm64": { "name": "libexample.dylib", "url": "https://…/libexample-darwin-arm64.dylib", "sha256": "…" },
        "linux-amd64": { "name": "libexample.so", "url": "https://…/libexample-linux-amd64.so", "sha256": "…" },
        "windows-amd64": { "name": "example.dll", "url": "https://…/example-windows-amd64.dll", "sha256": "…" }
      }
    }
  ]
}
```

`mygo build` and `mygo dev` find these files in the packages the app is
built from, download each library once into the user's cache
(`<cache>/mygo/natives/<sha256>/`), check its SHA-256, and install it at
the top of the app's resources under its name, signed like the other
code there; a universal macOS app gets both architectures in one file.
The package loads it from `PathResources`. Building for a platform the
package has no file for fails.

## macOS

`mygo build` makes an app bundle, signs it, and puts it in a disk image
whose window invites users to drag the app to Applications.

### Signing and notarization

Apps signed ad hoc, the default, run on the Mac that built them, but
Gatekeeper blocks them on others. To ship an app, sign it with a Developer
ID of the [Apple Developer Program](https://developer.apple.com/programs/)
and have Apple notarize it:

```ts
export default defineConfig({
  macos: {
    signingIdentity: "Developer ID Application: Jane Doe (TEAMID)",
    notarize: { keychainProfile: "notary" },
  },
});
```

Store the credentials of the notary service in the keychain once:

```sh
xcrun notarytool store-credentials notary
```

`mygo build` then signs the app with the hardened runtime, submits the disk
image to the notary service, and staples the ticket to the disk image and
to the app, which updates ship. `security
find-identity -v -p codesigning` lists your identities, and `mygo doctor`
counts the Developer IDs.

### Info.plist and entitlements

`macos.infoPlist` adds keys to the app's `Info.plist`, or replaces MyGo's,
for example the usage descriptions that pages using the camera or the
microphone need:

```ts
export default defineConfig({
  macos: {
    minimumSystemVersion: "13.0",
    infoPlist: {
      NSCameraUsageDescription: "Scan documents with the camera.",
      LSApplicationCategoryType: "public.app-category.productivity",
    },
    entitlements: "entitlements.plist",
    dmgTitle: "My App Installer",
  },
});
```

`entitlements` signs the app with a property list of entitlements, and
`helperEntitlements` [helper executables](#helper-executables).
`minimumSystemVersion` is the oldest macOS the app runs on, 12.0 by
default.

### Provisioning profiles

Restricted entitlements, such as Communication Notifications, need a
provisioning profile that authorizes them, including for Developer ID
distribution. Set `macos.provisioningProfile` to its path, relative to the
project or absolute. `build` and `dev` copy it to the bundle's
`Contents/embedded.provisionprofile` before signing, so the signed app,
disk image and update archive contain it.

```ts
export default defineConfig({
  macos: {
    signingIdentity: process.env.MACOS_SIGNING_IDENTITY,
    provisioningProfile: process.env.MACOS_PROVISIONING_PROFILE,
    entitlements: "entitlements.plist",
  },
});
```

The profile must match the signing certificate, bundle identifier and
entitlements, including the application and team identifiers. MyGo embeds
it as supplied; it does not create profiles or derive entitlements from
them. Apple validates their authorization when the app runs. A profile
cannot be used with ad-hoc signing (`-sign -`). `dev` signs ad hoc unless
it is given `-sign` (it does not use `macos.signingIdentity`), and its app's
identifier is `<identifier>.dev`, which the app's profile does not
authorize: leave the profile and the restricted entitlements out of
development builds with a [computed configuration](configuration.md#computed-configuration),

```ts
export default defineConfig(({ command }) => ({
  macos: {
    signingIdentity: process.env.MACOS_SIGNING_IDENTITY,
    ...(command === "build" && {
      provisioningProfile: process.env.MACOS_PROVISIONING_PROFILE,
      entitlements: "entitlements.plist",
    }),
  },
}));
```

or pass `-sign` and give `dev` a profile for `<identifier>.dev`. It
watches the profile and rebuilds when it changes.

Keep profiles out of version control and provide their paths through the
build environment. See Apple's
[Inside code signing: provisioning profiles](https://developer.apple.com/documentation/technotes/tn3125-inside-code-signing-provisioning-profiles).

## Windows

The executable carries the icon, the version information and a manifest
(per-monitor DPI awareness, modern controls), and runs without a console
window. A `.syso` file of your own in the main package replaces them.

### The installer

`mygo build` also makes `my-app-setup-1.2.0-amd64.exe`, named after the
app's [`slug`](configuration.md#the-app) and its architecture as the disk
images are, so that the installers of two
targets, uploaded to one release, keep both. It installs the app for
the current user in `%LOCALAPPDATA%\Programs\My App`, which needs no
administrator rights and lets the app [update itself](updates.md), adds a
Start menu shortcut and an uninstaller listed in Settings, and registers
the app's URL schemes and file associations. Its last page offers to run
the app and to create a desktop shortcut, which is checked; silent
installs (`/S`) create none.

The shortcuts, the entry in Settings > Apps and the installer's title
show the app's `name`, or
[`windows.displayName`](configuration.md#windows) when the executable and
its folder should keep a plain name, as `weather.exe` in
`%LOCALAPPDATA%\Programs\weather` for an app that shows as `天气助手`:

```ts
export default defineConfig({
  name: "weather",
  windows: { displayName: "天气助手" },
});
```

Characters that file names cannot hold, such as `:` and `?`, become `-`
in the names of the shortcuts only. The installer of a version that sets
`displayName` replaces the shortcuts that an earlier version made under
the name.

[`windows.installDirName`](configuration.md#windows) names the install
folder alone, for an app that keeps its name everywhere else, as an ASCII
folder for tools that handle only ASCII paths, such as scripts in the
console's code page. Here the app installs in
`%LOCALAPPDATA%\Programs\ZhiLian`, and its executable, shortcuts and entry
in Settings > Apps are named `智链`:

```ts
export default defineConfig({
  name: "智链",
  windows: { installDirName: "ZhiLian" },
});
```

An installed app stays in its folder: the installer of a newer version
installs in the folder of the installed one, which the user may have
chosen, as [`install.ps1`](#install-scripts-for-macos-and-windows) and
[updates](updates.md) do. A new `installDirName` applies to new installs
only, and to the app once it is uninstalled and installed again.

The installer is made with [NSIS](https://nsis.sourceforge.io). On
Windows, `mygo build` uses NSIS when it is installed, and otherwise
downloads it the first time it needs it: the official NSIS 3.13 zip, from
a copy on MyGo's GitHub releases or else from SourceForge, checked against
its SHA-256 and kept in `%LOCALAPPDATA%\mygo`. On macOS and
Linux, install NSIS to make Windows installers (`brew install makensis`,
`apt install nsis`); without it, `mygo build` skips the installer.

`mygo build -skip-installer` makes no installer, and neither looks for
NSIS nor downloads it: the build has the app's files to ship as you
choose, and with [updates](updates.md) the update archive and manifest,
which apps installed before keep updating from. It writes no `install.ps1`
either, which runs the installer, so the one-line install needs a release
that has both.

### Installer hooks

`windows.installerInclude` adds steps of the app's own to the installer:
an NSIS script, relative to the project's directory, that the installer
includes after MUI2.nsh and its languages, as electron-builder's `include`
does. Of the macros it defines, `customInstall` runs at the end of the
install, once the app's files are in `$INSTDIR`; `customUnInstall` runs in
the uninstall, before the uninstaller removes `$INSTDIR`; and `customInit`
runs in `.onInit`, when the installer starts, e.g. to check what the app
needs before anything is installed. A macro the script does not define is left out, and
`mygo build` stops when the script does not exist.

```ts
export default defineConfig({
  windows: { installerInclude: "build/installer.nsh" },
});
```

The script below copies a [helper executable](#helper-executables) that
the app ships as `resources/windows/bin/helper.exe`, installed as
`$INSTDIR\bin\helper.exe`, into the user's data directory, and removes it
on uninstall:

```nsis
; build/installer.nsh
!macro customInstall
  CreateDirectory "$LOCALAPPDATA\My App\bin"
  CopyFiles /SILENT "$INSTDIR\bin\helper.exe" "$LOCALAPPDATA\My App\bin"
!macroend

!macro customUnInstall
  Delete "$LOCALAPPDATA\My App\bin\helper.exe"
  RMDir "$LOCALAPPDATA\My App\bin"
!macroend
```

The macros run in silent installs and uninstalls (`/S`) too, as
[`install.ps1`](#install-scripts-for-macos-and-windows) runs the installer:
`IfSilent` tells them apart, and a message box needs `/SD` to answer
itself. [Updates](updates.md) replace the app's files without the
installer, so they run none of the macros. Functions that
`customUnInstall` calls are the uninstaller's, named with `un.`.

The installer installs for the current user without administrator rights,
so that the app can update itself: the hooks should keep to what the user
can write, such as `$LOCALAPPDATA` and `HKCU`. The installer already lets
the user choose the folder, and the uninstaller removes the app's folder
only, keeping the app's data. Signing applies to the installer and its
uninstaller as it does without hooks.

### Code signing

Unsigned apps make SmartScreen warn users. Sign the executable, the
[helper executables](#helper-executables), the installer and its
uninstaller with a certificate:

```ts
export default defineConfig({
  windows: { certificate: "certs/code-signing.pfx" },
});
```

The password comes from the `MYGO_WINDOWS_CERTIFICATE_PASSWORD`
environment variable. On Windows `signtool` signs, from the Windows SDK;
elsewhere `osslsigncode`. `timestampUrl` changes the time stamping server,
by default `http://timestamp.digicert.com`.

Certificates on hardware tokens or in cloud services, such as Azure Trusted
Signing, sign with a command of their own, in which `%1` is the file:

```ts
export default defineConfig({
  windows: {
    signCommand: "signtool sign /fd sha256 /tr http://timestamp.digicert.com /td sha256 /a %1",
  },
});
```

## Linux

`mygo build` writes the executable, named after the app's
[`slug`](configuration.md#the-app), which is its name in lower case by
default, a desktop entry and the icon, and a Debian package, which
installs the app in `/opt/my-app` with its desktop entry, icons, URL
schemes and file types:

```ts
export default defineConfig({
  linux: {
    maintainer: "Jane Doe <jane@example.com>",
    comment: "Take notes",
    categories: ["Office"],
    depends: ["libayatana-appindicator3-1"],
    command: "my-app",
  },
});
```

The package depends on GTK 3 and WebKitGTK; `depends` adds more packages.
An app that shows no web page, in a window or a `ui.WebView`, as an app of
[native UI](native.md), needs GTK alone: `webview: false` leaves WebKitGTK
out of its package, and out of what `install.sh` and `mygo doctor` (in the
project) check, as the app never loads it.
`maintainer` defaults to the `author` of package.json, else the name of
the app.
`comment` describes the app in its desktop entry and package, and
`categories` places it in application menus (`Utility` by default).
The app opens from the applications menu. `command` also gives it a
command, `/usr/bin/my-app` from the package and `~/.local/bin/my-app` from
[`install.sh`](#the-install-script), or several, as `["my-app", "ma"]`,
which adds a short name; without it, neither adds a command, which leaves
the name to a command-line tool of the same name, such as the app's own.

The desktop entry names the window class of the app, `StartupWMClass`,
as the executable of `command`: an X11 session groups the window with the
entry and shows its icon in the taskbar, where Wayland matches the same
name as the app id.

Keep the slug of a released app. Its Linux files are named after it, and
an update replaces the files of the same names: one to a version with
another slug leaves the old executable in place, which the desktop entry
that `install.sh` registered still runs, and the Debian package of the
new slug is another package, installed next to the old one. Choose the
slug before the first release, as `cloud-music` for `云音乐`, whose own
would be `yun-yin-le`.

### Arch Linux

Next to the archive, every Linux build also gets a `PKGBUILD`, from which
`makepkg` packages the archive for Arch Linux, checked against its
SHA-256, and installs it with pacman:

```sh
cd build/linux-amd64
makepkg -si
```

The package installs what the Debian package does: the app in
`/opt/my-app`, its commands in `/usr/bin`, its desktop entry, icons, URL
schemes and file types, which pacman's hooks register. The icons are
the files `my-app-256x256.png` and `my-app-512x512.png` next to it, as
the archive holds the original. Its fields come from the configuration:

| Field | From |
|---|---|
| `pkgname` | the slug |
| `pkgver` | the version without its hyphen, which a pkgver cannot hold: `1.2.0beta.1` of `1.2.0-beta.1`, which pacman sorts before `1.2.0` as it should (it would sort `1.2.0_beta.1` after it), `1.2.0pre1` of `1.2.0-1`, and `1.2.0_build.5` of `1.2.0+build.5` |
| `pkgdesc` | `linux.comment`, else the name |
| `arch` | `x86_64` of amd64, `aarch64` of arm64 |
| `url` | `homepage` in package.json, else the repository of `updates.github` |
| `license` | `license` in mygo.json, else in package.json, else `LicenseRef-custom` |
| `depends` | `gtk3` and, unless `linux.webview` is `false`, `webkit2gtk-4.1`, then the Arch Linux packages of `linux.depends` that MyGo knows, as `libnotify` of `libnotify4`; the build warns about a Debian package it knows no Arch Linux package for, and leaves it out |

`-upload` leaves the PKGBUILD out: it packages the archive next to it, and
every architecture has its own. Copy it, with the icons, next to the
archive downloaded from the release to package the release.

### The install script

Every Linux build also gets the app as an archive,
`my-app-1.2.0-linux-amd64.tar.gz`, and `install.sh`, which installs it for
the user without root: the app in `~/.local/my-app.app`, where it can
[update itself](updates.md), its desktop entry, icon, URL schemes and file
types, and with `linux.command` its commands in `~/.local/bin`, each
unless another program is there; it says to run the app by the first it
links. With [updates](updates.md) configured, it downloads the latest
version, which the update manifest of the machine names, and checks its
size and its Ed25519 signature against `updates.publicKey`, as the app's
updater does, before installing it, so one line installs the app:

```sh
curl -fsSL https://github.com/me/my-app/releases/latest/download/install.sh | sh
```

With a [`tagPrefix`](updates.md#set-up) other than `v`, the script of any
release, such as `…/releases/download/desktop-v1.2.0/install.sh`, installs
the newest release tagged with the prefix.

It checks the signature with `openssl pkeyutl -rawin`, which OpenSSL 3.0
and later have (Debian 12, Ubuntu 22.04, Fedora 36, RHEL 9 and later).
Without it, as with OpenSSL 1.1.1, it downloads nothing and says so; the
archive can then be downloaded from the release and given to the script,
which installs an archive it is given, or finds next to it, unchecked.

`-upload` publishes it with the archives; with `updates.url`, publish it
next to the manifests. Next to the archive of its version, as in
`build/linux-amd64/`, it installs that archive instead, and it installs
the one it is given:

```sh
sh install.sh my-app-1.2.0-linux-amd64.tar.gz
sh install.sh --uninstall   # keeps the settings and data of the app
```

Running it again installs the version it finds over the installed one; a
link of a name the app no longer has, other than its slug, stays, as one
the user made would, until `--uninstall` removes every link to the app in
`~/.local/bin`.
The app needs WebKitGTK, which `install.sh` does not install: when the
system's library cache has none, it says so after installing, with the
command that installs it on Debian, Ubuntu, Fedora, Arch and openSUSE,
unless `linux.webview` is `false`.
When the app updates itself, it registers the desktop entry and file types
of the new version, so a version that adds URL schemes or file types gets
them. The Debian package suits people who want the system's package manager to
update the app; `install.sh` suits the others, and is the Linux install
that updates itself.

### Install scripts for macOS and Windows

With [updates](updates.md), the same `install.sh` installs the app on
macOS, and Windows builds get `install.ps1`, so one line installs the app
on each system:

```sh
curl -fsSL https://github.com/me/my-app/releases/latest/download/install.sh | sh
```

```powershell
irm https://github.com/me/my-app/releases/latest/download/install.ps1 | iex
```

On macOS, `install.sh` downloads the archive that the update manifest of
the Mac's architecture, else of `darwin-universal`, names (Apple silicon's,
`darwin-arm64`, also from a shell under Rosetta), checks its size
and its Ed25519 signature against `updates.publicKey`, as the app's
updater does (macOS has no tool for Ed25519 signatures, so the script
hashes with `openssl` and `shasum` and does the arithmetic of the curve in
JavaScript for Automation), quits the app if it runs, and replaces the
bundle in `/Applications`, or in `~/Applications` when the user cannot
write to `/Applications` or the app is already there, then opens it. It
never asks for root, and an app of the same name with another bundle
identifier is left alone. `curl` sets no quarantine, so the app opens
without the prompt of Gatekeeper for downloaded apps.
`sh install.sh --uninstall` removes the app and keeps its settings and
data.

`install.ps1` runs in Windows PowerShell 5.1 and PowerShell 7. It
downloads the installer of the latest version for the PC's architecture
(`amd64` or `arm64`), checks its SHA-256 against `SHA256SUMS`, quits the
app if it runs, runs the installer silently (`/S`) in the folder of the
installed version, and opens the app. The installer uninstalls it as
usual, from Settings > Apps.

Builds with updates write `SHA256SUMS`, the SHA-256 of the installers,
archives and scripts in the format of `sha256sum`, to the output
directory, which `-upload` publishes with them, merging the lines of
earlier uploads of other platforms: those of the same GitHub release, or
those of the `SHA256SUMS` that `updates.url` serves for `updates.s3`,
without the files of earlier versions that the upload replaces. It names
the files as the build does; GitHub turns the spaces of uploaded names
into dots, which `install.ps1` follows. With `updates.url` and your own
server, publish the scripts and `SHA256SUMS` next to the manifests, with
the lines of every platform's latest files.

`SHA256SUMS` is not signed: it catches a download that is broken or cut
short, not a release whose files someone replaced. On Windows, sign the
installer ([code signing](#code-signing)) so that Windows shows who made it.

For testing, `MY_APP_DOWNLOAD_URL` (the [slug](configuration.md#the-app) of the app in capitals) is
where both scripts download from instead, `MY_APP_INSTALL_DIR` the folder
of the app on macOS, and `MY_APP_NO_OPEN=1` leaves the app closed.

## Languages

An app with [catalogs](i18n.md) declares their languages to each system,
its source language first, which the system falls back to:

- macOS: `CFBundleDevelopmentRegion`, `CFBundleLocalizations` and a
  localization folder per language in `Contents/Resources`, named as Xcode
  names them (`de.lproj`, `zh-Hans.lproj`, `zh-HK.lproj` for `zh-Hant-HK`),
  so that AppKit says its own texts (the Services menu, the panels) in the
  language the app shows, and System Settings lists the app's languages.
- Linux: `Name[xx]` and `Comment[xx]` in the desktop entry, by POSIX
  locale (`de`, `zh_CN`, `zh_TW`, `sr@latin`).
- Windows: the installer is in each of the languages, and picks the
  one of the user's display language, or else the source language; an
  app written in Chinese alone (`"i18n": { "source": "zh-Hans" }`) has
  an installer in Chinese only, but for MyGo's own few texts in it, the
  messages about the running app, which are in English. The finish
  page's check box that creates a desktop shortcut is MyGo's string
  `mygo.desktopShortcut`, in English and
  [MyGo's own languages](i18n.md#mygos-own-strings); an app translates
  it to its other languages, or changes it, in its catalogs. NSIS 3
  has files of some 60 languages, and a language it has none of stops
  the build, saying which it has. The file properties have a table of
  strings per language.

The app's name and description in a language are its catalog's
`mygo.appName` and `mygo.appDescription`; the configuration's `name` and
`linux.comment` are those of the source language, and on Windows
`windows.displayName` in place of `name` when it has one. An app without catalogs
declares no languages.

On macOS, the other strings of `Info.plist`, such as the reasons the
system shows as it asks for the microphone or the system's audio
(`NSMicrophoneUsageDescription`, `NSAudioCaptureUsageDescription`), are
translated as Xcode's projects translate them: by an `InfoPlist.strings`
of the language among the app's [resources](#platform-resources), which
the build keeps, adding the app's name where the file has none.

```
resources/darwin/
├── ja.lproj/InfoPlist.strings        "NSMicrophoneUsageDescription" = "音声メモを録音します。";
└── zh-Hans.lproj/InfoPlist.strings   "NSMicrophoneUsageDescription" = "用麦克风录语音笔记。";
```

The file is the text format Xcode writes, in UTF-8 or UTF-16, or a
property list in XML.

## URL schemes and file types

`urlSchemes` and `fileAssociations` in the configuration (see
[deep links](app.md#deep-links) and [file associations](app.md#file-associations))
are registered by each package: in the macOS app's `Info.plist`, by the
Windows installer, and by the Linux desktop entry, install script,
Debian package and PKGBUILD.

## Publishing

With `updates.github` set in the configuration, `mygo build -upload` uploads the
disk images, installers, packages, Linux archives and install script, and
update files to the GitHub release of the version, tagged `v1.2.0`,
creating it as a draft. Review the draft and publish it. It needs the
[GitHub CLI](https://cli.github.com) (`gh`), signed in. With `updates.s3`,
it uploads them to a bucket of Amazon S3 or of a compatible service, such
as Cloudflare R2, with the credentials in `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACCESS_KEY` (see [publishing to S3](updates.md#publishing-to-s3)).

Build on each platform in CI and upload to the same release: macOS runners
sign, notarize and make the disk images; any runner compiles and packages
Windows and Linux apps. Keep the signing secrets in CI secrets:
`MYGO_WINDOWS_CERTIFICATE_PASSWORD`, `MYGO_UPDATER_PRIVATE_KEY`, and the
notary credentials in a keychain of the macOS runner.
[GitHub Actions](github-actions.md) has a workflow that does it all.
