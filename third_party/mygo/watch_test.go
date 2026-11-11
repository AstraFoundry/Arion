//go:build !js

package mygo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
)

// fileWatchTestNotice reports notice for the target at path, found by
// path: on Windows the root's parent is enrolled before the root, so IDs
// differ.
func fileWatchTestNotice(t *testing.T, w *FileWatcher, path string, notice platform.FileWatchNotice) {
	t.Helper()
	native := w.native.(*fake.FileWatch)
	for _, target := range native.Targets() {
		if target.Path == filepath.Join(w.root, filepath.FromSlash(path)) {
			notice.Target = target.ID
			native.Notify(notice)
			return
		}
	}
	t.Fatalf("no watch target for %q", path)
}

// fileWatchTestEvents holds the events of batches not yet looked at.
var fileWatchTestEvents = map[*FileWatcher][]FileEvent{}

// fileWatchTestNext returns the next event, from the batch Next returned
// last or the next one.
func fileWatchTestNext(t *testing.T, w *FileWatcher) FileEvent {
	t.Helper()
	ctx := wait.Context(t)
	for len(fileWatchTestEvents[w]) == 0 {
		batch, err := w.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(batch) == 0 {
			t.Fatal("empty batch")
		}
		fileWatchTestEvents[w] = batch
	}
	e := fileWatchTestEvents[w][0]
	fileWatchTestEvents[w] = fileWatchTestEvents[w][1:]
	return e
}

// fileWatchTestBatch returns the next batch.
func fileWatchTestBatch(t *testing.T, w *FileWatcher) []FileEvent {
	t.Helper()
	ctx := wait.Context(t)
	batch, err := w.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return batch
}

func TestFileWatchLifecycle(t *testing.T) {
	dir := t.TempDir()
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if want, _ := filepath.EvalSymlinks(dir); w.Root() != want {
		t.Fatalf("root = %q, want %q", w.Root(), want)
	}
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{})
	// Directories written are not reported: their entries are.
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileCreate, Path: "a"}}) {
		t.Fatalf("batch = %+v", batch)
	}
	canceled, cancelWait := context.WithCancel(context.Background())
	cancelWait()
	if _, err = w.Next(canceled); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	w.Close()
	ctx := wait.Context(t)
	if _, err = w.Next(ctx); err != io.EOF || w.Err() != nil {
		t.Fatal(err, w.Err())
	}
	select {
	case <-w.Done():
	case <-ctx.Done():
		t.Fatal("cleanup timed out")
	}
}

// Canceling the context of a watch stops it as Close does.
func TestFileWatchContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w, err := WatchFiles(ctx, t.TempDir(), FileWatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	<-w.Done()
	if _, err = w.Next(context.Background()); err != io.EOF || w.Err() != nil {
		t.Fatal(err, w.Err())
	}
}

func TestFileWatchParentSentinel(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows enrolls a parent sentinel")
	}
	w, err := WatchFiles(context.Background(), t.TempDir(), FileWatchOptions{Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.WriteFile(filepath.Join(w.root, "a"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	// A hint for a sibling must not scan the root or expose the sibling.
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: filepath.Base(w.root) + "-sibling"})
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := w.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("sibling hint was not ignored: %v", err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: filepath.Base(w.root)})
	if e := fileWatchTestNext(t, w); e != (FileEvent{Op: FileCreate, Path: "a"}) {
		t.Fatalf("unexpected event: %+v", e)
	}
}

func TestFileWatchStaleFailureIgnored(t *testing.T) {
	w, err := WatchFiles(context.Background(), t.TempDir(), FileWatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	fb.LastFileWatch().Notify(platform.FileWatchNotice{Target: 99999, Err: errors.New("retired target")})
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := w.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stale failure stopped watcher: %v", err)
	}
}

func TestFileWatchValidation(t *testing.T) {
	if w, err := WatchFiles(context.Background(), t.TempDir(), FileWatchOptions{MaxEntries: -1}); err == nil {
		w.Close()
		t.Fatal("accepted a negative limit")
	}
	if _, err := WatchFiles(nil, t.TempDir(), FileWatchOptions{}); err == nil {
		t.Fatal("nil context")
	}
	if _, err := WatchFiles(context.Background(), "", FileWatchOptions{}); err == nil {
		t.Fatal("empty path")
	}
	// A MaxDelay shorter than Debounce is Debounce.
	w, err := WatchFiles(context.Background(), t.TempDir(), FileWatchOptions{Debounce: time.Second, MaxDelay: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if w.opts.MaxDelay != time.Second {
		t.Fatalf("MaxDelay = %v", w.opts.MaxDelay)
	}
}

func TestFileWatchRecursiveLimit(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, MaxEntries: 2}); !errors.Is(err, ErrWatchLimit) {
		t.Fatalf("limit: %v", err)
	}
}

// The watch doesn't look again while a batch waits for Next, and what
// changed meanwhile comes in the next batch.
func TestFileWatchBackpressure(t *testing.T) {
	dir := t.TempDir()
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
		fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: name, Kind: platform.FileWatchCreate})
		// The first look's batch waits for Next before the second notice.
		wait.Until(t, "a batch of "+name, w.waiting)
	}
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileCreate, Path: "a"}}) {
		t.Fatalf("first batch = %+v", batch)
	}
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileCreate, Path: "b"}}) {
		t.Fatalf("second batch = %+v", batch)
	}
	// Many notices while a batch waits look at everything once.
	for i := range 20 {
		name := fmt.Sprint("c", i)
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
		fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: name, Kind: platform.FileWatchCreate})
	}
	seen := 0
	for seen < 20 {
		seen += len(fileWatchTestBatch(t, w))
	}
	if seen != 20 {
		t.Fatalf("%d events", seen)
	}
}

// A watched directory removed ends the watch after an event saying so.
func TestFileWatchRootLoss(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "root")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Kind: platform.FileWatchRemove})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileRemove, Path: ".", IsDir: true}}) {
		t.Fatalf("batch = %+v", batch)
	}
	if _, err = w.Next(context.Background()); err != io.EOF || w.Err() != nil {
		t.Fatal(err, w.Err())
	}
}

// An entry replaced or removed between the scan seeing it and the backend
// enrolling it doesn't stop the watch.
func TestFileWatchEnrollmentRace(t *testing.T) {
	for _, recursive := range []bool{false, true} {
		for _, replace := range []bool{false, true} {
			for _, isDir := range []bool{false, true} {
				name := fmt.Sprintf("recursive=%v/replace=%v/directory=%v", recursive, replace, isDir)
				t.Run(name, func(t *testing.T) {
					dir := t.TempDir()
					w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: recursive, Debounce: -1})
					if err != nil {
						t.Fatal(err)
					}
					defer w.Close()
					path := filepath.Join(dir, "entry")
					create := func(path string) error {
						if isDir {
							return os.Mkdir(path, 0700)
						}
						return os.WriteFile(path, []byte("data"), 0600)
					}
					raced := false
					w.native.(*fake.FileWatch).AddHook = func(target platform.FileWatchTarget) error {
						if target.Path != filepath.Join(w.root, "entry") || raced {
							return nil
						}
						raced = true
						if !replace {
							return os.Remove(path)
						}
						replacement := filepath.Join(dir, "replacement")
						if err := create(replacement); err != nil {
							return err
						}
						if isDir {
							if err := os.Rename(path, filepath.Join(t.TempDir(), "retired")); err != nil {
								return err
							}
						}
						return os.Rename(replacement, path)
					}
					if err := create(path); err != nil {
						t.Fatal(err)
					}
					fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "entry", Kind: platform.FileWatchCreate})
					// An entry removed meanwhile was never there.
					for replace {
						if e := fileWatchTestNext(t, w); e.Path == "entry" {
							break
						}
					}
					if err := os.WriteFile(filepath.Join(dir, "later"), nil, 0600); err != nil {
						t.Fatal(err)
					}
					fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "later", Kind: platform.FileWatchCreate})
					for {
						if e := fileWatchTestNext(t, w); e.Path == "later" && e.Op == FileCreate {
							break
						}
					}
					if !raced {
						t.Fatal("the entry was not enrolled")
					}
				})
			}
		}
	}
}

// A notice for a file looks only at it, and one for a directory only at
// its entries.
func TestFileWatchLooksOnlyWhereNotified(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"a", "b/c", "d/e"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(p)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	// Changes the watch isn't told about stay unseen.
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("unseen"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "d", "f"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b", "c"), []byte("seen"), 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "b", platform.FileWatchNotice{Name: "c", Kind: platform.FileWatchWrite})
	if e := fileWatchTestNext(t, w); e != (FileEvent{Op: FileWrite, Path: "b/c"}) {
		t.Fatalf("event = %+v", e)
	}
	fileWatchTestNotice(t, w, "d", platform.FileWatchNotice{})
	if e := fileWatchTestNext(t, w); e != (FileEvent{Op: FileCreate, Path: "d/f"}) {
		t.Fatalf("event = %+v", e)
	}
	// Losing changes looks at everything.
	fb.LastFileWatch().Notify(platform.FileWatchNotice{Err: platform.ErrFileWatchOverflow})
	if e := fileWatchTestNext(t, w); e != (FileEvent{Op: FileWrite, Path: "a"}) {
		t.Fatalf("event = %+v", e)
	}
}

// Looking at everything again lists each directory from the one that has
// it, and a directory replaced by a symlink, which may lead out of the
// directory that has it or out of the watch, is a replacement, not the end
// of the watch.
func TestFileWatchRescan(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "dir")
	for _, p := range []string{"a/b/c/old", "a/b/gone", "a/pkg/index.js", "a/lib/index.js", "outside/x"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(p)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Rename(filepath.Join(dir, "outside"), filepath.Join(base, "outside")); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	replace := func(p, link string) {
		t.Helper()
		if err := os.RemoveAll(filepath.Join(dir, p)); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(link, filepath.Join(dir, p)); err != nil {
			if runtime.GOOS == "windows" {
				t.Skip(err) // symlinks need developer mode
			}
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a/b/c/new"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "a/b/gone")); err != nil {
		t.Fatal(err)
	}
	replace("a/pkg", "lib") // within the directory that has it
	fb.LastFileWatch().Notify(platform.FileWatchNotice{Err: platform.ErrFileWatchOverflow})
	want := []FileEvent{
		{Op: FileCreate, Path: "a/b/c/new"},
		{Op: FileRemove, Path: "a/b/gone"},
		{Op: FileRemove, Path: "a/pkg", IsDir: true},
		{Op: FileCreate, Path: "a/pkg"},
		{Op: FileRemove, Path: "a/pkg/index.js"},
	}
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, want) {
		t.Fatalf("batch = %+v, want %+v", batch, want)
	}
	// Out of the watch, and listed by itself.
	replace("a/lib", "../../outside")
	fileWatchTestNotice(t, w, "a/lib", platform.FileWatchNotice{})
	want = []FileEvent{
		{Op: FileRemove, Path: "a/lib", IsDir: true},
		{Op: FileCreate, Path: "a/lib"},
		{Op: FileRemove, Path: "a/lib/index.js"},
	}
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, want) {
		t.Fatalf("batch = %+v, want %+v", batch, want)
	}
	if err := w.Err(); err != nil {
		t.Fatal(err)
	}
}

// A directory renamed is one rename, not one for everything in it.
func TestFileWatchDirectoryRename(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "b", "c"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.Rename(filepath.Join(dir, "a"), filepath.Join(dir, "z")); err != nil {
		t.Fatal(err)
	}
	native := w.native.(*fake.FileWatch)
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "a", Kind: platform.FileWatchMoveFrom})
	if e := fileWatchTestNext(t, w); e != (FileEvent{Op: FileRename, OldPath: "a", Path: "z", IsDir: true}) {
		t.Fatalf("event = %+v", e)
	}
	// Its directories are enrolled again under their new paths.
	enrolled := map[string]bool{}
	for _, target := range native.Targets() {
		enrolled[target.Path] = true
	}
	for _, p := range []string{"z", "z/b", "z/b/c"} {
		if !enrolled[filepath.Join(w.root, filepath.FromSlash(p))] {
			t.Errorf("%s not enrolled", p)
		}
	}
	if enrolled[filepath.Join(w.root, "a")] {
		t.Error("a still enrolled")
	}
}

// A backend that watches a tree with one target is given only the root,
// and names what changes below by its path.
func TestFileWatchSubtree(t *testing.T) {
	fb.FileWatchSubtree = true
	defer func() { fb.FileWatchSubtree = false }()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0700); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	native := w.native.(*fake.FileWatch)
	for _, target := range native.Targets() {
		if target.Path == w.root {
			if !target.Subtree {
				t.Error("the root is not a subtree target")
			}
		} else if target.Path != filepath.Dir(w.root) {
			t.Errorf("%s enrolled", target.Path)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "a", "b", "c", "d"), 0700); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "a/b/c", Kind: platform.FileWatchCreate})
	want := []FileEvent{{Op: FileCreate, Path: "a/b/c", IsDir: true}, {Op: FileCreate, Path: "a/b/c/d", IsDir: true}}
	for _, want := range want {
		if e := fileWatchTestNext(t, w); e != want {
			t.Fatalf("event = %+v, want %+v", e, want)
		}
	}
	// The root's target lost enrolls it again and looks at everything.
	if err := os.WriteFile(filepath.Join(dir, "a", "f"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Kind: platform.FileWatchLost})
	if e := fileWatchTestNext(t, w); e != (FileEvent{Op: FileCreate, Path: "a/f"}) {
		t.Fatalf("event = %+v", e)
	}
	found := false
	for _, target := range native.Targets() {
		found = found || target.Path == w.root && target.Subtree
	}
	if !found {
		t.Fatal("the root was not enrolled again")
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "../x"})
	ctx := wait.Context(t)
	if _, err := w.Next(ctx); err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a name out of the tree: %v", err)
	}
}

type watchTestInfo struct{ os.FileInfo }

func TestFileWatchDiff(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	// Stat through a handle captures the identity now. On Windows os.Stat
	// defers fetching it until SameFile, when the old name no longer exists.
	info, err := f.Stat()
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	old := map[string]fileWatchEntry{"a": fileWatchEntryOf(info)}
	if err := os.Rename(a, b); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(b)
	if err != nil {
		t.Fatal(err)
	}
	events := fileWatchTestDiff(old, map[string]fileWatchEntry{"b": fileWatchEntryOf(info)}, map[string]bool{"a": true})
	if len(events) != 2 || events[0].Op != FileRename || events[0].OldPath != "a" || events[1].Op != FileWrite {
		t.Fatalf("%+v", events)
	}
}

// A file replaced at its path by another file, as an atomic save replaces
// it, is written, whatever its size and time: not removed and created.
// What isn't a file replaced by a file stays a removal and a creation, and
// the file that replaced it, gone from where it was, a removal there.
func TestFileWatchDiffReplacedFile(t *testing.T) {
	file := func(i uint64) fileWatchEntry {
		return fileWatchEntry{id: fileWatchID{volume: 1, index: i}, mode: 0600, size: 3}
	}
	dir := fileWatchEntry{id: fileWatchID{volume: 1, index: 9, typ: fs.ModeDir}, mode: fs.ModeDir | 0700}
	for _, tt := range []struct {
		name          string
		before, after map[string]fileWatchEntry
		want          []FileEvent
	}{
		{"replaced", map[string]fileWatchEntry{"a": file(1)}, map[string]fileWatchEntry{"a": file(2)},
			[]FileEvent{{Op: FileWrite, Path: "a"}}},
		{"from a temporary file", map[string]fileWatchEntry{"a": file(1), "a.tmp": file(2)}, map[string]fileWatchEntry{"a": file(2)},
			[]FileEvent{{Op: FileWrite, Path: "a"}, {Op: FileRemove, Path: "a.tmp"}}},
		{"kept beside", map[string]fileWatchEntry{"a": file(1)}, map[string]fileWatchEntry{"a": file(2), "a~": file(1)},
			[]FileEvent{{Op: FileWrite, Path: "a"}, {Op: FileCreate, Path: "a~"}}},
		{"by a directory", map[string]fileWatchEntry{"a": file(1)}, map[string]fileWatchEntry{"a": dir},
			[]FileEvent{{Op: FileRemove, Path: "a"}, {Op: FileCreate, Path: "a", IsDir: true}}},
	} {
		if events := fileWatchTestDiff(tt.before, tt.after, nil); !slices.Equal(events, tt.want) {
			t.Errorf("%s: events = %+v, want %+v", tt.name, events, tt.want)
		}
	}
}

// A file system such as ext4 gives a freed inode to the next file
// created: a directory replaced by a symlink with its inode is a removal
// and a creation, not a write.
func TestFileWatchDiffReusedInode(t *testing.T) {
	id := fileWatchID{volume: 1, index: 2}
	dir, link := id, id
	dir.typ, link.typ = fs.ModeDir, fs.ModeSymlink
	before := map[string]fileWatchEntry{"a": {id: dir, mode: fs.ModeDir | 0700}, "a/f": {id: fileWatchID{volume: 1, index: 3}}}
	after := map[string]fileWatchEntry{"a": {id: link, mode: fs.ModeSymlink | 0777}}
	want := []FileEvent{
		{Op: FileRemove, Path: "a", IsDir: true},
		{Op: FileCreate, Path: "a"},
		{Op: FileRemove, Path: "a/f"},
	}
	if events := fileWatchTestDiff(before, after, nil); !slices.Equal(events, want) {
		t.Fatalf("events = %+v, want %+v", events, want)
	}
}

// fileWatchTestInfos returns the stats of files in dir, as the scan keeps
// them: a, b, c, a hard link of a when the file system has them, and a
// directory d; then a, after a was renamed to a2, by another way.
func fileWatchTestInfos(t testing.TB, dir string) []os.FileInfo {
	t.Helper()
	for _, name := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	names := []string{"a", "b", "c", "d"}
	if err := os.Mkdir(filepath.Join(dir, "d"), 0700); err != nil {
		t.Fatal(err)
	}
	if os.Link(filepath.Join(dir, "a"), filepath.Join(dir, "link")) == nil {
		names = append(names, "link")
	}
	var infos []os.FileInfo
	for _, name := range names {
		info, err := os.Lstat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		infos = append(infos, StatIdentity(info))
	}
	if err := os.Rename(filepath.Join(dir, "a"), filepath.Join(dir, "a2")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	info, err := root.Lstat("a2")
	if err != nil {
		t.Fatal(err)
	}
	return append(infos, StatIdentity(info))
}

// The identity rename pairing looks up is what os.SameFile compares.
func TestFileWatchIdentity(t *testing.T) {
	infos := fileWatchTestInfos(t, t.TempDir())
	for i, a := range infos {
		ida, ok := fileWatchIdentity(a)
		if !ok {
			t.Fatalf("no identity for %s (%T)", a.Name(), a)
		}
		for _, b := range infos[i:] {
			idb, _ := fileWatchIdentity(b)
			if same := os.SameFile(a, b); same != (ida == idb) {
				t.Errorf("%s, %s: SameFile %v, identities %v and %v", a.Name(), b.Name(), same, ida, idb)
			}
		}
	}
	if _, ok := fileWatchIdentity(watchTestInfo{infos[0]}); ok {
		t.Error("an identity for a FileInfo of another type")
	}
}

// Pairing by identity pairs what comparing every entry with every other
// one would, for updates that rename, replace, hard link, move with a
// directory, create and remove.
func TestFileWatchDiffPairing(t *testing.T) {
	var infos []fileWatchEntry
	for _, info := range fileWatchTestInfos(t, t.TempDir()) {
		infos = append(infos, fileWatchEntryOf(info))
	}
	paths := []string{"a", "b", "c", "d", "d/x", "d/y", "e", "e/x", "e/y", "f"}
	rng := rand.New(rand.NewPCG(1, 2))
	pick := func() map[string]fileWatchEntry {
		m := map[string]fileWatchEntry{}
		for _, p := range paths {
			if rng.IntN(3) > 0 {
				m[p] = infos[rng.IntN(len(infos))]
			}
		}
		return m
	}
	// unique reports whether exactly one entry of m is a.
	unique := func(m map[string]fileWatchEntry, a fileWatchEntry) bool {
		count := 0
		for _, v := range m {
			if v.id == a.id {
				count++
			}
		}
		return count == 1
	}
	paired := 0
	for i := range 5000 {
		before, after := pick(), pick()
		changed, _ := fileWatchTestChanged(before, after)
		ids := newFileWatchIDs(changed, after)
		for old, a := range before {
			want, count := "", 0
			for p, b := range after {
				if b.id == a.id {
					if c, ok := before[p]; !ok || c.id != b.id {
						want = p
						count++
					}
				}
			}
			if count != 1 || !unique(before, a) || !unique(after, a) {
				want = ""
			}
			got := ids.pair(a)
			if got != "" {
				if c, ok := before[got]; ok && c.id == a.id {
					got = "" // not created: the same file at the same path
				}
			}
			if got != want {
				t.Fatalf("case %d, %s: by identity %q, by comparing all %q; before %v after %v", i, old, got, want, fileWatchTestNames(before), fileWatchTestNames(after))
			}
			if got != "" {
				paired++
			}
		}
	}
	if paired == 0 {
		t.Fatal("nothing paired")
	}
}

// fileWatchTestChanged is what an update that found before as after
// looked at: every path of either.
func fileWatchTestChanged(before, after map[string]fileWatchEntry) (map[string]fileWatchBefore, map[string]fileWatchEntry) {
	changed := map[string]fileWatchBefore{}
	for p, e := range before {
		changed[p] = fileWatchBefore{e, true}
	}
	for p := range after {
		if _, ok := changed[p]; !ok {
			changed[p] = fileWatchBefore{}
		}
	}
	return changed, after
}

// fileWatchTestDiff is fileWatchDiff of an update that found before as
// after.
func fileWatchTestDiff(before, after map[string]fileWatchEntry, writes map[string]bool) []FileEvent {
	changed, entries := fileWatchTestChanged(before, after)
	return fileWatchDiff(changed, entries, writes)
}

func fileWatchTestNames(m map[string]fileWatchEntry) map[string]uint64 {
	names := map[string]uint64{}
	for p, e := range m {
		names[p] = e.id.index
	}
	return names
}

// A directory of n files renamed is paired in time linear in n: by
// identity, not by comparing every entry with every other one.
func BenchmarkFileWatchDiffRename(b *testing.B) {
	for _, n := range []int{2500, 5000, 10000, 20000, 40000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			dir := b.TempDir()
			before := map[string]fileWatchEntry{}
			after := map[string]fileWatchEntry{}
			for i := range n {
				name := fmt.Sprint("f", i)
				if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
					b.Fatal(err)
				}
				info, err := os.Lstat(filepath.Join(dir, name))
				if err != nil {
					b.Fatal(err)
				}
				e := fileWatchEntryOf(info)
				before["a/"+name], after["b/"+name] = e, e
			}
			info, err := os.Lstat(dir)
			if err != nil {
				b.Fatal(err)
			}
			before["a"], after["b"] = fileWatchEntryOf(info), fileWatchEntryOf(info)
			changed, _ := fileWatchTestChanged(before, after)
			for b.Loop() {
				if events := fileWatchDiff(changed, after, nil); len(events) != 1 || events[0].Op != FileRename {
					b.Fatal(events)
				}
			}
		})
	}
}

// On Unix the scan takes what reading a directory opened in an os.Root says
// of an entry rather than stat it again: the os package stats every entry
// as it reads it, relative to the directory and without following a
// symlink, not later by a path that could lead out of the root.
func TestFileWatchEntryInfo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stat'ed by name in the directory's root")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a", filepath.Join(dir, "l")); err != nil {
		t.Fatal(err)
	}
	want, err := os.Lstat(filepath.Join(dir, "a"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	f, err := root.Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		t.Fatal(err)
	}
	// Gone once read: what the entries say was stat'ed already.
	for _, name := range []string{"a", "l"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range entries {
		info, err := fileWatchEntryInfo(root, entry)
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		switch entry.Name() {
		case "a":
			if !os.SameFile(info, want) || info.Size() != 1 {
				t.Fatalf("a: %v", info)
			}
		case "l":
			if info.Mode()&fs.ModeSymlink == 0 {
				t.Fatalf("l: %v, want the symlink", info.Mode())
			}
		}
	}
}

// A file is watched through its directory: written, replaced as editors
// save, removed and created again, then gone with its directory.
func TestFileWatchFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dir")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "config.json")
	if err := os.WriteFile(file, []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sibling"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), file, FileWatchOptions{Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	native := w.native.(*fake.FileWatch)
	enrolled := map[string]bool{}
	for _, target := range native.Targets() {
		enrolled[target.Path] = true
	}
	if !enrolled[w.Root()] || !enrolled[filepath.Dir(w.Root())] || len(enrolled) != 2 {
		t.Fatalf("enrolled %v", enrolled)
	}
	next := func(want FileEvent) {
		t.Helper()
		if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{want}) {
			t.Fatalf("batch = %+v, want %+v", batch, want)
		}
	}
	quiet := func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		if batch, err := w.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("batch = %+v, %v", batch, err)
		}
	}
	if err := os.WriteFile(file, []byte("bb"), 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Kind: platform.FileWatchWrite})
	next(FileEvent{Op: FileWrite, Path: "."})
	// Other files of the directory are not reported.
	if err := os.WriteFile(filepath.Join(dir, "sibling"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "sibling", Kind: platform.FileWatchWrite})
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{})
	quiet()
	// Replaced, as editors save.
	if err := os.WriteFile(filepath.Join(dir, "config.json.tmp"), []byte("ccc"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "config.json.tmp"), file); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "config.json", Kind: platform.FileWatchMoveTo})
	next(FileEvent{Op: FileWrite, Path: "."})
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "config.json", Kind: platform.FileWatchRemove})
	next(FileEvent{Op: FileRemove, Path: "."})
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "config.json", Kind: platform.FileWatchCreate})
	next(FileEvent{Op: FileCreate, Path: "."})
	// The new file is watched in its turn.
	found := false
	for _, target := range native.Targets() {
		found = found || target.Path == w.Root()
	}
	if !found {
		t.Fatal("the new file is not enrolled")
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Kind: platform.FileWatchLost})
	next(FileEvent{Op: FileRemove, Path: "."})
	if _, err := w.Next(context.Background()); err != io.EOF || w.Err() != nil {
		t.Fatal(err, w.Err())
	}
}

// A watched file that goes away comes with what it was when the watch
// last saw it, here the file that replaced it, so that a document finds
// it where it was renamed whenever it looks.
func TestFileWatchFileGone(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "doc.txt")
	if err := os.WriteFile(file, []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), file, FileWatchOptions{Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	next := func() fileResult {
		t.Helper()
		r := w.next(context.Background())
		if r.err != nil {
			t.Fatal(r.err)
		}
		return r
	}
	// Saved as editors save: a new file renamed over it.
	if err := os.WriteFile(file+".tmp", []byte("bb"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(file+".tmp", file); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "doc.txt", Kind: platform.FileWatchCreate})
	if r := next(); !slices.Equal(r.events, []FileEvent{{Op: FileWrite, Path: "."}}) || r.gone.ok {
		t.Fatalf("saved: %+v", r)
	}
	renamed := filepath.Join(dir, "renamed.txt")
	if err := os.Rename(file, renamed); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "doc.txt", Kind: platform.FileWatchRemove})
	r := next()
	if !slices.Equal(r.events, []FileEvent{{Op: FileRemove, Path: "."}}) || !r.gone.ok {
		t.Fatalf("renamed: %+v", r)
	}
	if found := findRenamed(filepath.Dir(w.Root()), []fileWatchGone{r.gone}); found != filepath.Join(filepath.Dir(w.Root()), "renamed.txt") {
		t.Fatalf("found %q", found)
	}
}

// What Ignore ignores is never enrolled, counted or reported.
func TestFileWatchIgnore(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"node_modules/a/b/c", "node_modules/d", "src/main.go", "debug.log"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(p)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var asked []string
	ignore := func(p string, isDir bool) bool {
		asked = append(asked, p)
		return p == "node_modules" && isDir || path.Ext(p) == ".log"
	}
	// Only the root, src, src/main.go and src/new.go, created below,
	// count.
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1, Ignore: ignore, MaxEntries: 4})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for _, p := range asked {
		if strings.HasPrefix(p, "node_modules/") {
			t.Errorf("asked about %s", p)
		}
	}
	for _, target := range w.native.(*fake.FileWatch).Targets() {
		if strings.Contains(target.Path, "node_modules") || strings.HasSuffix(target.Path, ".log") {
			t.Errorf("%s enrolled", target.Path)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "node_modules", "e"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.log"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "other.log", Kind: platform.FileWatchCreate})
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "node_modules", Kind: platform.FileWatchWrite})
	if err := os.WriteFile(filepath.Join(dir, "src", "new.go"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "src", platform.FileWatchNotice{Name: "new.go", Kind: platform.FileWatchCreate})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileCreate, Path: "src/new.go"}}) {
		t.Fatalf("batch = %+v", batch)
	}
}

// Notices in an ignored directory, which a Subtree target reports, are
// dropped.
func TestFileWatchIgnoreSubtree(t *testing.T) {
	fb.FileWatchSubtree = true
	defer func() { fb.FileWatchSubtree = false }()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0700); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1, Ignore: func(p string, isDir bool) bool { return p == ".git" }})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.WriteFile(filepath.Join(dir, ".git", "objects", "x"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: ".git/objects/x", Kind: platform.FileWatchCreate})
	if err := os.WriteFile(filepath.Join(dir, "y"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "y", Kind: platform.FileWatchCreate})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileCreate, Path: "y"}}) {
		t.Fatalf("batch = %+v", batch)
	}
}

// A backend that watches files through their directories, as every system
// backend does, is given only directories, and what changes in a file is
// named by its directory's target.
func TestFileWatchNoFiles(t *testing.T) {
	fb.FileWatchNoFiles = true
	defer func() { fb.FileWatchNoFiles = false }()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"f", "a/g", "a/b/h"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	native := w.native.(*fake.FileWatch)
	for _, target := range native.Targets() {
		if !target.Directory {
			t.Errorf("file %s enrolled", target.Path)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "b", "h"), []byte("xy"), 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "a/b", platform.FileWatchNotice{Name: "h", Kind: platform.FileWatchWrite})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileWrite, Path: "a/b/h"}}) {
		t.Fatalf("batch = %+v", batch)
	}
	// A file replaced, as editors save, is written.
	if err := os.WriteFile(filepath.Join(dir, "a", "g.tmp"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "a", "g.tmp"), filepath.Join(dir, "a", "g")); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "a", platform.FileWatchNotice{Name: "g", Kind: platform.FileWatchMoveTo})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileWrite, Path: "a/g"}}) {
		t.Fatalf("batch = %+v", batch)
	}
	// A watched file: only its directory is enrolled, and a replacement
	// is a write.
	file := filepath.Join(dir, "f")
	fw, err := WatchFiles(context.Background(), file, FileWatchOptions{Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()
	for _, target := range fw.native.(*fake.FileWatch).Targets() {
		if target.Path != filepath.Dir(fw.Root()) {
			t.Errorf("%s enrolled", target.Path)
		}
	}
	if err := os.WriteFile(file+".tmp", []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(file+".tmp", file); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, fw, "..", platform.FileWatchNotice{Name: "f", Kind: platform.FileWatchMoveTo})
	if batch := fileWatchTestBatch(t, fw); !slices.Equal(batch, []FileEvent{{Op: FileWrite, Path: "."}}) {
		t.Fatalf("batch = %+v", batch)
	}
}

// What the backend reports while the watch's channel is full waits for it
// rather than making it look at everything; only more than MaxEntries
// notices do.
func TestFileWatchSpill(t *testing.T) {
	w := &FileWatcher{ctx: context.Background(), opts: FileWatchOptions{MaxEntries: 3}, raw: make(chan platform.FileWatchNotice, 1), spilled: make(chan struct{}, 1), overflow: make(chan struct{}, 1)}
	for i := range 4 {
		w.notice(platform.FileWatchNotice{Target: uint64(i)})
	}
	if n := <-w.raw; n.Target != 0 {
		t.Fatalf("raw = %+v", n)
	}
	select {
	case <-w.spilled:
	default:
		t.Fatal("spill not signaled")
	}
	select {
	case <-w.overflow:
		t.Fatal("overflow within MaxEntries")
	default:
	}
	if spill := w.unspill(); len(spill) != 3 || spill[0].Target != 1 || spill[2].Target != 3 {
		t.Fatalf("spill = %+v", spill)
	}
	w.raw <- platform.FileWatchNotice{}
	for i := range 4 {
		w.notice(platform.FileWatchNotice{Target: uint64(10 + i)})
	}
	select {
	case <-w.overflow:
	default:
		t.Fatal("no overflow past MaxEntries")
	}
	if spill := w.unspill(); spill != nil {
		t.Fatalf("spill kept past an overflow: %+v", spill)
	}
}

// Changes of permissions alone are not writes.
func TestFileWatchMetadata(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.Chmod(file, 0400); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "a", platform.FileWatchNotice{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if batch, err := w.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("batch = %+v, %v", batch, err)
	}
	fileWatchTestNotice(t, w, "a", platform.FileWatchNotice{Kind: platform.FileWatchWrite})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileWrite, Path: "a"}}) {
		t.Fatalf("batch = %+v", batch)
	}
}

// What the watch can't look at, as a directory without permission or a
// file the system holds (pagefile.sys), is skipped rather than ending the
// watch.
func TestFileWatchInaccessible(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "busy"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	locked := runtime.GOOS != "windows" && os.Geteuid() != 0
	if locked {
		for _, p := range []string{"locked/a", "unsearchable/b"} {
			if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(p)), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, p), nil, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Chmod(filepath.Join(dir, "locked"), 0); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Join(dir, "unsearchable"), 0400); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			os.Chmod(filepath.Join(dir, "locked"), 0700)
			os.Chmod(filepath.Join(dir, "unsearchable"), 0700)
		})
	}
	var busy error = fs.ErrPermission
	if runtime.GOOS == "windows" {
		busy = syscall.Errno(32) // ERROR_SHARING_VIOLATION
	}
	fb.FileWatchAddHook = func(target platform.FileWatchTarget) error {
		if filepath.Base(target.Path) == "busy" {
			return busy
		}
		return nil
	}
	defer func() { fb.FileWatchAddHook = nil }()
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for _, target := range w.native.(*fake.FileWatch).Targets() {
		if filepath.Base(target.Path) == "busy" {
			t.Error("busy enrolled")
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "x"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "x", Kind: platform.FileWatchCreate})
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileCreate, Path: "x"}}) {
		t.Fatalf("batch = %+v", batch)
	}
	if locked {
		fileWatchTestNotice(t, w, "unsearchable", platform.FileWatchNotice{})
		if err := os.Remove(filepath.Join(dir, "x")); err != nil {
			t.Fatal(err)
		}
		fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "x"})
		if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileRemove, Path: "x"}}) {
			t.Fatalf("batch = %+v", batch)
		}
	}
}

// BenchmarkFileWatchScan times a recursive watch's first scan and a look
// at everything again, as after an overflow, with the fake backend: of
// the tree at MYGO_WATCH_BENCH_DIR (.git ignored), or of a generated one,
// nine directories deep.
func BenchmarkFileWatchScan(b *testing.B) {
	dir := os.Getenv("MYGO_WATCH_BENCH_DIR")
	if dir == "" {
		dir = b.TempDir()
		var fill func(p string, depth int)
		fill = func(p string, depth int) {
			for i := range 8 {
				if err := os.WriteFile(filepath.Join(p, fmt.Sprint("f", i)), nil, 0600); err != nil {
					b.Fatal(err)
				}
			}
			if depth == 9 {
				return
			}
			for i := range 2 {
				sub := filepath.Join(p, fmt.Sprint("d", i))
				if err := os.Mkdir(sub, 0700); err != nil {
					b.Fatal(err)
				}
				fill(sub, depth+1)
			}
		}
		fill(dir, 1)
	}
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		b.Fatal(err)
	}
	// Like the system backends, which watch files through their
	// directories.
	fb.FileWatchNoFiles = true
	defer func() { fb.FileWatchNoFiles = false }()
	start := func(b *testing.B) (*fileWatchScan, func()) {
		native, err := backend().NewFileWatch(func(platform.FileWatchNotice) {})
		if err != nil {
			b.Fatal(err)
		}
		ignore := func(p string, _ bool) bool { return p == ".git" }
		w := &FileWatcher{ctx: context.Background(), root: root, native: native, opts: FileWatchOptions{Recursive: true, MaxEntries: 1 << 20, Ignore: ignore}}
		s := newFileWatchScan(w)
		if _, err := s.start(); err != nil {
			b.Fatal(err)
		}
		return s, func() { s.close(); native.Close() }
	}
	b.Run("start", func(b *testing.B) {
		for b.Loop() {
			_, stop := start(b)
			stop()
		}
	})
	b.Run("rescan", func(b *testing.B) {
		s, stop := start(b)
		defer stop()
		for b.Loop() {
			s.all = true
			if events, err := s.update(); err != nil || len(events) > 0 {
				b.Fatal(events, err)
			}
		}
		b.ReportMetric(float64(len(s.entries)), "entries")
	})
}

// fileWatchTestHints returns the next batch, which must hold only hints.
func fileWatchTestHints(t *testing.T, w *FileWatcher) []string {
	t.Helper()
	var paths []string
	for _, e := range fileWatchTestBatch(t, w) {
		if e.Op != FileHint || !e.IsDir || e.OldPath != "" {
			t.Fatalf("a hints-only watch reported %+v", e)
		}
		paths = append(paths, e.Path)
	}
	return paths
}

// A hints-only watch enrolls directories, keeps no snapshot and says only
// in which directory something changed.
func TestFileWatchHintsOnly(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"a/b", "node_modules/pkg"} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(p)), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "b", "f"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	ignore := func(p string, isDir bool) bool { return p == "node_modules" }
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1, HintsOnly: true, Ignore: ignore})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	native := w.native.(*fake.FileWatch)
	enrolled := func() []string {
		var paths []string
		for _, target := range native.Targets() {
			if !target.Directory {
				t.Errorf("file %s enrolled", target.Path)
			}
			if rel, err := filepath.Rel(w.root, target.Path); err == nil && !strings.HasPrefix(rel, "..") {
				paths = append(paths, filepath.ToSlash(rel))
			}
		}
		slices.Sort(paths)
		return paths
	}
	if got := enrolled(); !slices.Equal(got, []string{".", "a", "a/b"}) {
		t.Fatalf("enrolled %v", got)
	}
	// A write to a file hints at its directory, not at the file.
	if err := os.WriteFile(filepath.Join(dir, "a", "b", "f"), []byte("xy"), 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "a/b", platform.FileWatchNotice{Name: "f", Kind: platform.FileWatchWrite})
	if got := fileWatchTestHints(t, w); !slices.Equal(got, []string{"a/b"}) {
		t.Fatalf("hints %v", got)
	}
	// A new directory is enrolled, and hinted at through its parent.
	if err := os.MkdirAll(filepath.Join(dir, "a", "new", "deeper"), 0700); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "a", platform.FileWatchNotice{Name: "new", Kind: platform.FileWatchCreate})
	if got := fileWatchTestHints(t, w); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("hints %v", got)
	}
	if got := enrolled(); !slices.Equal(got, []string{".", "a", "a/b", "a/new", "a/new/deeper"}) {
		t.Fatalf("enrolled %v", got)
	}
	// A directory moved: its targets follow it.
	if err := os.Rename(filepath.Join(dir, "a", "new"), filepath.Join(dir, "moved")); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "a", platform.FileWatchNotice{Name: "new", Kind: platform.FileWatchMoveFrom})
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "moved", Kind: platform.FileWatchMoveTo})
	// One batch or two: "." covers "a", and comes last.
	for got := fileWatchTestHints(t, w); !slices.Equal(got, []string{"."}); got = fileWatchTestHints(t, w) {
		if !slices.Equal(got, []string{"a"}) {
			t.Fatalf("hints %v", got)
		}
	}
	if got := enrolled(); !slices.Equal(got, []string{".", "a", "a/b", "moved", "moved/deeper"}) {
		t.Fatalf("enrolled %v", got)
	}
	// Lost changes hint at everything.
	native.Notify(platform.FileWatchNotice{Err: platform.ErrFileWatchOverflow})
	if got := fileWatchTestHints(t, w); !slices.Equal(got, []string{"."}) {
		t.Fatalf("hints %v", got)
	}
	// The directory gone ends the watch as any other.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Kind: platform.FileWatchLost})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileRemove, Path: ".", IsDir: true}}) {
		t.Fatalf("batch %+v", batch)
	}
	if _, err := w.Next(context.Background()); err != io.EOF {
		t.Fatal(err)
	}
}

// With a backend that watches a tree with one target, a hints-only watch
// enrolls nothing else, and hints at the directory of what changed deep
// down, but not in an ignored one.
func TestFileWatchHintsOnlySubtree(t *testing.T) {
	fb.FileWatchSubtree = true
	defer func() { fb.FileWatchSubtree = false }()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0700); err != nil {
		t.Fatal(err)
	}
	ignore := func(p string, isDir bool) bool { return path.Base(p) == ".git" }
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1, HintsOnly: true, Ignore: ignore})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for _, target := range w.native.(*fake.FileWatch).Targets() {
		if target.Path != w.root && target.Path != filepath.Dir(w.root) {
			t.Errorf("%s enrolled", target.Path)
		}
	}
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: ".git/objects/x", Kind: platform.FileWatchCreate})
	fileWatchTestNotice(t, w, ".", platform.FileWatchNotice{Name: "a/b/f", Kind: platform.FileWatchWrite})
	if got := fileWatchTestHints(t, w); !slices.Equal(got, []string{"a/b"}) {
		t.Fatalf("hints %v", got)
	}
}

// A hints-only watch of a file hints at it, not at the other entries of
// its directory.
func TestFileWatchHintsOnlyFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.json")
	if err := os.WriteFile(file, []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), file, FileWatchOptions{Debounce: -1, HintsOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "sibling", Kind: platform.FileWatchWrite})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if batch, err := w.Next(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("batch = %+v, %v", batch, err)
	}
	fileWatchTestNotice(t, w, "..", platform.FileWatchNotice{Name: "config.json", Kind: platform.FileWatchWrite})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileHint, Path: "."}}) {
		t.Fatalf("batch %+v", batch)
	}
}

// Without HintsOnly, the same change is the precise event it always was.
func TestFileWatchHintsOnlyOptIn(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "b", "f"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	w, err := WatchFiles(context.Background(), dir, FileWatchOptions{Recursive: true, Debounce: -1})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.WriteFile(filepath.Join(dir, "a", "b", "f"), []byte("xy"), 0600); err != nil {
		t.Fatal(err)
	}
	fileWatchTestNotice(t, w, "a/b", platform.FileWatchNotice{Name: "f", Kind: platform.FileWatchWrite})
	if batch := fileWatchTestBatch(t, w); !slices.Equal(batch, []FileEvent{{Op: FileWrite, Path: "a/b/f"}}) {
		t.Fatalf("batch %+v", batch)
	}
}
