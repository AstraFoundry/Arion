//go:build windows

package windows

// ConfigureOverlay keeps the status surface out of Alt+Tab as well as the
// taskbar, and prevents its window from activating even during an undo click.
// SetWindowLongPtr changes only the extended style; MyGo retains ownership.
func ConfigureOverlay(handle uintptr) {
	const (
		exStyle    = ^uintptr(19) // GWL_EXSTYLE (-20)
		toolWindow = 0x00000080
		appWindow  = 0x00040000
		noActivate = 0x08000000
	)
	get := user32.NewProc("GetWindowLongPtrW")
	set := user32.NewProc("SetWindowLongPtrW")
	style, _, _ := get.Call(handle, exStyle)
	set.Call(handle, exStyle, (style|toolWindow|noActivate)&^appWindow)
}
