# Auto-updates

Apps built with MyGo update themselves from signed releases, published on
GitHub or on any HTTPS server. `mygo build` signs each platform's build
with your key, and `mygo.Updater` in the app downloads a newer version,
checks the signature against the public key built into the app, and
replaces the app with it.

## Set up

Create the key pair that signs updates:

```sh
mygo keygen
```

It writes `mygo-update.key`, the secret key, and `mygo-update.pub`, the
public key, to `mygo/update-keys` in your configuration directory, and
prints where (`-o` chooses another directory). Keep the secret key out of the repository, in a password
manager and in the secrets of your CI (see
[GitHub Actions](github-actions.md)): installed apps accept only updates
signed with it, so losing it strands them, and anyone who has it can ship
code to your users.

Add the public key to the configuration, with where releases are
published, here in mygo.config.ts (in [mygo.json](configuration.md#mygojson),
the same fields in JSON):

```ts
export default defineConfig({
  version: "1.2.0",
  updates: {
    publicKey: "…the contents of mygo-update.pub…",
    github: "you/my-app",
  },
});
```

`github` is a public repository whose releases, tagged `v1.2.0` and so on,
hold the updates. Apps read the manifests of its latest release.

A repository may hold other releases too, such as a command-line tool's
that stays its latest release. Give the app's releases a `tagPrefix` of
their own, such as `desktop-v` for tags like `desktop-v1.2.0`, and apps,
`install.sh` and delta updates find the newest release with that prefix,
neither a draft nor a prerelease, through the GitHub API instead: one
request a check, within the API's 60 requests an hour for each address.
`mygo build` uses `GH_TOKEN` or `GITHUB_TOKEN` for it when there is one.

For your own server or an
[S3 bucket](#publishing-to-s3), set `url` instead, an HTTPS URL of a
directory:

```ts
export default defineConfig({
  updates: {
    publicKey: "…",
    url: "https://downloads.example.com/my-app",
  },
});
```

## Build and publish

Give `mygo build` the secret key, in the `MYGO_UPDATER_PRIVATE_KEY`
environment variable or as a file with `updates.privateKey` in the
configuration (a path, which may start with `~/`):

```sh
MYGO_UPDATER_PRIVATE_KEY="$(cat path/to/mygo-update.key)" bun run build        # a web frontend
MYGO_UPDATER_PRIVATE_KEY="$(cat path/to/mygo-update.key)" go tool mygo build   # native UI
```

Next to the installers of each platform it writes:

- `my-app-1.2.0-darwin-arm64.tar.gz`, the app as installed, named after
  the app's [`slug`](configuration.md#the-app),
- [delta updates](#delta-updates) from the versions before, such as
  `my-app-1.1.0-to-1.2.0-darwin-arm64.delta`, and
- `update-darwin-arm64.json`, the manifest: the version, its release
  notes, the date, and the URL, size and signature of the archive and of
  each delta.

Without a key, `mygo build` builds the apps and skips the update files,
but for the Linux archives, which [`install.sh`](distribution.md#the-install-script)
installs. `mygo build -upload` needs the key: it stops before building,
as a release without manifests is one that installed apps cannot update
to.

The release notes are the section of the version in `CHANGELOG.md`
(`updates.changelog` names another file), under a `## 1.2.0` heading, or
`## [1.2.0] - 2026-09-19` as in [Keep a Changelog](https://keepachangelog.com).
When the file exists, it must have that section.

Publish the files where `updates` points to:

- **GitHub**: `mygo build -upload` uploads them, with the installers, to a
  draft release of the version. Publishing the release makes it the latest,
  which apps check: they read the manifests from
  `https://github.com/you/my-app/releases/latest/download/`. With a
  `tagPrefix` of their own, the draft is not to become the latest, which
  stays the other releases': publish it with
  `gh release edit desktop-v1.2.0 --draft=false --latest=false`, or untick
  "Set as the latest release", and apps find it by its tag.
- **S3**: `mygo build -upload` uploads them, with the installers, to the
  bucket of `updates.s3`; see [below](#publishing-to-s3).
- **Your server**: upload the archives, deltas and manifests to the `url`
  directory, with `install.sh`, `install.ps1` and `SHA256SUMS` (with the
  lines of every platform), and keep the files of earlier versions there. Upload the manifests last, so apps never see a manifest
  whose files are missing.

Each platform, such as `darwin-arm64`, `darwin-universal` or
`windows-amd64`, has its own manifest, and a build only looks at its own.

### Prereleases

A release that GitHub marks a prerelease, such as `1.3.0-beta.1`, is never
the latest release, so installed apps skip it. Apps with a beta let the
users who chose it see prereleases with
`mygo.Updater.SetPrereleases(true)`: their checks then read the release
of the highest version that is not a draft, through the GitHub API, so
they get the next prerelease, and the release after it too. Versions
compare as semantic versioning does, `1.3.0-beta.2` before
`1.3.0-beta.10` before `1.3.0`. Turning it off installs no older version:
the app updates again once a release is newer than the prerelease it
runs. The choice lasts while the app runs; the app keeps it, such as in
its settings, and calls `SetPrereleases` at launch.

```go
mygo.Updater.SetPrereleases(settings.Beta)
```

Publish a prerelease as one, so that it does not become the latest
release, which every app checks:
`gh release edit v1.3.0-beta.1 --draft=false --prerelease`, or tick
"Set as a pre-release". The checks read the releases from the newest down
to the newest one that is not a prerelease, usually one request within
the API's 60 an hour for each address, with or without a `tagPrefix`.
Deltas start from the published release, so an app on a prerelease
downloads the next one whole. Apps updated from `url` read the manifest
the server has either way.

### Publishing to S3

When a bucket of Amazon S3, or of a compatible service such as Cloudflare
R2, serves `url`, directly or through a CDN, name it in `updates.s3`:

```ts
export default defineConfig({
  updates: {
    publicKey: "…",
    url: "https://downloads.example.com/my-app",
    s3: {
      bucket: "downloads",
      prefix: "my-app", // the directory in the bucket that url serves
      endpoint: "https://<account>.r2.cloudflarestorage.com", // not for Amazon S3
    },
  },
});
```

`mygo build -upload` then uploads the installers, archives, deltas and
install script to it, and the manifests last, which publishes the update
of each platform. It needs no other tool: the credentials come from
`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and, for temporary ones,
`AWS_SESSION_TOKEN`, and the region from `s3.region`, else `AWS_REGION`
(or `AWS_DEFAULT_REGION`), else `us-east-1`, which R2 accepts. The
manifests, the install scripts and `SHA256SUMS`, whose names stay from version to version, are
uploaded with `Cache-Control: no-cache`, so a CDN checks them again.

The bucket is named in the host name of requests on Amazon S3 and in their
path with `endpoint`; set `pathStyle` to `false` for services that only
take it in the host name, or to `true` to force the path.

## Delta updates

As with Sparkle, apps download only what changed since their version,
usually a small part of the whole app: a new version of a Go executable
mostly moves code around, which a binary patch describes in little space.

When `mygo build` signs an update, it reads the manifest of each platform
that is published, the one apps check, and downloads the archives of the
last 3 versions: the published one and the earlier ones its manifest lists
(`updates.deltas` changes how many, `0` makes none). It checks that they are
signed with your key and writes a delta from each to the new version:
files that did not change are copied from the installed app, changed ones
are patched ([bsdiff](https://www.daemonology.net/bsdiff/)), and the rest
are included. The notarization ticket stapled to a macOS app
(`Contents/CodeResources`) is always included whole: the archives' apps
have one, but the app copied from the disk image does not. The manifest
lists the deltas, and the archives of the versions before for the next
build. A delta that is not smaller than the
archive is left out. Builds need to download the published updates then;
when they cannot, such as for the first version, they make no deltas and
say why.

If the latest GitHub release has no manifest for the platform being built,
the build looks through earlier releases for that platform's manifest,
with the configured tag prefix, skipping drafts and prereleases. This
keeps delta updates and their history when platforms are published
separately. Installed apps still check the configured update feed.

An app whose version has a delta downloads it, checks its signature and
makes the new version from itself next to it, checking that every file has
the size and SHA-256 that the delta gives, so that the result is exactly
the version published, code signature included. When anything fails, for
example because the app was changed since it was installed, it downloads
the archive instead.

## The update window

The [updater plugin](plugins/updater.md) gives an app the update window
that Mac users know from Sparkle, on every platform: it checks for updates
in the background, shows the release notes of a new version and offers to
install it, skip it or remind the user later.

```go
import "github.com/egoist/mygo/plugins/updater"

mygo.Use(updater.Plugin)
```

Its page describes the "Check for Updates…" menu item, its options, the
preferences it keeps, its languages, and `native.Plugin`, the same window
in native UI for apps that show no web page.

## Your own update UI

`mygo.Updater` is what the plugin is built on. Use it for an interface of
your own, such as a banner in the app's page:

```go
// UpdateProgress tells pages how far the download got.
var UpdateProgress = mygo.NewEvent[float64]("update-progress")

func checkForUpdates(ctx context.Context) {
	if !mygo.Updater.Enabled() {
		return
	}
	up, err := mygo.Updater.Check(ctx)
	if err != nil || up == nil {
		return // offline, or up to date
	}
	res, _ := mygo.Dialog.Message(mygo.MessageOptions{
		Message: "Version " + up.Version + " is available.",
		Detail:  up.Notes, // Markdown
		Buttons: []string{"Install and Restart", "Later"},
	})
	if res.Button != 0 {
		return
	}
	err = up.Install(ctx, func(downloaded, total int64) {
		UpdateProgress.Broadcast(float64(downloaded) / float64(total))
	})
	if err == nil {
		mygo.App.Relaunch()
	}
}
```

- `Check` returns the newer version, with its `Version`, `Notes` and
  `Date`, or nil when the app is up to date.
- `Install` downloads the archive, or the delta of the running version,
  checks its signature, and replaces the app. The progress callback gets
  the bytes of the delta, and when the delta fails, starts over with those
  of the archive. The running app is not affected: the new version runs
  after `App.Relaunch`, or at the next launch. On macOS the new bundle
  replaces the app's once the app has exited, so that the Dock keeps the
  app's tile. On Linux and Windows it replaces the files of the app's
  folder, and removes those that the running version installed and the
  new one no longer has, which `.mygo-files` in the folder lists. Files
  that no version installed stay, such as the data a portable copy keeps
  next to it, and so does the executable the app runs from. An app built
  before MyGo wrote the list has none, so its first update removes
  nothing.
- `Enabled` reports whether the app can update itself: it was built with
  updates, and it can write where it is installed. Development builds
  cannot, and `Check` returns `mygo.ErrUpdatesDisabled` in them.
- `Check` and `Install` go through the proxy of the system's settings,
  or of `HTTPS_PROXY` when it is set (see [proxies](native.md#proxies)).

## Your own update service

`mygo.Updater` reads the feed of `updates` in mygo.config.ts. An app that
has its own update service, an endpoint that needs the user's
credentials, or a channel the user picks while it runs, gives the updater
a source instead with `mygo.Updater.SetSource`. The source returns the
manifest of the platform and the files the manifest names; the updater
checks them as it checks the feed's, so an archive or delta that is not
signed with the key of `updates.publicKey` is refused and the app stays as
it is. A source serves what `mygo build` writes, so the app is still
built with `updates`: the public key, and `github` or `url`, where the
files are published.

For example, with the files on a server that serves them to signed-in
users, set `updates.url` to its directory, so that the manifests name
its files, and add the user's token to every request:

```go
// accountUpdates downloads updates with the user's token.
type accountUpdates struct{ token string }

func (s accountUpdates) Manifest(ctx context.Context, prereleases bool) (io.ReadCloser, error) {
	// The manifest of the platform the app was built for; a universal
	// macOS build reads update-darwin-universal.json.
	return s.Open(ctx, "https://downloads.example.com/my-app/update-"+runtime.GOOS+"-"+runtime.GOARCH+".json")
}

func (s accountUpdates) Open(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return resp.Body, nil
}

// Once the user has signed in:
mygo.Updater.SetSource(accountUpdates{token: session.Token})
```

`Check` then reads the manifest from the source, and `Install` downloads
the archive or delta through its `Open`; `prereleases` is what
`SetPrereleases` set, for a source with a beta channel. The URLs are the
manifest's, so a service that writes its own manifests from those of
`mygo build` may name the files by ids of its own, as long as the sizes
and signatures stay. `SetSource(nil)` goes back to the feed.

## Where apps can update

An app replaces itself, so it must be able to write where it is installed:

- macOS: the app bundle, e.g. in `/Applications` for an administrator, or
  in the user's `~/Applications`.
- Windows: the per-user install of the [installer](distribution.md#the-installer),
  in `%LOCALAPPDATA%\Programs`.
- Linux: a directory of the user, such as `~/.local/my-app.app`, where
  [`install.sh`](distribution.md#the-install-script) installs the app, or
  the extracted build.

Apps installed by a package manager, such as the Debian package in
`/opt`, are updated by it instead: `Enabled` is false for them.
