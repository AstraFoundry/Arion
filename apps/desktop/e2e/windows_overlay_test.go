//go:build arion_e2e && windows

package e2e

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestWindowsOverlayWindowPolicy(t *testing.T) {
	var overlay *mygo.Window
	eventually(t, "overlay created", func() bool {
		overlay = app.Windows()["overlay"]
		return overlay != nil
	})
	get := user32.NewProc("GetWindowLongPtrW")
	style, _, _ := get.Call(overlay.NativeHandle(), ^uintptr(19))
	const want = 0x08000000 | 0x00000080 | 0x00000020 | 0x00080000 | 0x00200000
	if style&want != want || style&0x00040000 != 0 {
		t.Fatalf("overlay must be nonactivating, tool, click-through, layered and composed, without app-window style: %#x", style)
	}
}
