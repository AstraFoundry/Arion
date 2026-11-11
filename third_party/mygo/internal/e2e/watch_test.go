//go:build !js

package e2e

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

func TestFileWatchPluginPages(t *testing.T) {
	if _, err := os.Stat("../../plugins/watch/dist/index.js"); err != nil {
		t.Skip("run bun run build first")
	}
	a := newWindow(t, mygo.WindowOptions{Hidden: true})
	b := newWindow(t, mygo.WindowOptions{Hidden: true})
	start := func(w *mygo.Window) {
		t.Helper()
		if err := w.Page().LoadURL("app://localhost/plugins.html"); err != nil {
			t.Fatal(err)
		}
		waitFor(t, w, "window.plugins && !window.watcher")
		if _, err := w.Page().Eval(`(async()=>{window.watcher=await plugins.watch("project",{debounceMs:-1});window.seen=[];window.consume=(async()=>{for await(const batch of watcher)for(const e of batch)seen.push(e.path)})();return true})()`); err != nil {
			t.Fatal(err)
		}
	}
	start(a)
	start(b) // quota is one per page, not one per window service.
	path := filepath.Join(watchPluginRoot, "page-event")
	t.Cleanup(func() { _ = os.Remove(path) })
	if err := os.WriteFile(path, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	waitFor(t, a, `seen.includes("page-event")`)
	waitFor(t, b, `seen.includes("page-event")`)
	// Navigation cancels the old page's stream and grants a fresh quota generation.
	start(a)
	if _, err := b.Page().Eval(`seen=[]`); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	waitFor(t, a, `seen.includes("page-event")`)
	waitFor(t, b, `seen.includes("page-event")`)
	for _, w := range []*mygo.Window{a, b} {
		if _, err := w.Page().Eval(`watcher.close()`); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFileWatch(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), wait.Guard)
	defer cancel()
	w, err := mygo.WatchFiles(ctx, dir, mygo.FileWatchOptions{Recursive: true, Debounce: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		w.Close()
		select {
		case <-w.Done():
		case <-ctx.Done():
			t.Error("watch cleanup timed out")
		}
	}()
	var pending []mygo.FileEvent
	next := func() mygo.FileEvent {
		t.Helper()
		for len(pending) == 0 {
			batch, err := w.Next(ctx)
			if err != nil {
				t.Fatal(err)
			}
			pending = batch
		}
		e := pending[0]
		pending = pending[1:]
		return e
	}
	wait := func(op mygo.FileOp, path string) {
		t.Helper()
		for {
			if e := next(); e.Op == op && e.Path == path {
				return
			}
		}
	}
	file := filepath.Join(dir, "a")
	if err := os.WriteFile(file, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	wait(mygo.FileCreate, "a")
	if err := os.WriteFile(file, []byte("updated"), 0600); err != nil {
		t.Fatal(err)
	}
	wait(mygo.FileWrite, "a")
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	wait(mygo.FileRemove, "a")
	if err := os.Mkdir(filepath.Join(dir, "child"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "child", "nested"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	wait(mygo.FileCreate, "child/nested")
	// Directories in the tree can be renamed while it is watched, which
	// Windows refuses while a handle is open on a directory beneath.
	if err := os.MkdirAll(filepath.Join(dir, "child", "deeper", "deepest"), 0700); err != nil {
		t.Fatal(err)
	}
	wait(mygo.FileCreate, "child/deeper/deepest")
	if err := os.Rename(filepath.Join(dir, "child"), filepath.Join(dir, "moved")); err != nil {
		t.Fatal(err)
	}
	if e := next(); e != (mygo.FileEvent{Op: mygo.FileRename, OldPath: "child", Path: "moved", IsDir: true}) {
		t.Fatalf("event %+v", e)
	}
	if err := os.WriteFile(filepath.Join(dir, "moved", "deeper", "deepest", "file"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	wait(mygo.FileCreate, "moved/deeper/deepest/file")
	if err := os.RemoveAll(filepath.Join(dir, "moved")); err != nil {
		t.Fatal(err)
	}
	wait(mygo.FileRemove, "moved")
	if err := os.WriteFile(filepath.Join(dir, "last"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	wait(mygo.FileCreate, "last")
	// Permissions alone are not a write.
	if err := os.Chmod(filepath.Join(dir, "last"), 0400); err != nil {
		t.Fatal(err)
	}
	quiet, cancelQuiet := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancelQuiet()
	if batch, err := w.Next(quiet); err == nil {
		t.Fatalf("chmod reported %+v", batch)
	}
}

// A file is watched through its directory: written in place, saved as
// editors do, removed and created again.
func TestFileWatchFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.json")
	if err := os.WriteFile(file, []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait.Guard)
	defer cancel()
	w, err := mygo.WatchFiles(ctx, file, mygo.FileWatchOptions{Debounce: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	next := func(want mygo.FileEvent) {
		t.Helper()
		batch, err := w.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(batch) != 1 || batch[0] != want {
			t.Fatalf("batch %+v, want %+v", batch, want)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(file, "bb")
	next(mygo.FileEvent{Op: mygo.FileWrite, Path: "."})
	write(filepath.Join(dir, "sibling"), "x")
	write(file+".tmp", "ccc")
	if err := os.Rename(file+".tmp", file); err != nil {
		t.Fatal(err)
	}
	next(mygo.FileEvent{Op: mygo.FileWrite, Path: "."})
	// The replacement is watched in its turn.
	write(file, "dddd")
	next(mygo.FileEvent{Op: mygo.FileWrite, Path: "."})
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	next(mygo.FileEvent{Op: mygo.FileRemove, Path: "."})
	write(file, "e")
	next(mygo.FileEvent{Op: mygo.FileCreate, Path: "."})
}

// A file of a watched directory saved atomically, as editors and apps
// save, a temporary file renamed over it, is written, again and again:
// not removed and created, which would have an app close the document.
// The temporary file is one Ignore leaves out, as apps hide theirs.
func TestFileWatchAtomicSave(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "note.md")
	if err := os.WriteFile(file, []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait.Guard)
	defer cancel()
	w, err := mygo.WatchFiles(ctx, dir, mygo.FileWatchOptions{
		Recursive: true,
		Debounce:  10 * time.Millisecond,
		Ignore:    func(p string, isDir bool) bool { return strings.HasPrefix(path.Base(p), ".") },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	save := func(content string) {
		t.Helper()
		tmp := filepath.Join(dir, ".note.md.tmp")
		if err := os.WriteFile(tmp, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(tmp, file); err != nil {
			t.Fatal(err)
		}
	}
	written := func() {
		t.Helper()
		for {
			batch, err := w.Next(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range batch {
				if e.Path != "note.md" {
					continue
				}
				if e.Op != mygo.FileWrite {
					t.Fatalf("a save came as %+v", e)
				}
				return
			}
		}
	}
	for i := range 5 {
		save(strings.Repeat("x", i+2))
		written()
	}
	// The file that replaced it is watched in its turn.
	if err := os.WriteFile(file, []byte("in place"), 0600); err != nil {
		t.Fatal(err)
	}
	written()
}

// TestFileWatchManyFiles watches a directory with more entries than the
// other platforms' default limit of 8192, with default options: macOS
// covers a tree with one FSEvents stream, so its default MaxEntries is
// 65536.
func TestFileWatchManyFiles(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("watching more than 8192 entries is macOS-only")
	}
	dir := t.TempDir()
	for i := range 10001 {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file-%05d", i)), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait.Guard)
	defer cancel()
	w, err := mygo.WatchFiles(ctx, dir, mygo.FileWatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.WriteFile(filepath.Join(dir, "file-00000"), []byte("written"), 0600); err != nil {
		t.Fatal(err)
	}
	for {
		batch, err := w.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range batch {
			if e.Op == mygo.FileWrite && e.Path == "file-00000" {
				return
			}
		}
	}
}
