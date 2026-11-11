package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

const demoGreeter = `package greeter

// Greet greets name, or the world.
func Greet(name string) string {
	// TODO: trim spaces.
	if name == "" {
		name = "world"
	}
	return "Hello, " + name + "!"
}
`

const demoGreeterEdited = `package greeter

import "strings"

// Greet greets name, or everyone.
func Greet(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "everyone"
	}
	return "Hello, " + name + "!"
}
`

func init() {
	golang := editor.LanguageFor("greeter.go")
	code := editor.NewBuffer(demoGreeter)
	code.SetLanguage(golang)
	st := &editor.State{Buffer: code, Label: "greeter.go", WrapGuides: []int{80}}
	demo("code-editor", func(c *ui.Context) {
		ui.Box(c).AlignSelf(ui.Stretch).Height(230).Border(1, c.Theme().Border).Radius(8).Clip().Children(func() {
			editor.View(c, st).Fill()
		})
		// The TODO comments are warnings, as a linter's would be.
		todos := todosOf(code)
		st.SetDiagnostics(code.Version(), todos)
		pos := st.CursorPos()
		carets := ""
		if n := len(st.Selections()); n > 1 {
			carets = fmt.Sprintf(" · %d carets", n)
		}
		status(c, fmt.Sprintf("Ln %d, Col %d%s · %d lines · %d TODO", pos.Line+1, pos.Col+1, carets, code.LineCount(), len(todos)))
	})

	before, after := editor.NewBuffer(demoGreeter), editor.NewBuffer(demoGreeterEdited)
	before.SetLanguage(golang)
	after.SetLanguage(golang)
	diff := &editor.DiffState{Old: before, New: after, Editable: true, HunkActions: true}
	demo("diff-viewer", func(c *ui.Context) {
		editor.DiffView(c, diff).AlignSelf(ui.Stretch).Height(250).Border(1, c.Theme().Border).Radius(8).Clip()
		added, deleted := diff.Stats()
		status(c, fmt.Sprintf("+%d −%d · edit the right side, or revert and keep its changes", added, deleted))
	})
}

// todosOf marks the TODO comments of a buffer as warnings.
func todosOf(b *editor.Buffer) []editor.Diagnostic {
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
