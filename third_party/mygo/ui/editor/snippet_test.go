package editor

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func snippetFor(t *testing.T, source string) *Snippet {
	t.Helper()
	s, err := ParseSnippet(source)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSnippetTabstops(t *testing.T) {
	ct := newEditorTest(t, "", "before | after")
	if !ct.st.InsertSnippet(snippetFor(t, "call(${2:second}, ${1:first})$0")) {
		t.Fatal("snippet not inserted")
	}
	ct.tt.Frame()
	ct.want("before call(second, [first]) after")
	ct.typ("你好")
	ct.key(0, ui.KeyTab)
	ct.want("before call([second], 你好) after")
	ct.key(ui.Shift, ui.KeyTab)
	ct.want("before call(second, [你好]) after")
	ct.key(0, ui.KeyTab)
	ct.typ("ok")
	ct.key(0, ui.KeyTab)
	ct.want("before call(ok, 你好)| after")
}

func TestSnippetMirrors(t *testing.T) {
	ct := newEditorTest(t, "", "|")
	ct.st.InsertSnippet(snippetFor(t, "$1 = ${1:名前}; ${2:value}; $0"))
	ct.tt.Frame()
	ct.want("[名前] = [名前]; value; ")
	ct.typ("長い名")
	ct.want("長い名| = 長い名|; value; ")
	ct.key(0, ui.KeyTab)
	ct.want("長い名 = 長い名; [value]; ")
	ct.key(ui.Shift, ui.KeyTab)
	ct.want("[長い名] = [長い名]; value; ")
	ct.key(0, ui.KeyBackspace)
	ct.typ("x")
	ct.key(0, ui.KeyTab)
	ct.want("x = x; [value]; ")
}

func TestSnippetMultipleCaretsAndUndo(t *testing.T) {
	ct := newEditorTest(t, "", "one |\ntwo |")
	ct.st.InsertSnippet(snippetFor(t, "f(${1:x}, ${2:y})$0"))
	ct.tt.Frame()
	ct.want("one f([x], y)\ntwo f([x], y)")
	ct.typ("a")
	ct.key(0, ui.KeyTab)
	ct.typ("b")
	ct.key(0, ui.KeyTab)
	ct.want("one f(a, b)|\ntwo f(a, b)|")
	ct.run(Undo)
	ct.want("one f(a, [y])\ntwo f(a, [y])")
	ct.run(Undo)
	ct.run(Undo)
	ct.want("one |\ntwo |")
	ct.run(Redo)
	ct.want("one f([x], y)\ntwo f([x], y)")
}

func TestSnippetCompletion(t *testing.T) {
	ct := newEditorTest(t, "go", "pri|")
	ct.st.ShowCompletions(0, []Completion{{Label: "print", Kind: CompletionSnippet, Snippet: snippetFor(t, "print(${1:value})$0")}})
	ct.tt.Frame()
	ct.key(0, ui.KeyTab)
	ct.want("print([value])")
	ct.typ("name")
	ct.key(0, ui.KeyTab)
	ct.want("print(name)|")
}

func TestSnippetParsing(t *testing.T) {
	for _, tc := range []struct{ source, text string }{
		{"${2:b} $1 ${1:a} $2 $0", "b a a b "},
		{`\$x \\ ${1:a\}b} $1`, `$x \ a}b a}b`},
		{"${1:你好}\r\n$0", "你好\n"},
		{"plain text", "plain text"},
	} {
		if got := snippetFor(t, tc.source).Text(); got != tc.text {
			t.Errorf("%q: got %q, want %q", tc.source, got, tc.text)
		}
	}
	for _, source := range []string{"${", "${1", "${1:x", "$HOME", "${HOME}", "${1|a,b|}", "${1/x/y/}", "${1:${2:x}}", "${1:a} ${1:b}", "$99999999999999999999999999999999", "$2147483648", "\xff"} {
		if _, err := ParseSnippet(source); err == nil {
			t.Errorf("accepted unsupported or invalid snippet %q", source)
		}
	}
}

func TestSnippetCancelAndExternalEdit(t *testing.T) {
	for _, cancel := range []func(*editorTest){
		func(ct *editorTest) { ct.key(0, ui.KeyEscape) },
		func(ct *editorTest) { ct.st.Select(0, 0) },
		func(ct *editorTest) { ct.st.Buffer.Edit(Edit{Start: 0, Text: ">"}); ct.tt.Frame() },
		func(ct *editorTest) { ct.run(Undo) },
	} {
		ct := newEditorTest(t, "", "prefix |")
		ct.st.InsertSnippet(snippetFor(t, "${1:x} ${2:y}$0"))
		ct.tt.Frame()
		cancel(ct)
		if ct.st.ed.snippet != nil {
			t.Fatal("snippet survived cancellation")
		}
	}
}

func TestSnippetAdjacentFields(t *testing.T) {
	for _, source := range []string{"${1:a}$1${2:b}$0", "$2${1:a}$1$0"} {
		ct := newEditorTest(t, "", "|")
		ct.st.InsertSnippet(snippetFor(t, source))
		ct.tt.Frame()
		ct.typ("xy")
		ct.key(0, ui.KeyTab)
		if source[1] == '{' {
			ct.want("xyxy[b]")
		} else {
			ct.want("|xyxy")
		}
	}
	// Empty stops at the same byte coalesce, as ordinary carets do.
	ct := newEditorTest(t, "", "|")
	ct.st.InsertSnippet(snippetFor(t, "$1$1-$2$0"))
	ct.tt.Frame()
	ct.typ("x")
	ct.key(0, ui.KeyTab)
	ct.want("x-|")
}

func TestSnippetReadOnlyAndPlainCompletion(t *testing.T) {
	ct := newEditorTest(t, "", "a|")
	ct.st.ReadOnly = true
	ct.tt.Frame()
	before := ct.st.Selections()
	if ct.st.InsertSnippet(snippetFor(t, "${1:x}")) || ct.st.Buffer.String() != "a" || !slices.Equal(before, ct.st.Selections()) {
		t.Fatal("read-only editor changed")
	}
	ct.st.ReadOnly = false
	ct.tt.Frame()
	ct.st.ShowCompletions(0, []Completion{{Label: "abc", Insert: "literal $1", Kind: CompletionSnippet}})
	ct.tt.Frame()
	ct.key(0, ui.KeyTab)
	ct.want("literal $1|")
}

func TestSnippetComposition(t *testing.T) {
	ct := newEditorTest(t, "", "|")
	ct.st.InsertSnippet(snippetFor(t, "${1:name} = $1; ${2:value}$0"))
	ct.tt.Frame()
	ct.tt.Compose("に", 1)
	ct.tt.Compose("にほ", 2)
	ct.tt.Type("日本語")
	ct.want("日本語| = 日本語|; value")
	ct.key(0, ui.KeyTab)
	ct.want("日本語 = 日本語; [value]")
}

func TestSnippetDeletesCoincidentMirrors(t *testing.T) {
	ct := newEditorTest(t, "", "|")
	ct.st.InsertSnippet(snippetFor(t, "${1:a}$2$1$0"))
	ct.tt.Frame()
	ct.key(0, ui.KeyBackspace)
	if ct.st.ed.snippet == nil {
		t.Fatal("session lost after deletion")
	}
	ct.typ("x")
	if ct.st.ed.snippet == nil {
		t.Fatal("session lost after typing")
	}
	ct.key(0, ui.KeyTab)
	ct.want("x|")
}

func TestSnippetNoopAndReplacement(t *testing.T) {
	ct := newEditorTest(t, "", "[x] suffix")
	ct.st.InsertSnippet(snippetFor(t, "${1:x}$2$0"))
	ct.tt.Frame()
	ct.want("[x] suffix")
	ct.typ("y")
	ct.st.InsertSnippet(snippetFor(t, "${1:z}-$2$0"))
	ct.tt.Frame()
	ct.want("y[z]- suffix")
	ct.typ("q")
	ct.key(0, ui.KeyTab)
	ct.want("yq-| suffix")
	ct = newEditorTest(t, "", "|")
	if !ct.st.InsertSnippet(&Snippet{}) {
		t.Fatal("zero snippet was not inserted")
	}
	ct.tt.Frame()
	ct.want("|")
	if ct.st.InsertSnippet(nil) {
		t.Fatal("nil snippet was inserted")
	}
}

func TestSnippetArrowsAndBufferSwap(t *testing.T) {
	ct := newEditorTest(t, "", "|")
	s := snippetFor(t, "${1:ab}-$1-$2$0")
	ct.st.InsertSnippet(s)
	ct.tt.Frame()
	ct.key(0, ui.KeyRight)
	ct.key(0, ui.KeyLeft)
	ct.typ("X")
	ct.key(0, ui.KeyTab)
	ct.want("aXb-aXb-|")
	ct.st.Buffer = NewBuffer("other")
	ct.tt.Frame()
	if ct.st.ed.snippet != nil {
		t.Fatal("buffer swap retained snippet")
	}
	// Parsed templates are reusable; one editor's edits do not change one.
	if s.Text() != "ab-ab-" {
		t.Fatal("snippet was mutated")
	}
}

func FuzzParseSnippet(f *testing.F) {
	for _, source := range []string{"", "${1:你好} $1 $2$0", `\$ ${1:a\}b}`, "$1$2$1", "${1:${2:nested}}", "${HOME}", "\xff"} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		s, err := ParseSnippet(source)
		if err != nil {
			return
		}
		values := map[int]string{}
		for _, field := range s.fields {
			if field.start < 0 || field.end < field.start || field.end > len(s.Text()) {
				t.Fatal("field outside text")
			}
			text := s.Text()[field.start:field.end]
			if old, ok := values[field.id]; ok && old != text {
				t.Fatal("mirrors have different defaults")
			}
			values[field.id] = text
		}
		if len(s.stops) == 0 || s.stops[len(s.stops)-1] != 0 || !slices.IsSorted(s.stops[:len(s.stops)-1]) {
			t.Fatal("invalid traversal order")
		}
	})
}

func FuzzSnippetEdits(f *testing.F) {
	for _, source := range []string{"${1:a}$2$1$0", "a${1:你好}-$1-$2$0", "$1$2$1", "${2:b}$0${1:x}"} {
		f.Add(source, []byte{0, 1, 2, 3, 0, 4, 0})
	}
	f.Fuzz(func(t *testing.T, source string, actions []byte) {
		if len(source) > 1024 || len(actions) > 64 {
			return
		}
		snippet, err := ParseSnippet(source)
		if err != nil {
			return
		}
		st := &State{Buffer: NewBuffer("")}
		st.InsertSnippet(snippet)
		for _, action := range actions {
			switch action % 5 {
			case 0:
				st.Insert("x")
			case 1:
				st.Run(Tab)
			case 2:
				st.Run(Backtab)
			case 3:
				st.Run(Backspace)
			case 4:
				st.Run(Delete)
			}
			if s := st.ed.snippet; s != nil {
				text := st.Buffer.String()
				values := map[int]string{}
				for _, field := range s.fields {
					if field.start < 0 || field.end < field.start || field.end > len(text) {
						t.Fatal("field outside edited text")
					}
					value := text[field.start:field.end]
					if old, ok := values[field.id]; ok && old != value {
						t.Fatal("mirrored values diverged")
					}
					values[field.id] = value
				}
			}
		}
	})
}

func TestSnippetCompletionOverlappingPrefixes(t *testing.T) {
	ct := newEditorTest(t, "", "f|oo|")
	ct.st.ShowCompletions(0, []Completion{{Label: "foobar", Snippet: snippetFor(t, "f(${1:x})$0")}})
	ct.tt.Frame()
	ct.key(0, ui.KeyTab)
	ct.want("f([x])")
	ct.typ("y")
	ct.key(0, ui.KeyTab)
	ct.want("f(y)|")
}

func TestSnippetPreservesPrimarySite(t *testing.T) {
	ct := newEditorTest(t, "", "a| b|")
	ct.st.SetSelections(Selection{Anchor: 3, Head: 3}, Selection{Anchor: 1, Head: 1})
	ct.st.InsertSnippet(snippetFor(t, "${1:x}=$1-$2$0"))
	ct.tt.Frame()
	if got := ct.st.Cursor(); got != 2 {
		t.Fatalf("primary caret at %d, want first field of original primary site (2)", got)
	}
	ct.key(0, ui.KeyTab)
	if got := ct.st.Cursor(); got != 5 {
		t.Fatalf("next field scrolled to another site: %d", got)
	}
}
