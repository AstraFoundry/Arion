package ui

import (
	"strconv"
	"testing"
)

func TestOnPaste(t *testing.T) {
	text := "hello world"
	var seen []string
	attached := 0
	tt := NewTester(func(c *Context) {
		Text(c, strconv.Itoa(attached)+" attached")
		TextArea(c, &text).Width(200).Label("draft").OnPaste(func(s string) bool {
			seen = append(seen, s)
			if len(s) > 5 {
				attached++
				return true
			}
			return false
		})
	}, 300, 100)
	tt.Click("draft")
	tt.Key(Cmd, KeyEnd)

	// Left: the paste inserts as usual.
	tt.SetClipboard("!")
	tt.Key(Cmd, KeyV)
	if text != "hello world!" {
		t.Fatalf("a paste left became %q", text)
	}

	// Taken, by Cmd+V and by the menu: nothing changes, the selection
	// stays, as when a web page prevents a paste's default.
	tt.SetClipboard("a large block")
	tt.Key(Cmd, KeyV)
	if !tt.HasText("1 attached") {
		t.Fatalf("a paste taken showed %q", tt.Texts())
	}
	tt.Key(Cmd, KeyA)
	tt.Command("paste")
	if text != "hello world!" {
		t.Fatalf("a paste taken became %q", text)
	}
	if !tt.HasText("2 attached") {
		t.Fatalf("a paste taken showed %q", tt.Texts())
	}
	tt.Command("delete")
	if text != "" {
		t.Fatalf("the selection a paste taken left became %q", text)
	}
	// A paste taken leaves nothing to undo.
	tt.Key(Cmd, KeyZ)
	if text != "hello world!" {
		t.Fatalf("undone, it became %q", text)
	}
	tt.Key(Cmd, KeyZ)
	if text != "hello world" {
		t.Fatalf("undone twice, it became %q", text)
	}

	// An empty clipboard, as with an image, is seen too.
	tt.SetClipboard("")
	tt.Command("paste")
	want := []string{"!", "a large block", "a large block", ""}
	if len(seen) != len(want) {
		t.Fatalf("saw %q", seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("saw %q", seen)
		}
	}
}
