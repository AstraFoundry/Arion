//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu/d3d11"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/scene"
)

func TestTransparentSurfaceRequiresComposition(t *testing.T) {
	b := New()
	// A device creation failure is in backoff. No native window may be
	// created with an opaque rendering path in its place.
	b.compRetry = time.Now().Add(time.Hour)
	w, err := b.NewWindow(&platform.WindowOptions{Surface: true, Transparent: true}, nil)
	if w != nil || err == nil || !strings.Contains(err.Error(), "require DirectComposition") {
		t.Fatalf("NewWindow = %v, %v; want an explicit composition error", w, err)
	}
	if len(b.windows) != 0 || len(b.surfaces) != 0 {
		t.Fatal("failed transparency setup left native windows behind")
	}
}

// TestTransparentSurfaceDesktopPixels observes DWM's actual desktop output,
// rather than the frame buffer, which retains alpha even on the broken
// opaque presentation path. Run in an unlocked Windows desktop session:
// MYGO_E2E=1 go test ./internal/windows -run TransparentSurface -count=1.
func TestTransparentSurfaceDesktopPixels(t *testing.T) {
	if os.Getenv("MYGO_E2E") == "" {
		t.Skip("set MYGO_E2E=1 in an unlocked Windows desktop session")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	previous := theBackend
	b := New()
	if err := b.Init(&transparencyAppHandler{}, platform.AppOptions{Name: "MyGo transparency test"}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, w := range b.windows {
			w.destroy()
		}
		if b.comp != nil {
			b.comp.free()
		}
		procDestroyWindow.Call(b.appHwnd)
		theBackend = previous
	}()
	makeWindow := func(transparent bool, color uint32) *window {
		t.Helper()
		h := &transparencyWindowHandler{color: color, transparent: transparent}
		pw, err := b.NewWindow(&platform.WindowOptions{
			Surface: true, Transparent: transparent, Frameless: true,
			Width: 240, Height: 160, X: 80, Y: 80,
		}, h)
		if err != nil {
			t.Fatal(err)
		}
		w := pw.(*window)
		h.surface = w.surface
		w.ShowInactive()
		procRedrawWindow.Call(w.surface.hwnd, 0, 0, rdwInvalidate|rdwUpdateNow)
		return w
	}
	background := makeWindow(false, 0xFF296FC7)
	background.SetAlwaysOnTop(true)
	foreground := makeWindow(true, 0xFFE83A5D)
	foreground.SetAlwaysOnTop(true)
	foreground.SetIgnoreMouseEvents(true)
	var client rect
	procGetClientRect.Call(foreground.surface.hwnd, uintptr(unsafe.Pointer(&client)))
	centerX, centerY := client.Right/2, client.Bottom/2
	if !foreground.noRedirect || windowLong(foreground.hwnd, gwlExStyle)&wsExNoRedirect == 0 {
		t.Fatal("transparent window retains an opaque redirection bitmap")
	}
	if n := foreground.surface.Native(); !n.Composed {
		t.Fatal("GPU renderer was given an opaque swap-chain target")
	}
	// Native releases the CPU compositor in preparation for GPU ownership;
	// draw again through the CPU path for the desktop pixel assertions.
	procRedrawWindow.Call(foreground.surface.hwnd, 0, 0, rdwInvalidate|rdwUpdateNow)
	assertPixels := func(stage string) {
		t.Helper()
		for deadline := time.Now().Add(3 * time.Second); ; {
			dwmapi.NewProc("DwmFlush").Call()
			corner := transparencyDesktopPixel(foreground.surface.hwnd, 8, 8)
			center := transparencyDesktopPixel(foreground.surface.hwnd, centerX, centerY)
			if corner == 0xC76F29 && center == 0x5D3AE8 {
				t.Logf("%s: transparent corner=%#06x, opaque center=%#06x", stage, corner, center)
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s: corner=%#08x center=%#08x; want background=%#06x content=%#06x", stage, corner, center, 0xC76F29, 0x5D3AE8)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	assertPixels("first frame with click-through")
	style := windowLong(foreground.hwnd, gwlExStyle)
	if style&(wsExTransparent|wsExLayered|wsExNoActivate) != wsExTransparent|wsExLayered|wsExNoActivate {
		t.Fatalf("click-through/nonactivation styles missing: %#x", style)
	}
	// Real OS hit testing must select the underlying native window even
	// over an opaque pixel of the transparent overlay.
	pt := point{centerX, centerY}
	procClientToScreen.Call(foreground.surface.hwnd, uintptr(unsafe.Pointer(&pt)))
	hit, _, _ := user32.NewProc("WindowFromPoint").Call(uintptr(uint32(pt.X)) | uintptr(uint32(pt.Y))<<32)
	root, _, _ := user32.NewProc("GetAncestor").Call(hit, 2) // GA_ROOT
	if root != background.hwnd {
		t.Fatalf("overlay intercepted desktop hit testing: root=%#x want background=%#x", root, background.hwnd)
	}
	// When the product enables its undo action, deliver the native mouse
	// messages to the owned surface and preserve the current keyboard focus.
	// Desktop hit testing above independently covers click-through; this
	// does not claim physical mouse injection.
	activeBefore, _, _ := procGetForegroundWindow.Call()
	focusBefore, _, _ := user32.NewProc("GetFocus").Call()
	foreground.SetIgnoreMouseEvents(false)
	lp := uintptr(uint16(centerX)) | uintptr(uint16(centerY))<<16
	procSendMessageW.Call(foreground.surface.hwnd, wmLButtonDown, 1, lp)
	procSendMessageW.Call(foreground.surface.hwnd, wmLButtonUp, 0, lp)
	if clicks := foreground.h.(*transparencyWindowHandler).clicks; clicks != 1 {
		t.Fatalf("native pointer events=%d; want 1", clicks)
	}
	if active, _, _ := procGetForegroundWindow.Call(); active != activeBefore {
		t.Fatalf("overlay stole foreground on native click: %#x, want %#x", active, activeBefore)
	}
	if focus, _, _ := user32.NewProc("GetFocus").Call(); focus != focusBefore {
		t.Fatalf("overlay stole keyboard focus on native click: %#x, want %#x", focus, focusBefore)
	}
	foreground.SetIgnoreMouseEvents(true)
	b.loseComposition(errors.New("transparent surface test device loss"))
	procRedrawWindow.Call(foreground.surface.hwnd, 0, 0, rdwInvalidate|rdwUpdateNow)
	if c := foreground.surface.comp; c == nil || c.dev == nil || c.dev != b.comp {
		t.Fatal("CPU compositor did not recreate the lost device")
	}
	assertPixels("after compositor recreation")

	// The UI's default renderer uses the same native target contract,
	// but owns its own composition device and swap chain.
	n := foreground.surface.Native()
	gpu, err := d3d11.NewComposed(n.HWND)
	if err != nil {
		t.Fatal(err)
	}
	defer gpu.Release()
	var r rect
	procGetClientRect.Call(n.HWND, uintptr(unsafe.Pointer(&r)))
	s := &scene.Scene{Width: int(r.Right), Height: int(r.Bottom), Ops: []scene.Op{{
		Kind: scene.OpFill, Color: scene.Color{R: 0xE8, G: 0x3A, B: 0x5D, A: 0xFF},
		Rect: scene.Rect{X: float32(r.Right) / 4, Y: float32(r.Bottom) / 4, W: float32(r.Right) / 2, H: float32(r.Bottom) / 2},
	}}}
	if err := gpu.Render(s); err != nil {
		t.Fatal(err)
	}
	assertPixels("GPU swap chain with click-through")
}

type transparencyAppHandler struct{ platform.AppHandler }

func (*transparencyAppHandler) DidBecomeActive() {}
func (*transparencyAppHandler) DidResignActive() {}
func (*transparencyAppHandler) ThemeChanged()    {}
func (*transparencyAppHandler) DisplaysChanged() {}
func (*transparencyAppHandler) Ready()           {}
func (*transparencyAppHandler) Dispatch()        {}

type transparencyWindowHandler struct {
	platform.WindowHandler
	surface     *surface
	color       uint32
	transparent bool
	clicks      int
}

func (*transparencyWindowHandler) Closed()  {}
func (*transparencyWindowHandler) Focused() {}
func (*transparencyWindowHandler) Blurred() {}
func (*transparencyWindowHandler) Resized() {}
func (*transparencyWindowHandler) Moved()   {}
func (h *transparencyWindowHandler) SurfaceEvent(ev platform.SurfaceEvent) bool {
	if ev.Kind == platform.PointerDown {
		h.clicks++
	}
	if ev.Kind != platform.SurfaceFrame || h.surface == nil {
		return false
	}
	var r rect
	procGetClientRect.Call(h.surface.hwnd, uintptr(unsafe.Pointer(&r)))
	w, height := int(r.Right), int(r.Bottom)
	px := make([]uint32, w*height)
	for y := range height {
		for x := range w {
			if !h.transparent || x >= w/4 && x < w*3/4 && y >= height/4 && y < height*3/4 {
				px[y*w+x] = h.color
			}
		}
	}
	if len(px) != 0 {
		h.surface.PresentPixels(unsafe.Slice((*byte)(unsafe.Pointer(&px[0])), len(px)*4), w*4, w, height)
	}
	return false
}

func transparencyDesktopPixel(hwnd uintptr, x, y int32) uintptr {
	pt := point{x, y}
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
	dc, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, dc)
	px, _, _ := gdi32.NewProc("GetPixel").Call(dc, uintptr(pt.X), uintptr(pt.Y))
	return px
}
