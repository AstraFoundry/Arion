//go:build !darwin && !(linux && (amd64 || arm64)) && !(windows && (amd64 || arm64)) && !js

package e2e

import (
	"errors"
	"testing"

	"github.com/egoist/mygo"
)

// Platforms without a backend run no GUI tests.
func activateMenu(*mygo.Window, ...string) error { return errors.New("unsupported") }

func menuItemEnabled(*mygo.Window, ...string) (bool, error) { return false, errors.ErrUnsupported }

func pressShortcut(string, bool) (bool, bool) { return false, false }

func endSheet(*mygo.Window) (bool, bool) { return false, false }

func recentDocumentRecorded(string) (recorded, supported bool) { return false, false }

func click(*mygo.Window, float64, float64) bool { return false }

func appClick(*mygo.Window, float64, float64) bool { return false }

func clickPage(*mygo.Window, float64, float64) bool { return false }

func heldAppClick(*mygo.Window, float64, float64) bool { return false }

func drag(*mygo.Window, [][2]float64) bool { return false }

func sideButton(*mygo.Window, bool) bool { return false }

func pinch(*mygo.Window, float64, float64, float64) (float64, bool) { return 0, false }

func wheel(*mygo.Window, float64, float64) bool { return false }

func webViewAttached(*mygo.Window) (bool, bool) { return false, false }

// Collection behaviors and deactivating the app are macOS's.
func collectionBehavior(*mygo.Window) (uint, bool)               { return 0, false }
func deactivateApp() bool                                        { return false }
func material(*mygo.Window) (string, int, float64, bool, bool)   { return "", 0, 0, false, false }
func materialState(*mygo.Window) (int, bool)                     { return 0, false }
func roundedBacking(*mygo.Window) (bool, float64, float64, bool) { return false, 0, 0, false }
func windowBacking(*mygo.Window) (bool, float64, bool)           { return false, 0, false }
func appActive() (bool, bool)                                    { return false, false }

func dockDevTools(*mygo.Window) bool { return false }

func dockedDevToolsPlace(*mygo.Window) string { return "" }

func dismissPopups() (int, bool) { return 0, false }

func setDroppedFiles(*mygo.Window, []string) bool { return false }

// The progress bar is shown by the shell, out of the app's reach.
func dockTileImage() ([]byte, bool) { return nil, false }

func defaultURLHandler(string) (string, bool) { return "", false }

func dockMenu(int) ([]string, bool) { return nil, false }

// Posting key events needs the accessibility permission on macOS.
func pressCtrlShiftK() bool { return false }

// Only Linux binds global shortcuts through a desktop portal.
func usePortalShortcuts() (func(), bool) { return nil, false }

func pressKeys(...string) bool { return false }

// Only macOS windows have a toolbar.
func fullScreenHidesToolbar(*mygo.Window) (bool, bool) { return false, false }

// Only macOS windows have traffic lights.
func trafficLights(*mygo.Window) (float64, float64, bool) { return 0, 0, false }
func setWindowAppearance(*mygo.Window, bool)              {}

func movePointer(int, int) bool                { return false }
func pressButton(bool) bool                    { return false }
func resizeCursor(*mygo.Window) (string, bool) { return "", false }

func menuBarShown(*mygo.Window) (bool, bool)                { return false, false }
func activateAccelerator(*mygo.Window, string) (bool, bool) { return false, false }
func enterMenuBar(*mygo.Window, string) (bool, bool, bool)  { return false, false, false }
func openMenus(*mygo.Window, byte) bool                     { return false }
func closeMenus()                                           {}

func titleButtons(*mygo.Window) ([]string, bool) { return nil, false }
func pressTitleButton(*mygo.Window, string) bool { return false }

func topNonClient(*mygo.Window) (int32, int32, bool) { return 0, 0, false }

func menuBarColors(*mygo.Window) (uint32, uint32, bool) { return 0, 0, false }

func titleButtonColor(*mygo.Window, string) (uint8, uint8, uint8, bool) { return 0, 0, 0, false }

func composition(*mygo.Window) (bool, bool, bool) { return false, false, false }
func canCompose() bool                            { return false }
func loseComposition(bool) bool                   { return false }

func failWebViews(int) bool { return false }

// A Control-click is a secondary click on macOS only.
func controlClick(*mygo.Window, float64, float64) bool { return false }

func secondaryClickAsPrimary(*mygo.Window, float64, float64) bool { return false }

// Input methods, file drops and assistive technology are only automated
// on macOS.
func composeOver(*mygo.Window, string, int, bool, int, int) bool { return false }
func inputClient(*mygo.Window) ([2]int, string, bool)            { return [2]int{}, "", false }
func dropFiles(*mygo.Window, float64, float64, []string) (bool, bool, bool) {
	return false, false, false
}
func accessibility(*mygo.Window) ([]accessNode, bool)         { return nil, false }
func accessPerform(*mygo.Window, string, string, string) bool { return false }

const roleText, roleButton, roleCheckBox, roleTextField, roleSlider = "text", "button", "check box", "text field", "slider"

// roleListItem is the role of the rows of a List.
const roleListItem = "list item"

// Typing into native UI is only automated on macOS.
func clickAndType(*mygo.Window, float64, float64, string) bool { return false }
func compose(*mygo.Window, string, int, bool) bool             { return false }

// Only Linux draws native UI in a GtkGLArea, nor waits to load the GPU's
// driver.
func glSurface(*mygo.Window) (string, []byte, int, int, bool) { return "", nil, 0, 0, false }
func memoryUI(*testing.T) bool                                { return false }
func surfaceOnScreen(*mygo.Window) ([]byte, bool)             { return nil, false }

// Only Windows reads the screen over native UI.
func screenColor(*mygo.Window, float64, float64) (uint8, uint8, uint8, bool) { return 0, 0, 0, false }

// Only macOS has key-value observing.
func observe(*mygo.Window) (func(), bool) { return nil, false }

func rightClick(*mygo.Window, float64, float64) bool { return false }
func popupMenus() ([][]string, bool)                 { return nil, false }
func choosePopupItem(string) bool                    { return false }
func popupItemEnabled(string) (bool, bool)           { return false, false }

// Input methods keep the keys typed while they compose from the content
// themselves here (GTK's filtering, IMM32's VK_PROCESSKEY), which a
// composition the test makes up does not show.
func pressKey(*mygo.Window, uint16, string) bool { return false }
func pressTab(*mygo.Window, bool) bool           { return false }

func pressKeyComposing(*mygo.Window, uint16, string, string, string) bool { return false }

func dragFiles(*mygo.Window, float64, float64, []string) (bool, bool) { return false, false }

// AppKit's older accessibility API is macOS's.
func axAttribute(*mygo.Window, string, string) (string, bool, bool, bool) {
	return "", false, false, false
}

func pressEscape(*mygo.Window) bool { return false }

func documentChrome(*mygo.Window) (string, string, bool, bool, bool) {
	return "", "", false, false, false
}
func sheet(*mygo.Window) ([]string, []string, bool, bool) { return nil, nil, false, false }
func endSheetButton(*mygo.Window, int) bool               { return false }

// Panels' window styles are Windows'.
func panelStyles(*mygo.Window) (bool, bool, bool, bool) { return false, false, false, false }

func throttlingOff(*mygo.Window, bool) (bool, int)                   { return false, 0 }
func mockCapture(*mygo.Window) bool                                  { return false }
func pageGestures(*mygo.Window, int) (bool, bool, bool)              { return false, false, false }
func pinchPage(*mygo.Window, int, float64, float64, ...float64) bool { return false }

// Right-clicking a page and reading the menu it opens are not hooked here.
const pageMenusHooked = false

func rightClickPage(*mygo.Window, float64, float64) bool { return false }
func pageMenu() ([]string, bool)                         { return nil, false }
func choosePageMenuItem(string) bool                     { return false }
func dismissPageMenu()                                   {}

func takeForeground(*mygo.Window) {}

func pressLetter(*mygo.Window, byte, bool) bool { return false }

func glassTint(*mygo.Window) (red, green, blue, alpha float64, ok bool) { return 0, 0, 0, 0, false }

func nativeGlassSupported() bool { return false }

func nativeGlass(*mygo.Window) (class string, frame [4]float64, style int, ok bool) {
	return "", frame, 0, false
}

func nativeGlassTint(*mygo.Window) (red, green, blue, alpha float64) { return 0, 0, 0, 0 }

func activateApp() {}

func hitView(*mygo.Window, float64, float64) string { return "" }

// Only X servers keep a keyboard's state for the tests after a test: see
// left_behind_linux_test.go.
func keyboardState() (state string, restore func()) { return "", func() {} }
