//go:build !js

// testDocs, which these use, is document_test.go's and that file builds
// everywhere but js, so this follows it.

package mygo

import (
	"bytes"
	"io"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

// An Equal that reads the document it compares for: Set and saving once
// called it holding docBase.mu, which Content takes too, and deadlocked.
func TestDocumentEqualReadsDocument(t *testing.T) {
	writing, release := make(chan struct{}, 1), make(chan struct{})
	var block bool
	var doc *Document[string]
	ds := testDocs(t, DocumentsOptions[string]{
		Autosave: AutosaveOff,
		Equal:    func(a, b string) bool { _ = doc.Content(); return a == b },
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
	doc, _ = ds.Open(path)
	returns(t, "Set", func() { doc.Set("two") })
	if !doc.IsEdited() {
		t.Error("content different from the file's did not edit the document")
	}
	returns(t, "Set", func() { doc.Set("one") })
	if doc.IsEdited() {
		t.Error("content equal to the file's left the document edited")
	}

	// Edited while it is written, back to what was written: saving asks
	// Equal, which reads the document, and leaves it unedited.
	doc.Set("three")
	block = true
	done := make(chan error, 1)
	go func() { done <- doc.Save() }()
	<-writing
	doc.Set("four")
	doc.Set("three")
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if doc.IsEdited() {
		t.Error("saved what Equal finds the content to be, and still edited")
	}
}

func TestDocumentSetDiscardsStaleEqual(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	ds := testDocs(t, DocumentsOptions[string]{
		Autosave: AutosaveOff,
		Equal: func(a, b string) bool {
			if a == "one" && b == "one" {
				close(entered)
				<-release
			}
			return a == b
		},
	})
	path := writeTemp(t, "stale-equal.txt", "one")
	doc, err := ds.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		doc.Set("one")
		close(done)
	}()
	<-entered
	doc.Set("two")
	close(release)
	<-done

	if got := doc.Content(); got != "two" {
		t.Fatalf("content = %q, want two", got)
	}
	if !doc.IsEdited() {
		t.Fatal("the newer edit was marked saved by an older Equal result")
	}
}

// Equal is for contents that == can't compare, as []byte.
func TestDocumentEqualUncomparable(t *testing.T) {
	ds := &Documents[[]byte]{opts: DocumentsOptions[[]byte]{Equal: bytes.Equal}}
	d := ds.newDocument([]byte("a"), "x")
	d.mu.Lock()
	d.content, d.changes = []byte("b"), 2
	settle := d.markSaved(docSnapshot{at: 1, value: []byte("b")})
	d.savedAt = 1
	d.mu.Unlock()
	if settle == nil {
		t.Fatal("edited while written, and Equal not asked")
	}
	settle()
	if d.edited {
		t.Error("content Equal to what was written left the document edited")
	}
}

// returns runs f, which fails the test should it never return.
func returns(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); f() }()
	wait.Until(t, what+" to return", func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})
}
