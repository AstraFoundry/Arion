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
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/egoist/mygo/internal/update"
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

// Check asks the update feed for a version newer than the running one and
// returns it, or nil when the app is up to date.
func (u *UpdaterModule) Check(ctx context.Context) (*Update, error) {
	if !u.Enabled() {
		return nil, ErrUpdatesDisabled
	}
	return checkUpdate(ctx, packageUpdateFeed, packageUpdateKey, App.Version())
}

// UpdateCheckOptions selects which published releases are eligible.
type UpdateCheckOptions struct{ IncludePrereleases bool }

// ErrNoPublishedUpdates means the selected channel has no published release.
var ErrNoPublishedUpdates = update.ErrNotPublished

// CheckWithOptions uses the built-in key, version, signature and installer.
func (u *UpdaterModule) CheckWithOptions(ctx context.Context, options UpdateCheckOptions) (*Update, error) {
	if !u.Enabled() {
		return nil, ErrUpdatesDisabled
	}
	return checkChannel(ctx, packageUpdateFeed, packageUpdateKey, App.Version(), options)
}

func checkChannel(ctx context.Context, feed, key, current string, options UpdateCheckOptions) (*Update, error) {
	up, err := checkUpdateWithOptions(ctx, channelUpdateFeed(feed, options), key, current, options)
	// A channel server answers 404 while the channel has no release.
	var status *httpStatusError
	if errors.As(err, &status) && status.code == http.StatusNotFound {
		return nil, fmt.Errorf("mygo: checking for updates: %w", ErrNoPublishedUpdates)
	}
	return up, err
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

func checkUpdate(ctx context.Context, feed, key, current string) (*Update, error) {
	return checkUpdateWithOptions(ctx, feed, key, current, UpdateCheckOptions{})
}

func checkUpdateWithOptions(ctx context.Context, feed, key, current string, options UpdateCheckOptions) (*Update, error) {
	feed, err := update.ResolveFeedWithPrereleases(ctx, feed, httpGet, options.IncludePrereleases)
	if err != nil {
		return nil, fmt.Errorf("mygo: checking for updates: %w", err)
	}
	body, err := httpGet(ctx, feed)
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
	up := &Update{Version: m.Version, Notes: m.Notes, manifest: m, key: key, current: current}
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
	return prepared.Acknowledge()
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
// version current: from its delta for current when there is one, else, or
// when the delta fails, from its archive.
func installUpdate(ctx context.Context, m update.Manifest, key, current, target string, progress func(downloaded, total int64)) error {
	prepared, err := prepareUpdate(ctx, m, key, current, target, progress)
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
func applyDelta(ctx context.Context, d update.Delta, pub ed25519.PublicKey, version, target, work, dir string, progress func(downloaded, total int64)) error {
	if d.Size <= 0 || d.Size > update.MaxArchiveSize {
		return fmt.Errorf("invalid size %d", d.Size)
	}
	path := filepath.Join(work, "update.delta")
	if err := download(ctx, d.URL, d.Size, d.Signature, pub, path, progress); err != nil {
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

// download fetches url into the file path and checks its size and
// signature.
func download(ctx context.Context, url string, size int64, signature string, pub ed25519.PublicKey, path string, progress func(downloaded, total int64)) error {
	body, err := httpGet(ctx, url)
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

// rename renames a file of the app, retrying for a few seconds while
// Windows reports it in use: antivirus software opens new executables, such
// as the app that just started and the one of the update, to scan them.
func rename(from, to string) error {
	for i := 1; ; i++ {
		err := os.Rename(from, to)
		if err == nil || i == 10 || runtime.GOOS != "windows" ||
			// ERROR_SHARING_VIOLATION, or ERROR_ACCESS_DENIED for a directory
			// holding an open file.
			!errors.Is(err, syscall.Errno(32)) && !errors.Is(err, syscall.Errno(5)) {
			return err
		}
		time.Sleep(time.Duration(i) * 100 * time.Millisecond)
	}
}

// relaunchNow starts the app again; it runs after the quit sequence.
func relaunchNow() {
	if launchedByDev() {
		os.Exit(devRelaunchCode)
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
// interface, for tests).
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
	req.Header.Set("User-Agent", "MyGo/"+Version+" "+App.Name()+"/"+App.Version())
	resp, err := http.DefaultClient.Do(req)
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
