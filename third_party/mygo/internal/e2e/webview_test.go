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

// TestContentWindowWebViewScrollToContainer turns the wheel over two web
// views side by side, each in a scroll container of its own and taller
// than it, with pages taller still: over the one with ScrollToContainer
// its container scrolls and its page does not, and over the other one
// its page scrolls and its container does not.
func TestContentWindowWebViewScrollToContainer(t *testing.T) {
	var reader, page *mygo.WebView
	var readerScroll, pageScroll ui.ScrollState
	var readerY, pageY atomic.Value // float32: where the containers are
	readerY.Store(float32(0))
	pageY.Store(float32(0))
	w := newWindow(t, mygo.WindowOptions{Title: "Web view wheel", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Scroll(c).Grow(1).TrackScroll(&pageScroll).Children(func() {
				if page != nil {
					ui.WebView(c, page).Height(600)
				}
			})
			// On the right, where the wheel's point in the web view is
			// not the same in the window.
			ui.Scroll(c).Grow(1).TrackScroll(&readerScroll).Children(func() {
				if reader != nil {
					ui.WebView(c, reader).Height(600)
				}
			})
		})
		readerY.Store(readerScroll.Y)
		pageY.Store(pageScroll.Y)
	})})
	r, err := w.NewWebView(mygo.WebViewOptions{ScrollToContainer: true})
	if err != nil {
		t.Fatal(err)
	}
	p, err := w.NewWebView(mygo.WebViewOptions{})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { reader, page = r, p })
	// Ready once the page has drawn: a wheel that comes sooner, while its
	// first frame is still on the way, can find nothing to scroll yet.
	const tall = `<body style="margin:0;height:3000px"><script>requestAnimationFrame(() => requestAnimationFrame(() => { window.ready = true }))</script></body>`
	r.Page().LoadHTML(tall, "")
	p.Page().LoadHTML(tall, "")
	waitForPage(t, r.Page(), "window.ready")
	waitForPage(t, p.Page(), "window.ready")
	container := func(y *atomic.Value) float32 {
		w.Invalidate()
		return y.Load().(float32)
	}
	scrolled := func(v *mygo.WebView) float64 {
		y, _ := mygo.EvalAs[float64](v.Page(), "scrollY")
		return y
	}

	if !wheel(w, 300, 150) {
		t.Skip("wheel automation not available on this platform")
	}
	eventually(t, "the wheel over the reader to scroll its container", func() bool { return container(&readerY) > 0 })
	if !wheel(w, 100, 150) {
		t.Fatal("the second wheel")
	}
	eventually(t, "the wheel over the page to scroll the page", func() bool { return scrolled(p) > 0 })
	if y := scrolled(r); y != 0 {
		t.Errorf("the reader's page scrolled to %v, want its container to take the wheel", y)
	}
	if y := container(&pageY); y != 0 {
		t.Errorf("the page's container scrolled to %v, want the page to take the wheel", y)
	}
}

// TestContentWindowWebViewUnhandledKeys presses keys in the untrusted page
// of a web view that has the keyboard: those the page leaves unhandled
// reach the shortcuts of the element around the web view and of the
// window, as if the web view's element had the focus, while the page
// keeps those it cancels and those typed into its text field.
func TestContentWindowWebViewUnhandledKeys(t *testing.T) {
	var view *mygo.WebView
	var js, ks, ls, cmdKs, shown atomic.Int32
	w := newWindow(t, mygo.WindowOptions{Title: "Web view keys", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		if c.Shortcut(ui.Cmd, ui.KeyK) {
			cmdKs.Add(1)
		}
		if c.Shortcut(0, ui.KeyL) {
			ls.Add(1)
		}
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Text(c, "List").Width(100)
			reader := ui.Column(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
				if view != nil {
					ui.WebView(c, view).Grow(1)
					shown.Add(1)
				}
			})
			if reader.Shortcut(0, ui.KeyJ) {
				js.Add(1)
			}
			if reader.Shortcut(0, ui.KeyK) {
				ks.Add(1)
			}
		})
	})})
	v, err := w.NewWebView(mygo.WebViewOptions{Page: mygo.PageOptions{Untrusted: true}})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view = v })
	v.Page().LoadHTML(`<p>Text to read.</p><input id=i>
<script>
addEventListener("keydown", (e) => { if (e.key === "l") { e.preventDefault(); window.took = (window.took || "") + "l"; } });
window.ready = true;
</script>`, "")
	waitForPage(t, v.Page(), "window.ready")
	count := func(n *atomic.Int32) int32 {
		w.Invalidate()
		return n.Load()
	}
	// The window learns where a frame showed the web view once it
	// presented it: the frame after that one has.
	eventually(t, "two frames showing the web view", func() bool { return count(&shown) >= 2 })
	w.Focus()
	v.Focus()
	waitForPage(t, v.Page(), "document.hasFocus()")
	press := func(letter byte, cmd bool) {
		t.Helper()
		if !pressLetter(w, letter, cmd) {
			t.Skip("key automation not available on this platform")
		}
	}

	press('J', false)
	eventually(t, "J the page leaves to reach the shortcut around the web view", func() bool { return count(&js) == 1 })
	press('L', false)
	waitForPage(t, v.Page(), "window.took === 'l'")
	press('K', false)
	eventually(t, "K the page leaves to reach the shortcut around the web view", func() bool { return count(&ks) == 1 })
	if n := count(&ls); n != 0 {
		t.Errorf("L, which the page cancels, reached the window's shortcut %d times", n)
	}
	press('K', true)
	eventually(t, "Cmd/Ctrl+K the page leaves to reach the window's shortcut", func() bool { return count(&cmdKs) == 1 })

	if _, err := v.Page().Eval("document.getElementById('i').focus()"); err != nil {
		t.Fatal(err)
	}
	press('J', false)
	waitForPage(t, v.Page(), "document.getElementById('i').value === 'j'")
	if _, err := v.Page().Eval("document.getElementById('i').blur()"); err != nil {
		t.Fatal(err)
	}
	press('K', false)
	eventually(t, "K after the text field to reach the shortcut", func() bool { return count(&ks) == 2 })
	if n := count(&js); n != 1 {
		t.Errorf("J typed into the page's text field reached the shortcut too: %d times", n)
	}
}

// TestContentWindowWebViewUnhandledKeysInFrame presses a key in a frame of
// a web view's untrusted page, which the frame leaves unhandled: it
// reaches the window's shortcut as the page's own keys do.
func TestContentWindowWebViewUnhandledKeysInFrame(t *testing.T) {
	var view *mygo.WebView
	var js, shown atomic.Int32
	w := newWindow(t, mygo.WindowOptions{Title: "Web view frame keys", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		if c.Shortcut(0, ui.KeyJ) {
			js.Add(1)
		}
		if view != nil {
			ui.WebView(c, view).Fill()
			shown.Add(1)
		}
	})})
	v, err := w.NewWebView(mygo.WebViewOptions{Page: mygo.PageOptions{Untrusted: true}})
	if err != nil {
		t.Fatal(err)
	}
	w.Update(func() { view = v })
	v.Page().LoadHTML(`<p>Outside the frame.</p><iframe srcdoc="<p id=x tabindex=0>In the frame.</p>"></iframe>
<script>window.ready = true</script>`, "")
	waitForPage(t, v.Page(), "window.ready && document.querySelector('iframe').contentDocument?.getElementById('x')")
	eventually(t, "two frames showing the web view", func() bool {
		w.Invalidate()
		return shown.Load() >= 2
	})
	w.Focus()
	v.Focus()
	if _, err := v.Page().Eval("document.querySelector('iframe').contentDocument.getElementById('x').focus()"); err != nil {
		t.Fatal(err)
	}
	waitForPage(t, v.Page(), "document.hasFocus() && document.activeElement.tagName === 'IFRAME'")
	if !pressLetter(w, 'J', false) {
		t.Skip("key automation not available on this platform")
	}
	eventually(t, "J the frame leaves to reach the window's shortcut", func() bool {
		w.Invalidate()
		return js.Load() == 1
	})
}
