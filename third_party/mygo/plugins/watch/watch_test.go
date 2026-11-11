package watch

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

type fakeWatcher struct {
	root    string
	once    sync.Once
	closed  chan struct{}
	batches [][]mygo.FileEvent
	err     error
}

func (w *fakeWatcher) Next(context.Context) ([]mygo.FileEvent, error) {
	if len(w.batches) > 0 {
		b := w.batches[0]
		w.batches = w.batches[1:]
		return b, nil
	}
	if w.err != nil {
		return nil, w.err
	}
	return nil, io.EOF
}
func (w *fakeWatcher) Close() error { w.once.Do(func() { close(w.closed) }); return nil }
func (w *fakeWatcher) Root() string { return w.root }
func (w *fakeWatcher) isClosed() bool {
	select {
	case <-w.closed:
		return true
	default:
		return false
	}
}

// pluginFixture returns the plugin's service with the root "project", and
// a page.
func pluginFixture(t *testing.T, opts Options) (*service, string, any) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if opts.Roots == nil {
		opts.Roots = map[string]Root{"project": {Path: dir}}
	}
	p := New(opts)
	if err = p.Setup(); err != nil {
		t.Fatal(err)
	}
	return p.Service.(*service), dir, new(int)
}

// collect runs req for page and returns the parts sent, and as the last
// one the outcome, as an "error" part.
func collect(s *service, ctx context.Context, page any, req request) []result {
	var parts []result
	o := s.run(ctx, page, req, func(p part) error { parts = append(parts, result{Type: p.Type, Events: p.Events}); return nil }, func() {})
	if o.Code != "" {
		parts = append(parts, result{Type: "error", Code: o.Code, Message: o.Message})
	}
	return parts
}

type result struct {
	Type          string
	Events        []mygo.FileEvent
	Code, Message string
}

func TestStreamReadyAndCleanup(t *testing.T) {
	s, dir, page := pluginFixture(t, Options{})
	w := &fakeWatcher{root: dir, closed: make(chan struct{}), batches: [][]mygo.FileEvent{{{Op: mygo.FileCreate, Path: "a"}, {Op: mygo.FileWrite, Path: "b"}}}}
	s.factory = func(_ context.Context, path string, o mygo.FileWatchOptions) (watcher, error) {
		if path != dir || o.Debounce != -time.Millisecond {
			t.Fatalf("watch %s %+v", path, o)
		}
		return w, nil
	}
	parts := collect(s, context.Background(), page, request{Root: "project", DebounceMS: -1})
	if len(parts) != 2 || parts[0].Type != "ready" || len(parts[1].Events) != 2 {
		t.Fatalf("parts %+v", parts)
	}
	if !w.isClosed() {
		t.Fatal("not closed")
	}
	if len(s.active) != 0 {
		t.Fatal("quota leaked")
	}
}

func TestStreamSanitizesFailures(t *testing.T) {
	s, _, page := pluginFixture(t, Options{})
	s.factory = func(context.Context, string, mygo.FileWatchOptions) (watcher, error) {
		return nil, &os.PathError{Op: "watch", Path: "/secret/private", Err: errors.New("secret")}
	}
	parts := collect(s, context.Background(), page, request{Root: "project"})
	if len(parts) != 1 || parts[0].Type != "error" || parts[0].Code != "io" || strings.Contains(parts[0].Message, "secret") {
		t.Fatalf("parts %+v", parts)
	}
	s.factory = func(context.Context, string, mygo.FileWatchOptions) (watcher, error) {
		return nil, mygo.ErrWatchLimit
	}
	if parts := collect(s, context.Background(), page, request{Root: "project"}); parts[0].Code != "limit" {
		t.Fatalf("parts %+v", parts)
	}
}

func TestPermissionsAndQuota(t *testing.T) {
	s, dir, page := pluginFixture(t, Options{MaxWatchesPerPage: 1})
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	s.factory = func(context.Context, string, mygo.FileWatchOptions) (watcher, error) {
		t.Fatal("unauthorized factory call")
		return nil, nil
	}
	// A root without Recursive lets pages watch it and its files, not its
	// directories.
	for _, req := range []request{{Root: "project", Recursive: true}, {Root: "project", Path: "sub"}} {
		if parts := collect(s, context.Background(), page, req); parts[0].Code != "denied" {
			t.Fatalf("%+v: %+v", req, parts)
		}
	}
	block := make(chan struct{})
	s.factory = func(_ context.Context, path string, _ mygo.FileWatchOptions) (watcher, error) {
		return &fakeWatcher{root: path, closed: make(chan struct{}), err: errors.New("blocked")}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.run(ctx, page, request{Root: "project", Path: "file"}, func(p part) error {
			if p.Type == "ready" {
				<-block
			}
			return nil
		}, func() {})
	}()
	for {
		s.mu.Lock()
		n := len(s.active)
		s.mu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if parts := collect(s, context.Background(), page, request{Root: "project"}); parts[0].Code != "limit" {
		t.Fatalf("parts %+v", parts)
	}
	// Another page has its own quota, and a page's watches whose context
	// is done no longer count, before they have stopped.
	if parts := collect(s, context.Background(), new(int), request{Root: "project"}); parts[0].Type != "ready" {
		t.Fatalf("parts %+v", parts)
	}
	cancel()
	if parts := collect(s, context.Background(), page, request{Root: "project"}); parts[0].Type != "ready" {
		t.Fatalf("parts %+v", parts)
	}
	close(block)
	<-done
}

func TestPaths(t *testing.T) {
	s, dir, page := pluginFixture(t, Options{})
	s.opts.Roots["project"] = Root{Path: dir, Recursive: true}
	for _, p := range []string{"a/b", "a/b/c"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(p)), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(dir, "a"), filepath.Join(dir, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	var watched []string
	s.factory = func(_ context.Context, path string, _ mygo.FileWatchOptions) (watcher, error) {
		watched = append(watched, path)
		return &fakeWatcher{root: path, closed: make(chan struct{})}, nil
	}
	for _, p := range []string{"", ".", "a", "a/b/c"} {
		if parts := collect(s, context.Background(), page, request{Root: "project", Path: p, Recursive: true}); parts[0].Type != "ready" {
			t.Fatalf("%q: %+v", p, parts)
		}
	}
	want := []string{dir, dir, filepath.Join(dir, "a"), filepath.Join(dir, "a", "b", "c")}
	if strings.Join(watched, "\n") != strings.Join(want, "\n") {
		t.Fatalf("watched %v", watched)
	}
	for _, p := range []string{"..", "../x", "/etc", "a/../..", "a//b", "./a", `a\b`, "link", "link/b", "missing"} {
		parts := collect(s, context.Background(), page, request{Root: "project", Path: p})
		if parts[0].Type != "error" || p == "missing" && parts[0].Code != "not-found" {
			t.Fatalf("%q: %+v", p, parts)
		}
	}
	// What Ignore is asked is relative to the root.
	var asked string
	s.opts.Roots["project"] = Root{Path: dir, Recursive: true, Ignore: func(p string, isDir bool) bool { asked = p; return false }}
	s.factory = func(_ context.Context, path string, o mygo.FileWatchOptions) (watcher, error) {
		o.Ignore("c", true)
		return &fakeWatcher{root: path, closed: make(chan struct{})}, nil
	}
	collect(s, context.Background(), page, request{Root: "project", Path: "a/b"})
	if asked != "a/b/c" {
		t.Fatalf("asked %q", asked)
	}
}

// Allow gets the request, and Resolve grants roots at run time.
func TestAllowAndResolve(t *testing.T) {
	picked, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var requests []Request
	s, _, page := pluginFixture(t, Options{
		Resolve: func(ctx context.Context, name string) (Root, bool) {
			return Root{Path: picked, Recursive: true}, name == "picked"
		},
		Allow: func(ctx context.Context, r Request) bool {
			requests = append(requests, r)
			return r.Path != "secret"
		},
	})
	if err := os.Mkdir(filepath.Join(picked, "secret"), 0700); err != nil {
		t.Fatal(err)
	}
	s.factory = func(_ context.Context, path string, _ mygo.FileWatchOptions) (watcher, error) {
		return &fakeWatcher{root: path, closed: make(chan struct{})}, nil
	}
	if parts := collect(s, context.Background(), page, request{Root: "picked", Recursive: true}); parts[0].Type != "ready" {
		t.Fatalf("parts %+v", parts)
	}
	if parts := collect(s, context.Background(), page, request{Root: "picked", Path: "secret"}); parts[0].Code != "denied" {
		t.Fatalf("parts %+v", parts)
	}
	if parts := collect(s, context.Background(), page, request{Root: "other"}); parts[0].Code != "denied" {
		t.Fatalf("parts %+v", parts)
	}
	if len(requests) != 2 || requests[0] != (Request{Root: "picked", Path: ".", Recursive: true}) || requests[1].Path != "secret" {
		t.Fatalf("requests %+v", requests)
	}
}

// A root replaced since setup is not watched.
func TestReplacedRoot(t *testing.T) {
	s, dir, page := pluginFixture(t, Options{})
	if err := os.Rename(dir, dir+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if parts := collect(s, context.Background(), page, request{Root: "project"}); parts[0].Code != "io" {
		t.Fatalf("parts %+v", parts)
	}
}

func TestDenyUnknownRoot(t *testing.T) {
	p := New(Options{})
	if err := p.Setup(); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{"project", "../", "/tmp", `C:\secret`, "a:b"} {
		parts := collect(p.Service.(*service), context.Background(), new(int), request{Root: root})
		if len(parts) != 1 || parts[0].Code != "denied" {
			t.Fatalf("%q: %+v", root, parts)
		}
	}
}

func TestCanonicalRoot(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "real")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "alias")
	if err := os.Symlink(dir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	p := New(Options{Roots: map[string]Root{"project": {Path: link}}})
	if err := p.Setup(); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := p.Service.(*service)
	if got := s.opts.Roots["project"].Path; got != want {
		t.Fatalf("root = %q, want %q", got, want)
	}
}

func TestConfiguration(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "config.json")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	roots := map[string]Root{"project": {Path: dir}, "config": {Path: file}}
	p := New(Options{Roots: roots})
	roots["project"] = Root{Path: "relative"}
	if err := p.Setup(); err != nil {
		t.Fatal(err)
	}
	for _, opts := range []Options{{Roots: map[string]Root{"../": {Path: dir}}}, {Roots: map[string]Root{"project": {Path: "relative"}}}, {MaxWatchesPerPage: -1}, {MaxWatches: -1}} {
		if err := New(opts).Setup(); err == nil {
			t.Fatalf("accepted %+v", opts)
		}
	}
}

// A file root replaced, as editors save, is still watched.
func TestFileRoot(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	s, _, page := pluginFixture(t, Options{Roots: map[string]Root{"settings": {Path: file}}})
	s.factory = func(_ context.Context, path string, _ mygo.FileWatchOptions) (watcher, error) {
		return &fakeWatcher{root: path, closed: make(chan struct{})}, nil
	}
	if err := os.WriteFile(file+".tmp", []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(file+".tmp", file); err != nil {
		t.Fatal(err)
	}
	if parts := collect(s, context.Background(), page, request{Root: "settings"}); parts[0].Type != "ready" {
		t.Fatalf("parts %+v", parts)
	}
	for _, req := range []request{{Root: "settings", Path: "x"}, {Root: "settings", Recursive: true}} {
		if parts := collect(s, context.Background(), page, req); parts[0].Code == "" {
			t.Fatalf("%+v: %+v", req, parts)
		}
	}

	// A file grant must not expand to a directory and its children.
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(file, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(file, "child"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	s.factory = func(context.Context, string, mygo.FileWatchOptions) (watcher, error) {
		t.Fatal("a file root replaced by a directory reached the watcher")
		return nil, nil
	}
	for _, path := range []string{".", "child"} {
		parts := collect(s, context.Background(), page, request{Root: "settings", Path: path})
		if len(parts) != 1 || parts[0].Code != "io" {
			t.Fatalf("%q: %+v", path, parts)
		}
	}
}
