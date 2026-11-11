package editor

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

const testPatch = `diff --git a/greet.go b/greet.go
index 3b18e51..a2e8f0c 100644
--- a/greet.go
+++ b/greet.go
@@ -1,5 +1,6 @@ package main
 package main

-func greet() string {
-	return "hello"
+// greet greets.
+func greet(name string) string {
+	return "hello " + name
 }
diff --git a/old.txt b/new.txt
similarity 90%
rename from old.txt
rename to new.txt
--- a/old.txt
+++ b/new.txt
@@ -1 +1 @@
-a
\ No newline at end of file
+b
diff --git a/added.md b/added.md
new file mode 100644
--- /dev/null
+++ b/added.md
@@ -0,0 +1,2 @@
+# Title
+text
`

func TestParsePatch(t *testing.T) {
	files, err := ParsePatch(testPatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("%d files", len(files))
	}
	f := files[0]
	if f.OldName != "greet.go" || f.NewName != "greet.go" || len(f.Hunks) != 1 {
		t.Fatalf("first file %+v", f)
	}
	h := f.Hunks[0]
	if h.OldStart != 1 || h.OldLines != 5 || h.NewStart != 1 || h.NewLines != 6 || h.Section != "package main" || len(h.Lines) != 8 {
		t.Fatalf("hunk %+v", h)
	}
	old := "package main\n\nfunc greet() string {\n\treturn \"hello\"\n}\n"
	got, err := f.Apply(old)
	if err != nil {
		t.Fatal(err)
	}
	want := "package main\n\n// greet greets.\nfunc greet(name string) string {\n\treturn \"hello \" + name\n}\n"
	if got != want {
		t.Fatalf("applied %q", got)
	}
	r := files[1]
	if r.OldName != "old.txt" || r.NewName != "new.txt" {
		t.Fatalf("rename %+v", r)
	}
	if got, err := r.Apply("a"); err != nil || got != "b\n" {
		t.Fatalf("applied rename %q, %v", got, err)
	}
	n := files[2]
	if n.OldName != "" || n.NewName != "added.md" || n.NewMode != "100644" {
		t.Fatalf("new file %+v", n)
	}
	if got, err := n.Apply(""); err != nil || got != "# Title\ntext\n" {
		t.Fatalf("applied new file %q, %v", got, err)
	}
	if _, err := f.Apply("something else\n"); err == nil {
		t.Fatal("applied to the wrong text")
	}
}

// A patch shows without the files, numbered as they are.
func TestDiffViewerPatch(t *testing.T) {
	files, _ := ParsePatch(`--- a/x.go
+++ b/x.go
@@ -10,3 +10,3 @@ func f() {
 	a()
-	b()
+	c()
 	d()
@@ -40,2 +40,3 @@
 	e()
+	f()
 }
`)
	d := &DiffState{}
	d.SetPatch(files[0])
	tt := ui.NewTester(func(c *ui.Context) {
		DiffView(c, d).Fill()
	}, 800, 400)
	tt.Frame()
	if hs := d.Hunks(); len(hs) != 2 {
		t.Fatalf("hunks %+v", hs)
	}
	if n := d.sides[1].number(3); n != 40 {
		t.Fatalf("line 3 of the new text is line %d of the file", n)
	}
	if n := d.sides[0].number(1); n != 11 {
		t.Fatalf("line 1 of the old text is line %d of the file", n)
	}
	if d.New.Language() == nil || !strings.HasPrefix(d.sides[1].blks[0].id.(string), "@@ -10,3 +10,3 @@") {
		t.Fatalf("language %v, blocks %+v", d.New.Language(), d.sides[1].blks)
	}
}

// The patch of issue #1047: its first hunk ends in a doc comment.
const commentPatch = `diff --git a/compare.cpp b/compare.cpp
--- a/compare.cpp
+++ b/compare.cpp
@@ -1,7 +1,7 @@
 #include <cstdlib>
 #include <vector>
 
-int threshold = 24;
+int threshold = 32;
 
 /**
  * Compares two RGBA frames pixel by pixel.
@@ -27,3 +27,3 @@ int compare(...)
     // padding 9
-    for (int i = 0; i < cd.size() && i < fd.size(); i += 4) {
+    for (size_t i = 0; i < cd.size() && i < fd.size(); i += 4) {
         const int dr = std::abs(cd[i] - fd[i]);
`

// firstKind returns the kind of the first token of the line of b holding
// s, past its indentation, as b's highlighter makes it once done.
func firstKind(t *testing.T, b *Buffer, s string) TokenKind {
	t.Helper()
	line := -1
	for i := range b.LineCount() {
		if strings.Contains(b.Line(i), s) {
			line = i
			break
		}
	}
	if line < 0 {
		t.Fatalf("no line holds %q", s)
	}
	h := b.highlighter()
	if !h.Highlight(b.LineCount()-1, time.Hour) {
		t.Fatal("highlighter not done")
	}
	text := b.Line(line)
	toks, final := h.Tokens(line, text, nil)
	if !final {
		t.Fatalf("tokens of %q not final", text)
	}
	indent := len(text) - len(strings.TrimLeft(text, " \t"))
	for _, tk := range toks {
		if tk.End > indent {
			return tk.Kind
		}
	}
	t.Fatalf("no tokens past the indentation of %q: %+v", text, toks)
	return 0
}

// Each hunk of a patch is highlighted from state 0, not from the state
// the hunk before ends in (#1047).
func TestPatchHunksHighlightApart(t *testing.T) {
	files, err := ParsePatch(commentPatch)
	if err != nil {
		t.Fatal(err)
	}
	d := &DiffState{}
	d.SetPatch(files[0])
	if k := firstKind(t, d.New, "for (size_t"); k != TokenKeyword {
		t.Fatalf("new side's for is %v, want a keyword", k)
	}
	if k := firstKind(t, d.Old, "for (int"); k != TokenKeyword {
		t.Fatalf("old side's for is %v, want a keyword", k)
	}
	// The doc comment the first hunk ends in is one still.
	if k := firstKind(t, d.New, "Compares two"); k != TokenComment {
		t.Fatalf("doc comment is %v", k)
	}
	// So the views draw it.
	tt := ui.NewTester(func(c *ui.Context) {
		DiffView(c, d).Fill()
	}, 800, 400)
	tt.Frame()
	tt.Frame()
	n := d.New.LineCount() - 2
	if l := d.eds[1].view.line(n); !strings.Contains(l.text, "for (size_t") || len(l.toks) == 0 || l.toks[0].Kind == TokenComment {
		t.Fatalf("line %d %q drawn with tokens %+v", n, l.text, l.toks)
	}

	// Another language, as an app sets, starts the hunks afresh too.
	cpp := *d.New.Language()
	d.New.SetLanguage(&cpp)
	if k := firstKind(t, d.New, "for (size_t"); k != TokenKeyword {
		t.Fatalf("for is %v in another language", k)
	}

	// The hunks' first lines move with edits: lines added to the first
	// hunk, and its last line deleted.
	d.New.Edit(Edit{Start: 0, Text: "/* a\n b */\n"})
	if k := firstKind(t, d.New, "for (size_t"); k != TokenKeyword {
		t.Fatalf("for is %v after lines added above", k)
	}
	at := d.New.Offset(Pos{Line: d.New.LineCount() - 4})
	d.New.Edit(Edit{Start: at, End: d.New.Offset(Pos{Line: d.New.LineCount() - 3}), Text: ""})
	if k := firstKind(t, d.New, "for (size_t"); k != TokenKeyword || strings.Contains(d.New.String(), "Compares") {
		t.Fatalf("for is %v after the comment's line went: %q", k, d.New.String())
	}
	// A new highlighter, after edits, takes the lines where they are.
	d.New.SetLanguage(LanguageFor("compare.cpp"))
	if k := firstKind(t, d.New, "for (size_t"); k != TokenKeyword {
		t.Fatalf("for is %v in a highlighter made after the edits", k)
	}
	// An edit opening a comment in the first hunk stops at the second,
	// tokenizing no line past its first.
	calls := 0
	counted := *LanguageFor("compare.cpp")
	inner := counted.Tokenizer
	counted.Tokenizer = TokenizerFunc(func(line string, state uint64, toks []Token) ([]Token, uint64) {
		calls++
		return inner.Tokenize(line, state, toks)
	})
	d.New.SetLanguage(&counted)
	firstKind(t, d.New, "for (size_t")
	second := 0
	for i := range d.New.LineCount() {
		if strings.Contains(d.New.Line(i), "padding 9") {
			second = i
		}
	}
	at = d.New.Offset(Pos{Line: 2})
	d.New.Edit(Edit{Start: at, End: at, Text: "/* "})
	calls = 0
	h := d.New.highlighter()
	h.Highlight(d.New.LineCount()-1, time.Hour)
	if calls != second-2 {
		t.Fatalf("tokenized %d lines after the edit, want the %d from it to the second hunk", calls, second-2)
	}
	if k := firstKind(t, d.New, "padding 9"); k != TokenComment {
		t.Fatalf("second hunk's first line is %v", k)
	}
	if k := firstKind(t, d.New, "for (size_t"); k != TokenKeyword {
		t.Fatalf("for is %v after a comment opened in the first hunk", k)
	}
	if k := firstKind(t, d.New, "include <vector>"); k != TokenComment {
		t.Fatalf("the first hunk is %v under the comment opened", k)
	}
}

// A string over lines stops at the end of its hunk too.
func TestPatchHunksHighlightApartStrings(t *testing.T) {
	files, err := ParsePatch(`--- a/app.py
+++ b/app.py
@@ -1,4 +1,4 @@
-import os
+import sys
 
 def f():
     """Does f.
@@ -20,3 +20,3 @@ def f():
 def g():
-    return 1
+    return 2
`)
	if err != nil {
		t.Fatal(err)
	}
	d := &DiffState{}
	d.SetPatch(files[0])
	for _, b := range []*Buffer{d.Old, d.New} {
		if k := firstKind(t, b, "Does f"); k != TokenString {
			t.Fatalf("docstring is %v", k)
		}
		if k := firstKind(t, b, "def g"); k != TokenKeyword {
			t.Fatalf("def after the docstring's hunk is %v", k)
		}
		if k := firstKind(t, b, "return"); k != TokenKeyword {
			t.Fatalf("return is %v", k)
		}
	}
	// Texts of the app's are one text, highlighted as such.
	d.Old = NewBuffer(d.Old.String())
	d.Old.SetLanguage(LanguageFor("app.py"))
	if k := firstKind(t, d.Old, "def g"); k != TokenString {
		t.Fatalf("def in the docstring of a whole text is %v", k)
	}
	// A Language.Highlighter of the app's sees the patch's text whole.
	var seen Snapshot
	d.New.SetLanguage(&Language{Name: "Own", Highlighter: func(text Snapshot, _ func()) Highlighter {
		seen = text
		return NewTokenHighlighter(LanguageFor("app.py").Tokenizer, text)
	}})
	if k := firstKind(t, d.New, "def g"); k != TokenString || seen.String() != d.New.String() {
		t.Fatalf("def is %v with the app's highlighter of %q", k, seen.String())
	}
}

// Lines restarting the highlighter move with their first byte, and go
// when an edit replaces it.
func TestMoveRestarts(t *testing.T) {
	text := "a\nb\nc\nd\ne"
	for _, c := range []struct {
		edit Edit
		want []int
	}{
		{Edit{Start: 0, End: 0, Text: "x\n"}, []int{0, 2, 5}},  // at the first restart: it stays
		{Edit{Start: 2, End: 2, Text: "x\n"}, []int{0, 1, 5}},  // at a restart's start: it stays
		{Edit{Start: 2, End: 4, Text: ""}, []int{0, 1, 3}},     // b's line, with its newline
		{Edit{Start: 1, End: 3, Text: ""}, []int{0, 3}},        // b's start, joined to a
		{Edit{Start: 0, End: 9, Text: "z"}, []int{0}},          // everything
		{Edit{Start: 6, End: 6, Text: "\n\n"}, []int{0, 1, 6}}, // after the last
	} {
		b := NewBuffer(text)
		b.setRestarts([]int{0, 1, 4})
		b.Edit(c.edit)
		if got := b.restarts; !slices.Equal(got, c.want) {
			t.Errorf("%+v: restarts %v, want %v", c.edit, got, c.want)
		}
	}
	// Undo moves them back.
	b := NewBuffer(text)
	b.setRestarts([]int{1, 4})
	b.Edit(Edit{Start: 0, Text: "x\ny\n"})
	b.Undo()
	if !slices.Equal(b.restarts, []int{1, 4}) {
		t.Errorf("restarts %v after undo", b.restarts)
	}
	// Two meeting are one.
	b = NewBuffer(text)
	b.setRestarts([]int{1, 2})
	b.Edit(Edit{Start: 2, End: 4})
	if !slices.Equal(b.restarts, []int{1}) {
		t.Errorf("restarts %v after the line between went", b.restarts)
	}
}
