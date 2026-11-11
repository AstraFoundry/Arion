//go:build windows

package e2e

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestAppFocusKeepsFront brings the app forward with the window in front
// staying there, as macOS does, rather than another of the app's windows.
func TestAppFocusKeepsFront(t *testing.T) {
	var ws []*mygo.Window
	for i := range 3 {
		ws = append(ws, newWindow(t, mygo.WindowOptions{
			Title: fmt.Sprint("Window ", i), Width: 300, Height: 200, X: 100 + 40*i, Y: 100 + 40*i,
			Content: ui.View(func(c *ui.Context) { ui.Text(c, "window") }),
		}))
	}
	for _, w := range ws {
		w.Focus()
		eventually(t, w.Title()+" focused", w.IsFocused)
		mygo.App.Focus()
		for _, other := range ws {
			if other != w && other.IsFocused() {
				t.Errorf("with %s in front, App.Focus brought %s forward", w.Title(), other.Title())
			}
		}
		if !w.IsFocused() {
			t.Errorf("App.Focus took the focus from %s", w.Title())
		}
	}
}
