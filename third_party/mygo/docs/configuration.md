# Configuration

`mygo.config.ts`, at the root of a project, describes the app and how the
CLI develops and builds it. Its default export is the configuration, typed
by `defineConfig` from the `mygo-cli` package that projects depend on, so
editors complete and document every field:

```ts
import { defineConfig } from "mygo-cli";

export default defineConfig({
  name: "My App",
  slug: "my-app",
  identifier: "com.example.myapp",
  version: "1.2.0",
  copyright: "© 2026 Example Inc.",

  devUrl: "http://localhost:5173",
  devCommand: "bun run dev:web",
  buildCommand: "bun run build:web",
  frontendDist: "dist",
  bindings: "src/mygo.ts",
  out: "build",

  resources: ["third_party/licenses"],
  urlSchemes: ["myapp"],
  fileAssociations: [{ ext: ["md"], name: "Markdown Document", mimeType: "text/markdown" }],

  i18n: { dir: "locales", source: "en" },
  updates: { publicKey: "…", github: "you/my-app" },
  macos: { signingIdentity: "Developer ID Application: Jane Doe (TEAMID)" },
  windows: { certificate: "certs/code-signing.pfx" },
  linux: { maintainer: "Jane Doe <jane@example.com>" },
});
```

Every field is optional. The configuration can also be JSON, in
[mygo.json](#mygojson), as projects of [native UI](ui/README.md) have it: they
have no JavaScript, and no frontend to develop or build, so they leave out
the fields of [the frontend](#development-and-the-frontend).

## The app

| Field | Default | |
|---|---|---|
| `name` | the directory's name | the name users see: the app, its executable, menus, installers; Windows can show another, [`windows.displayName`](#windows) |
| `slug` | the name in lower case, with hyphens between its words, e.g. `my-app` of `My App`; Chinese characters take their most common Mandarin reading without tones, each its own word, and kana and Hangul their romanization, e.g. `yun-yin-le` of `云音乐`, `ramen` of `ラーメン` | the name of the app's files where a name must be plain ASCII: the disk images of macOS and the installers of Windows, the Linux executable, its desktop entry, icon, Debian package and `/opt` directory, the [update archives](updates.md) of every platform, the variables of the [install scripts](distribution.md#install-scripts-for-macos-and-windows) and the MIME types the app defines; lower-case letters and digits with single hyphens between them. Set it to choose another, as `cloud-music` for `云音乐`, and keep it once the app is released (see [Linux](distribution.md#linux)) |
| `identifier` | `com.mygo.` and the name's letters and digits, e.g. `com.mygo.myapp` | a reverse DNS name unique to the app, e.g. `com.example.myapp`; systems key preferences, permissions and registrations on it |
| `version` | `0.1.0` | the version of the app, which `App.Version()` returns and updates compare |
| `copyright` | | the copyright notice of the app, in its `Info.plist` (macOS) and file properties (Windows) |
| `publisher` | the copyright notice without its "©" and years, e.g. `Example Inc.` of `© 2026 Example Inc.` | who makes the app: the publisher that Settings > Apps lists for the installed app on Windows, and its company in the file properties |
| `license` | `license` in package.json | the SPDX license expression of the app, such as `MIT` or `Apache-2.0 WITH LLVM-exception`, which the PKGBUILD of Arch Linux records |
| `icon` | `resources/icon.png` when it exists | a square PNG, ideally 1024×1024; its name with `-dev`, as `icon-dev.png`, beside it is the development app's, when it exists |
| `main` | `.` | the Go package of the app, relative to the project |

## Development and the frontend

| Field | Default | |
|---|---|---|
| `devUrl` | | the dev server that `mygo dev` points the app at, an `http(s)` URL; windows load it for URLs without a scheme, such as `/` |
| `devCommand` | | runs in the project directory while `mygo dev` runs, e.g. a Vite dev server; the app starts once `devUrl` answers |
| `buildCommand` | | builds the frontend before `mygo build` compiles the app |
| `frontendDist` | | the directory of the built frontend, which `mygo build` embeds into the app; `mygo dev` serves it from disk when there is no `devUrl` |
| `bindings` | `frontend/src/mygo.ts` when `frontend/package.json` depends on `mygo-runtime`, else `src/mygo.ts` when package.json at the root does, else `mygo.ts` when the configuration has a frontend (`devUrl`, `devCommand`, `buildCommand` or `frontendDist`); none for an app without one, such as one of [native UI](ui/README.md) | where `mygo generate` writes the TypeScript client |
| `out` | `dist` | where `mygo build` writes the builds, one directory per platform |

See [the frontend](frontend.md#how-pages-load) for how they fit together.
New projects set `out` to `build`, apart from Vite's `dist`.

## Integration

| Field | |
|---|---|
| `resources` | files and directories copied into the app of every platform, next to the contents of the `resources` directory, under their base names; see [resources](distribution.md#resources), and [platform resources](distribution.md#platform-resources) for the files of one platform |
| `resourcesExclude` | files of the `resources` directory left out of the apps: patterns of their paths there, such as `["**/*.psd", "design"]`, where `**` matches any number of directories; Go source files are always left out. See [leaving files out](distribution.md#leaving-files-out) |
| `urlSchemes` | the URL schemes the app opens, such as `myapp` for `myapp://…`, which reach `App.OnOpenURL`; see [deep links](app.md#deep-links) |
| `fileAssociations` | the file types the app opens, which reach `App.OnOpenFile`; see [file associations](app.md#file-associations) |

Each file association has:

| Field | |
|---|---|
| `ext` | the file name extensions, without dots: `["md", "markdown"]` |
| `name` | what the files are: `"Markdown Document"` |
| `mimeType` | their MIME type, by which Linux identifies files; the app defines an unregistered one (`x-`), and without one `application/x-<app>-<ext>` |
| `role` | `Editor` (the default) or `Viewer`, on macOS |

## i18n

The catalogs of the app's messages, a JSON file per language; see
[Languages](i18n.md). Without `i18n`, the app ships English only.

| Field | Default | |
|---|---|---|
| `dir` | `locales` | the directory of the catalogs, `<tag>.json` for each language the app ships (`de.json`, `zh-Hant.json`); a Go package of its own, where `mygo generate` writes the Go functions of the messages |
| `source` | `en` | the language the app is written in, whose catalog has every message and which the others fall back to |

## updates

Signed [auto-updates](updates.md). `publicKey` and one of `github` and
`url` are required.

| Field | Default | |
|---|---|---|
| `publicKey` | | the contents of `mygo-update.pub`, from `mygo keygen` |
| `github` | | a public repository, `owner/name`, whose releases hold the updates |
| `tagPrefix` | `v` | what precedes the version in release tags; another prefix, such as `desktop-v`, makes apps find the newest release tagged with it rather than the repository's latest |
| `url` | | instead of `github`, the HTTPS URL of a directory holding the updates |
| `s3` | | the bucket that serves `url`, which `mygo build -upload` uploads to: `bucket`, and optionally `prefix`, its directory, `region` (default: `AWS_REGION`, else `us-east-1`), `endpoint`, of a compatible service such as Cloudflare R2, and `pathStyle` (see [publishing to S3](updates.md#publishing-to-s3)) |
| `privateKey` | | the path of `mygo-update.key`, for `mygo build`; the `MYGO_UPDATER_PRIVATE_KEY` environment variable, holding the key, takes precedence |
| `changelog` | `CHANGELOG.md` when it exists | a Markdown file whose `## <version>` section becomes the release notes |
| `deltas` | `3` | how many earlier versions get a [delta update](updates.md#delta-updates); `0` for none |

## macos

| Field | Default | |
|---|---|---|
| `icon` | | the icon as an asset catalog, an `Assets.car` or an Icon Composer `.icon`, beside the `.icns` of `icon`; or an `.icns` in its place. See [the icon](distribution.md#name-icon-and-version) |
| `minimumSystemVersion` | `12.0` | the oldest macOS the app runs on |
| `signingIdentity` | `-`, ad hoc | the identity that signs the app and its disk image, e.g. `Developer ID Application: Jane Doe (TEAMID)` |
| `entitlements` | | a property list of entitlements to sign the app with |
| `provisioningProfile` | | a profile embedded as `Contents/embedded.provisionprofile` before signing; must match the app's identifier, entitlements and signing identity; cannot be used with ad-hoc signing. See [provisioning profiles](distribution.md#provisioning-profiles) |
| `helperEntitlements` | | property lists of entitlements for code among the resources, by its path there, e.g. `{ "bin/server": "server.entitlements.plist" }`; other code keeps the entitlements it is signed with. See [helper executables](distribution.md#helper-executables) |
| `infoPlist` | | keys added to the app's `Info.plist`, replacing MyGo's, e.g. `NSCameraUsageDescription` |
| `dmgTitle` | `name` | the volume name of the disk image |
| `notarize` | | notarizes production disk images: `keychainProfile`, the profile of `xcrun notarytool store-credentials`, and optionally `keychain`, the keychain holding it |

See [macOS](distribution.md#macos).

## windows

| Field | Default | |
|---|---|---|
| `displayName` | `name` | the name Windows shows the app by: its Start menu and desktop shortcuts, its entry in Settings > Apps, the installer's title and messages, and the product name in the executable's file properties. The executable, its install folder and the files named after `name` or `slug` keep their names, e.g. `天气助手` for an app named `weather`, whose executable stays `weather.exe` |
| `installDirName` | `name` | the name of the folder the installer installs the app in, `%LOCALAPPDATA%\Programs\<installDirName>`, e.g. an ASCII `ZhiLian` for an app named `智链`; the executable, the shortcuts, the entry in Settings > Apps and everything else users see keep the name. No path separators, `:*?"<>\|`, or `.` and `..`. Changing it moves new installs only. See [the installer](distribution.md#the-installer) |
| `certificate` | | a code signing certificate (`.pfx`) for the executable, the unsigned executables and libraries among the resources, and the installer; its password comes from `MYGO_WINDOWS_CERTIFICATE_PASSWORD` |
| `signCommand` | | instead of `certificate`, a command that signs a file, with `%1` for its path |
| `timestampUrl` | `http://timestamp.digicert.com` | the time stamping server of `certificate` |
| `installerInclude` | | an NSIS script that the installer includes, e.g. `build/installer.nsh`, whose `customInit`, `customInstall` and `customUnInstall` macros run when the installer starts, at the end of the install, and in the uninstall before the app's folder is removed. See [installer hooks](distribution.md#installer-hooks) |

See [Windows](distribution.md#windows).

## linux

| Field | Default | |
|---|---|---|
| `maintainer` | `author` in package.json, else `name` | `Name <email>`, the maintainer of the Debian package and of the PKGBUILD |
| `comment` | | a short description, for the desktop entry and the package |
| `categories` | `["Utility"]` | the categories of the desktop entry, which place it in application menus |
| `depends` | | Debian packages the app needs besides GTK and WebKitGTK; the PKGBUILD depends on the [Arch Linux packages](distribution.md#arch-linux) of those MyGo knows |
| `webview` | `true` | `false` for an app that shows no web page, as one of native UI: the Debian package and the PKGBUILD depend on GTK alone, and install.sh and `mygo doctor` do not ask for WebKitGTK |
| `command` | none | a command that runs the app, or a list of names, as `["my-app", "ma"]`, which adds a short one: `/usr/bin/<command>` from the Debian package and the PKGBUILD, `~/.local/bin/<command>` from install.sh, for each |

See [Linux](distribution.md#linux).

## Computed configuration

The configuration is code, so it can compute fields: read the version from
package.json, or a signing identity from the environment of CI. The default
export may also be a function, which can be async, of the command running
(`"dev"`, `"build"`, `"generate"` or `"init"`):

```ts
import { defineConfig } from "mygo-cli";
import pkg from "./package.json" with { type: "json" };

export default defineConfig(({ command }) => ({
  name: command === "dev" ? "My App (dev)" : "My App",
  version: pkg.version,
  macos: {
    signingIdentity: process.env.MACOS_SIGNING_IDENTITY ?? "-",
  },
}));
```

## How it runs

The CLI evaluates `mygo.config.ts` with [Bun](https://bun.sh), or else
Node.js 22.6 or later, in the project directory, and checks the result.
Node.js runs TypeScript by removing the types, so the file may use type
annotations but not syntax that generates code, such as `enum`. What the
file prints goes to the terminal, apart from the CLI's output. `mygo dev`
reloads the app when the file changes, but not when files it imports do.

## mygo.json

The configuration can be JSON instead, in `mygo.json`, which needs no
JavaScript runtime to read; `mygo init -template native` makes one:

```json
{
  "name": "My App",
  "identifier": "com.example.myapp",
  "devUrl": "http://localhost:5173",
  "macos": { "signingIdentity": "Developer ID Application: Jane Doe (TEAMID)" }
}
```

It has the same fields. A project has either `mygo.config.ts` or
`mygo.json`, not both.
