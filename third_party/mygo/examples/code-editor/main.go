// Code editor shows MyGo's code editor and diff viewer: files of several
// languages to edit, with completions of the words of the file, hovers,
// diagnostics of TODO comments and the changes against the file as it
// was in the gutter; and the diff of those changes, side by side or
// unified, which edits the same buffer.
//
//	go run ./examples/code-editor
//
// In a Go file, complete funcdef or forrange with Tab to insert a snippet;
// Tab and Shift+Tab visit its fields, and Escape ends snippet editing.
package main

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// file is a file of the example: its text as it was, and the buffer and
// editor state editing it, kept across frames.
type file struct {
	name     string
	original *editor.Buffer
	editor   *editor.State
	diff     *editor.DiffState
}

type app struct {
	win      *mygo.Window
	files    []*file
	selected string
	page     int // 0 the editor, 1 the diff
	layout   int // the diff's: 0 split, 1 unified
	inline   string
	wrap     bool
}

func newFile(name, text string) *file {
	f := &file{name: name, original: editor.NewBuffer(text)}
	buf := editor.NewBuffer(text)
	lang := editor.LanguageFor(name)
	buf.SetLanguage(lang)
	f.original.SetLanguage(lang)
	f.editor = &editor.State{Buffer: buf, DiffBase: f.original, WrapGuides: []int{100}}
	f.diff = &editor.DiffState{Old: f.original, New: buf, Editable: true, HunkActions: true, LineSelection: true}
	return f
}

func (a *app) current() *file {
	for _, f := range a.files {
		if f.name == a.selected {
			return f
		}
	}
	return a.files[0]
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	f := a.current()
	f.editor.SoftWrap = a.wrap
	f.diff.Wrap = a.wrap
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Width(200).PaddingY(12).Background(t.Surface).Shrink(0).Children(func() {
			ui.Sidebar(c, &a.selected, func() {
				for _, f := range a.files {
					label := f.name
					if f.editor.Buffer.Modified() {
						label += " •"
					}
					ui.SidebarItem(c, f.name, nil, label)
				}
			}).Grow(1).Label("Files")
		})
		ui.Column(c).Grow(1).FillHeight().Children(func() {
			ui.Row(c).Gap(12).Padding(8, 12).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
				ui.Segmented(c, &a.page, "Editor", "Changes")
				if a.page == 1 {
					ui.Segmented(c, &a.layout, "Split", "Unified")
					ui.Select(c, &a.inline, []string{"Words", "Characters", "Off"})
				}
				ui.Box(c).Grow(1)
				ui.Text(c, "Wrap").TextColor(t.TextMuted)
				ui.Switch(c, &a.wrap).Label("Soft wrap")
			})
			if a.page == 0 {
				a.editorPage(c, f)
			} else {
				a.diffPage(c, f)
			}
		})
	})
}

func (a *app) editorPage(c *ui.Context, f *file) {
	t := c.Theme()
	st := f.editor
	st.OnCompletion = func(r editor.CompletionRequest) {
		st.ShowCompletions(r.Offset-len(r.Prefix), wordsOf(st.Buffer, r.Prefix))
	}
	st.OnHover = func(r editor.HoverRequest) {
		word, start, end := wordAt(st.Buffer, r.Offset)
		if word == "" {
			return
		}
		n := strings.Count(st.Buffer.String(), word)
		st.ShowHover(start, end, fmt.Sprintf("%s: %d occurrence%s in %s", word, n, map[bool]string{true: "", false: "s"}[n == 1], f.name), nil)
	}
	st.SetDiagnostics(st.Buffer.Version(), todos(st.Buffer))
	editor.View(c, st).Grow(1).FillWidth().AutoFocus()
	// The status bar.
	pos := st.CursorPos()
	sels := len(st.Selections())
	lang := "Plain Text"
	if l := st.Buffer.Language(); l != nil {
		lang = l.Name
	}
	status := fmt.Sprintf("Ln %d, Col %d", pos.Line+1, pos.Col+1)
	if sels > 1 {
		status += fmt.Sprintf(" · %d selections", sels)
	}
	added, deleted := f.diff.Stats()
	ui.Row(c).Gap(16).Padding(4, 12).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Text(c, status).FontSize(12).TextColor(t.TextMuted)
		ui.Text(c, fmt.Sprintf("+%d −%d", added, deleted)).FontSize(12).TextColor(t.TextMuted)
		ui.Box(c).Grow(1)
		ui.Text(c, fmt.Sprintf("%d lines · %s", st.Buffer.LineCount(), lang)).FontSize(12).TextColor(t.TextMuted)
	})
}

func (a *app) diffPage(c *ui.Context, f *file) {
	d := f.diff
	d.Layout = editor.DiffLayout(a.layout)
	switch a.inline {
	case "Characters":
		d.Inline = editor.DiffInlineChar
	case "Off":
		d.Inline = editor.DiffInlineNone
	default:
		d.Inline = editor.DiffInlineWordAlt
	}
	if len(d.Hunks()) == 0 {
		ui.Column(c).Grow(1).Center().Children(func() {
			ui.Text(c, "No changes").FontSize(18).Bold()
			ui.Text(c, "Edit "+f.name+" to see its changes here.").TextColor(c.Theme().TextMuted)
		})
		return
	}
	editor.DiffView(c, d).Grow(1).FillWidth()
}

// wordsOf completes prefix with the words of the buffer, as editors do
// without a language server.
func wordsOf(b *editor.Buffer, prefix string) []editor.Completion {
	seen := map[string]bool{}
	var out []editor.Completion
	words := strings.FieldsFunc(b.String(), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' })
	for _, w := range words {
		if len(w) < 3 || seen[w] || w == prefix {
			continue
		}
		seen[w] = true
		out = append(out, editor.Completion{Label: w, Kind: editor.CompletionText, Detail: "word"})
	}
	if l := b.Language(); l != nil {
		for _, k := range []string{"func", "return", "import", "package", "const", "struct", "interface", "range"} {
			if !seen[k] {
				out = append(out, editor.Completion{Label: k, Kind: editor.CompletionKeyword, Detail: "keyword"})
			}
		}
	}
	if l := b.Language(); l != nil && l.Name == "Go" {
		out = append(out, goSnippets...)
	}
	slices.SortFunc(out, func(x, y editor.Completion) int { return strings.Compare(x.Label, y.Label) })
	return out
}

// Templates are parsed once and shared by completions, not per frame.
var goSnippets = []editor.Completion{
	exampleSnippet("funcdef", "func ${1:name}(${2:args}) ${3:error} {\n\t$0\n}"),
	exampleSnippet("forrange", "for ${1:index}, ${2:value} := range ${3:items} {\n\t$0\n}"),
}

func exampleSnippet(label, source string) editor.Completion {
	s, err := editor.ParseSnippet(source)
	if err != nil {
		panic(err)
	}
	return editor.Completion{Label: label, Kind: editor.CompletionSnippet, Detail: "snippet · Tab visits fields", Snippet: s}
}

func wordAt(b *editor.Buffer, off int) (string, int, int) {
	p := b.Pos(off)
	line := b.Line(p.Line)
	isWord := func(r byte) bool {
		return r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
	}
	a, z := p.Col, p.Col
	for a > 0 && isWord(line[a-1]) {
		a--
	}
	for z < len(line) && isWord(line[z]) {
		z++
	}
	start := off - (p.Col - a)
	return line[a:z], start, start + z - a
}

// todos marks the TODO comments as warnings.
func todos(b *editor.Buffer) []editor.Diagnostic {
	var out []editor.Diagnostic
	for i := range b.LineCount() {
		line := b.Line(i)
		if at := strings.Index(line, "TODO"); at >= 0 {
			start := b.Offset(editor.Pos{Line: i, Col: at})
			out = append(out, editor.Diagnostic{Start: start, End: start + len(line) - at, Severity: editor.SeverityWarning,
				Message: strings.TrimSpace(line[at:]), Source: "todo"})
		}
	}
	return out
}

func main() {
	a := &app{inline: "Words", selected: "server.go"}
	for _, s := range samples {
		a.files = append(a.files, newFile(s.name, s.text))
	}
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleAppMenu}, {Role: mygo.RoleEditMenu}, {Role: mygo.RoleWindowMenu}}))
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "Code editor", Width: 1200, Height: 800, Content: ui.View(a.view)})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
