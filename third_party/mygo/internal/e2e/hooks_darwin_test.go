//go:build darwin

package e2e

import (
	"log"
	"os/exec"
	"slices"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/internal/wait"
)

// recentDocumentRecorded reports whether path is among the app's recent
// documents.
func recentDocumentRecorded(path string) (recorded, supported bool) {
	var paths []string
	mygo.RunOnMain(func() { paths = darwin.TestRecentDocuments() })
	return slices.Contains(paths, path), true
}

func activateMenu(w *mygo.Window, path ...string) (err error) {
	mygo.RunOnMain(func() { err = darwin.TestPerformMenuItem(path...) })
	return err
}

func menuItemEnabled(_ *mygo.Window, path ...string) (enabled bool, err error) {
	mygo.RunOnMain(func() { enabled, err = darwin.TestMenuItemEnabled(path...) })
	return enabled, err
}

func pressShortcut(key string, shift bool) (handled bool, supported bool) {
	mygo.RunOnMain(func() { handled = darwin.TestPerformKeyEquivalent(key, shift) })
	return handled, true
}

func endSheet(w *mygo.Window) (ok bool, supported bool) {
	mygo.RunOnMain(func() { ok = darwin.TestEndSheet(w.NativeHandle()) })
	return ok, true
}

func click(w *mygo.Window, x, y float64) bool {
	waitOnScreen(w)
	mygo.RunOnMain(func() { darwin.TestClick(w.NativeHandle(), x, y) })
	return true
}

// clickPage clicks a window's page: click reaches web views too.
func clickPage(w *mygo.Window, x, y float64) bool { return click(w, x, y) }

// appClick clicks through the application, whose event monitors see it.
func appClick(w *mygo.Window, x, y float64) bool {
	waitOnScreen(w)
	mygo.RunOnMain(func() { darwin.TestAppClick(w.NativeHandle(), x, y) })
	return true
}

// heldAppClick is appClick with the button held down, as a user clicks:
// what the press did runs before the release, the work it left for the
// main thread too.
func heldAppClick(w *mygo.Window, x, y float64) bool {
	waitOnScreen(w)
	mygo.RunOnMain(func() { darwin.TestAppPress(w.NativeHandle(), x, y, true) })
	mygo.RunOnMain(func() {}) // after what the press posted
	mygo.RunOnMain(func() { darwin.TestAppPress(w.NativeHandle(), x, y, false) })
	return true
}

// waitOnScreen waits for a window to show on screen, as a user sees a
// window before clicking it: a window shows a while after it is ordered
// front and made key, longer on a busy runner, and AppKit puts a click
// made for it before then elsewhere in it, as outside its content, where
// it clicks nothing. wait.Guard is a hang guard: it says so if the window
// did not show by then, before the click that will miss.
func waitOnScreen(w *mygo.Window) {
	if !wait.Poll(func() (on bool) {
		mygo.RunOnMain(func() { on = darwin.TestOnScreen(w.NativeHandle()) })
		return on
	}) {
		log.Printf("e2e: the window %q did not show on screen in %v: a click made for it misses", w.Title(), wait.Guard)
	}
}

func sideButton(w *mygo.Window, back bool) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestSideButton(w.NativeHandle(), back) })
	return ok
}

// pinch pinches the touchpad at (x, y) by scale, and returns the scale
// the platform reports.
func pinch(w *mygo.Window, x, y, scale float64) (got float64, ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestPinch(w.NativeHandle(), x, y, scale-1) })
	return scale, ok
}

func drag(w *mygo.Window, points [][2]float64) bool {
	mygo.RunOnMain(func() { darwin.TestDrag(w.NativeHandle(), points) })
	return true
}

// wheel turns the wheel at (x, y) of a window's content, through the
// window's own hit testing.
// The wheel over a web view is covered by internal/darwin's
// TestEmbeddedWebViewScrollReachesSurface, with AppKit's events.
func wheel(*mygo.Window, float64, float64) bool { return false }

func hitView(w *mygo.Window, x, y float64) (class string) {
	mygo.RunOnMain(func() { class = darwin.TestHitView(w.NativeHandle(), x, y) })
	return class
}

func webViewAttached(w *mygo.Window) (attached bool, supported bool) {
	mygo.RunOnMain(func() { attached = darwin.TestWebViewAttached(w.NativeHandle()) })
	return attached, true
}

// deactivateApp activates the Finder, as the user would.
func deactivateApp() bool {
	return exec.Command("open", "-a", "Finder").Run() == nil
}

func appActive() (active bool, supported bool) {
	mygo.RunOnMain(func() { active = darwin.TestIsFrontmost() })
	return active, true
}

func collectionBehavior(w *mygo.Window) (behavior uint, supported bool) {
	mygo.RunOnMain(func() { behavior = darwin.TestCollectionBehavior(w.NativeHandle()) })
	return behavior, true
}

func material(w *mygo.Window) (class string, style int, radius float64, active bool, supported bool) {
	mygo.RunOnMain(func() { class, style, radius, active = darwin.TestMaterial(w.NativeHandle()) })
	return class, style, radius, active, true
}

// materialState returns the NSVisualEffectState of the window's material.
func materialState(w *mygo.Window) (state int, supported bool) {
	mygo.RunOnMain(func() { state = darwin.TestMaterialState(w.NativeHandle()) })
	return state, true
}

func roundedBacking(w *mygo.Window) (opaque bool, alpha, red float64, supported bool) {
	mygo.RunOnMain(func() { opaque, alpha, red = darwin.TestRoundedBacking(w.NativeHandle()) })
	return opaque, alpha, red, true
}

// windowBacking reports whether the window is opaque, and the alpha of its
// background color.
func windowBacking(w *mygo.Window) (opaque bool, alpha float64, supported bool) {
	mygo.RunOnMain(func() { opaque, alpha = darwin.TestWindowBacking(w.NativeHandle()) })
	return opaque, alpha, true
}

func dockDevTools(w *mygo.Window) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestDockInspector(w.NativeHandle()) })
	return ok
}

func dockedDevToolsPlace(w *mygo.Window) (place string) {
	mygo.RunOnMain(func() { place = darwin.TestInspectorPlace(w.NativeHandle()) })
	return place
}

// Context menus track the mouse in a modal loop; not automated.
func dismissPopups() (int, bool) { return 0, false }

func setDroppedFiles(w *mygo.Window, paths []string) bool {
	mygo.RunOnMain(func() { darwin.TestSetDroppedFiles(w.NativeHandle(), paths) })
	return true
}

func dockTileImage() (png []byte, supported bool) {
	mygo.RunOnMain(func() { png = darwin.TestDockTileImage() })
	return png, true
}

func defaultURLHandler(string) (string, bool) { return "", false }

func dockMenu(click int) (titles []string, supported bool) {
	mygo.RunOnMain(func() { titles = darwin.TestDockMenu(click) })
	return titles, true
}

// Posting key events needs the accessibility permission on macOS.
func pressCtrlShiftK() bool { return false }

// Only Linux binds global shortcuts through a desktop portal.
func usePortalShortcuts() (func(), bool) { return nil, false }

func pressKeys(...string) bool { return false }

func fullScreenHidesToolbar(w *mygo.Window) (hides bool, supported bool) {
	mygo.RunOnMain(func() { hides = darwin.TestFullScreenHidesToolbar(w.NativeHandle()) })
	return hides, true
}

func trafficLights(w *mygo.Window) (x, y float64, supported bool) {
	mygo.RunOnMain(func() { x, y = darwin.TestTrafficLights(w.NativeHandle()) })
	return x, y, true
}

func setWindowAppearance(w *mygo.Window, dark bool) {
	mygo.RunOnMain(func() { darwin.TestSetWindowAppearance(w.NativeHandle(), dark) })
}

// Frameless windows keep native resize borders here.
func movePointer(int, int) bool                { return false }
func pressButton(bool) bool                    { return false }
func resizeCursor(*mygo.Window) (string, bool) { return "", false }

// The menu bar belongs to the application on macOS.
func menuBarShown(*mygo.Window) (bool, bool)                { return false, false }
func activateAccelerator(*mygo.Window, string) (bool, bool) { return false, false }
func enterMenuBar(*mygo.Window, string) (bool, bool, bool)  { return false, false, false }
func openMenus(*mygo.Window, byte) bool                     { return false }
func closeMenus()                                           {}

// The traffic lights are AppKit's own.
func titleButtons(*mygo.Window) ([]string, bool) { return nil, false }
func pressTitleButton(*mygo.Window, string) bool { return false }

func topNonClient(*mygo.Window) (int32, int32, bool) { return 0, 0, false }

func menuBarColors(*mygo.Window) (uint32, uint32, bool) { return 0, 0, false }

func titleButtonColor(*mygo.Window, string) (uint8, uint8, uint8, bool) { return 0, 0, 0, false }

// AppKit composes the window: there is no redirection bitmap to go without.
func composition(*mygo.Window) (bool, bool, bool) { return false, false, false }
func canCompose() bool                            { return false }
func loseComposition(bool) bool                   { return false }

// WebKit creates web views with their windows.
func failWebViews(int) bool { return false }

func clickAndType(w *mygo.Window, x, y float64, text string) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestClickAndType(w.NativeHandle(), x, y, text) })
	return ok
}

// pressKey presses the key of a virtual key code in a window of native UI,
// through the input method.
func pressKey(w *mygo.Window, code uint16, chars string) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestKey(w.NativeHandle(), code, chars) })
	return ok
}

// AppKit routes a drag to the view under the pointer, which
// dropFiles, dropping on the content's, skips.
func dragFiles(*mygo.Window, float64, float64, []string) (bool, bool) { return false, false }

// pressTab presses Tab in a window of native UI, with Shift when back, as
// the keyboard does: in the content or in the page of a web view.
func pressTab(w *mygo.Window, back bool) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestKeyShift(w.NativeHandle(), 48, "\t", back) })
	return ok
}

// pressKeyComposing presses a key in a window of native UI with the test
// standing in for the input method: it commits commit, unless empty, then
// passes the key on as command (doCommandBySelector:), unless empty.
func pressKeyComposing(w *mygo.Window, code uint16, chars, commit, command string) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestKeyComposing(w.NativeHandle(), code, chars, commit, command) })
	return ok
}

func compose(w *mygo.Window, text string, caret int, commit bool) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestCompose(w.NativeHandle(), text, caret, commit) })
	return ok
}

func controlClick(w *mygo.Window, x, y float64) bool {
	mygo.RunOnMain(func() { darwin.TestControlClick(w.NativeHandle(), x, y) })
	return true
}

func secondaryClickAsPrimary(w *mygo.Window, x, y float64) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestSecondaryClickAsPrimary(w.NativeHandle(), x, y) })
	return ok
}

func composeOver(w *mygo.Window, text string, caret int, commit bool, from, length int) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestComposeOver(w.NativeHandle(), text, caret, commit, from, length) })
	return ok
}

func inputClient(w *mygo.Window) (selected [2]int, document string, ok bool) {
	mygo.RunOnMain(func() { selected, document = darwin.TestInputClient(w.NativeHandle()) })
	return selected, document, true
}

func dropFiles(w *mygo.Window, x, y float64, paths []string) (over, dropped, ok bool) {
	mygo.RunOnMain(func() { over, dropped = darwin.TestDropFiles(w.NativeHandle(), x, y, paths) })
	return over, dropped, true
}

// The roles of elements in the accessibility API of AppKit.
const roleText, roleButton, roleCheckBox, roleTextField, roleSlider = "AXStaticText", "AXButton", "AXCheckBox", "AXTextField", "AXSlider"

// roleListItem is the role of the rows of a List.
const roleListItem = "AXRow/AXTableRow"

func accessibility(w *mygo.Window) (nodes []accessNode, ok bool) {
	mygo.RunOnMain(func() {
		for _, n := range darwin.TestAccessibility(w.NativeHandle()) {
			role := n.Role
			if n.Subrole != "" {
				role += "/" + n.Subrole
			}
			nodes = append(nodes, accessNode{role, n.Label, n.Value})
		}
	})
	return nodes, true
}

// axAttribute reads an attribute of an element through AppKit's older
// accessibility API, which only macOS has.
func axAttribute(w *mygo.Window, label, attr string) (value string, settable, named, ok bool) {
	mygo.RunOnMain(func() { value, settable, named, ok = darwin.TestAccessibilityAttribute(w.NativeHandle(), label, attr) })
	return value, settable, named, ok
}

func accessPerform(w *mygo.Window, label, action, value string) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestAccessibilityPerform(w.NativeHandle(), label, action, value) })
	return ok
}

// observe has key-value observing watch the view of a window showing
// native UI and its elements, until stop.
func observe(w *mygo.Window) (stop func(), supported bool) {
	var end func()
	mygo.RunOnMain(func() { end = darwin.TestObserve(w.NativeHandle()) })
	return func() { mygo.RunOnMain(end) }, true
}

// Only Linux draws native UI in a GtkGLArea, nor waits to load the GPU's
// driver.
func glSurface(*mygo.Window) (string, []byte, int, int, bool) { return "", nil, 0, 0, false }
func memoryUI(*testing.T) bool                                { return false }
func surfaceOnScreen(*mygo.Window) ([]byte, bool)             { return nil, false }

// Only Windows reads the screen over native UI.
func screenColor(*mygo.Window, float64, float64) (uint8, uint8, uint8, bool) { return 0, 0, 0, false }

// Context menus are not automated on macOS: one shown waits for the user.
func rightClick(*mygo.Window, float64, float64) bool { return false }
func popupMenus() ([][]string, bool)                 { return nil, false }
func choosePopupItem(string) bool                    { return false }
func popupItemEnabled(string) (bool, bool)           { return false, false }

// pressEscape presses Escape in the key window, as the keyboard does.
func pressEscape(*mygo.Window) bool {
	mygo.RunOnMain(func() { darwin.TestAppKey(53, "\x1b") })
	return true
}

func documentChrome(w *mygo.Window) (title, file string, icon, edited, supported bool) {
	mygo.RunOnMain(func() { title, file, icon, edited = darwin.TestDocumentChrome(w.NativeHandle()) })
	return title, file, icon, edited, true
}

func sheet(w *mygo.Window) (texts, buttons []string, ok, supported bool) {
	mygo.RunOnMain(func() { texts, buttons, ok = darwin.TestSheet(w.NativeHandle()) })
	return texts, buttons, ok, true
}

func endSheetButton(w *mygo.Window, i int) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestEndSheetButton(w.NativeHandle(), i) })
	return ok
}

// Panels' window styles are Windows'.
func panelStyles(*mygo.Window) (bool, bool, bool, bool) { return false, false, false, false }

// throttlingOff reports whether the page of w, or of its first web view,
// is set up to keep running while it can't be seen, and how many of the
// settings that take the platform has.
func throttlingOff(w *mygo.Window, webView bool) (off bool, settings int) {
	mygo.RunOnMain(func() { off, settings = darwin.TestBackgroundThrottlingOff(w.NativeHandle(), webView) })
	return off, settings
}

// mockCapture gives a window's page WebKit's mock camera and microphone.
func mockCapture(w *mygo.Window) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestMockCaptureDevices(w.NativeHandle()) })
	return ok
}

// pageGestures reports whether a swipe navigates and a pinch magnifies the
// page of a window, or with webView 0 or more, of its web view of that
// index.
func pageGestures(w *mygo.Window, webView int) (swipe, pinch, ok bool) {
	mygo.RunOnMain(func() { swipe, pinch, ok = darwin.TestPageGestures(w.NativeHandle(), webView) })
	return swipe, pinch, ok
}

// A WKWebView's magnification follows a trackpad's gesture through its
// phases, which a test can't make.
func pinchPage(*mygo.Window, int, float64, float64, ...float64) bool { return false }

// pageMenusHooked tells that the tests can right-click a page and read
// the menu it opens, through rightClickPage, pageMenu, choosePageMenuItem
// and dismissPageMenu.
const pageMenusHooked = true

// rightClickPage right-clicks (x, y) in a window's page.
func rightClickPage(w *mygo.Window, x, y float64) bool {
	waitOnScreen(w)
	mygo.RunOnMain(func() { darwin.TestRightClickPage(w.NativeHandle(), x, y) })
	return true
}

// pageMenu returns the titles of the items of the web view's context menu
// shown over a page, "-" for separators, and whether one is shown. AppKit
// tracks the menu in a loop of its own, which runs the main thread's work.
func pageMenu() (labels []string, shown bool) {
	mygo.RunOnMain(func() { labels, shown = darwin.TestPageMenu() })
	return labels, shown
}

// choosePageMenuItem chooses an item of that menu by its title.
func choosePageMenuItem(label string) (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestChoosePageMenuItem(label) })
	return ok
}

func dismissPageMenu() { mygo.RunOnMain(darwin.TestDismissPageMenu) }

// Only Windows' tests start a process that takes the foreground.
func takeForeground(*mygo.Window) {}

// pressLetter presses the key of an uppercase letter, with Command held
// when cmd is set, as the keyboard does in a window of native UI: in the
// content, or in the page of a web view that has the keyboard.
func pressLetter(w *mygo.Window, letter byte, cmd bool) (ok bool) {
	code, known := map[byte]uint16{'J': 0x26, 'K': 0x28, 'L': 0x25}[letter]
	if !known {
		return false
	}
	chars := string(rune(letter - 'A' + 'a'))
	mygo.RunOnMain(func() {
		if cmd {
			ok = darwin.TestKeyCmd(w.NativeHandle(), code, chars)
		} else {
			ok = darwin.TestKey(w.NativeHandle(), code, chars)
		}
	})
	return ok
}

func glassTint(w *mygo.Window) (red, green, blue, alpha float64, ok bool) {
	mygo.RunOnMain(func() { red, green, blue, alpha = darwin.TestGlassTint(w.NativeHandle()) })
	return red, green, blue, alpha, true
}

func nativeGlassSupported() (ok bool) {
	mygo.RunOnMain(func() { ok = darwin.TestNativeGlassSupported() })
	return ok
}

func nativeGlass(w *mygo.Window) (class string, frame [4]float64, style int, ok bool) {
	mygo.RunOnMain(func() { class, frame, style, ok = darwin.TestNativeGlass(w.NativeHandle()) })
	return class, frame, style, ok
}

func nativeGlassTint(w *mygo.Window) (red, green, blue, alpha float64) {
	mygo.RunOnMain(func() { red, green, blue, alpha = darwin.TestNativeGlassTint(w.NativeHandle()) })
	return red, green, blue, alpha
}

func activateApp() { mygo.RunOnMain(darwin.TestActivate) }

// Only X servers keep a keyboard's state for the tests after a test: see
// left_behind_linux_test.go.
func keyboardState() (state string, restore func()) { return "", func() {} }
