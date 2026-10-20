//go:build !js

package e2e

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const webViewPage = `<!doctype html><html><body style="margin:0;height:100vh;background:#fff"
	onmousedown="window.clicks = (window.clicks || 0) + 1">
<script>
mygo.call("Greeter.WindowID").then((id) => { window.windowId = id; window.ready = true; });
</script></body></html>`

// TestContentWindowWebView shows a web page in a window of native UI,
// under a popover the view opens over it: the page calls Go as a window's
// does, with the window as caller, clicks on the page reach it, those on
// the popover over it do not, and a click on the page closes the popover.
func TestContentWindowWebView(t *testing.T) {
	var view *mygo.WebView
	var open, frames, items atomic.Int32
	w := newWindow(t, mygo.WindowOptions{Title: "Web view", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
			menu := ui.Button(c, "Menu").Height(40).Width(100)
			if menu.Clicked() {
				open.Store(1)
			}
			shown := open.Load() == 1
			ui.Popover(c, menu, &shown, func() {
				if ui.Box(c).Size(150, 100).Background(ui.RGB(255, 0, 0)).Clicked() {
					items.Add(1)
				}
			})
			if !shown {
				open.Store(0)
			}
			if view != nil {
				ui.WebView(c, view).Grow(1)
			}
		})
	})})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	v, err := w.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if v.Window() != w || v.Page().WebView() != v {
		t.Error("the web view's window or page")
	}
	w.Update(func() { view = v })
	v.Page().LoadHTML(webViewPage, "")
	waitForPage(t, v.Page(), "window.ready")
	if id, err := mygo.EvalAs[float64](v.Page(), "window.windowId"); err != nil || int(id) != w.ID() {
		t.Errorf("the page called as window %v (%v), want %d", id, err, w.ID())
	}

	w.Focus()
	clicks := func() float64 {
		n, _ := mygo.EvalAs[float64](v.Page(), "window.clicks || 0")
		return n
	}
	if !click(w, 300, 200) {
		t.Skip("click automation not available on this platform")
	}
	eventually(t, "the click on the page", func() bool { return clicks() == 1 })

	click(w, 50, 20)
	eventually(t, "the popover", func() bool { return open.Load() == 1 })
	click(w, 50, 80) // on the popover, over the page
	eventually(t, "the click on the popover", func() bool { return items.Load() == 1 })
	if n := clicks(); n != 1 {
		t.Errorf("the page got %v clicks under the popover", n)
	}
	click(w, 300, 200)
	eventually(t, "the second click on the page", func() bool { return clicks() == 2 })
	eventually(t, "the popover closing", func() bool { return open.Load() == 0 })

	v.Destroy()
	if !v.IsDestroyed() {
		t.Error("Destroy left the web view")
	}
}

// TestContentWindowWebViewTab tabs through a window of native UI with a
// web view between two text inputs: Tab gives the page the keyboard at its
// first element, moves among its elements, and past the last one goes back
// to the native UI after the web view; Shift+Tab goes back through the
// page from its last element.
func TestContentWindowWebViewTab(t *testing.T) {
	var view *mygo.WebView
	var before, after string
	var frames atomic.Int32
	var native atomic.Value // the label of the input with the focus
	w := newWindow(t, mygo.WindowOptions{Title: "Web view Tab", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		frames.Add(1)
		focused := ""
		ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
			if ui.TextInput(c, &before).Label("Before").Focused() {
				focused = "Before"
			}
			if view != nil {
				ui.WebView(c, view).Grow(1)
			}
			if ui.TextInput(c, &after).Label("After").Focused() {
				focused = "After"
			}
		})
		native.Store(focused)
	})})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	v, err := w.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view = v })
	v.Page().LoadHTML(`<input id=a> <input id=b><script>window.ready = true</script>`, "")
	waitForPage(t, v.Page(), "window.ready")
	w.Focus()
	focused := func() string {
		id, _ := mygo.EvalAs[string](v.Page(), "document.hasFocus() && document.activeElement ? document.activeElement.id : ''")
		return id
	}
	nativeFocus := func() string {
		w.Invalidate()
		s, _ := native.Load().(string)
		return s
	}
	tab := func(back bool) {
		t.Helper()
		if !pressTab(w, back) {
			t.Skip("key automation not available on this platform")
		}
	}

	tab(false)
	eventually(t, "Tab to the input before the page", func() bool { return nativeFocus() == "Before" })
	tab(false)
	eventually(t, "Tab into the page's first input", func() bool { return focused() == "a" })
	tab(false)
	eventually(t, "Tab to the page's second input", func() bool { return focused() == "b" })
	tab(false)
	eventually(t, "Tab out of the page, to the input after it", func() bool { return focused() == "" && nativeFocus() == "After" })

	tab(true)
	eventually(t, "Shift+Tab into the page's last input", func() bool { return focused() == "b" })
	tab(true)
	eventually(t, "Shift+Tab to the page's first input", func() bool { return focused() == "a" })
	tab(true)
	eventually(t, "Shift+Tab out of the page, to the input before it", func() bool { return focused() == "" && nativeFocus() == "Before" })
}

// TestContentWindowWebViewDrop drops files on the page of a web view in a
// window of native UI, which gets them as it would in a window of its own,
// and beside it, where the native UI does.
func TestContentWindowWebViewDrop(t *testing.T) {
	var view *mygo.WebView
	var zone []string
	var frames atomic.Int32
	w := newWindow(t, mygo.WindowOptions{Title: "Web view drop", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		frames.Add(1)
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			if files := ui.Box(c).Width(100).Background(ui.RGB(200, 200, 200)).DroppedFiles(); files != nil {
				zone = files
			}
			if view != nil {
				ui.WebView(c, view).Grow(1)
			}
		})
	})})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	v, err := w.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view = v })
	v.Page().LoadHTML(`<body style="margin:0;height:100vh"><script>
document.addEventListener("dragover", (e) => e.preventDefault());
document.addEventListener("drop", (e) => {
	e.preventDefault();
	window.dropped = [...e.dataTransfer.files].map((f) => f.name).join(",");
});
window.ready = true;
</script></body>`, "")
	waitForPage(t, v.Page(), "window.ready")
	dir := t.TempDir()
	file := filepath.Join(dir, "page.txt")
	if err := os.WriteFile(file, []byte("page"), 0o644); err != nil {
		t.Fatal(err)
	}
	dropped, ok := dragFiles(w, 250, 150, []string{file})
	if !ok {
		t.Skip("drags routed by the window's drop target are Windows's")
	}
	if !dropped {
		t.Error("the page did not take the drop")
	}
	waitForPage(t, v.Page(), "window.dropped === 'page.txt'")
	// Beside the page, the drop target hands the drag to the native UI,
	// whose element takes the files as in TestContentWindowFileDrop.
	if over, dropped, _ := dropFiles(w, 50, 150, []string{file}); !over || !dropped {
		t.Errorf("the native UI beside the page took the drag %v and the drop %v", over, dropped)
	}
	eventually(t, "the drop beside the page", func() bool {
		var n int
		mygo.RunOnMain(func() { n = len(zone) })
		return n == 1
	})
}

// TestContentWindowWebViewEditMenu makes a web view before its window drew
// a frame, and edits its page through the Edit menu once the page has the
// keyboard: the menu's actions go to the page, as in a window of its own.
func TestContentWindowWebViewEditMenu(t *testing.T) {
	var view *mygo.WebView
	w := newWindow(t, mygo.WindowOptions{Title: "Web view edit", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		ui.Column(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Button(c, "Native").Height(40)
			if view != nil {
				ui.WebView(c, view).Grow(1)
			}
		})
	})})
	v, err := w.NewWebView(mygo.WebViewOptions{}) // before the first frame
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view = v })
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleEditMenu}}))
	defer mygo.App.SetMenu(nil)
	clipboard := mygo.Clipboard.ReadText()
	defer mygo.Clipboard.WriteText(clipboard)
	v.Page().LoadHTML(`<body style="margin:0;height:100vh;background:#f00">
<div id=e contenteditable style="height:100px">hello</div><script>window.ready = true</script></body>`, "")
	waitForPage(t, v.Page(), "window.ready")
	w.Focus()
	v.Focus()
	if _, err := v.Page().Eval("document.getElementById('e').focus()"); err != nil {
		t.Fatal(err)
	}
	text := func() string {
		s, _ := mygo.EvalAs[string](v.Page(), "document.getElementById('e').textContent")
		return s
	}
	menu := func(item string) {
		t.Helper()
		if err := activateMenu(w, "Edit", item); err != nil {
			t.Skipf("menu automation: %v", err)
		}
	}
	menu("Select All")
	waitForPage(t, v.Page(), "getSelection().toString() === 'hello'")
	mygo.Clipboard.WriteText("before copy")
	menu("Copy")
	eventually(t, "Copy to copy the page's selection", func() bool { return mygo.Clipboard.ReadText() == "hello" })
	menu("Cut")
	eventually(t, "Cut to take the text out of the page", func() bool { return text() == "" })
	menu("Paste")
	eventually(t, "Paste to put it back", func() bool { return text() == "hello" })
	menu("Undo")
	eventually(t, "Undo to take the paste back", func() bool { return text() == "" })
	menu("Undo")
	eventually(t, "Undo to take the cut back", func() bool { return text() == "hello" })
	if _, err := v.Page().Eval("getSelection().collapseToEnd()"); err != nil {
		t.Fatal(err)
	}
	if handled, ok := pressShortcut("a", false); ok {
		if !handled {
			t.Error("Select All's shortcut found no menu item")
		}
		waitForPage(t, v.Page(), "getSelection().toString() === 'hello'")
	}
}
