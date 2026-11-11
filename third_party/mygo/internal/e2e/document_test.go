//go:build !js

package e2e

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// notes are the documents of the tests: plain text, asking to save when
// they close, as on Windows and Linux.
var notes = sync.OnceValue(func() *mygo.Documents[string] {
	return mygo.NewDocuments(mygo.DocumentsOptions[string]{
		Types:      []mygo.DocumentType{{Name: "E2E Note", Extensions: []string{"e2enote"}}},
		Autosave:   mygo.AutosaveOff,
		NoUntitled: true,
		Read: func(r io.Reader) (string, error) {
			b, err := io.ReadAll(r)
			return string(b), err
		},
		Write: func(w io.Writer, s string) error {
			_, err := io.WriteString(w, s)
			return err
		},
		Window: func(*mygo.Document[string]) *mygo.Window {
			return mygo.NewWindow(mygo.WindowOptions{Width: 420, Height: 260})
		},
	})
})

// TestDocumentWindow opens a document and checks what its window shows
// of it: its name in the title, its file's icon in the title bar and the
// dot of an edited document on macOS, a "*" before the title elsewhere.
// Saving from the File menu writes it, and closing it edited asks first
// (macOS, whose sheet the test answers).
func TestDocumentWindow(t *testing.T) {
	mygo.App.SetPath(mygo.PathUserData, t.TempDir())
	defer mygo.App.SetPath(mygo.PathUserData, "")
	docs := notes()
	old := mygo.App.Menu()
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleAppMenu}, {Role: mygo.RoleFileMenu}}))
	defer mygo.App.SetMenu(old)

	dir, _ := filepath.EvalSymlinks(t.TempDir())
	path := filepath.Join(dir, "chrome.e2enote")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := docs.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	w := doc.Window()
	t.Cleanup(func() {
		doc.SetEdited(false)
		if !w.IsDestroyed() {
			w.Destroy()
		}
	})
	want := "chrome.e2enote"
	if runtime.GOOS != "darwin" {
		want += " - " + mygo.App.Name()
	}
	if _, _, _, _, ok := documentChrome(w); !ok {
		t.Skip("no window title to read on " + runtime.GOOS)
	}
	eventually(t, "the document's title", func() bool {
		title, file, icon, _, _ := documentChrome(w)
		return title == want && (runtime.GOOS != "darwin" || file == path && icon)
	})

	doc.Set("two")
	eventually(t, "the edited mark", func() bool {
		title, _, _, edited, _ := documentChrome(w)
		if runtime.GOOS == "darwin" {
			return edited && title == want
		}
		return title == "*"+want
	})
	if w.Title() != want {
		t.Errorf("Title() = %q, want it without the mark", w.Title())
	}

	if err := activateMenu(w, "File", "Save"); err != nil {
		t.Fatalf("File > Save: %v", err)
	}
	eventually(t, "the document to be saved", func() bool {
		b, _ := os.ReadFile(path)
		return string(b) == "two" && !doc.IsEdited()
	})
	eventually(t, "the edited mark to go", func() bool {
		title, _, _, edited, _ := documentChrome(w)
		return !edited && title == want
	})

	if _, _, _, supported := sheet(w); !supported {
		return
	}
	// Revert asks, then reads the file again.
	doc.Set("unwanted")
	eventually(t, "the edited mark", func() bool { _, _, _, edited, _ := documentChrome(w); return edited })
	if err := activateMenu(w, "File", "Revert to Saved"); err != nil {
		t.Fatalf("File > Revert to Saved: %v", err)
	}
	eventually(t, "the sheet asking to revert", func() bool {
		texts, buttons, ok, _ := sheet(w)
		return ok && slices.Contains(texts, "Do you want to revert “chrome.e2enote” to the last saved version?") && len(buttons) == 2
	})
	endSheetButton(w, 0)
	eventually(t, "the document to revert", func() bool { return doc.Content() == "two" && !doc.IsEdited() })

	// Closing it edited asks first, in a sheet; Cancel keeps it open.
	doc.Set("three")
	w.Close()
	var texts, buttons []string
	eventually(t, "the sheet asking to save", func() bool {
		var ok bool
		texts, buttons, ok, _ = sheet(w)
		return ok && len(buttons) == 3
	})
	if !slices.Contains(texts, "Do you want to save the changes made to the document “chrome.e2enote”?") ||
		!slices.Equal(buttons, []string{"Save", "Cancel", "Don’t Save"}) {
		t.Errorf("the sheet says %q with %q", texts, buttons)
	}
	if !endSheetButton(w, slices.Index(buttons, "Cancel")) {
		t.Fatal("no sheet to end")
	}
	eventually(t, "the sheet to close", func() bool { _, _, ok, _ := sheet(w); return !ok })
	if w.IsDestroyed() {
		t.Fatal("Cancel closed the window")
	}
	// Save saves it, then closes it.
	w.Close()
	eventually(t, "the sheet asking again", func() bool { _, b, ok, _ := sheet(w); return ok && len(b) == 3 })
	endSheetButton(w, 0)
	eventually(t, "the window to close", w.IsDestroyed)
	if b, _ := os.ReadFile(path); string(b) != "three" {
		t.Errorf("the file has %q after Save", b)
	}
	if !slices.ContainsFunc(mygo.App.RecentDocuments(), func(p string) bool { return strings.HasSuffix(p, "chrome.e2enote") }) {
		t.Errorf("recent documents %q", mygo.App.RecentDocuments())
	}
}

// TestDocumentOtherApps checks that a document follows what other apps do
// to its file, through the watch of the file: a save as editors do it, a
// rename and a removal.
func TestDocumentOtherApps(t *testing.T) {
	mygo.App.SetPath(mygo.PathUserData, t.TempDir())
	defer mygo.App.SetPath(mygo.PathUserData, "")
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	path := filepath.Join(dir, "other.e2enote")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := notes().Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if w := doc.Window(); w != nil && !w.IsDestroyed() {
			w.Destroy()
		}
	})
	if err := os.WriteFile(path+".tmp", []byte("two, longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := renameAsApps(path+".tmp", path); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the document to read the new file", func() bool { return doc.Content() == "two, longer" })
	renamed := filepath.Join(dir, "renamed.e2enote")
	if err := renameAsApps(path, renamed); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the document to follow the rename", func() bool { return doc.Path() == renamed })
	if err := os.WriteFile(renamed, []byte("three"), 0o644); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the document to read the renamed file", func() bool { return doc.Content() == "three" })
	if err := os.Remove(renamed); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the document to be deleted", func() bool { return doc.State().Deleted })
}

// renameAsApps renames as the apps saving a document do on Windows: they
// try again while the system refuses, which it does as long as any
// program holds either file open without sharing its deletion, as a
// virus scanner reading a file just written, or the shell a recent
// document. The guard only stops a rename that never succeeds.
func renameAsApps(from, to string) error {
	deadline := time.Now().Add(time.Minute)
	for {
		err := os.Rename(from, to)
		if err == nil || runtime.GOOS != "windows" || time.Now().After(deadline) ||
			// ERROR_ACCESS_DENIED, ERROR_SHARING_VIOLATION
			!errors.Is(err, syscall.Errno(5)) && !errors.Is(err, syscall.Errno(32)) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
