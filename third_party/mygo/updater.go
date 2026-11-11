//go:build !js

package mygo

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/internal/update"
	"github.com/egoist/mygo/netproxy"
)

// Set by `mygo build` with -ldflags -X when mygo.config.ts configures updates:
// the manifest of this build's target, or with a tag prefix on GitHub its
// feed in the newest tagged release (update.TaggedFeed), and the public key
// updates must be signed with.
var (
	packageUpdateFeed string
	packageUpdateKey  string
)

// startExe is the executable the app started from. On Linux, os.Executable
// reports the old file once an update replaced it.
var startExe, _ = os.Executable()

// UpdaterModule installs new versions of the app, which `mygo build`
// publishes when mygo.config.ts configures updates: signed archives and a
// manifest per platform. Use the Updater singleton:
//
//	update, err := mygo.Updater.Check(ctx)
//	if err == nil && update != nil {
//		if err := update.Install(ctx, nil); err == nil {
//			mygo.App.Relaunch()
//		}
//	}
type UpdaterModule struct{}

// Updater installs new versions of the app.
var Updater = &UpdaterModule{}

// prereleases is set while Check offers prereleases (SetPrereleases).
var prereleases atomic.Bool

// updateSource is the source of SetSource, nil for the feed of
// mygo.config.ts.
var updateSource atomic.Pointer[UpdateSource]

// UpdateSource is where Updater gets updates from instead of the feed that
// mygo.config.ts configures (SetSource): an update service the app already
// has, an endpoint that needs the user's credentials, or a channel the app
// chooses while it runs. It serves what `mygo build` publishes, from
// wherever the app keeps it, and Updater checks it as it checks the feed's:
// the archive and deltas must be signed with the key of updates.publicKey
// that the app was built with, or they are not installed.
type UpdateSource interface {
	// Manifest returns the manifest of the running platform: the JSON that
	// `mygo build` writes as update-<target>.json, such as
	// update-darwin-arm64.json, of at most 1 MiB.
	// prereleases reports whether the app offers them (SetPrereleases).
	Manifest(ctx context.Context, prereleases bool) (io.ReadCloser, error)
	// Open returns the file at url, which the manifest names: the archive
	// of the update or one of its deltas. The URL is the manifest's, so it
	// may be anything the source understands, such as an id of its own.
	Open(ctx context.Context, url string) (io.ReadCloser, error)
}

// SetSource makes Check read the manifest from s, and Install download the
// files it names through s, instead of from the feed of mygo.config.ts; nil
// restores the feed. Updates are still enabled only in builds with updates
// in mygo.config.ts (Enabled), which give the key that updates are checked
// with. An Update that Check returned keeps downloading from the source it
// came from. It may be called from any goroutine.
func (u *UpdaterModule) SetSource(s UpdateSource) {
	if s == nil {
		updateSource.Store(nil)
		return
	}
	updateSource.Store(&s)
}

// ErrUpdatesDisabled is returned by Check for builds without updates:
// development builds, and builds of apps without updates in mygo.config.ts.
var ErrUpdatesDisabled = errors.New("mygo: updates are not enabled for this build")

// Update is a newer version of the app, found by Updater.Check.
type Update struct {
	// Version of the update, and its release notes in Markdown.
	Version string
	Notes   string
	// Date is when the update was built.
	Date time.Time

	manifest update.Manifest
	key      string
	// source is where the manifest came from, which serves its files.
	source UpdateSource
	// current is the version that Check compared with, which delta updates
	// update.
	current string
}

// Enabled reports whether the app can update itself: it was built with
// updates and can write where it is installed. Apps installed by a package
// manager, such as a .deb in /opt, are updated by it instead.
func (u *UpdaterModule) Enabled() bool {
	if packageUpdateFeed == "" || packageUpdateKey == "" || IsDev() {
		return false
	}
	target, err := installTarget()
	if err != nil {
		return false
	}
	probe, err := os.MkdirTemp(filepath.Dir(target), ".mygo-update-")
	if err != nil {
		return false
	}
	os.Remove(probe)
	return true
}

// Check asks the update feed, or the source of SetSource, for a version
// newer than the running one and returns it, or nil when the app is up to
// date.
func (u *UpdaterModule) Check(ctx context.Context) (*Update, error) {
	if !u.Enabled() {
		return nil, ErrUpdatesDisabled
	}
	var src UpdateSource = feedSource(packageUpdateFeed)
	if s := updateSource.Load(); s != nil {
		src = *s
	}
	return checkUpdate(ctx, src, packageUpdateKey, App.Version(), prereleases.Load())
}

// UpdateCheckOptions selects which published releases are eligible.
type UpdateCheckOptions struct{ IncludePrereleases bool }

// ErrNoPublishedUpdates means the selected channel has no published release.
var ErrNoPublishedUpdates = update.ErrNotPublished

// CheckWithOptions uses the built-in key, version, signature and installer.
// It reads the channel of options rather than SetPrereleases and SetSource.
func (u *UpdaterModule) CheckWithOptions(ctx context.Context, options UpdateCheckOptions) (*Update, error) {
	if !u.Enabled() {
		return nil, ErrUpdatesDisabled
	}
	return checkChannel(ctx, packageUpdateFeed, packageUpdateKey, App.Version(), options)
}

func checkChannel(ctx context.Context, feed, key, current string, options UpdateCheckOptions) (*Update, error) {
	up, err := checkUpdate(ctx, channelSource(channelUpdateFeed(feed, options)), key, current, options.IncludePrereleases)
	// A channel server answers 404 while the channel has no release.
	var status *httpStatusError
	if errors.As(err, &status) && status.code == http.StatusNotFound {
		return nil, fmt.Errorf("mygo: checking for updates: %w", ErrNoPublishedUpdates)
	}
	return up, err
}

// channelSource is the feed of an update channel: with prereleases, a
// GitHub feed offers the newest release, draft or not, that has a manifest
// for the running platform (update.ResolveFeedWithPrereleases).
type channelSource string

func (f channelSource) Manifest(ctx context.Context, pre bool) (io.ReadCloser, error) {
	feed, err := update.ResolveFeedWithPrereleases(ctx, string(f), httpGet, pre)
	if err != nil {
		return nil, err
	}
	return httpGet(ctx, feed)
}

func (f channelSource) Open(ctx context.Context, url string) (io.ReadCloser, error) {
	return httpGet(ctx, url)
}

// Update channels of a URL feed (updates.url) are selected with the
// channel query parameter: "stable", or "beta" when prereleases are
// included. The server answers with the manifest of the newest release of
// the channel, or 404 when the channel has none.
const (
	stableChannel = "stable"
	betaChannel   = "beta"
)

func channelUpdateFeed(feed string, options UpdateCheckOptions) string {
	u, err := url.Parse(feed)
	if err != nil {
		return feed
	}
	if u.Host == "github.com" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) == 6 && parts[2] == "releases" && parts[3] == "latest" && parts[4] == "download" {
			feed = update.TaggedFeed(parts[0]+"/"+parts[1], "v", parts[5])
		}
		return feed
	}
	query := u.Query()
	query.Set("channel", stableChannel)
	if options.IncludePrereleases {
		query.Set("channel", betaChannel)
	}
	u.RawQuery = query.Encode()
	return u.String()
}

// SetPrereleases makes Check offer prereleases too, such as 1.3.0-beta.1,
// for the users of an app who chose its beta: with updates on GitHub,
// Check then reads the release of the highest version that is not a
// draft, through the GitHub API (usually one request a check), so that a
// release newer than the prereleases is offered too. Check never offers a
// version older than the running one, so turning it off waits for the
// release after the prerelease the app runs. Updates of updates.url are
// the manifest the server has, whichever is set. It is off at launch; the
// app keeps the user's choice and sets it again. It may be called from
// any goroutine.
func (u *UpdaterModule) SetPrereleases(on bool) { prereleases.Store(on) }

// Prereleases reports whether Check offers prereleases (SetPrereleases).
func (u *UpdaterModule) Prereleases() bool { return prereleases.Load() }

// feedSource is the update feed that mygo.config.ts configures, the URL of
// a manifest or a feed that update.ResolveFeed resolves.
type feedSource string

func (f feedSource) Manifest(ctx context.Context, pre bool) (io.ReadCloser, error) {
	resolve := update.ResolveFeed
	if pre {
		resolve = update.ResolvePrereleaseFeed
	}
	feed, err := resolve(ctx, string(f), httpGet)
	if err != nil {
		return nil, err
	}
	return httpGet(ctx, feed)
}

func (f feedSource) Open(ctx context.Context, url string) (io.ReadCloser, error) {
	return httpGet(ctx, url)
}

func checkUpdate(ctx context.Context, src UpdateSource, key, current string, pre bool) (*Update, error) {
	body, err := src.Manifest(ctx, pre)
	if err != nil {
		return nil, fmt.Errorf("mygo: checking for updates: %w", err)
	}
	defer body.Close()
	var m update.Manifest
	if err := json.UnmarshalRead(io.LimitReader(body, update.MaxManifestSize), &m); err != nil {
		return nil, fmt.Errorf("mygo: checking for updates: %w", err)
	}
	if m.Version == "" || m.URL == "" || m.Signature == "" {
		return nil, errors.New("mygo: checking for updates: incomplete manifest")
	}
	if update.Compare(m.Version, current) <= 0 {
		return nil, nil
	}
	up := &Update{Version: m.Version, Notes: m.Notes, manifest: m, key: key, current: current, source: src}
	up.Date, _ = time.Parse(time.RFC3339, m.Date)
	return up, nil
}

// Install prepares and applies the update. Applications that need an idle or
// restart transaction should call Prepare first and Apply only inside that gate.
// Applying replaces resources on disk; the host must not load old-version
// resources or launch workers until it has restarted.
func (up *Update) Install(ctx context.Context, progress func(downloaded, total int64)) error {
	prepared, err := up.Prepare(ctx, progress)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(err, prepared.Discard())
	}
	if err := prepared.Apply(); err != nil {
		return errors.Join(err, prepared.Discard())
	}
	if err := prepared.Acknowledge(); err != nil {
		return err
	}
	refreshDesktopEntry(prepared.record.Target)
	return nil
}

// refreshDesktopEntry registers the desktop entry of the version installed
// at target on Linux, as install.sh did for the first one.
func refreshDesktopEntry(target string) {
	if runtime.GOOS != "linux" {
		return
	}
	if err := update.RefreshDesktopEntry(target, filepath.Base(startExe)); err != nil {
		log.Printf("mygo: registering the desktop entry of the update: %v", err)
	}
}

// getter returns how the update's files are downloaded: from its source.
func (up *Update) getter() update.Getter {
	if up.source != nil {
		return up.source.Open
	}
	return httpGet
}

// installTarget returns what an update replaces: the app bundle on macOS,
// the directory of the executable elsewhere.
func installTarget() (string, error) {
	exe := startExe
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	if runtime.GOOS == "darwin" {
		bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
		if !strings.HasSuffix(bundle, ".app") {
			return "", errors.New("mygo: only app bundles can update themselves")
		}
		return bundle, nil
	}
	if os.Getenv("APPIMAGE") != "" {
		return "", errors.New("mygo: AppImages cannot update themselves")
	}
	return filepath.Dir(exe), nil
}

// installUpdate installs the version of m in place of target, the app of
// version current, through the staged transaction: from its delta for
// current when there is one, else, or when the delta fails, from its
// archive, both downloaded with get.
func installUpdate(ctx context.Context, get update.Getter, m update.Manifest, key, current, target string, progress func(downloaded, total int64)) error {
	prepared, err := prepareUpdate(ctx, get, m, key, current, target, progress)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(err, prepared.Discard())
	}
	if err := prepared.Apply(); err != nil {
		return errors.Join(err, prepared.Discard())
	}
	return prepared.Acknowledge()
}

// applyDelta downloads the delta d into work and makes the new version of
// the app at target from it, in dir: its bundle on macOS, in dir too, as
// the bundle of an archive is.
func applyDelta(ctx context.Context, get update.Getter, d update.Delta, pub ed25519.PublicKey, version, target, work, dir string, progress func(downloaded, total int64)) error {
	if d.Size <= 0 || d.Size > update.MaxArchiveSize {
		return fmt.Errorf("invalid size %d", d.Size)
	}
	path := filepath.Join(work, "update.delta")
	if err := download(ctx, get, d.URL, d.Size, d.Signature, pub, path, progress); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if runtime.GOOS == "darwin" {
		if err := os.Mkdir(dir, 0o755); err != nil {
			return err
		}
		dir = filepath.Join(dir, filepath.Base(target))
	}
	return update.ApplyDelta(f, d.Size, d.From, version, target, dir)
}

// download fetches url with get into the file path and checks its size and
// signature.
func download(ctx context.Context, get update.Getter, url string, size int64, signature string, pub ed25519.PublicKey, path string, progress func(downloaded, total int64)) error {
	body, err := get(ctx, url)
	if err != nil {
		return err
	}
	defer body.Close()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	sum := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, sum), &progressReader{r: io.LimitReader(body, size+1), total: size, fn: progress})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	if n != size {
		return fmt.Errorf("downloaded %d bytes, want %d", n, size)
	}
	return update.Verify(pub, sum.Sum(nil), signature)
}

// relaunchNow starts the app again; it runs after the quit sequence.
func relaunchNow() {
	if launchedByDev() {
		os.Exit(devRelaunchCode)
	}
	if runtime.GOOS == "darwin" && openBundle() {
		return
	}
	// The executable the app started from, which an update may have
	// replaced.
	cmd := exec.Command(startExe, os.Args[1:]...)
	cmd.Dir = startDir
	if runtime.GOOS != "windows" {
		// GUI executables on Windows have no standard streams to pass on.
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	}
	if cmd.Start() == nil {
		_ = cmd.Process.Release()
	}
}

// openBundle starts the app bundle again through Launch Services, as the
// Dock does, reporting whether it did. A child of this process would stay
// in this instance's coalition, which has no app once this instance exits:
// when an app that the new instance started quits, as one a terminal's
// shell ran, loginwindow takes the coalition's other processes for that
// app's and ends them with SIGTERM.
func openBundle() bool {
	bundle, err := installTarget()
	if err != nil {
		return false
	}
	// -n: this instance still runs, and would only be brought forward.
	args := []string{"-n", bundle}
	if len(os.Args) > 1 {
		args = append(append(args, "--args"), os.Args[1:]...)
	}
	return exec.Command("/usr/bin/open", args...).Run() == nil
}

// cleanUpdateLeftovers removes what an update could not remove while the
// previous version ran (Windows keeps running executables).
func cleanUpdateLeftovers() {
	if startExe != "" {
		cleanPreparedUpdateLeftovers()
	}
	if runtime.GOOS == "darwin" || startExe == "" {
		return
	}
	olds, _ := filepath.Glob(filepath.Join(filepath.Dir(startExe), ".*.old-*"))
	for _, p := range olds {
		_ = os.RemoveAll(p)
	}
}

// httpGet fetches a URL over HTTPS (or plain HTTP from the loopback
// interface, for tests), through the system's proxy (package netproxy).
func httpGet(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopback(u.Hostname())) {
		return nil, fmt.Errorf("refusing to download %s: updates need HTTPS", rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	// Not the app's name: GitHub's release assets come from a store that
	// answers 400 to a User-Agent with bytes outside ASCII, as 云音乐, for
	// every file its CDN has not cached yet.
	req.Header.Set("User-Agent", "mygo-updater")
	resp, err := (&http.Client{Transport: netproxy.Transport()}).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, &httpStatusError{url: rawURL, code: resp.StatusCode, status: resp.Status}
	}
	return resp.Body, nil
}

// httpStatusError is a response other than 200 OK.
type httpStatusError struct {
	url, status string
	code        int
}

func (e *httpStatusError) Error() string { return fmt.Sprintf("GET %s: %s", e.url, e.status) }

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// progressReader reports how much of a download was read.
type progressReader struct {
	r     io.Reader
	n     int64
	total int64
	fn    func(downloaded, total int64)
	last  time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if p.fn != nil && (err != nil || p.n == p.total || time.Since(p.last) > 100*time.Millisecond) {
		p.last = time.Now()
		p.fn(p.n, p.total)
	}
	return n, err
}
