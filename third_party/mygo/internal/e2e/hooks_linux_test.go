//go:build linux && (amd64 || arm64)

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/internal/wait"
)

func activateMenu(w *mygo.Window, path ...string) (err error) {
	mygo.RunOnMain(func() { err = linux.TestActivateMenuItem(w.NativeHandle(), path...) })
	return err
}

func menuItemEnabled(w *mygo.Window, path ...string) (enabled bool, err error) {
	mygo.RunOnMain(func() { enabled, err = linux.TestMenuItemEnabled(w.NativeHandle(), path...) })
	return enabled, err
}

// Key presses and dialogs need an input device under Xvfb; not covered.
func pressShortcut(string, bool) (bool, bool) { return false, false }

func endSheet(*mygo.Window) (bool, bool) { return false, false }

// fake runs a hook that fakes input through XTEST, and returns once the
// app handled that input, as the hooks of macOS and Windows do: the X
// server makes the events and the app gets them later, on the main
// thread, after other work queued there. A test that went on at once
// would race them: Edit > Copy while the press of a drag was handled but
// not its release, a composition the click's press then ended.
func fake(hook func() bool) (ok bool) {
	mygo.RunOnMain(func() { ok = hook() })
	if !ok {
		return false
	}
	// The hook synced with the X server: the events wait in the app's
	// queue, and it has handled them once it has no events left. All but
	// a last motion, which GDK holds until its next frame to merge it with
	// the ones that follow: a test moving the pointer alone waits for
	// what the move does.
	wait.Poll(func() (pending bool) {
		mygo.RunOnMain(func() { pending = linux.TestInputPending() })
		return !pending
	})
	return true
}

// Clicks reach windows showing native UI only, through XTEST.
func click(w *mygo.Window, x, y float64) bool {
	return fake(func() bool { return linux.TestClickSurface(w.NativeHandle(), x, y) })
}

// clickPage clicks a window's page, through XTEST.
func clickPage(w *mygo.Window, x, y float64) bool {
	return fake(func() bool { return linux.TestClickPage(w.NativeHandle(), x, y) })
}

// appClick is click, already through XTEST.
func appClick(w *mygo.Window, x, y float64) bool { return click(w, x, y) }

// heldAppClick is appClick: the grab of a focusable flyout eats the press
// that closes it, however long it is held.
func heldAppClick(w *mygo.Window, x, y float64) bool { return appClick(w, x, y) }

func sideButton(w *mygo.Window, back bool) bool {
	return fake(func() bool { return linux.TestSideButton(w.NativeHandle(), back) })
}

// pinch pinches the touchpad at (x, y) by scale with GDK's events, which
// only Wayland makes.
func pinch(w *mygo.Window, x, y, scale float64) (got float64, ok bool) {
	mygo.RunOnMain(func() { ok = linux.TestPinchSurface(w.NativeHandle(), x, y, scale) })
	return scale, ok
}

func drag(w *mygo.Window, points [][2]float64) bool {
	return fake(func() bool { return linux.TestDragSurface(w.NativeHandle(), points) })
}

// wheel turns the mouse's wheel a notch down at (x, y), through XTEST.
func wheel(w *mygo.Window, x, y float64) bool {
	return fake(func() bool { return linux.TestWheelSurface(w.NativeHandle(), x, y) })
}

func webViewAttached(*mygo.Window) (bool, bool) { return false, false }

// Collection behaviors and deactivating the app are macOS's.
func collectionBehavior(*mygo.Window) (uint, bool)               { return 0, false }
func deactivateApp() bool                                        { return false }
func material(*mygo.Window) (string, int, float64, bool, bool)   { return "", 0, 0, false, false }
func materialState(*mygo.Window) (int, bool)                     { return 0, false }
func roundedBacking(*mygo.Window) (bool, float64, float64, bool) { return false, 0, 0, false }
func windowBacking(*mygo.Window) (bool, float64, bool)           { return false, 0, false }
func appActive() (bool, bool)                                    { return false, false }

// WebKitGTK docks the inspector inside the web view.
func dockDevTools(*mygo.Window) bool { return false }

func dockedDevToolsPlace(*mygo.Window) string { return "" }

func dismissPopups() (n int, supported bool) {
	mygo.RunOnMain(func() { n = linux.TestDismissPopups() })
	return n, true
}

func setDroppedFiles(w *mygo.Window, paths []string) bool {
	mygo.RunOnMain(func() { linux.TestSetDroppedFiles(w.NativeHandle(), paths) })
	return true
}

// The progress bar is shown by the shell, out of the app's reach.
func dockTileImage() ([]byte, bool) { return nil, false }

func defaultURLHandler(scheme string) (id string, supported bool) {
	mygo.RunOnMain(func() { id = linux.TestDefaultURLHandler(scheme) })
	return id, true
}

func dockMenu(int) ([]string, bool) { return nil, false }

// recentDocumentRecorded reports whether path is among the user's recently
// used files.
func recentDocumentRecorded(path string) (recorded, supported bool) {
	mygo.RunOnMain(func() { recorded = linux.TestHasRecentItem(path) })
	return recorded, true
}

// pressCtrlShiftK presses Ctrl+Shift+K like a keyboard.
func pressCtrlShiftK() bool {
	return fake(func() bool { return linux.TestPressKeys("Control_L", "Shift_L", "k") })
}

// usePortalShortcuts makes global shortcuts bind through the XDG desktop
// portal, as on Wayland, when the desktop offers it.
func usePortalShortcuts() (restore func(), ok bool) {
	mygo.RunOnMain(func() { ok = linux.TestUsePortalShortcuts(true) })
	return func() { mygo.RunOnMain(func() { linux.TestUsePortalShortcuts(false) }) }, ok
}

// pressKeys presses keys, X keysym names such as "Control_L", together.
func pressKeys(keys ...string) bool {
	return fake(func() bool { return linux.TestPressKeys(keys...) })
}

// Only macOS windows have a toolbar.
func fullScreenHidesToolbar(*mygo.Window) (bool, bool) { return false, false }

// Only macOS windows have traffic lights.
func trafficLights(*mygo.Window) (float64, float64, bool) { return 0, 0, false }
func setWindowAppearance(*mygo.Window, bool)              {}

// movePointer moves the pointer to a point of the screen, and pressButton
// presses or releases the first mouse button, like a mouse.
func movePointer(x, y int) bool {
	return fake(func() bool { return linux.TestMovePointer(x, y) })
}

func pressButton(press bool) bool {
	return fake(func() bool { return linux.TestPressButton(press) })
}

func resizeCursor(w *mygo.Window) (name string, supported bool) {
	mygo.RunOnMain(func() { name = linux.TestResizeCursor(w.NativeHandle()) })
	return name, true
}

func menuBarShown(w *mygo.Window) (shown, supported bool) {
	mygo.RunOnMain(func() { shown, _ = linux.TestMenuBarState(w.NativeHandle()) })
	return shown, true
}

func activateAccelerator(w *mygo.Window, accel string) (handled, supported bool) {
	mygo.RunOnMain(func() { handled = linux.TestActivateAccelerator(w.NativeHandle(), accel) })
	return handled, true
}

// enterMenuBar presses key, an X keysym name, on the window: the bar should
// show with its first menu open. Then it closes the menus, as Escape does.
func enterMenuBar(w *mygo.Window, key string) (during, after, supported bool) {
	if !w.IsFocused() || !pressKeys(key) {
		return false, false, false
	}
	state := func() (shown, open bool) {
		mygo.RunOnMain(func() { shown, open = linux.TestMenuBarState(w.NativeHandle()) })
		return shown, open
	}
	during = wait.Poll(func() bool { shown, open := state(); return shown && open })
	mygo.RunOnMain(func() { linux.TestCloseMenuBar(w.NativeHandle()) })
	wait.Poll(func() bool { after, _ = state(); return !after })
	return during, after, true
}

// Alt opens the menus in the bar here (enterMenuBar), and the items of GTK
// menus have no letters.
func openMenus(*mygo.Window, byte) bool { return false }
func closeMenus()                       {}

// titleButtons returns GTK's title buttons over the page of a window with
// a hidden title bar.
func titleButtons(w *mygo.Window) (names []string, supported bool) {
	mygo.RunOnMain(func() { names = linux.TestTitleButtons(w.NativeHandle()) })
	return names, true
}

func pressTitleButton(w *mygo.Window, name string) (ok bool) {
	mygo.RunOnMain(func() { ok = linux.TestPressTitleButton(w.NativeHandle(), name) })
	return ok
}

func topNonClient(*mygo.Window) (int32, int32, bool) { return 0, 0, false }

func menuBarColors(*mygo.Window) (uint32, uint32, bool) { return 0, 0, false }

func titleButtonColor(*mygo.Window, string) (uint8, uint8, uint8, bool) { return 0, 0, 0, false }

// Linux shows no material.
func composition(*mygo.Window) (bool, bool, bool) { return false, false, false }
func canCompose() bool                            { return false }
func loseComposition(bool) bool                   { return false }

// WebKitGTK creates web views with their windows.
func failWebViews(int) bool { return false }

// A Control-click is a secondary click on macOS only.
func controlClick(*mygo.Window, float64, float64) bool { return false }

func secondaryClickAsPrimary(*mygo.Window, float64, float64) bool { return false }

// composeOver does what an input method that edits typed text does through
// GTK: it deletes the text around the caret it composes over, then
// composes or commits.
func composeOver(w *mygo.Window, text string, caret int, commit bool, from, length int) (ok bool) {
	mygo.RunOnMain(func() { ok = linux.TestComposeOver(w.NativeHandle(), text, caret, commit, from, length) })
	return ok
}

// inputClient returns the caret and the text around it that input methods
// get through GTK, which knows no selection, and no composition in it.
func inputClient(w *mygo.Window) (selected [2]int, document string, ok bool) {
	mygo.RunOnMain(func() {
		var caret int
		document, caret, ok = linux.TestSurrounding(w.NativeHandle())
		selected = [2]int{caret, 0}
	})
	return selected, document, ok
}

func dropFiles(w *mygo.Window, x, y float64, paths []string) (over, dropped, ok bool) {
	mygo.RunOnMain(func() { over, dropped = linux.TestDropFiles(w.NativeHandle(), x, y, paths) })
	return over, dropped, true
}

// The roles of elements in ATK.
const roleText, roleButton, roleCheckBox, roleTextField, roleSlider = "label", "button", "check box", "entry", "slider"

// roleListItem is the role of the rows of a List.
const roleListItem = "list item"

func accessibility(w *mygo.Window) (nodes []accessNode, ok bool) {
	mygo.RunOnMain(func() {
		var list []linux.TestAccessNode
		list, ok = linux.TestAccessibility(w.NativeHandle())
		for _, n := range list {
			nodes = append(nodes, accessNode{n.Role, n.Label, n.Value})
		}
	})
	return nodes, ok
}

func accessPerform(w *mygo.Window, label, action, value string) (ok bool) {
	mygo.RunOnMain(func() { ok = linux.TestAccessibilityPerform(w.NativeHandle(), label, action, value) })
	return ok
}

// Typing into native UI is only automated on macOS.
func clickAndType(*mygo.Window, float64, float64, string) bool { return false }
func compose(*mygo.Window, string, int, bool) bool             { return false }

// glSurface returns how the native UI of a window draws ("opengl",
// "memory", "cairo") and, for a GtkGLArea, the pixels of its framebuffer,
// premultiplied BGRA rows from the top.
func glSurface(w *mygo.Window) (how string, pix []byte, width, height int, supported bool) {
	mygo.RunOnMain(func() { how, pix, width, height = linux.TestSurfaceGL(w.NativeHandle()) })
	return how, pix, width, height, true
}

// memoryUI has the windows of native UI created from now on draw in
// memory, in a GtkDrawingArea, as without a GPU.
func memoryUI(t *testing.T) bool {
	t.Setenv("MYGO_GPU", "0")
	return true
}

// surfaceOnScreen returns, as a PNG, what the display shows of a window's
// native UI.
// Only Windows reads the screen over native UI.
func screenColor(*mygo.Window, float64, float64) (uint8, uint8, uint8, bool) { return 0, 0, 0, false }

func surfaceOnScreen(w *mygo.Window) (png []byte, supported bool) {
	mygo.RunOnMain(func() { png = linux.TestSurfaceOnScreen(w.NativeHandle()) })
	return png, true
}

// rightClick clicks (x, y) in a window showing native UI with the
// secondary button, through XTEST.
func rightClick(w *mygo.Window, x, y float64) bool {
	return fake(func() bool { return linux.TestRightClickSurface(w.NativeHandle(), x, y) })
}

// popupMenus returns the labels of the items of the context menus shown.
func popupMenus() (menus [][]string, supported bool) {
	mygo.RunOnMain(func() { menus = linux.TestPopups() })
	return menus, true
}

func choosePopupItem(label string) (ok bool) {
	mygo.RunOnMain(func() { ok = linux.TestChoosePopupItem(label) })
	return ok
}

func popupItemEnabled(label string) (enabled, ok bool) {
	mygo.RunOnMain(func() { enabled, ok = linux.TestPopupItemEnabled(label) })
	return enabled, ok
}

// Only macOS has key-value observing.
func observe(*mygo.Window) (func(), bool) { return nil, false }

// Input methods keep the keys typed while they compose from the content
// themselves here (GTK's filtering, IMM32's VK_PROCESSKEY), which a
// composition the test makes up does not show.
func pressKey(*mygo.Window, uint16, string) bool { return false }

func pressKeyComposing(*mygo.Window, uint16, string, string, string) bool { return false }

// X routes a drag to the window under the pointer, which
// dropFiles, dropping on the content's, skips.
func dragFiles(*mygo.Window, float64, float64, []string) (bool, bool) { return false, false }

// pressTab presses Tab, with Shift when back, as the keyboard does.
func pressTab(_ *mygo.Window, back bool) bool {
	if back {
		return pressKeys("Shift_L", "Tab")
	}
	return pressKeys("Tab")
}

// AppKit's older accessibility API is macOS's.
func axAttribute(*mygo.Window, string, string) (string, bool, bool, bool) {
	return "", false, false, false
}

// pressEscape presses Escape, as the keyboard does.
func pressEscape(*mygo.Window) bool { return pressKeys("Escape") }

// GTK shows no document but the title.
func documentChrome(w *mygo.Window) (title, file string, icon, edited, supported bool) {
	mygo.RunOnMain(func() { title = linux.TestWindowTitle(w.NativeHandle()) })
	return title, "", false, false, true
}

func sheet(*mygo.Window) ([]string, []string, bool, bool) { return nil, nil, false, false }
func endSheetButton(*mygo.Window, int) bool               { return false }

// Panels' window styles are Windows'.
func panelStyles(*mygo.Window) (bool, bool, bool, bool) { return false, false, false, false }

// throttlingOff reports whether the page of w, or of its first web view,
// is set up to keep running while it can't be seen, and how many of the
// settings that take the platform has.
func throttlingOff(w *mygo.Window, webView bool) (off bool, settings int) {
	mygo.RunOnMain(func() { off, settings = linux.TestBackgroundThrottlingOff(w.NativeHandle(), webView) })
	return off, settings
}

// mockCapture gives a window's page WebKit's mock camera and microphone.
func mockCapture(w *mygo.Window) (ok bool) {
	mygo.RunOnMain(func() { ok = linux.TestMockCaptureDevices(w.NativeHandle()) })
	return ok
}

// pageGestures reports whether a swipe navigates and a pinch zooms the page
// of a window, or with webView 0 or more, of its web view of that index.
func pageGestures(w *mygo.Window, webView int) (swipe, pinch, ok bool) {
	mygo.RunOnMain(func() { swipe, pinch, ok = linux.TestPageGestures(w.NativeHandle(), webView) })
	return swipe, pinch, ok
}

// pinchPage pinches the touchpad over the page of a window, or of its web
// view, from 1 through scales, with GDK's events, which only Wayland
// makes.
func pinchPage(w *mygo.Window, webView int, x, y float64, scales ...float64) (ok bool) {
	mygo.RunOnMain(func() { ok = linux.TestPinchPage(w.NativeHandle(), webView, x, y, scales) })
	return ok
}

// pageMenusHooked tells that the tests can right-click a page and read
// the menu it opens, through rightClickPage, pageMenu, choosePageMenuItem
// and dismissPageMenu.
const pageMenusHooked = true

// rightClickPage clicks (x, y) in a window's page with the secondary
// button, through XTEST.
func rightClickPage(w *mygo.Window, x, y float64) bool {
	return fake(func() bool { return linux.TestRightClickPage(w.NativeHandle(), x, y) })
}

// pageMenu returns the labels of the items of the web view's context menu
// shown over a page, "-" for separators, and whether one is shown.
func pageMenu() (labels []string, shown bool) {
	mygo.RunOnMain(func() { labels, shown = linux.TestPageMenu() })
	return labels, shown
}

// choosePageMenuItem chooses an item of that menu by its label.
func choosePageMenuItem(label string) (ok bool) {
	mygo.RunOnMain(func() { ok = linux.TestChoosePageMenuItem(label) })
	return ok
}

func dismissPageMenu() { mygo.RunOnMain(linux.TestDismissPageMenu) }

// Only Windows' tests start a process that takes the foreground.
func takeForeground(*mygo.Window) {}

// pressLetter presses the key of an uppercase letter, with Ctrl held when
// cmd is set, like a keyboard.
func pressLetter(_ *mygo.Window, letter byte, cmd bool) bool {
	key := string(rune(letter - 'A' + 'a'))
	if cmd {
		return pressKeys("Control_L", key)
	}
	return pressKeys(key)
}

func glassTint(*mygo.Window) (red, green, blue, alpha float64, ok bool) { return 0, 0, 0, 0, false }

func nativeGlassSupported() bool { return false }

func nativeGlass(*mygo.Window) (class string, frame [4]float64, style int, ok bool) {
	return "", frame, 0, false
}

func nativeGlassTint(*mygo.Window) (red, green, blue, alpha float64) { return 0, 0, 0, 0 }

func activateApp() {}

func hitView(*mygo.Window, float64, float64) string { return "" }
