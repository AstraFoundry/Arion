//go:build !js

package mygo

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/internal/wait"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
)

// testDocs makes documents of plain text, .txt, in their own user data,
// and closes their windows when the test ends.
func testDocs(t *testing.T, opts DocumentsOptions[string]) *Documents[string] {
	t.Helper()
	App.SetPath(PathUserData, t.TempDir())
	t.Cleanup(func() { App.SetPath(PathUserData, "") })
	if opts.Types == nil {
		opts.Types = []DocumentType{{Name: "Text " + t.Name(), Extensions: []string{"txt"}}}
	}
	opts.NoUntitled = true
	opts.Read = func(r io.Reader) (string, error) { b, err := io.ReadAll(r); return string(b), err }
	if opts.Write == nil {
		opts.Write = func(w io.Writer, s string) error { _, err := io.WriteString(w, s); return err }
	}
	opts.Window = func(doc *Document[string]) *Window { return NewWindow(WindowOptions{}) }
	ds := NewDocuments(opts)
	t.Cleanup(func() {
		for _, d := range ds.Documents() {
			d.SetEdited(false)
			if w := d.Window(); w != nil {
				w.Destroy()
			}
		}
		docWork.Wait()
		// Windows deletes no file a process holds open: the session's lock.
		ds.kind.recovery.close()
		onMain(func() { fb.OnMessageBox, fb.OnSaveDialog, fb.OnOpenDialog = nil, nil, nil })
	})
	return ds
}

// fakeWindow returns the fake native window of w.
func fakeWindow(w *Window) *fake.Window {
	return onMainValue(func() *fake.Window { return w.native.(*fake.Window) })
}

// waitFor waits until cond holds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	wait.Until(t, what, cond)
}

// asked records the messages of the message boxes shown.
type asked struct {
	mu   sync.Mutex
	list []string
}

func (a *asked) all() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.list)
}

// answer makes message boxes answer with the button labeled label, and
// records their messages.
func answer(label string) *asked {
	a := &asked{}
	onMain(func() {
		fb.OnMessageBox = func(o *platform.MessageBoxOptions) platform.MessageBoxResult {
			a.mu.Lock()
			a.list = append(a.list, o.Message)
			a.mu.Unlock()
			return platform.MessageBoxResult{Response: slices.Index(o.Buttons, label)}
		}
	})
	return a
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func readString(t *testing.T, path string) string {
	t.Helper()
	b, err := readAppFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// fileIs reports whether the file at path holds want.
func fileIs(path, want string) bool {
	b, err := readAppFile(path)
	return err == nil && string(b) == want
}

func TestDocumentOpen(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff, Equal: func(a, b string) bool { return a == b }})
	path := writeTemp(t, "notes.txt", "hello")
	d, err := ds.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if d.Content() != "hello" || d.Path() != path || d.Name() != "notes.txt" || d.IsEdited() {
		t.Fatalf("opened %q %q %q edited=%v", d.Content(), d.Path(), d.Name(), d.IsEdited())
	}
	w := d.Window()
	if ds.ForWindow(w) != d {
		t.Error("ForWindow")
	}
	fw := fakeWindow(w)
	want := "notes.txt"
	if runtime.GOOS != "darwin" {
		want += " - MyGoTest"
	}
	waitFor(t, "the window's title", func() bool { p, _ := fw.Document(); return w.Title() == want && p == path })

	// Opening it again focuses it.
	again, err := ds.Open(path)
	if err != nil || again != d || len(ds.Documents()) != 1 {
		t.Fatalf("opened again: %v, %d documents", err, len(ds.Documents()))
	}

	d.Set("hello!")
	waitFor(t, "the edited dot", func() bool { _, e := fw.Document(); return e })
	if !d.IsEdited() || !w.IsDocumentEdited() {
		t.Fatal("not edited")
	}
	if runtime.GOOS != "darwin" {
		if got := fw.Title(); got != "*"+want {
			t.Errorf("native title %q, want the mark", got)
		}
		if w.Title() != want {
			t.Errorf("Title() = %q, want it without the mark", w.Title())
		}
	}
	// Equal: back to what the file has is unedited.
	d.Set("hello")
	waitFor(t, "the dot to go", func() bool { _, e := fw.Document(); return !e })
	if d.IsEdited() {
		t.Error("edited after setting the saved content again")
	}
	if !slices.Contains(App.RecentDocuments(), path) {
		t.Errorf("recent documents %q lack %s", App.RecentDocuments(), path)
	}
}

func TestDocumentSave(t *testing.T) {
	release := make(chan struct{})
	writing := make(chan struct{}, 1)
	var block bool
	ds := testDocs(t, DocumentsOptions[string]{
		Autosave: AutosaveOff,
		Write: func(w io.Writer, s string) error {
			if block {
				writing <- struct{}{}
				<-release
			}
			_, err := io.WriteString(w, s)
			return err
		},
	})
	path := writeTemp(t, "a.txt", "one")
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	d, _ := ds.Open(path)
	d.Set("two")
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	if readString(t, path) != "two" || d.IsEdited() {
		t.Fatalf("saved %q, edited %v", readString(t, path), d.IsEdited())
	}
	if info, _ := os.Stat(path); runtime.GOOS != "windows" && info.Mode().Perm() != 0o640 {
		t.Errorf("permissions %v after saving", info.Mode().Perm())
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Errorf("files left next to the document: %v", entries)
	}

	// Edits made while it is written stay unsaved.
	block = true
	d.Set("three")
	done := make(chan error)
	go func() { done <- d.Save() }()
	<-writing
	d.Set("four")
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if readString(t, path) != "three" || !d.IsEdited() {
		t.Fatalf("file %q, edited %v: the edit made while writing was lost", readString(t, path), d.IsEdited())
	}
}

// A system that removes the file and can't put the new one in its place
// (ReplaceFileW's ERROR_UNABLE_TO_MOVE_REPLACEMENT) leaves the new contents
// only in the sibling: the save keeps it and names it in its error. When the
// file is still there, the sibling goes as before.
func TestDocumentSaveKeepsTheNewContentsWhenReplacingFails(t *testing.T) {
	for _, removed := range []bool{true, false} {
		t.Run(fmt.Sprintf("removed=%v", removed), func(t *testing.T) {
			ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
			path := writeTemp(t, "a.txt", "one")
			d, _ := ds.Open(path)
			d.Set("two")
			onMain(func() {
				fb.ReplaceFileHook = func(target, replacement string) error {
					if removed {
						if err := os.Remove(target); err != nil {
							return err
						}
					}
					return errors.New("cannot move the replacement")
				}
			})
			err := d.Save()
			onMain(func() { fb.ReplaceFileHook = nil })
			entries, _ := os.ReadDir(filepath.Dir(path))
			if err == nil || len(entries) != 1 {
				t.Fatalf("save error %v, files %v", err, entries)
			}
			left := filepath.Join(filepath.Dir(path), entries[0].Name())
			// The error names the sibling by the file's real path, which on
			// Windows may spell a temporary directory's 8.3 name in full.
			if removed && (readString(t, left) != "two" || !strings.Contains(err.Error(), entries[0].Name())) {
				t.Errorf("kept %s with %q; error %v", left, readString(t, left), err)
			}
			if !removed && (left != path || readString(t, path) != "one") {
				t.Errorf("left %s with %q; error %v", left, readString(t, left), err)
			}
		})
	}
}

// A sibling that could not be written goes, even when the file went
// meanwhile: it holds no new contents to keep.
func TestDocumentSaveDropsASiblingNotWritten(t *testing.T) {
	var path string
	ds := testDocs(t, DocumentsOptions[string]{
		Autosave: AutosaveOff,
		Write: func(w io.Writer, s string) error {
			if err := os.Remove(path); err != nil {
				return err
			}
			return errors.New("cannot write")
		},
	})
	path = writeTemp(t, "a.txt", "one")
	d, _ := ds.Open(path)
	d.Set("two")
	err := d.Save()
	entries, _ := os.ReadDir(filepath.Dir(path))
	if err == nil || len(entries) != 0 {
		t.Fatalf("save error %v, files %v", err, entries)
	}
}

func TestDocumentSaveKeepsLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hard links and symlinks need privileges")
	}
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	dir := t.TempDir()
	target := writeTemp(t, "target.txt", "x")
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	hard := filepath.Join(dir, "hard.txt")
	if err := os.Link(target, hard); err != nil {
		t.Fatal(err)
	}
	d, _ := ds.Open(link)
	d.Set("y")
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink was replaced by a file")
	}
	if readString(t, target) != "y" || readString(t, hard) != "y" {
		t.Errorf("target %q, hard link %q", readString(t, target), readString(t, hard))
	}
}

func TestDocumentAutosave(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{AutosaveDelay: 20 * time.Millisecond})
	path := writeTemp(t, "auto.txt", "a")
	d, _ := ds.Open(path)
	d.Set("b")
	waitFor(t, "the autosave", func() bool { return fileIs(path, "b") && !d.IsEdited() })

	// A saved document closes without asking, saving first.
	asked := answer("Cancel")
	d.Set("c")
	d.Close()
	waitFor(t, "the window to close", func() bool { return d.Window() == nil })
	if readString(t, path) != "c" || len(asked.all()) != 0 {
		t.Errorf("file %q, asked %q", readString(t, path), asked.all())
	}
}

func TestDocumentUntitledClose(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{})
	d1, d2 := ds.New(), ds.New()
	if d1.Name() != "Untitled" || d2.Name() != "Untitled 2" || d1.Path() != "" {
		t.Fatalf("names %q %q", d1.Name(), d2.Name())
	}
	d1.Set("draft")
	asked := answer("Cancel")
	d1.Close()
	waitFor(t, "the question", func() bool { return len(asked.all()) == 1 })
	if asked.all()[0] != "Do you want to keep this new document “Untitled”?" {
		t.Errorf("asked %q", asked.all()[0])
	}
	time.Sleep(20 * time.Millisecond)
	if d1.Window() == nil {
		t.Fatal("Cancel closed the window")
	}
	answer("Delete")
	d1.Close()
	waitFor(t, "the window to close", func() bool { return d1.Window() == nil })
	// An unedited one closes at once.
	d2.Close()
	waitFor(t, "the other window to close", func() bool { return d2.Window() == nil })
}

func TestDocumentCloseAsks(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "ask.txt", "1")
	d, _ := ds.Open(path)
	d.Set("2")
	asked := answer("Save")
	d.Close()
	waitFor(t, "the window to close", func() bool { return d.Window() == nil })
	if readString(t, path) != "2" || len(asked.all()) != 1 || !strings.Contains(asked.all()[0], "“ask.txt”") {
		t.Errorf("file %q, asked %q", readString(t, path), asked.all())
	}

	// Saving an untitled document asks where.
	u := ds.New()
	u.Set("new")
	target := filepath.Join(t.TempDir(), "named")
	onMain(func() { fb.OnSaveDialog = func(*platform.SaveDialogOptions) string { return target } })
	answer("Save…")
	u.Close()
	waitFor(t, "the untitled window to close", func() bool { return u.Window() == nil })
	if readString(t, target+".txt") != "new" {
		t.Error("the untitled document was not saved with its extension")
	}
}

func TestDocumentOutsideChanges(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "out.txt", "mine")
	d, _ := ds.Open(path)
	reloaded := make(chan struct{}, 1)
	d.OnReload(func() { reloaded <- struct{}{} })

	// Unedited: read again.
	future := time.Now().Add(time.Hour)
	os.WriteFile(path, []byte("theirs"), 0o644)
	os.Chtimes(path, future, future)
	d.checkDisk()
	<-reloaded
	if d.Content() != "theirs" || d.IsEdited() {
		t.Fatalf("content %q edited %v", d.Content(), d.IsEdited())
	}

	// Edited: in conflict, which saving asks about.
	d.Set("mine again")
	os.WriteFile(path, []byte("theirs again"), 0o644)
	os.Chtimes(path, future.Add(time.Hour), future.Add(time.Hour))
	d.checkDisk()
	if !d.HasConflict() || d.Content() != "mine again" {
		t.Fatalf("conflict %v, content %q", d.HasConflict(), d.Content())
	}
	asked := answer("Save Anyway")
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	if readString(t, path) != "mine again" || d.HasConflict() || len(asked.all()) != 1 {
		t.Errorf("file %q, conflict %v, asked %q", readString(t, path), d.HasConflict(), asked.all())
	}

	// Without asking, Save reports the change.
	os.Chtimes(path, future.Add(2*time.Hour), future.Add(2*time.Hour))
	d.Set("x")
	if _, err := d.write(path, writeSave); !errors.Is(err, ErrDocumentChanged) {
		t.Errorf("write over a changed file: %v", err)
	}

	// Deleted: saving writes it again.
	os.Remove(path)
	d.checkDisk()
	if !d.State().Deleted {
		t.Fatal("not deleted")
	}
	if err := d.Save(); err != nil || readString(t, path) != "x" {
		t.Errorf("saving a deleted document: %v", err)
	}
}

func TestDocumentRecovery(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	d := ds.New()
	d.Set("unsaved")
	root, _ := ds.kind.recovery.root()
	// Backed up once the copy and the index are written.
	waitFor(t, "the recovery copy", func() bool { return d.backedUpAt() != 0 })
	m, _ := filepath.Glob(filepath.Join(root, "*", d.id))
	if len(m) != 1 || readString(t, m[0]) != "unsaved" {
		t.Fatalf("recovery copies %q", m)
	}
	copyPath := m[0]
	var entries []recoveryEntry
	if err := json.Unmarshal([]byte(readString(t, filepath.Join(filepath.Dir(copyPath), "index.json"))), &entries); err != nil || len(entries) != 1 || entries[0].Name != "Untitled" {
		t.Fatalf("index %v: %v", entries, err)
	}
	// Discarded: gone.
	answer("Don’t Save")
	d.Close()
	waitFor(t, "the copy to go", func() bool { _, err := os.Stat(copyPath); return os.IsNotExist(err) })

	// A session no instance holds, left by a crash, comes back edited, and
	// its file, changed meanwhile, in conflict.
	path := writeTemp(t, "crashed.txt", "on disk")
	orphan := filepath.Join(root, "orphan")
	os.MkdirAll(orphan, 0o700)
	stamp, _, _ := statStamp(path)
	stamp.ModTime--
	index, _ := json.Marshal([]recoveryEntry{{ID: "a", Name: "Untitled 3"}, {ID: "b", Path: path, Stamp: stamp}})
	os.WriteFile(filepath.Join(orphan, "index.json"), index, 0o600)
	os.WriteFile(filepath.Join(orphan, "a"), []byte("draft"), 0o600)
	os.WriteFile(filepath.Join(orphan, "b"), []byte("changes"), 0o600)
	if n := ds.kind.restore(); n != 2 {
		t.Fatalf("restored %d documents", n)
	}
	var untitled, file *Document[string]
	for _, d := range ds.Documents() {
		if d.Path() == path {
			file = d
		} else if d.Name() == "Untitled 3" {
			untitled = d
		}
	}
	if untitled == nil || file == nil {
		t.Fatalf("restored %v", ds.Documents())
	}
	if untitled.Content() != "draft" || !untitled.IsEdited() || file.Content() != "changes" || !file.IsEdited() || !file.HasConflict() {
		t.Errorf("restored %q %v, %q %v conflict %v", untitled.Content(), untitled.IsEdited(), file.Content(), file.IsEdited(), file.HasConflict())
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Error("the claimed session is still there")
	}
}

func TestDocumentQuit(t *testing.T) {
	quits := make(chan struct{}, 1)
	quitAgain = func() {
		docQuit.resolved = false
		quits <- struct{}{}
	}
	t.Cleanup(func() { quitAgain = func() { App.Quit() } })

	// Kept for the next launch, without asking.
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	d := ds.New()
	d.Set("keep me")
	asked := answer("Cancel")
	if onMainValue(fb.QuitRequested) {
		t.Fatal("quit with an unsaved document not kept yet")
	}
	<-quits
	if len(asked.all()) != 0 || d.backedUpAt() == 0 {
		t.Errorf("asked %q, backed up %v", asked.all(), d.backedUpAt())
	}

	// Asked about.
	ask := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff, AskOnQuit: true})
	a := ask.New()
	a.Set("ask")
	asked = answer("Cancel")
	if onMainValue(fb.QuitRequested) {
		t.Fatal("quit without asking")
	}
	waitFor(t, "the question", func() bool { return len(asked.all()) == 1 })
	select {
	case <-quits:
		t.Fatal("quit after Cancel")
	case <-time.After(20 * time.Millisecond):
	}
	answer("Don’t Save")
	onMainValue(fb.QuitRequested)
	<-quits
}

func (b *docBase) backedUpAt() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.backedUp
}

func TestDocumentRenameMove(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "old.txt", "x")
	d, _ := ds.Open(path)
	if err := d.Rename("new.txt"); err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(filepath.Dir(path), "new.txt")
	if d.Path() != renamed || readString(t, renamed) != "x" {
		t.Fatalf("renamed to %s", d.Path())
	}
	dir := t.TempDir()
	if err := d.MoveTo(dir); err != nil {
		t.Fatal(err)
	}
	if d.Path() != filepath.Join(dir, "new.txt") {
		t.Fatalf("moved to %s", d.Path())
	}
	fw := fakeWindow(d.Window())
	waitFor(t, "the represented file to follow", func() bool { p, _ := fw.Document(); return p == d.Path() })

	dup := d.Duplicate()
	if dup.Name() != "new copy" || dup.Content() != "x" || !dup.IsEdited() || dup.Path() != "" {
		t.Errorf("duplicate %q %q %v", dup.Name(), dup.Content(), dup.IsEdited())
	}
}

// TestDocumentFollowsReplacedRename renames a file that another app
// replaced, after the document read the new file but before its watch
// looked again: the watch knows only the file replaced, so the document
// finds the new one by what it read.
func TestDocumentFollowsReplacedRename(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "doc.txt", "one")
	path, _ = filepath.EvalSymlinks(path)
	d, _ := ds.Open(path)
	// The watch saw the first file once it enrolled the file itself.
	var watch *fake.FileWatch
	waitFor(t, "the watch to see the file", func() bool {
		watch = fb.LastFileWatch()
		return slices.ContainsFunc(watch.Targets(), func(t platform.FileWatchTarget) bool { return t.Path == path && !t.Directory })
	})
	if err := os.WriteFile(path+".tmp", []byte("two, longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		t.Fatal(err)
	}
	// Read by the look when the watch started or the window was focused.
	d.checkDisk()
	waitFor(t, "the document to read the new file", func() bool { return d.Content() == "two, longer" })
	renamed := filepath.Join(filepath.Dir(path), "renamed.txt")
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	for _, target := range watch.Targets() {
		if target.Path == filepath.Dir(path) {
			watch.Notify(platform.FileWatchNotice{Target: target.ID, Name: "doc.txt", Kind: platform.FileWatchRemove})
		}
	}
	// Following it records it as recent, in the user data the test removes.
	waitFor(t, "the document to follow the rename", func() bool {
		return d.Path() == renamed && slices.Contains(App.RecentDocuments(), renamed)
	})
	if d.State().Deleted {
		t.Error("deleted")
	}
}

// TestDocumentWatchStartFollowsRename renames the file before its watch
// starts, which then can't: the document follows it all the same.
func TestDocumentWatchStartFollowsRename(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "start.txt", "one")
	path, _ = filepath.EvalSymlinks(path)
	d, _ := ds.Open(path)
	renamed := filepath.Join(filepath.Dir(path), "started.txt")
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	watchDocumentFile(context.Background(), path)
	// The recent documents too: the move writes them in the user data
	// directory after the path, which the test's cleanup removes.
	waitFor(t, "the document to follow the rename", func() bool {
		return d.Path() == renamed && slices.Contains(App.RecentDocuments(), renamed)
	})
	if d.State().Deleted {
		t.Error("the document is deleted")
	}
}

// TestDocumentLookFollowsRename looks at a document's file after another
// app renamed it and before the watch did, as a window focused then does:
// the look follows the file by what the document read rather than taking
// it for deleted, which the watch would then never follow. It is the order
// in which TestDocumentFollowsReplacedRename's look when the watch started
// sometimes ran.
func TestDocumentLookFollowsRename(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	path := writeTemp(t, "look.txt", "one")
	path, _ = filepath.EvalSymlinks(path)
	d, _ := ds.Open(path)
	renamed := filepath.Join(filepath.Dir(path), "looked.txt")
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	// The watch is never told: only a look can follow it.
	d.checkDisk()
	waitFor(t, "the look to follow the rename", func() bool {
		return d.Path() == renamed && slices.Contains(App.RecentDocuments(), renamed)
	})
	if d.State().Deleted {
		t.Error("deleted")
	}
}

func TestDocumentMenus(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	file := NewMenu([]*MenuItem{{Role: RoleFileMenu}}).Items()[0]
	var roles []MenuRole
	for _, it := range file.Submenu {
		roles = append(roles, it.Role)
	}
	for _, r := range []MenuRole{RoleNew, RoleOpen, RoleOpenRecent, RoleSave, RoleRevert, RoleClose} {
		if !slices.Contains(roles, r) {
			t.Errorf("the File menu lacks %s: %v", r, roles)
		}
	}
	path := writeTemp(t, "menu.txt", "1")
	d, _ := ds.Open(path)
	d.Set("2")
	documentRole(RoleSave, d.Window())
	waitFor(t, "Save", func() bool { return fileIs(path, "2") })
	n := len(allDocuments())
	documentRole(RoleNew, nil)
	waitFor(t, "New", func() bool { return len(allDocuments()) == n+1 })
	for _, b := range allDocuments() {
		if b.kind != ds.kind && b.parent() != nil {
			b.parent().Destroy() // the first kind's
		}
	}

	recent := NewMenu([]*MenuItem{{Role: RoleOpenRecent}}).Items()[0]
	if len(recent.Submenu) < 3 || recent.Submenu[0].Label != "menu.txt" {
		t.Errorf("Open Recent: %v", recent.Submenu)
	}
}

var serviceRuns int

func TestDocumentService(t *testing.T) {
	ds := testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
	serviceRuns++
	note := fmt.Sprintf("Note%d", serviceRuns)
	if err := bind(note, ds.Service(), false); err != nil {
		t.Fatal(err)
	}
	path := writeTemp(t, "page.txt", "from Go")
	d, _ := ds.Open(path)
	fw := fakeWindow(d.Window())
	onMain(func() { fw.H.NavigationCommitted("about:blank") })
	page(fw, `{"t":"dom-ready"}`)
	if r := call(t, fw, 1, note+".Content"); r["v"] != "from Go" {
		t.Fatalf("content: %v", r)
	}
	call(t, fw, 2, note+".SetContent", "from the page")
	if d.Content() != "from the page" || !d.IsEdited() {
		t.Fatalf("content %q edited %v", d.Content(), d.IsEdited())
	}
	received(t, fw, func(m map[string]any) bool {
		p, _ := m["p"].(map[string]any)
		return m["n"] == documentStateEvent && p["edited"] == true && p["name"] == "page.txt"
	})
	if r := call(t, fw, 3, note+".State"); r["v"].(map[string]any)["path"] != path {
		t.Errorf("state: %v", r)
	}
}

func TestUntitledNumber(t *testing.T) {
	for name, want := range map[string]int{
		"Untitled": 1, "Untitled 2": 2, "Untitled 12": 12, "Untitled x": 0, "Notes": 0, "Untitled 1": 0,
	} {
		if got := untitledNumber(name); got != want {
			t.Errorf("untitledNumber(%q) = %d, want %d", name, got, want)
		}
	}
	// A language whose digits are others: the number reads back.
	if got := l10n.In("fa").UntitledNumbered(12); got != "Untitled ۱۲" || untitledNumberIn(got, "fa", "en") != 12 {
		t.Errorf("Persian: %q", got)
	}
	if n, ok := decimalValue("٣٤"); !ok || n != 34 {
		t.Errorf("Arabic-Indic 34: %d %v", n, ok)
	}
	if got := untitledName(3); got != "Untitled 3" || untitledNumber(got) != 3 {
		t.Errorf("untitledName(3) = %q", got)
	}
}
