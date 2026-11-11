//go:build !js

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

// TestRichTextEditor drives a rich text editor in a window through the
// system: the Format menu bolds what an input method then composes, the
// Edit menu pastes HTML another app copied with its formats, and undoes
// it.
func TestRichTextEditor(t *testing.T) {
	prev := mygo.App.Menu()
	defer mygo.App.SetMenu(prev)
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: "App", Submenu: []*mygo.MenuItem{{Role: mygo.RoleQuit}}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleFormatMenu},
	}))
	doc := richtext.NewDocument(richtext.NewText("Hello "))
	var frames atomic.Int32
	view := func(c *ui.Context) {
		richtext.Editor(c, doc, richtext.EditorOptions{Label: "Note"}).Fill().AutoFocus()
		frames.Add(1)
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Rich text", Width: 420, Height: 220, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	// In the middle of the content, whatever the frame takes of the
	// window: Windows 11's menu bar, which MyGo draws taller, left the
	// content short of where a fixed point was.
	b := w.ContentBounds()
	if !click(w, float64(b.Width)/2, float64(b.Height)/2) {
		t.Skip("native click automation unavailable")
	}
	runs := func() string {
		var s string
		mygo.RunOnMain(func() {
			for _, r := range doc.Text().Runs(0, doc.Text().Len()) {
				a := r.Attributes
				text := r.Text
				if a.Bold() {
					text = "*" + text + "*"
				}
				if a.Italic {
					text = "_" + text + "_"
				}
				s += text + "|"
			}
		})
		return s
	}
	// Bold at the caret, then an input method's text takes it.
	if err := activateMenu(w, "Format", "Bold"); err != nil {
		t.Skipf("menu automation unavailable: %v", err)
	}
	eventually(t, "bold typing attributes", func() bool {
		var bold bool
		mygo.RunOnMain(func() { bold = doc.Formatting().Bold == richtext.FormatOn })
		return bold
	})
	// A composition at the caret, then its commit, as an input method
	// sends them.
	if !composeOver(w, "世界", 2, false, -1, 0) {
		t.Skip("input method automation unavailable")
	}
	composeOver(w, "世界", 2, true, -1, 0)
	eventually(t, "the composition committed in bold", func() bool { return runs() == "Hello |*世界*|" })
	// HTML pasted keeps its formats; one undo takes it back.
	err := mygo.Clipboard.Write(transfer.New(transfer.NewItem(
		transfer.Bytes(transfer.HTML, []byte("<p><i>again</i></p>")),
		transfer.Bytes(transfer.Text, []byte("again")),
	)))
	if err != nil {
		t.Fatal(err)
	}
	if err := activateMenu(w, "Edit", "Paste"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the HTML pasted", func() bool { return runs() == "Hello |*世界*|_again_|" })
	if err := activateMenu(w, "Edit", "Undo"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the paste undone", func() bool { return runs() == "Hello |*世界*|" })
	// RTF alone, as TextEdit copies it, pastes with its formats too.
	err = mygo.Clipboard.Write(transfer.New(transfer.NewItem(
		transfer.Bytes(richtext.RTF, []byte(`{\rtf1\ansi{\fonttbl\f0 Helvetica;}\f0 \b rtf\b0 }`)),
		transfer.Bytes(transfer.Text, []byte("rtf")),
	)))
	if err != nil {
		t.Fatal(err)
	}
	if err := activateMenu(w, "Edit", "Paste"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the RTF pasted", func() bool { return runs() == "Hello |*世界rtf*|" })
}
