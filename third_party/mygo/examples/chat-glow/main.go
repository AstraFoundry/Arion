// Chat-glow is an assistant's window, lit as Apple Intelligence lights its
// own: while the assistant works, the window's edges glow in flowing
// colors, the field to write in glows in its ring, and "Thinking…"
// shimmers; the assistant's orb, a glass sphere of drifting light,
// quickens as it thinks.
//
//	go run ./examples/chat-glow
//
// Each is a pattern of its own for an app. The edges' glow is views: the
// window's outline as borders painted with a conic gradient
// (BorderConicGradient), a sharp one and blurred ones over it (Blur), in
// boxes over the whole window that the pointer goes through (Absolute,
// PassThrough), as SwiftUI's recreations of the glow stack theirs. The
// rest is shaders written in Go (package ui/shader): a glowing focus ring
// drawn around a field, text filled with a shader (shader.Text), and an
// avatar drawn by one. The replies are canned; Escape stops one.
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:         "Chat Glow",
			Width:         860,
			Height:        700,
			TitleBarStyle: mygo.TitleBarHiddenInset,
			Content:       ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
