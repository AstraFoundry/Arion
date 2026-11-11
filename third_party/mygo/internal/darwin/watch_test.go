//go:build darwin

package darwin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
)

func TestFSEventsEventMapping(t *testing.T) {
	sub := &fseStream{id: 7, path: "/w", subtree: true}
	plain := &fseStream{id: 7, path: "/w"}
	for _, tc := range []struct {
		name  string
		s     *fseStream
		path  string
		flags uint32
		want  string
		kind  platform.FileWatchKind
		err   error
		ok    bool
	}{
		{"root", sub, "/w", fseItemModified | fseItemIsFile, "", platform.FileWatchWrite, nil, true},
		{"child", sub, "/w/child", fseItemModified | fseItemIsFile, "child", platform.FileWatchWrite, nil, true},
		{"subtree deep", sub, "/w/a/b", fseItemCreated, "a/b", platform.FileWatchDirty, nil, true},
		{"plain deep", plain, "/w/a/b", fseItemCreated, "", 0, nil, false},
		{"plain deep dropped", plain, "/w/a/b", fseKernelDropped | fseMustScanSubDirs, "", 0, platform.ErrFileWatchOverflow, true},
		{"created", sub, "/w/a", fseItemCreated, "a", platform.FileWatchDirty, nil, true},
		{"removed", sub, "/w/a", fseItemRemoved, "a", platform.FileWatchDirty, nil, true},
		{"renamed", sub, "/w/a", fseItemRenamed, "a", platform.FileWatchDirty, nil, true},
		{"metadata", sub, "/w/a", fseItemInodeMeta | fseItemIsFile, "a", platform.FileWatchDirty, nil, true},
		{"created and modified", sub, "/w/a", fseItemCreated | fseItemModified | fseItemIsFile, "a", platform.FileWatchDirty, nil, true},
		{"removed and modified", sub, "/w/a", fseItemRemoved | fseItemModified | fseItemIsFile, "a", platform.FileWatchDirty, nil, true},
		{"modified directory", sub, "/w/a", fseItemModified | fseItemIsDir, "a", platform.FileWatchDirty, nil, true},
		{"root changed", sub, "/w", fseRootChanged, "", platform.FileWatchDirty, nil, true},
		{"mount", sub, "/w/a", fseMount, "a", platform.FileWatchDirty, nil, true},
		{"unmount", sub, "/w/a", fseUnmount, "a", platform.FileWatchLost, nil, true},
		{"must scan", sub, "/w/a", fseMustScanSubDirs, "", 0, platform.ErrFileWatchOverflow, true},
		{"user dropped", sub, "/w/a", fseUserDropped, "", 0, platform.ErrFileWatchOverflow, true},
		{"kernel dropped", sub, "/w/a", fseKernelDropped, "", 0, platform.ErrFileWatchOverflow, true},
		{"ids wrapped", sub, "/w/a", fseEventIdsWrapped, "", 0, platform.ErrFileWatchOverflow, true},
		{"history done", sub, "/w/a", fseHistoryDone, "", 0, nil, false},
		{"outside", sub, "/other/a", fseItemCreated, "", 0, nil, false},
		{"sibling prefix", sub, "/w2/a", fseItemCreated, "", 0, nil, false},
		{"volume root", &fseStream{id: 7, path: "/", subtree: true}, "/a/b", fseItemCreated, "a/b", platform.FileWatchDirty, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, ok := tc.s.event(tc.path, tc.flags)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			if n.Target != 7 || n.Kind != tc.kind || n.Name != tc.want || !errors.Is(n.Err, tc.err) {
				t.Fatalf("notice = %+v, want kind %v name %q err %v", n, tc.kind, tc.want, tc.err)
			}
		})
	}
}

// watchTestDir is a temp directory with its symlinks resolved, as the core
// resolves what it watches (macOS's /tmp is /private/tmp).
func watchTestDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// watchTest makes a backend watch, closed when the test ends.
func watchTest(t *testing.T) (platform.FileWatch, chan platform.FileWatchNotice) {
	t.Helper()
	notices := make(chan platform.FileWatchNotice, 256)
	w, err := (&Backend{}).NewFileWatch(func(n platform.FileWatchNotice) { notices <- n })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Close)
	return w, notices
}

// watchTestNext waits for a notice of kind about a target naming one of
// names, and fails when none comes.
func watchTestNext(t *testing.T, notices chan platform.FileWatchNotice, target uint64, kind platform.FileWatchKind, names ...string) {
	t.Helper()
	watchTestNextKinds(t, notices, target, []platform.FileWatchKind{kind}, names...)
}

// watchTestNextKinds is watchTestNext for events whose flags vary: a write
// on APFS carries ItemCreated and ItemModified at once, for example, which
// the mapping calls dirty.
func watchTestNextKinds(t *testing.T, notices chan platform.FileWatchNotice, target uint64, kinds []platform.FileWatchKind, names ...string) {
	t.Helper()
	deadline := time.Now().Add(wait.Guard)
	for time.Now().Before(deadline) {
		select {
		case n := <-notices:
			if n.Target == target && slices.Contains(kinds, n.Kind) && slices.Contains(names, n.Name) {
				return
			}
		case <-time.After(time.Until(deadline)):
		}
	}
	t.Fatalf("no notice of %v for %q on target %d", kinds, names, target)
}

func watchTestAdd(t *testing.T, w platform.FileWatch, id uint64, path string, subtree bool) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Add(platform.FileWatchTarget{ID: id, Path: path, Directory: info.IsDir(), Info: info, Subtree: subtree}); err != nil {
		t.Fatal(err)
	}
}

func watchTestWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// watchTestWritten is what a write is reported as: written for FSEvents'
// plain content events, dirty for the writes APFS also calls created.
var watchTestWritten = []platform.FileWatchKind{platform.FileWatchWrite, platform.FileWatchDirty}

func TestFSEventsWatch(t *testing.T) {
	dir := watchTestDir(t)
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "file")
	watchTestWrite(t, file, "one")
	w, notices := watchTest(t)
	// FSEvents is recursive by nature: a Subtree target must be watched,
	// not refused.
	watchTestAdd(t, w, 1, dir, true)
	// A target without Subtree watches the directory and its direct
	// children, and drops events deeper down.
	other := watchTestDir(t)
	otherFile := filepath.Join(other, "file")
	watchTestWrite(t, otherFile, "one")
	watchTestAdd(t, w, 2, other, false)

	watchTestWrite(t, file, "two, longer")
	watchTestNextKinds(t, notices, 1, watchTestWritten, "sub/file")
	watchTestWrite(t, otherFile, "two, longer")
	watchTestNextKinds(t, notices, 2, watchTestWritten, "file")

	// A directory beneath a watched tree needs no stream of its own: the
	// core enrolls children of a directory it doesn't watch recursively,
	// which the stream above names, and what changes in them is dropped.
	child := filepath.Join(other, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	watchTestNext(t, notices, 2, platform.FileWatchDirty, "child")
	watchTestAdd(t, w, 3, child, false)
	watchTestWrite(t, filepath.Join(child, "inside"), "one")
	watchTestWrite(t, otherFile, "after inside")
	for timeout := time.After(10 * time.Second); ; {
		var n platform.FileWatchNotice
		select {
		case n = <-notices:
		case <-timeout:
			t.Fatal("no notice for file")
		}
		if n.Target == 2 && n.Name == "child" {
			t.Fatalf("change inside a child reported: %+v", n)
		}
		if n.Target == 2 && n.Name == "file" {
			break
		}
	}

	if err := os.WriteFile(filepath.Join(sub, "new"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	watchTestNext(t, notices, 1, platform.FileWatchDirty, "sub/new")
	if err := os.Remove(filepath.Join(sub, "new")); err != nil {
		t.Fatal(err)
	}
	watchTestNext(t, notices, 1, platform.FileWatchDirty, "sub/new")
	if err := os.Mkdir(filepath.Join(sub, "deeper"), 0700); err != nil {
		t.Fatal(err)
	}
	watchTestNext(t, notices, 1, platform.FileWatchDirty, "sub/deeper")
	if err := os.Remove(filepath.Join(sub, "deeper")); err != nil {
		t.Fatal(err)
	}
	watchTestNext(t, notices, 1, platform.FileWatchDirty, "sub/deeper")
	// A rename reports both names.
	if err := os.Rename(file, filepath.Join(sub, "moved")); err != nil {
		t.Fatal(err)
	}
	watchTestNext(t, notices, 1, platform.FileWatchDirty, "sub/file", "sub/moved")
	// Removing a target doesn't panic, and its stream stops.
	w.Remove(1)
	watchTestWrite(t, filepath.Join(sub, "after-remove"), "x")
	watchTestWrite(t, otherFile, "three, longer")
	watchTestNextKinds(t, notices, 2, watchTestWritten, "file")
}

// A file is watched through the directory that holds it, whose stream
// reports it by name: a file target is refused, so that the core enrolls
// no files.
func TestFSEventsWatchFile(t *testing.T) {
	dir := watchTestDir(t)
	file := filepath.Join(dir, "config.json")
	watchTestWrite(t, file, "one")
	w, notices := watchTest(t)
	watchTestAdd(t, w, 1, dir, false)
	info, err := os.Lstat(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Add(platform.FileWatchTarget{ID: 2, Path: file, Info: info}); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("file target: %v", err)
	}
	watchTestWrite(t, file, "two, longer")
	watchTestNextKinds(t, notices, 1, watchTestWritten, "config.json")
}

func TestFSEventsWatchIdentity(t *testing.T) {
	w, _ := watchTest(t)
	dir := filepath.Join(watchTestDir(t), "a")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	err = w.Add(platform.FileWatchTarget{ID: 1, Path: dir, Directory: true, Info: info})
	if err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("Add of a replaced directory: %v", err)
	}
}

// TestFSEventsWatchManyFiles watches a directory with more entries than
// other backends' limit of 8192: one stream covers it all.
func TestFSEventsWatchManyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("creates 10001 files")
	}
	dir := watchTestDir(t)
	start := time.Now()
	for i := range 10001 {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file-%05d", i)), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Logf("created 10001 files in %v", d)
	}
	w, notices := watchTest(t)
	watchTestAdd(t, w, 1, dir, true)
	watchTestWrite(t, filepath.Join(dir, "file-00042"), "written")
	watchTestNextKinds(t, notices, 1, watchTestWritten, "file-00042")
}

// A path given in another case than the disk's, on a case-insensitive
// volume, is watched as the disk names it, which is how FSEvents reports it.
func TestFSEventsWatchCase(t *testing.T) {
	dir := watchTestDir(t)
	if err := os.Mkdir(filepath.Join(dir, "Proj"), 0700); err != nil {
		t.Fatal(err)
	}
	lower := filepath.Join(dir, "proj")
	if _, err := os.Lstat(lower); err != nil {
		t.Skip("the volume is case-sensitive")
	}
	w, notices := watchTest(t)
	watchTestAdd(t, w, 1, lower, true)
	watchTestWrite(t, filepath.Join(dir, "Proj", "file"), "one")
	watchTestNextKinds(t, notices, 1, append(watchTestWritten, platform.FileWatchDirty), "file")
}
