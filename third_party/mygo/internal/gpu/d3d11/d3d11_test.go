//go:build windows

package d3d11

import (
	"runtime"
	"slices"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu"
	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/wait"
)

// hiddenWindow creates a window that never shows, for a swap chain.
func hiddenWindow(t *testing.T, w, h int) uintptr {
	hwnd, err := newHiddenWindow(w, h)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { destroyWindow(hwnd) })
	return hwnd
}

// readBack copies the back buffer to memory: BGRA rows and their stride.
func (r *Renderer) readBack(t *testing.T) ([]byte, int) {
	pix, stride, err := r.read()
	if err != nil {
		t.Fatal(err)
	}
	return pix, stride
}

func TestDrawsAsTheCPURenderer(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// With the shaders compiled ahead of time, and with those compiled
	// from shader.hlsl when the renderer starts, as when they are older.
	for _, compile := range []bool{false, true} {
		compileShaders = compile
		s := gputest.Scene()
		r, err := New(hiddenWindow(t, s.Width, s.Height))
		compileShaders = false
		if err != nil {
			t.Skip("no Direct3D 11:", err)
		}
		// Twice: the second frame updates what the first uploaded.
		for frame := range 2 {
			if err := r.draw(s); err != nil {
				t.Fatal(err)
			}
			pix, stride := r.readBack(t)
			gputest.Compare(t, "d3d11", pix, stride, s)
			if err := r.present(1); err != nil {
				t.Fatalf("frame %d: %v", frame, err)
			}
		}

		s = gputest.TransformedScene()
		for range 2 {
			if err := r.draw(s); err != nil {
				t.Fatal(err)
			}
			pix, stride := r.readBack(t)
			gputest.Compare(t, "d3d11-affine", pix, stride, s)
		}
		s = gputest.PerspectiveScene()
		for range 2 {
			if err := r.draw(s); err != nil {
				t.Fatal(err)
			}
			pix, stride := r.readBack(t)
			gputest.Compare(t, "d3d11-perspective", pix, stride, s)
		}
		s = gputest.BehindOriginScene()
		for range 2 {
			if err := r.draw(s); err != nil {
				t.Fatal(err)
			}
			pix, stride := r.readBack(t)
			gputest.Compare(t, "d3d11-behind-origin", pix, stride, s)
		}
		for i, s := range gputest.TranslatedScenes() {
			if err := r.draw(s); err != nil {
				t.Fatal(err)
			}
			pix, stride := r.readBack(t)
			gputest.Compare(t, "d3d11-translated"+[]string{"", "-between-pixels"}[i], pix, stride, s)
		}
		s = gputest.MaskedClipScene()
		for range 2 {
			if err := r.draw(s); err != nil {
				t.Fatal(err)
			}
			pix, stride := r.readBack(t)
			gputest.Compare(t, "d3d11-masked-clips", pix, stride, s)
		}
		s = gputest.CornerClipScene()
		for range 2 {
			if err := r.draw(s); err != nil {
				t.Fatal(err)
			}
			pix, stride := r.readBack(t)
			gputest.Compare(t, "d3d11-corner-clips", pix, stride, s)
		}
		r.Release()
	}
}

// TestBorderedCardDrawsAsTheCPURenderer draws the card of #531 (a large
// opaque card with a border of 2 pixels at scale 2, and a small button like
// it) with round corners, as Direct3D draws continuous ones, and requires the
// CPU renderer's pixels along their straight edges, exactly.
func TestBorderedCardDrawsAsTheCPURenderer(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := gputest.BorderedCardScene(false)
	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	for range 2 {
		if err := r.draw(s); err != nil {
			t.Fatal(err)
		}
		pix, stride := r.readBack(t)
		gputest.CompareStraightEdges(t, "d3d11-bordered-card", pix, stride, s)
	}
}

// TestResizeSettles checks that while the window changes size the swap
// chain draws into part of larger buffers, resized only when the window
// outgrows them, and that once the settle timer asked for a frame, the
// frame gives it buffers of the window's size again. A swap chain that
// stretches its frames over the window shows that part of its buffers; one
// for composition or an unscaled one (#855), which would stretch the part
// over the buffers' size, magnifying the frame (#892), shows them whole,
// and an unscaled one the content's background beyond the frame.
func TestResizeSettles(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer func(d time.Duration) { settleDelay = d }(settleDelay)
	settleDelay = 50 * time.Millisecond
	for _, newRenderer := range []func(uintptr) (*Renderer, error){New, NewComposed} {
		r, err := newRenderer(hiddenWindow(t, 200, 150))
		if err != nil {
			t.Skip("no Direct3D 11:", err)
		}
		resizeSettles(t, r)
		r.Release()
	}
}

func resizeSettles(t *testing.T, r *Renderer) {
	t.Helper()
	kind := "window"
	if r.composed {
		kind = "composition"
	}
	bg := scene.Color{R: 24, G: 24, B: 27, A: 255}
	frame := func(w, h int) {
		t.Helper()
		if err := r.draw(&scene.Scene{Width: w, Height: h, Background: bg}); err != nil {
			t.Fatal(err)
		}
		if r.unscaled {
			// What the buffers hold beyond the frame shows where the
			// window grows before the next frame presents.
			pix, stride, err := r.readPart(r.bw, r.bh)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range [][2]int{{r.bw - 1, r.bh - 1}, {w, 0}, {r.bw - 1, 0}, {0, h}, {w - 1, r.bh - 1}} {
				if p[0] < w && p[1] < h || p[0] >= r.bw || p[1] >= r.bh {
					continue // in the frame, or beyond the buffers
				}
				px := pix[p[1]*stride+p[0]*4:]
				if got := [3]byte{px[2], px[1], px[0]}; got != [3]byte{bg.R, bg.G, bg.B} {
					t.Fatalf("%s: a %dx%d frame in %dx%d buffers: pixel %v beyond it is %v; want the background %v", kind, w, h, r.bw, r.bh, p, got, bg)
				}
			}
		}
		if err := r.present(0); err != nil {
			t.Fatal(err)
		}
		if r.swapChain2 == 0 {
			return
		}
		const scGetSourceSize = 30 // IDXGISwapChain2
		var sw, sh uint32
		call(r.swapChain2, scGetSourceSize, uintptr(unsafe.Pointer(&sw)), uintptr(unsafe.Pointer(&sh)))
		want := [2]int{r.w, r.h}
		if r.composed || r.unscaled {
			want = [2]int{r.bw, r.bh}
		}
		if got := [2]int{int(sw), int(sh)}; got != want {
			t.Fatalf("%s: a %dx%d frame in %dx%d buffers shows %dx%d of them; want %dx%d", kind, w, h, r.bw, r.bh, sw, sh, want[0], want[1])
		}
	}
	frame(200, 150)
	if r.swapChain2 == 0 {
		t.Skip("no IDXGISwapChain2")
	}
	if r.bw != 200 || r.bh != 150 || r.resizing {
		t.Fatalf("%s: first frame: buffers %dx%d, resizing %v; want 200x150 and not resizing", kind, r.bw, r.bh, r.resizing)
	}
	frame(220, 160)
	bw, bh := r.bw, r.bh
	if !r.resizing || bw <= 220 || bh <= 160 {
		t.Fatalf("%s: resized: buffers %dx%d, resizing %v; want larger than 220x160 and resizing", kind, bw, bh, r.resizing)
	}
	frame(230, 155)
	if r.bw != bw || r.bh != bh || r.w != 230 || r.h != 155 {
		t.Fatalf("%s: resized within the buffers: buffers %dx%d drawing %dx%d; want %dx%d drawing 230x155", kind, r.bw, r.bh, r.w, r.h, bw, bh)
	}
	// The timer fires as messages are dispatched.
	user32 := syscall.NewLazyDLL("user32.dll")
	peek, dispatch := user32.NewProc("PeekMessageW"), user32.NewProc("DispatchMessageW")
	var msg [64]byte
	for deadline := time.Now().Add(wait.Guard); settling[r.hwnd]; time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("%s: the settle timer did not fire", kind)
		}
		for {
			if ok, _, _ := peek.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1); ok == 0 {
				break
			}
			dispatch.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
	frame(230, 155)
	if r.bw != 230 || r.bh != 155 || r.resizing {
		t.Fatalf("%s: settled: buffers %dx%d, resizing %v; want 230x155 and not resizing", kind, r.bw, r.bh, r.resizing)
	}
}

// TestBeyondFrame checks the parts of a swap chain's buffers beyond the
// frame, which an unscaled one fills with the content's background.
func TestBeyondFrame(t *testing.T) {
	for _, c := range []struct {
		w, h, bw, bh int
		want         []gpu.Scissor
	}{
		{200, 150, 200, 150, nil},
		{200, 150, 314, 251, []gpu.Scissor{{Left: 200, Right: 314, Bottom: 251}, {Top: 150, Right: 200, Bottom: 251}}},
		{200, 150, 200, 251, []gpu.Scissor{{Top: 150, Right: 200, Bottom: 251}}},
		{200, 150, 314, 150, []gpu.Scissor{{Left: 200, Right: 314, Bottom: 150}}},
	} {
		parts, n := beyondFrame(c.w, c.h, c.bw, c.bh)
		if got := parts[:n]; !slices.Equal(got, c.want) {
			t.Errorf("beyondFrame(%d, %d, %d, %d) = %v; want %v", c.w, c.h, c.bw, c.bh, got, c.want)
		}
	}
}

// TestSwapChainBackground checks that the window's swap chain shows its
// frames unscaled, and the scene's background beyond them (#222): where the
// window grew before the frame of its size presents, DWM shows that, not
// black.
func TestSwapChainBackground(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	r, err := New(hiddenWindow(t, 200, 150))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	const scGetDesc1, scGetBackgroundColor = 18, 26 // IDXGISwapChain1
	for _, bg := range []scene.Color{{R: 255, G: 255, B: 255, A: 255}, {R: 24, G: 24, B: 27, A: 255}, {}} {
		if err := r.draw(&scene.Scene{Width: 200, Height: 150, Background: bg}); err != nil {
			t.Fatal(err)
		}
		var desc swapChainDesc1
		call(r.swapChain, scGetDesc1, uintptr(unsafe.Pointer(&desc)))
		if desc.Scaling != scalingNone {
			t.Fatalf("the swap chain scales frames as %d; want DXGI_SCALING_NONE", desc.Scaling)
		}
		var got [4]float32
		call(r.swapChain, scGetBackgroundColor, uintptr(unsafe.Pointer(&got)))
		want := [4]float32{float32(bg.R) / 255, float32(bg.G) / 255, float32(bg.B) / 255, 1}
		if bg.A == 0 {
			want = [4]float32{0, 0, 0, 1}
		}
		for i := range 3 {
			if d := got[i] - want[i]; d < -0.002 || d > 0.002 {
				t.Fatalf("background %v: the swap chain shows %v beyond the frame; want %v", bg, got, want)
			}
		}
	}
}

// releaseImages arms every image of a scene to keep only the renderer's
// copy of its pixels: the restore re-makes them from a copy it took, and
// dropped counts the uploads and drops them again.
func releaseImages(s *scene.Scene, restores, drops *int) int {
	armed := 0
	for i := range s.Ops {
		if img := s.Ops[i].Image; img != nil && img.Scene == nil && img.Restore == nil {
			pix := slices.Clone(img.Pix)
			img.Restore = func() []byte { *restores++; return pix }
			img.Dropped = func() { *drops++; img.Pix = nil }
			armed++
		}
	}
	return armed
}

// TestReleasedImageUploadsFromItsRestore checks issue #664's contract with
// the renderer: an image whose pixels an owner dropped uploads them again
// from the restore, drawing as before, as a renderer that lost or evicted
// its texture does. The first draw takes the new-texture path, which an
// eviction after 240 frames takes too.
func TestReleasedImageUploadsFromItsRestore(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := gputest.Scene()
	restores, drops := 0, 0
	armed := releaseImages(s, &restores, &drops)
	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	if err := r.draw(s); err != nil {
		t.Fatal(err)
	}
	if restores != 0 {
		t.Fatalf("%d restores for pixels the first draw had", restores)
	}
	if drops != armed {
		t.Fatalf("%d drops of %d images", drops, armed)
	}
	// The CPU renderer the comparison draws with re-makes the pixels of
	// the images it draws.
	pix, stride := r.readBack(t)
	gputest.Compare(t, "d3d11 released", pix, stride, s)
	if restores != armed {
		t.Fatalf("%d restores of %d images in the CPU render", restores, armed)
	}
	// The textures stay for the frames that follow: the restore runs no
	// more, and the dropped pixels go.
	if err := r.draw(s); err != nil {
		t.Fatal(err)
	}
	if restores != armed || drops != armed {
		t.Fatalf("%d restores, %d drops after a frame with the textures live", restores, drops)
	}
}

// TestReleasedImageWithNewRenderer uploads from the restore again in a
// renderer of its own, as a device that was made again does.
func TestReleasedImageWithNewRenderer(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := gputest.Scene()
	restores, drops := 0, 0
	armed := releaseImages(s, &restores, &drops)
	for i := range 2 {
		r, err := New(hiddenWindow(t, s.Width, s.Height))
		if err != nil {
			t.Skip("no Direct3D 11:", err)
		}
		if err := r.draw(s); err != nil {
			r.Release()
			t.Fatal(err)
		}
		// The first renderer uploaded the pixels it had; the second
		// found them dropped by the first and re-made them.
		if want := i * armed; restores != want {
			t.Fatalf("renderer %d: %d restores, want %d", i, restores, want)
		}
		if drops != (i+1)*armed {
			t.Fatalf("renderer %d: %d drops, want %d", i, drops, (i+1)*armed)
		}
		pix, stride := r.readBack(t)
		gputest.Compare(t, "d3d11 released, new renderer", pix, stride, s)
		r.Release()
	}
}

// TestShaderBytecode checks that the shaders compiled ahead of time come
// from shader.hlsl as it is, and that the compiler renderers fall back to
// compiles it.
func TestShaderBytecode(t *testing.T) {
	if sourceSum() != shaderSum {
		t.Fatal("shader.hlsl changed since shaders.go was generated: run go generate ./internal/gpu/d3d11 on Windows")
	}
	for _, s := range []struct{ entry, target string }{{"vs", "vs_4_0"}, {"ps", "ps_4_0"}} {
		code, err := compileShader(s.entry, s.target)
		if err != nil {
			t.Fatal(err)
		}
		// DXBC starts with its magic.
		if len(code) < 4 || string(code[:4]) != "DXBC" {
			t.Errorf("%s: not DXBC", s.entry)
		}
	}
}

// TestComposed draws into a swap chain for DirectComposition, which shows
// on the window with its alpha: frames draw as into the window's own, what
// a scene leaves transparent stays so, and once a renderer is released
// another composes the window, as after a GPU failure: a window has one
// DirectComposition target at most.
func TestComposed(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := gputest.Scene()
	hwnd := hiddenWindow(t, s.Width, s.Height)
	for i := range 2 {
		r, err := NewComposed(hwnd)
		if err != nil {
			t.Skip("no Direct3D 11:", err)
		}
		if err := r.draw(s); err != nil {
			r.Release()
			t.Fatalf("renderer %d: %v", i, err)
		}
		if r.target == 0 || r.visual == 0 {
			t.Fatalf("renderer %d: the swap chain shows through no DirectComposition target", i)
		}
		pix, stride := r.readBack(t)
		gputest.Compare(t, "d3d11 composed", pix, stride, s)
		if err := r.present(1); err != nil {
			t.Fatal(err)
		}
		// A frame of another size, transparent.
		if err := r.draw(&scene.Scene{Width: s.Width + 20, Height: s.Height + 10}); err != nil {
			t.Fatal(err)
		}
		if pix, _ := r.readBack(t); len(pix) < 4 || pix[0]|pix[1]|pix[2]|pix[3] != 0 {
			t.Errorf("renderer %d: a transparent frame reads %v", i, pix[:min(len(pix), 4)])
		}
		if err := r.present(0); err != nil {
			t.Fatal(err)
		}
		r.Release()
	}
}
