//go:build !js

package e2e

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

const contextMenuPage = `<!doctype html><body style="margin:0;font:16px sans-serif">
<a id=link href="https://example.com/doc" style="position:absolute;left:0;top:0;width:200px;height:40px;display:block">Example link</a>
<input id=field value="hello" style="position:absolute;left:0;top:100px;width:200px;height:30px">
<p id=text style="position:absolute;left:0;top:180px;margin:0;font-size:24px">Some text</p>
</body>`

// contextMenuSeen is what a test's listener heard of a context menu.
type contextMenuSeen struct {
	link, linkText, selection string
	editable                  bool
	ids                       []string
}

func menuIDs(m *mygo.Menu) []string {
	var ids []string
	for _, it := range m.Items() {
		ids = append(ids, it.ID)
	}
	return ids
}

// A page's context menu tells its listeners what was right-clicked, and
// shows the menu they make of the web view's: without the items they
// removed, with those they renamed doing what they do, and with those
// they added, whose Click runs as one is chosen. PreventDefault shows
// none.
func TestPageContextMenu(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "Context menu", Width: 400, Height: 300})
	if !pageMenusHooked {
		t.Skip("right-clicking a page is not hooked here")
	}
	w.Page().LoadHTML(contextMenuPage, "")
	waitFor(t, w, "document.readyState === 'complete' && window.mygo")

	seen := make(chan contextMenuSeen, 4)
	clicked := make(chan *mygo.Window, 4)
	off := w.Page().OnContextMenu(func(e *mygo.ContextMenuEvent) {
		seen <- contextMenuSeen{e.LinkURL, e.LinkText, e.SelectionText, e.Editable, menuIDs(e.Menu)}
		e.Menu.Remove(mygo.ContextMenuReload, mygo.ContextMenuBack, mygo.ContextMenuForward, mygo.ContextMenuInspect)
		if it := e.Menu.ItemByID(mygo.ContextMenuSelectAll); it != nil {
			it.SetLabel("Select Everything")
		}
		e.Menu.Append(mygo.Separator(), &mygo.MenuItem{Label: "Open in Browser", Click: func(_ *mygo.MenuItem, w *mygo.Window) {
			clicked <- w
		}})
	})
	open := func(what string, x, y float64) (contextMenuSeen, []string) {
		t.Helper()
		if !rightClickPage(w, x, y) {
			t.Fatalf("%s: right-clicking the page failed", what)
		}
		var got contextMenuSeen
		select {
		case got = <-seen:
		case <-time.After(wait.Guard):
			t.Fatalf("%s: the listener never heard of the menu", what)
		}
		var labels []string
		wait.Until(t, what+": the menu shown", func() bool {
			var shown bool
			labels, shown = pageMenu()
			return shown
		})
		t.Logf("%s: %+v, shown %q", what, got, labels)
		return got, labels
	}

	got, labels := open("the link", 20, 20)
	if got.link != "https://example.com/doc" || got.linkText != "Example link" || got.editable {
		t.Errorf("the link: heard %+v", got)
	}
	if !slices.Contains(got.ids, mygo.ContextMenuCopyLink) {
		t.Errorf("the link: the web view's items %q have no %q", got.ids, mygo.ContextMenuCopyLink)
	}
	if !slices.Contains(labels, "Open in Browser") || slices.Contains(labels, "Reload") {
		t.Errorf("the link: the menu shows %q", labels)
	}
	if !choosePageMenuItem("Open in Browser") {
		t.Fatalf("the link: no item Open in Browser in %q", labels)
	}
	select {
	case cw := <-clicked:
		if cw != w {
			t.Errorf("Click got window %v, want the page's", cw)
		}
	case <-time.After(wait.Guard):
		t.Fatal("the app's item was chosen, and its Click never ran")
	}
	wait.Until(t, "the menu closed", func() bool { _, shown := pageMenu(); return !shown })

	got, _ = open("the field", 50, 115)
	if !got.editable || got.link != "" {
		t.Errorf("the field: heard %+v", got)
	}
	if !slices.Contains(got.ids, mygo.ContextMenuPaste) {
		t.Errorf("the field: the web view's items %q have no %q", got.ids, mygo.ContextMenuPaste)
	}
	// The web view's item, renamed, still does what it does. WebKit on
	// macOS has no Select All there.
	if slices.Contains(got.ids, mygo.ContextMenuSelectAll) {
		mustEval(t, w, "field.setSelectionRange(5, 5), true")
		if !choosePageMenuItem("Select Everything") {
			t.Fatal("the field: no item Select Everything")
		}
		waitFor(t, w, "field.selectionStart === 0 && field.selectionEnd === 5")
	} else {
		dismissPageMenu()
	}
	wait.Until(t, "the menu closed", func() bool { _, shown := pageMenu(); return !shown })

	// Pressing outside the field the menu focused would clear a selection
	// made while it has the focus.
	mustEval(t, w, `document.activeElement.blur(), getSelection().selectAllChildren(document.getElementById("text")), true`)
	got, _ = open("the selection", 30, 192)
	if got.selection != "Some text" || got.editable {
		t.Errorf("the selection: heard %+v", got)
	}
	if !slices.Contains(got.ids, mygo.ContextMenuCopy) {
		t.Errorf("the selection: the web view's items %q have no %q", got.ids, mygo.ContextMenuCopy)
	}
	dismissPageMenu()
	off()

	w.Page().OnContextMenu(func(e *mygo.ContextMenuEvent) {
		seen <- contextMenuSeen{link: e.LinkURL}
		e.PreventDefault()
	})
	wait.Until(t, "the menu closed", func() bool { _, shown := pageMenu(); return !shown })
	if !rightClickPage(w, 20, 20) {
		t.Fatal("right-clicking the page failed")
	}
	select {
	case <-seen:
	case <-time.After(wait.Guard):
		t.Fatal("the listener that prevents the menu never heard of it")
	}
	// A menu would show at once, as the listener returns.
	for range 10 {
		if labels, shown := pageMenu(); shown {
			dismissPageMenu()
			t.Fatalf("PreventDefault: a menu shows %q", labels)
		}
		time.Sleep(30 * time.Millisecond)
	}
}

// A page whose own contextmenu handler cancels the event shows no menu,
// and its listeners hear of none.
func TestPageContextMenuCancelled(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "Cancelled context menu", Width: 400, Height: 300})
	if !pageMenusHooked {
		t.Skip("right-clicking a page is not hooked here")
	}
	w.Page().LoadHTML(contextMenuPage+`<script>addEventListener("contextmenu", (e) => { e.preventDefault(); window.cancelled = (window.cancelled || 0) + 1 })</script>`, "")
	waitFor(t, w, "document.readyState === 'complete' && window.mygo")
	heard := make(chan bool, 1)
	w.Page().OnContextMenu(func(*mygo.ContextMenuEvent) { heard <- true })
	if !rightClickPage(w, 20, 20) {
		t.Fatal("right-clicking the page failed")
	}
	waitFor(t, w, "window.cancelled === 1")
	select {
	case <-heard:
		t.Error("the listener heard of a menu the page cancelled")
	case <-time.After(300 * time.Millisecond):
	}
	if labels, shown := pageMenu(); shown {
		dismissPageMenu()
		t.Errorf("a menu shows %q", labels)
	}
}
