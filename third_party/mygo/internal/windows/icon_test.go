//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"
)

// TestFileIconAlpha checks that the shell's icons come with straight
// alpha, as imageOfDIB takes them: the colors of partly transparent
// pixels go above their alpha, which premultiplied ones never do.
func TestFileIconAlpha(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procCoInitializeEx.Call(0, coinitApartmentThreaded)
	paths := []string{
		filepath.Join(os.Getenv("SystemRoot"), "System32", "notepad.exe"),
		filepath.Join(os.Getenv("SystemRoot"), "explorer.exe"),
		os.Getenv("SystemRoot"),
	}
	for _, path := range paths {
		var factory uintptr
		if r, _, _ := procSHCreateItemFromParsingName.Call(uintptr(unsafe.Pointer(u16(path))), 0, uintptr(unsafe.Pointer(&iidIShellItemImageFactory)), uintptr(unsafe.Pointer(&factory))); failed(r) {
			t.Fatalf("%s: %#x", path, r)
		}
		var hbm uintptr
		if r := comCall(factory, 3, uintptr(48)|uintptr(48)<<32, 0x4, uintptr(unsafe.Pointer(&hbm))); failed(r) {
			t.Fatalf("%s: GetImage %#x", path, r)
		}
		release(factory)
		img, err := imageOfDIB(hbm)
		procDeleteObject.Call(hbm)
		if err != nil {
			t.Fatal(err)
		}
		partial, over := 0, 0
		for i := 0; i < len(img.Pix); i += 4 {
			r, g, b, a := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
			if a > 0 && a < 255 {
				partial++
			}
			if r > a || g > a || b > a {
				over++
			}
		}
		t.Logf("%s: %dx%d, %d partly transparent pixels, %d above their alpha", path, img.Rect.Dx(), img.Rect.Dy(), partial, over)
		if partial > 0 && over == 0 {
			t.Errorf("%s: no partly transparent pixel has a channel above its alpha: premultiplied?", path)
		}
	}
}
