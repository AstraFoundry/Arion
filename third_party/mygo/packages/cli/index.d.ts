/** The platforms with a prebuilt binary, as `${process.platform}-${process.arch}`. */
export declare const platforms: readonly string[];

/**
 * Returns the name of the package holding the mygo binary of a platform,
 * by default this one. Throws for platforms without a prebuilt binary.
 */
export declare function platformPackage(platform?: string, arch?: string): string;

/**
 * Returns the path of the mygo binary to run: $MYGO_CLI_BINARY when set,
 * else the binary of this platform's package. Throws when that package is
 * not installed.
 */
export declare function binaryPath(): string;

/**
 * Types the configuration of mygo.config.ts, which takes the place of
 * mygo.json:
 *
 * ```ts
 * import { defineConfig } from "mygo-cli";
 *
 * export default defineConfig({
 *   name: "My App",
 *   identifier: "com.example.myapp",
 *   devUrl: "http://localhost:5173",
 * });
 * ```
 *
 * A function gets the command running and returns the configuration, or a
 * promise of it.
 */
export declare function defineConfig(config: Config): Config;
export declare function defineConfig(config: Promise<Config>): Promise<Config>;
export declare function defineConfig(config: ConfigFn): ConfigFn;

/** A configuration computed for the command running. */
export type ConfigFn = (env: ConfigEnv) => Config | Promise<Config>;

/** What a configuration function gets. */
export interface ConfigEnv {
  /** The mygo command running. */
  command: "dev" | "build" | "generate" | "init";
}

/** The configuration of a MyGo project. Every field is optional. */
export interface Config {
  /** The name users see: the app, its executable, menus and installers (default: the directory's name). */
  name?: string;
  /**
   * The name of the app's files where a name must be plain ASCII: the disk
   * images of macOS and the installers of Windows, the Linux executable,
   * desktop entry, icon and Debian package, the update archives of every
   * platform, and the variables of the install scripts. Lower-case
   * letters and digits with single hyphens between them (default: the name
   * in lower case, with hyphens between its words and Chinese, Japanese and
   * Korean in Latin letters, as `my-app` of "My App" and `yun-yin-le` of
   * "云音乐").
   */
  slug?: string;
  /**
   * A reverse DNS name unique to the app, such as `com.example.myapp`, on
   * which systems key preferences, permissions and registrations (default:
   * `com.mygo.` and the name's letters and digits).
   */
  identifier?: string;
  /** The version of the app, which updates compare (default: `0.1.0`). */
  version?: string;
  /** The copyright notice of the app. */
  copyright?: string;
  /**
   * Who makes the app, which Windows lists with the installed app (default:
   * the copyright notice without its "©" and years).
   */
  publisher?: string;
  /**
   * The SPDX license expression of the app, such as `MIT`, which the PKGBUILD
   * of Arch Linux records (default: `license` in `package.json`).
   */
  license?: string;
  /** A square PNG, ideally 1024×1024 (default: `resources/icon.png` when it exists). */
  icon?: string;
  /** The Go package of the app, relative to the project (default: `.`). */
  main?: string;
  /** Where `mygo build` writes the builds, one directory per platform (default: `dist`). */
  out?: string;
  /**
   * Where `mygo generate` writes the TypeScript client (default:
   * `src/mygo.ts` when package.json at the project root depends on
   * mygo-runtime; an app without a frontend gets no client).
   */
  bindings?: string;

  /**
   * The dev server `mygo dev` points the app at, such as
   * `http://localhost:5173`: windows load it for URLs without a scheme.
   */
  devUrl?: string;
  /** Runs in the project while `mygo dev` runs, e.g. the dev server; the app starts once `devUrl` answers. */
  devCommand?: string;
  /** Builds the frontend before `mygo build` compiles the app. */
  buildCommand?: string;
  /**
   * The directory of the built frontend, which `mygo build` embeds into the
   * app; `mygo dev` serves it from disk without `devUrl`.
   */
  frontendDist?: string;

  /**
   * Files and directories copied into the app of every platform under their
   * base names, next to the contents of the project's resources directory,
   * whose directories named after a platform, such as
   * `resources/linux-amd64`, hold the files of one platform.
   */
  resources?: string[];
  /**
   * Files of the project's resources directory left out of the apps:
   * patterns of their paths there, with slashes, where `*` matches within a
   * name and `**` any number of directories, as in `design/**`. A directory
   * that matches is left out with what it holds. Go source files are always
   * left out; listed `resources` ship whole.
   */
  resourcesExclude?: string[];
  /** The URL schemes the app opens, such as `myapp` for `myapp://…`, which reach `App.OnOpenURL`. */
  urlSchemes?: string[];
  /** The types of files the app opens, which reach `App.OnOpenFile`. */
  fileAssociations?: FileAssociation[];
  /**
   * The catalogs of the app's messages, which make the languages it ships.
   * Without it, the app ships English only.
   */
  i18n?: I18nConfig;
  /** Signed updates, which `mygo.Updater` installs. */
  updates?: UpdatesConfig;
  /** macOS packaging. */
  macos?: MacOSConfig;
  /** Windows packaging. */
  windows?: WindowsConfig;
  /** Linux packaging. */
  linux?: LinuxConfig;
}

/**
 * The catalogs of the app's messages: a JSON file per language in `dir`,
 * named after its BCP 47 tag (`de.json`, `zh-Hant.json`), whose messages
 * `mygo generate` writes typed Go and TypeScript functions for.
 */
export interface I18nConfig {
  /** The directory of the catalogs, a Go package of its own (default `locales`). */
  dir?: string;
  /** The language the app is written in, which the others fall back to (default `en`). */
  source?: string;
}

/** A type of file the app opens. */
export interface FileAssociation {
  /** The file name extensions, without dots, e.g. `["md"]`. */
  ext: string[];
  /** What the files are, e.g. `"Markdown Document"`. */
  name?: string;
  /** The MIME type of the files, by which Linux identifies them; the app defines an unregistered one (`x-`) (default: `application/x-<app>-<ext>`). */
  mimeType?: string;
  /** The app's role for the files on macOS (default: `"Editor"`). */
  role?: "Editor" | "Viewer";
}

/** Signed updates: `publicKey` and one of `github` and `url` are required. */
export interface UpdatesConfig {
  /** The contents of mygo-update.pub, from `mygo keygen`. */
  publicKey: string;
  /** A public GitHub repository, `owner/name`, whose releases hold the updates. */
  github?: string;
  /**
   * What precedes the version in release tags (default: `v`). Another prefix,
   * such as `desktop-v`, lets the repository hold other releases: apps find
   * the newest release tagged with it through the GitHub API instead of
   * reading the latest release.
   */
  tagPrefix?: string;
  /** Instead of `github`, the HTTPS URL of a directory holding the updates. */
  url?: string;
  /** The bucket that serves `url`, which `mygo build -upload` uploads to. */
  s3?: S3Config;
  /** The path of mygo-update.key; `MYGO_UPDATER_PRIVATE_KEY`, holding the key, takes precedence. */
  privateKey?: string;
  /** A Markdown file whose `## <version>` section becomes the release notes (default: `CHANGELOG.md` when it exists). */
  changelog?: string;
  /** How many earlier versions get a delta update, a smaller download of what changed (default: `3`; `0` for none). */
  deltas?: number;
}

/**
 * A bucket of Amazon S3, or of a compatible service such as Cloudflare R2.
 * The credentials come from `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and
 * `AWS_SESSION_TOKEN`.
 */
export interface S3Config {
  /** The name of the bucket. */
  bucket: string;
  /** The directory of the files in the bucket, e.g. `my-app` (default: its root). */
  prefix?: string;
  /** The region of the bucket (default: `AWS_REGION`, else `AWS_DEFAULT_REGION`, else `us-east-1`). */
  region?: string;
  /** The endpoint of a compatible service, e.g. `https://<account>.r2.cloudflarestorage.com` (default: Amazon S3's). */
  endpoint?: string;
  /** Names the bucket in the path of URLs rather than in the host name (default: with `endpoint`, or a bucket name with dots). */
  pathStyle?: boolean;
}

/** macOS packaging. */
export interface MacOSConfig {
  /**
   * The icon as an asset catalog: an `Assets.car` compiled with actool (its
   * app icon named `AppIcon`, unless `infoPlist` sets `CFBundleIconName`), or
   * an Icon Composer `.icon`, which builds compile on a Mac with Xcode 26 or
   * later and skip elsewhere. The `.icns` made from `icon` remains the
   * fallback. Or an `.icns` in place of the one made from `icon`, for an
   * icon drawn per size.
   */
  icon?: string;
  /** The oldest macOS the app runs on (default: `12.0`). */
  minimumSystemVersion?: string;
  /** The identity that signs the app and its disk image, e.g. `Developer ID Application: Jane Doe (TEAMID)` (default: `-`, ad hoc). */
  signingIdentity?: string;
  /** A property list of entitlements to sign the app with. */
  entitlements?: string;
  /**
   * A provisioning profile embedded before signing, for restricted entitlements.
   * Must match the app's identifier, entitlements and signing identity.
   * Cannot be used with ad-hoc signing. Paths are relative to the project.
   */
  provisioningProfile?: string;
  /**
   * Property lists of entitlements for code among the resources, by its path
   * there, e.g. `{ "bin/server": "server.entitlements.plist" }` for a helper
   * that needs `com.apple.security.cs.allow-jit` under the hardened runtime.
   * Other code keeps the entitlements it is signed with.
   */
  helperEntitlements?: Record<string, string>;
  /** Keys added to the app's Info.plist, replacing MyGo's, e.g. `NSCameraUsageDescription`. */
  infoPlist?: Record<string, unknown>;
  /** The volume name of the disk image (default: the name). */
  dmgTitle?: string;
  /** Notarizes production disk images. */
  notarize?: NotarizeConfig;
}

/** Credentials of `xcrun notarytool store-credentials <profile>`. */
export interface NotarizeConfig {
  /** The profile holding the credentials. */
  keychainProfile: string;
  /** The keychain holding the profile (default: the login keychain). */
  keychain?: string;
}

/** Windows packaging. */
export interface WindowsConfig {
  /** Register native WinRT toast activation. Call mygo.HandleNotificationRegistration first in main. */
  notifications?: boolean;
  /**
   * The name Windows shows the app by: its shortcuts, its entry in
   * Settings > Apps, the installer's title and the executable's product
   * name (default: the name). The executable and the install folder keep
   * the name's, e.g. `天气助手` for an app named `weather`.
   */
  displayName?: string;
  /**
   * The name of the folder the installer installs the app in, in
   * `%LOCALAPPDATA%\Programs` (default: the name), e.g. an ASCII `ZhiLian`
   * for an app named `智链`, whose shortcuts and entry in Settings > Apps
   * keep the name. Changing it moves new installs only.
   */
  installDirName?: string;
  /** A code signing certificate (.pfx); its password comes from `MYGO_WINDOWS_CERTIFICATE_PASSWORD`. */
  certificate?: string;
  /** Instead of `certificate`, a command that signs a file, with `%1` for its path. */
  signCommand?: string;
  /** The time stamping server of `certificate` (default: `http://timestamp.digicert.com`). */
  timestampUrl?: string;
  /**
   * An NSIS script that the installer includes, e.g. `build/installer.nsh`:
   * the macros it defines among `customInit`, `customInstall` and
   * `customUnInstall` run when the installer starts, at the end of the
   * install, and in the uninstall before the app's folder is removed.
   */
  installerInclude?: string;
}

/** Linux packaging. */
export interface LinuxConfig {
  /** `Name <email>`, the maintainer of the Debian package and of the PKGBUILD (default: `author` in package.json, else the app's name). */
  maintainer?: string;
  /** A short description, for the desktop entry and the package. */
  comment?: string;
  /** The categories of the desktop entry (default: `["Utility"]`). */
  categories?: string[];
  /** Debian packages the app needs besides GTK and WebKitGTK; the PKGBUILD depends on the Arch Linux packages of those MyGo knows. */
  depends?: string[];
  /**
   * `false` for an app that shows no web page, in a window or a `ui.WebView`,
   * as an app of native UI: its Debian package and PKGBUILD depend on GTK alone, and
   * install.sh and `mygo doctor` do not ask for WebKitGTK (default: `true`).
   */
  webview?: boolean;
  /**
   * A command that runs the app, such as `my-app`, or a list of names, such
   * as `["my-app", "ma"]`, which adds a short one: a link in `/usr/bin` from
   * the Debian package and the PKGBUILD, and in `~/.local/bin` from install.sh, for each,
   * which says to run the app by the first it links (default: none; the app
   * opens from the applications menu).
   */
  command?: string | string[];
}
