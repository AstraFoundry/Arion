//go:build !js

package e2e

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const flyoutPage = `<body style="margin:0;background:transparent">` +
	`<button id="b" style="width:100%;height:100px;background:#fc0" onclick="window.clicked=(window.clicked||0)+1">x</button></body>`

// TestFlyout places a flyout below an anchor in its parent, beyond the
// parent's bottom edge, flips it above where the work area ends, and
// keeps it next to the anchor as the parent moves.
func TestFlyout(t *testing.T) {
	work := mygo.Screen.PrimaryDisplay().WorkArea
	parent := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200, X: work.X + 100, Y: work.Y + 100})
	anchor := mygo.Rectangle{X: 20, Y: 150, Width: 80, Height: 24}
	fl := mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Anchor: anchor, Gap: 4, Width: 220, Height: 160})
	defer fl.Destroy()
	if !fl.IsFlyout() || fl.Parent() != parent || !fl.IsVisible() {
		t.Fatalf("flyout: IsFlyout %v, parent %v, visible %v", fl.IsFlyout(), fl.Parent() == parent, fl.IsVisible())
	}
	below := func() mygo.Rectangle {
		c := parent.ContentBounds()
		return mygo.Rectangle{X: c.X + anchor.X, Y: c.Y + anchor.Y + anchor.Height + 4, Width: 220, Height: 160}
	}
	eventuallyAt(t, "the flyout below its anchor", fl, below)
	if p, f := parent.Bounds(), fl.Bounds(); f.Y+f.Height <= p.Y+p.Height {
		t.Errorf("flyout %+v does not extend beyond its parent %+v", f, p)
	}

	// A click in a flyout that is not focusable reaches its page and
	// leaves the keyboard in the parent.
	fl.Page().LoadHTML(flyoutPage, "")
	waitFor(t, fl, "document.getElementById('b')")
	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	if click(fl, 50, 50) {
		waitFor(t, fl, "window.clicked === 1")
		if fl.IsFocused() || !parent.IsFocused() {
			t.Errorf("after a click in the flyout: flyout focused %v, parent focused %v", fl.IsFocused(), parent.IsFocused())
		}
	}

	// It follows the parent.
	parent.SetPosition(work.X+160, work.Y+140)
	eventuallyAt(t, "the flyout following its parent", fl, below)

	// Where the work area has no room below, it goes above.
	parent.SetPosition(work.X+100, work.Y+work.Height-210)
	eventuallyAt(t, "the flyout above its anchor", fl, func() mygo.Rectangle {
		c := parent.ContentBounds()
		return mygo.Rectangle{X: c.X + anchor.X, Y: c.Y + anchor.Y - 4 - 160, Width: 220, Height: 160}
	})

	// SetAnchor and SetSize place it again.
	parent.SetPosition(work.X+100, work.Y+100)
	anchor = mygo.Rectangle{X: 200, Y: 10, Width: 40, Height: 20}
	fl.SetAnchor(anchor)
	fl.SetSize(120, 90)
	eventuallyAt(t, "the flyout at its new anchor and size", fl, func() mygo.Rectangle {
		c := parent.ContentBounds()
		return mygo.Rectangle{X: c.X + anchor.X, Y: c.Y + anchor.Y + anchor.Height + 4, Width: 120, Height: 90}
	})

	// It closes with its parent.
	parent.Destroy()
	eventually(t, "the flyout closed with its parent", fl.IsDestroyed)
}

// eventuallyAt waits for a flyout to be at the bounds want returns.
func eventuallyAt(t *testing.T, what string, fl *mygo.Window, want func() mygo.Rectangle) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for fl.Bounds() != want() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s: at %+v, not %+v", what, fl.Bounds(), want())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestFocusableFlyout gives a focusable flyout the keyboard and closes it
// when the parent takes the keyboard back, unless OnClose prevents it.
func TestFocusableFlyout(t *testing.T) {
	parent := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200})
	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	fl := mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Anchor: mygo.Rectangle{X: 10, Y: 10}, Width: 200, Height: 120, Focusable: true})
	defer fl.Destroy()
	eventually(t, "the flyout focused", fl.IsFocused)

	keep := true
	closes := 0
	fl.OnClose(func(e *mygo.CloseEvent) {
		closes++
		if keep {
			e.PreventDefault()
		}
	})
	parent.Focus()
	eventually(t, "the flyout asked to close", func() bool { return closes == 1 })
	if fl.IsDestroyed() {
		t.Fatal("the flyout closed though OnClose prevented it")
	}

	keep = false
	fl.Focus()
	eventually(t, "the flyout focused again", fl.IsFocused)
	parent.Focus()
	eventually(t, "the flyout closed", fl.IsDestroyed)
	eventually(t, "the parent focused", parent.IsFocused)
}

// TestFlyoutModalWindow opens a window modal to a focusable flyout, which
// takes the keyboard and keeps the flyout open, over it; the flyout gets
// the keyboard back as it closes, and closes it as it closes.
func TestFlyoutModalWindow(t *testing.T) {
	parent := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200})
	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	fl := mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Anchor: mygo.Rectangle{X: 10, Y: 10}, Width: 160, Height: 100, Focusable: true, Content: ui.View(func(c *ui.Context) {})})
	defer fl.Destroy()
	eventually(t, "the flyout focused", fl.IsFocused)
	var closes atomic.Int32
	fl.OnClose(func(*mygo.CloseEvent) { closes.Add(1) })

	open := func() *mygo.Window {
		t.Helper()
		dialog := mygo.NewWindow(mygo.WindowOptions{Parent: fl, Modal: true, Width: 360, Height: 160, Content: ui.View(func(c *ui.Context) {})})
		eventually(t, "the modal window focused", dialog.IsFocused)
		return dialog
	}
	dialog := open()
	if b, f := dialog.Bounds(), fl.Bounds(); b.X > f.X || b.X+b.Width < f.X+f.Width {
		t.Errorf("the modal window at %+v does not cover the flyout at %+v", b, f)
	}
	dialog.Close()
	eventually(t, "the modal window closed", dialog.IsDestroyed)
	eventually(t, "the flyout focused again", fl.IsFocused)
	if fl.IsDestroyed() || closes.Load() != 0 {
		t.Fatalf("the flyout closed (%d times asked) for its modal window", closes.Load())
	}

	dialog = open()
	fl.Close()
	eventually(t, "the modal window closed with the flyout", dialog.IsDestroyed)
	eventually(t, "the flyout closed", fl.IsDestroyed)
}

// TestPopoverFlyout shows a flyout in a popover of the system where there
// is one (macOS), which closes as the user clicks elsewhere when it is
// focusable, and is a plain flyout elsewhere.
func TestPopoverFlyout(t *testing.T) {
	parent := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200})
	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	fl := mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Anchor: mygo.Rectangle{X: 100, Y: 20, Width: 80, Height: 24}, Width: 200, Height: 100, Popover: true})
	defer fl.Destroy()
	fl.Page().LoadHTML(flyoutPage, "")
	waitFor(t, fl, "document.getElementById('b')")
	if !fl.IsVisible() {
		t.Fatal("the popover does not show")
	}
	b, c := fl.Bounds(), parent.ContentBounds()
	if b.Width < 200 || b.Height < 100 || b.Y < c.Y+20+24 {
		t.Errorf("popover at %+v, below the anchor at %+v in %+v", b, mygo.Rectangle{X: 100, Y: 20, Width: 80, Height: 24}, c)
	}
	fl.Hide()
	eventually(t, "the popover hidden", func() bool { return !fl.IsVisible() })
	fl.ShowInactive()
	eventually(t, "the popover shown again", fl.IsVisible)
	if fl.IsDestroyed() {
		t.Fatal("hiding the popover closed it")
	}
	if appClick(fl, 50, 50) {
		waitFor(t, fl, "window.clicked === 1")
	}
	if fl.IsDestroyed() {
		t.Fatal("a click in the popover closed it")
	}

	focusable := mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Anchor: mygo.Rectangle{X: 10, Y: 150}, Width: 150, Height: 80, Popover: true, Focusable: true})
	defer focusable.Destroy()
	eventually(t, "the focusable popover focused", focusable.IsFocused)
	closes := 0
	focusable.OnClose(func(*mygo.CloseEvent) { closes++ })
	if appClick(parent, 280, 10) {
		eventually(t, "the focusable popover closed", focusable.IsDestroyed)
		if closes != 1 {
			t.Errorf("OnClose heard %d closes", closes)
		}
	}
}

// TestFlyoutToggle opens a focusable flyout from a button of its parent,
// and closes it with a second click on the button, which the closing
// does not turn into opening it again: quick, or held as a user clicks,
// whose press closes the flyout before the release would click the button
// where the press goes on (macOS) but for the flyout's anchor. The click
// hooks return once the app handled the click: a frame the parent builds
// after that reports a click the button got.
func TestFlyoutToggle(t *testing.T) {
	var parent, fl *mygo.Window // main thread only
	var shown atomic.Pointer[mygo.Window]
	var builds atomic.Int64
	opened := make(chan struct{}, 4)
	closed := make(chan struct{}, 4)
	view := func(c *ui.Context) {
		builds.Add(1)
		if !ui.Button(c.Key("toggle"), "Toggle").Width(120).Height(30).Clicked() {
			return
		}
		if fl != nil {
			fl.Close()
			return
		}
		fl = mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Anchor: mygo.Rectangle{Width: 120, Height: 30}, Width: 150, Height: 100, Focusable: true, Content: ui.View(func(c *ui.Context) {})})
		shown.Store(fl)
		fl.OnClosed(func() {
			fl = nil
			closed <- struct{}{}
		})
		opened <- struct{}{}
	}
	drawn := make(chan struct{}, 1)
	mygo.RunOnMain(func() {
		parent = mygo.NewWindow(mygo.WindowOptions{Width: 300, Height: 200, Hidden: true, Content: ui.View(view)})
		parent.OnReadyToShow(func() { drawn <- struct{}{} })
		parent.Show()
	})
	defer parent.Destroy()
	wait := func(ch chan struct{}, what string) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for %s", what)
		}
	}
	// built waits for the parent to build a frame after the clicks the
	// hooks handled, which reports a click of the button.
	built := func() {
		t.Helper()
		n := builds.Load()
		parent.Invalidate()
		eventually(t, "a frame of the parent built", func() bool { return builds.Load() > n })
	}
	wait(drawn, "the parent drawn")
	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	// The button lays out at the top-left of the window.
	for i, closing := range []func(*mygo.Window, float64, float64) bool{heldAppClick, appClick} {
		if !appClick(parent, 60, 15) {
			t.Skip("clicks not available on this platform")
		}
		wait(opened, "the flyout opened")
		// A user clicks again once the flyout shows, with the keyboard.
		eventually(t, "the flyout focused", func() bool { return shown.Load().IsFocused() || len(closed) > 0 })
		select {
		case <-closed:
			// It closes as another app takes the keyboard.
			t.Fatal("the flyout closed before the click that closes it: was the app deactivated?")
		default:
		}
		closing(parent, 60, 15)
		wait(closed, "the flyout closed")
		built()
		select {
		case <-opened:
			t.Fatalf("the click that closed the flyout opened it again (held %v)", i == 0)
		default:
		}
	}
}

// TestFlyoutEscape closes a focusable flyout of native UI, or showing a
// page, on an Escape its content does not handle, and keeps one whose
// content does.
func TestFlyoutEscape(t *testing.T) {
	parent := newWindow(t, mygo.WindowOptions{Width: 300, Height: 200})
	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	var handled, received atomic.Bool
	native := mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Width: 150, Height: 100, Focusable: true, Content: ui.View(func(c *ui.Context) {
		if handled.Load() && c.Shortcut(0, ui.KeyEscape) {
			received.Store(true)
		}
	})})
	defer native.Destroy()
	eventually(t, "the flyout focused", native.IsFocused)
	// The view asks for Escape from the frame that handles the key on,
	// whether or not a frame built since showed it.
	handled.Store(true)
	if !pressEscape(native) {
		t.Skip("keys not available on this platform")
	}
	// Posting the native key does not wait for the content to receive it.
	eventually(t, "the content handling Escape", func() bool { return received.Load() || native.IsDestroyed() })
	if native.IsDestroyed() {
		t.Fatal("an Escape the content handled closed the flyout")
	}
	// It no longer does, from the frame that handles the key on.
	handled.Store(false)
	pressEscape(native)
	eventually(t, "the flyout of native UI closed", native.IsDestroyed)

	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	page := mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Width: 150, Height: 100, Focusable: true})
	defer page.Destroy()
	page.Page().LoadHTML(`<input id=i autofocus>`, "")
	waitFor(t, page, "document.activeElement && document.activeElement.id === 'i'")
	eventually(t, "the page's flyout focused", page.IsFocused)
	pressEscape(page)
	eventually(t, "the page's flyout closed", page.IsDestroyed)

	// A flyout of native UI showing a web view whose page has the
	// keyboard closes on an Escape the page leaves.
	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	var view *mygo.WebView
	var shown atomic.Int32
	reader := mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Width: 150, Height: 100, Focusable: true, Content: ui.View(func(c *ui.Context) {
		if view != nil {
			ui.WebView(c, view).Fill()
			shown.Add(1)
		}
	})})
	defer reader.Destroy()
	v, err := reader.NewWebView(mygo.WebViewOptions{Page: mygo.PageOptions{Untrusted: true}})
	if err != nil {
		t.Fatal(err)
	}
	reader.Update(func() { view = v })
	v.Page().LoadHTML(`<p>Text to read.</p><script>window.ready = true</script>`, "")
	waitForPage(t, v.Page(), "window.ready")
	eventually(t, "two frames showing the web view", func() bool {
		reader.Invalidate()
		return shown.Load() >= 2
	})
	eventually(t, "the reader focused", reader.IsFocused)
	v.Focus()
	waitForPage(t, v.Page(), "document.hasFocus()")
	pressEscape(reader)
	eventually(t, "the reader closed", reader.IsDestroyed)

	// A popover of the system's (macOS) closes so too.
	parent.Focus()
	eventually(t, "the parent focused", parent.IsFocused)
	popover := mygo.NewFlyout(mygo.FlyoutOptions{Parent: parent, Width: 150, Height: 100, Focusable: true, Popover: true, Content: ui.View(func(c *ui.Context) {})})
	defer popover.Destroy()
	eventually(t, "the popover focused", popover.IsFocused)
	pressEscape(popover)
	eventually(t, "the popover closed", popover.IsDestroyed)
}

// TestTrayFlyout places a flyout next to a tray icon, below the macOS menu
// bar's or above an icon of a taskbar at the bottom, and in the middle of
// the primary display's work area where the icon has no bounds. (Clicks
// on the icon, which toggle it, are TestTrayPopoverToggle's on macOS.)
func TestTrayFlyout(t *testing.T) {
	tray, err := mygo.NewTray(mygo.TrayOptions{Icon: squarePNG(), ToolTip: "flyout"})
	if err != nil {
		t.Skip("no tray: ", err)
	}
	defer tray.Destroy()
	waitTrayLaidOut(tray)
	for _, popover := range []bool{false, true} {
		fl := mygo.NewFlyout(mygo.FlyoutOptions{Tray: tray, Placement: mygo.PlacementBottom, Gap: 4, Width: 200, Height: 120, Focusable: true, Popover: popover,
			Content: ui.View(func(c *ui.Context) {})})
		eventually(t, "the flyout shown", fl.IsVisible)
		if popover && runtime.GOOS == "darwin" {
			// AppKit places a popover, its arrow pointing at the icon.
			eventually(t, "the popover below the icon", func() bool {
				b, icon := fl.Bounds(), tray.Bounds()
				return b.Y > icon.Y && b.Y <= icon.Y+icon.Height+30 && b.X < icon.X+icon.Width/2 && b.X+b.Width > icon.X+icon.Width/2
			})
		} else if icon := tray.Bounds(); icon.Width > 0 && icon.Height > 0 {
			eventually(t, "the flyout next to the icon", func() bool {
				b := fl.Bounds()
				return (b.Y == icon.Y+icon.Height+4 || b.Y+b.Height == icon.Y-4) && b.Width == 200 && b.Height == 120
			})
		} else if runtime.GOOS != "linux" || os.Getenv("WAYLAND_DISPLAY") == "" {
			work := mygo.Screen.PrimaryDisplay().WorkArea
			want := mygo.Rectangle{X: work.X + (work.Width-200)/2, Y: work.Y + (work.Height-120)/2, Width: 200, Height: 120}
			eventuallyAt(t, "the flyout in the middle of the work area", fl, func() mygo.Rectangle { return want })
		}
		fl.Destroy()
	}
}

// waitTrayLaidOut waits for the menu bar to lay a tray icon out, which it
// does after the icon was made, off the screen until then, as a click on
// it would find it.
func waitTrayLaidOut(tray *mygo.Tray) {
	onScreen := func() bool {
		r := tray.Bounds()
		for _, d := range mygo.Screen.Displays() {
			b := d.Bounds
			if r.Width > 0 && r.Height > 0 && r.X >= b.X && r.X+r.Width <= b.X+b.Width && r.Y >= b.Y-2 && r.Y+r.Height <= b.Y+b.Height {
				return true
			}
		}
		return false
	}
	last, still := tray.Bounds(), 0
	for deadline := time.Now().Add(3 * time.Second); still < 10 && time.Now().Before(deadline); {
		time.Sleep(30 * time.Millisecond)
		if b := tray.Bounds(); b == last && onScreen() {
			still++
		} else {
			last, still = b, 0
		}
	}
}

// squarePNG returns a black square, 32 pixels wide.
func squarePNG() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	draw.Draw(img, image.Rect(4, 4, 28, 28), image.Black, image.Point{}, draw.Src)
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}
