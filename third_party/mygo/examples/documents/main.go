// Documents is a notes editor of Markdown and text files in native UI,
// built on mygo.NewDocuments: the File menu, a window per note, autosave,
// recovery after a crash, the proxy icon and the edited dot on macOS, and
// notes changed by other apps.
//
//	go run ./examples/documents [file.md…]
package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// note is the window of a document: the text its editor shows.
type note struct {
	doc  *mygo.Document[string]
	win  *mygo.Window
	text string
}

func (n *note) view(c *ui.Context) {
	t := c.Theme()
	state := n.doc.State()
	ui.Column(c).Fill().Children(func() {
		if state.Conflict {
			ui.Row(c).Padding(10).Gap(8).Background(t.Warning.Alpha(0.15)).Children(func() {
				ui.Text(c, "Another app changed this note. Save to keep yours, or revert to theirs.").Grow(1)
				if ui.Button(c.Key("revert"), "Revert").Clicked() {
					go n.doc.Revert()
				}
				if ui.PrimaryButton(c.Key("save"), "Save").Clicked() {
					go n.doc.SaveAnyway()
				}
			})
		}
		ui.TextArea(c.Key("text"), &n.text).Fill().Font("monospace").Label("Note").AutoFocus().
			OnChange(func() { n.doc.Set(n.text) })
		ui.Row(c).Padding(6, 12).Gap(12).Children(func() {
			ui.Text(c, fmt.Sprintf("%d words", len(strings.Fields(n.text)))).FontSize(12).TextColor(t.TextMuted).Grow(1)
			status := "Saved"
			switch {
			case state.Path == "":
				status = "Not saved"
			case state.ReadOnly:
				status = "Read-only"
			case state.Edited:
				status = "Edited"
			}
			ui.Text(c, status).FontSize(12).TextColor(t.TextMuted)
		})
	})
}

func main() {
	mygo.NewDocuments(mygo.DocumentsOptions[string]{
		Types: []mygo.DocumentType{
			{Name: "Markdown Document", Extensions: []string{"md", "markdown"}},
			{Name: "Plain Text", Extensions: []string{"txt"}},
		},
		Read: func(r io.Reader) (string, error) {
			b, err := io.ReadAll(r)
			return string(b), err
		},
		Write: func(w io.Writer, text string) error {
			_, err := io.WriteString(w, text)
			return err
		},
		// Undoing back to what the file has leaves the note unedited.
		Equal: func(a, b string) bool { return a == b },
		Window: func(doc *mygo.Document[string]) *mygo.Window {
			n := &note{doc: doc, text: doc.Content()}
			n.win = mygo.NewWindow(mygo.WindowOptions{Width: 640, Height: 480, StateKey: "note", Content: ui.View(n.view)})
			// Reverted, or changed by another app: show the file's text.
			doc.OnReload(func() { n.text = doc.Content(); n.win.Update(func() {}) })
			doc.OnStateChange(func(mygo.DocumentState) { n.win.Update(func() {}) })
			return n.win
		},
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
