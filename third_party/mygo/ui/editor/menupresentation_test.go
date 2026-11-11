package editor

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The editor's context menu draws in the window while the window draws its
// menus (ui.Context.SetMenuPresentation).
func TestCodeEditorMenuDrawnInTheWindow(t *testing.T) {
	st := &State{Buffer: NewBuffer("hello world")}
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetMenuPresentation(ui.PopoverMenus)
		View(c, st).Fill().AutoFocus()
	}, 600, 400)
	tt.Frame()
	tt.RightClickAt(300, 200)
	if menu := tt.Menu(); !slices.Contains(menu, "Select All") || !tt.HasText("Select All") {
		t.Fatalf("Menu() %q, drawn %v", menu, tt.HasText("Select All"))
	}
	if err := tt.ChooseMenuItem("Select All"); err != nil {
		t.Fatal(err)
	}
	ct := &editorTest{t: t, st: st, tt: tt}
	ct.want("[hello world]")
	if tt.HasText("Select All") {
		t.Fatal("the menu stayed open after a choice")
	}
}
