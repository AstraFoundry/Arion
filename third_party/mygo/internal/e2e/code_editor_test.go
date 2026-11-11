//go:build !js

package e2e

import (
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// The code editor takes the system's input methods: a composition
// replacing text the input method names, its commit, and text typed.
func TestContentWindowCodeEditor(t *testing.T) {
	var frames atomic.Int32
	st := &editor.State{Buffer: editor.NewBuffer("first\ncafe")}
	w := newWindow(t, mygo.WindowOptions{Title: "Code editor", Width: 500, Height: 240, Content: ui.View(func(c *ui.Context) {
		editor.View(c, st).Fill().AutoFocus()
		frames.Add(1)
	})})
	eventually(t, "an editor frame", func() bool { return frames.Load() > 0 })
	if _, _, ok := inputClient(w); !ok {
		t.Skip("native input automation unavailable")
	}
	// The end of the second line.
	if !click(w, 400, 30) {
		t.Skip("click automation unavailable")
	}
	eventually(t, "the caret at the end", func() bool {
		var head int
		mygo.RunOnMain(func() { head = st.Cursor() })
		return head == len("first\ncafe")
	})
	// The e, in UTF-16 units on macOS, runes elsewhere: the same here.
	composeOver(w, "e", 1, false, 9, 1)
	composeOver(w, "é", 1, true, -1, 0)
	eventually(t, "the composition committed", func() bool {
		var text string
		mygo.RunOnMain(func() { text = st.Buffer.String() })
		return text == "first\ncafé"
	})
	if runtime.GOOS == "darwin" {
		// Typed text, and Return indenting nothing.
		if !clickAndType(w, 400, 30, "!") {
			return
		}
		eventually(t, "text typed", func() bool {
			var text string
			mygo.RunOnMain(func() { text = st.Buffer.String() })
			return text == "first\ncafé!"
		})
		// Up arrow (key code 126) moves the caret to the first line.
		pressKey(w, 126, "")
		eventually(t, "the caret up a line", func() bool {
			var p editor.Pos
			mygo.RunOnMain(func() { p = st.CursorPos() })
			return p.Line == 0
		})
	}
}

// With NoInputMethod, keys type the characters of the layout, which come
// to OnInput as text, without going through the input method.
func TestContentWindowCodeEditorNoInputMethod(t *testing.T) {
	var frames atomic.Int32
	var typed atomic.Value
	typed.Store("")
	st := &editor.State{Buffer: editor.NewBuffer("abc"), NoInputMethod: true, CursorShape: editor.CursorBlock}
	st.OnInput = func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputText {
			typed.Store(typed.Load().(string) + ev.Text)
			return true
		}
		return false
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Code editor", Width: 500, Height: 240, Content: ui.View(func(c *ui.Context) {
		editor.View(c, st).Fill().AutoFocus()
		frames.Add(1)
	})})
	eventually(t, "an editor frame", func() bool { return frames.Load() > 0 })
	if !click(w, 400, 30) {
		t.Skip("click automation unavailable")
	}
	eventually(t, "the editor focused", func() bool {
		var focused bool
		mygo.RunOnMain(func() { focused = st.Focused() })
		return focused
	})
	if !pressKey(w, 38, "j") && !pressKeys("j") { // kVK_ANSI_J on macOS
		t.Skip("key automation unavailable")
	}
	eventually(t, "j for OnInput", func() bool { return typed.Load().(string) == "j" })
	var text string
	mygo.RunOnMain(func() { text = st.Buffer.String() })
	if text != "abc" {
		t.Fatalf("the text is %q, want it untouched", text)
	}
}
