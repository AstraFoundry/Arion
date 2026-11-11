//go:build windows && (amd64 || arm64)

package windows

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

func TestWatchDirectoryParser(t *testing.T) {
	record := make([]byte, 16)
	binary.LittleEndian.PutUint32(record[4:], 1)
	binary.LittleEndian.PutUint32(record[8:], 2)
	binary.LittleEndian.PutUint16(record[12:], 'a')
	got, err := parseDirectoryChanges(record, 42)
	if err != nil || len(got) != 1 || got[0].Name != "a" || got[0].Target != 42 {
		t.Fatalf("%+v %v", got, err)
	}
	nested := make([]byte, 18)
	binary.LittleEndian.PutUint32(nested[4:], 3)
	binary.LittleEndian.PutUint32(nested[8:], 6)
	for i, c := range `a\b` {
		binary.LittleEndian.PutUint16(nested[12+2*i:], uint16(c))
	}
	if got, err := parseDirectoryChanges(nested, 42); err != nil || len(got) != 1 || got[0].Name != "a/b" || got[0].Kind != platform.FileWatchWrite {
		t.Fatalf("%+v %v", got, err)
	}
	for _, mutate := range []func([]byte){
		func(b []byte) { binary.LittleEndian.PutUint32(b[8:], 3) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[8:], 100) },
		func(b []byte) { binary.LittleEndian.PutUint32(b, 3) },
		func(b []byte) { binary.LittleEndian.PutUint16(b[12:], 0xd800) },
		func(b []byte) { binary.LittleEndian.PutUint16(b[12:], 0xdc00) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[4:], 99) },
	} {
		b := append([]byte(nil), record...)
		mutate(b)
		if _, err := parseDirectoryChanges(b, 42); err == nil {
			t.Fatalf("accepted malformed %v", b)
		}
	}
	for i := 0; i < 14; i++ {
		if _, err := parseDirectoryChanges(record[:i], 42); err == nil {
			t.Fatalf("accepted length %d", i)
		}
	}
}

func TestDirectoryWatch(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{`tree\a\b`, `single\c`} {
		if err := os.MkdirAll(filepath.Join(dir, p), 0700); err != nil {
			t.Fatal(err)
		}
	}
	notices := make(chan platform.FileWatchNotice, 64)
	w, err := (&Backend{}).NewFileWatch(func(n platform.FileWatchNotice) { notices <- n })
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	add := func(id uint64, path string, subtree bool) {
		t.Helper()
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Add(platform.FileWatchTarget{ID: id, Path: path, Directory: info.IsDir(), Info: info, Subtree: subtree}); err != nil {
			t.Fatal(err)
		}
	}
	add(1, filepath.Join(dir, "tree"), true)
	add(2, filepath.Join(dir, "single"), false)
	add(3, filepath.Join(dir, "single", "c"), false)
	next := func(want platform.FileWatchNotice) {
		t.Helper()
		timeout := time.After(5 * time.Second)
		for {
			select {
			case n := <-notices:
				if n == want {
					return
				}
				if n.Err != nil {
					t.Fatalf("notice %+v", n)
				}
			case <-timeout:
				t.Fatalf("no notice %+v", want)
			}
		}
	}
	// A subtree names what changes deeper down by its path.
	if err := os.WriteFile(filepath.Join(dir, "tree", "a", "b", "f"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	next(platform.FileWatchNotice{Target: 1, Kind: platform.FileWatchCreate, Name: "a/b/f"})
	// Its directories can still be renamed, which a handle open beneath
	// them would prevent.
	if err := os.Rename(filepath.Join(dir, "tree", "a"), filepath.Join(dir, "tree", "z")); err != nil {
		t.Fatal(err)
	}
	next(platform.FileWatchNotice{Target: 1, Kind: platform.FileWatchMoveTo, Name: "z"})
	// A directory deleted loses its watch.
	if err := os.Remove(filepath.Join(dir, "single", "c")); err != nil {
		t.Fatal(err)
	}
	next(platform.FileWatchNotice{Target: 3, Kind: platform.FileWatchLost})
	next(platform.FileWatchNotice{Target: 2, Kind: platform.FileWatchRemove, Name: "c"})
	// Removing a target closes its handle, on the goroutine of the watch.
	w.Remove(2)
	for start := time.Now(); ; time.Sleep(10 * time.Millisecond) {
		err := os.Rename(filepath.Join(dir, "single"), filepath.Join(dir, "renamed"))
		if err == nil {
			break
		}
		if time.Since(start) > 5*time.Second {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "renamed", "g"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tree", "h"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	next(platform.FileWatchNotice{Target: 1, Kind: platform.FileWatchCreate, Name: "h"})
	// Closing doesn't wait for I/O that already completed: waiting for it,
	// it would give up on the session after 5 s and leak it.
	leaked := func() int {
		leakedSessions.Lock()
		defer leakedSessions.Unlock()
		return len(leakedSessions.sessions)
	}
	before := leaked()
	w.Close()
	if n := leaked() - before; n > 0 {
		t.Fatalf("closing gave up waiting for the I/O of %d sessions", n)
	}
}

// Closing waits for the canceled I/O and leaves nothing behind.
func TestDirectoryWatchClose(t *testing.T) {
	dir := t.TempDir()
	for range 50 {
		w, err := (&Backend{}).NewFileWatch(func(platform.FileWatchNotice) {})
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Lstat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Add(platform.FileWatchTarget{ID: 1, Path: dir, Directory: true, Info: info, Subtree: true}); err != nil {
			t.Fatal(err)
		}
		w.Close()
	}
	if err := os.Rename(dir, dir+"-renamed"); err != nil {
		t.Fatal(err)
	}
	os.Rename(dir+"-renamed", dir)
}

// A junction is not watched in place of the directory it points to.
func TestDirectoryWatchJunction(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Skipf("mklink: %v: %s", err, out)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	w, err := (&Backend{}).NewFileWatch(func(platform.FileWatchNotice) {})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.Add(platform.FileWatchTarget{ID: 1, Path: link, Directory: true, Info: info}); err == nil {
		t.Fatal("watched a junction")
	}
}
