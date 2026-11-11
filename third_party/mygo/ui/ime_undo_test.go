package ui

import "testing"

// compose runs one composition through the widget's native client: the
// preedit, then the commit, with the empty preedit GTK's fcitx module
// sends on one side of the commit or the other.
func composeOver(t *testing.T, c interface {
	SetMarkedText(*TextInputRange, string, TextInputRange)
	ReplaceText(*TextInputRange, string)
}, text string, preeditAfter bool) {
	t.Helper()
	c.SetMarkedText(nil, text, TextInputRange{})
	if preeditAfter {
		c.ReplaceText(nil, text)
		c.SetMarkedText(nil, "", TextInputRange{})
		return
	}
	c.SetMarkedText(nil, "", TextInputRange{})
	c.ReplaceText(nil, text)
}

// A composition is one undo step, whichever side of the commit the input
// method clears its preedit on: the first Ctrl+Z takes the whole word
// back, not a step that changes nothing.
func TestWidgetCompositionUndoesOnce(t *testing.T) {
	for _, preeditAfter := range []bool{false, true} {
		s := "cafe"
		tt := inputTester(&s, false)
		c := tt.h.ime.Client
		composeOver(t, c, "你好", preeditAfter)
		tt.Frame()
		if s != "cafe你好" {
			t.Fatalf("preedit after %v: the commit did not reach the text: %q", preeditAfter, s)
		}
		tt.Key(Cmd, KeyZ)
		tt.Frame()
		if s != "cafe" {
			t.Fatalf("preedit after %v: one undo must take the whole composition back, got %q", preeditAfter, s)
		}
		tt.rt.close()
	}
}
