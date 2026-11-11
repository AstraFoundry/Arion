//go:build windows

package d3d11

import (
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/gpu"
	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/wait"
)

// drawClasses draws s with a new renderer that set changes, and returns its
// pixels.
func drawClasses(t *testing.T, s *scene.Scene, set func(r *Renderer)) []byte {
	t.Helper()
	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	set(r)
	if err := r.draw(s); err != nil {
		t.Fatal(err)
	}
	pix, _ := r.readBack(t)
	return pix
}

// compareToGeneral fails when a channel of pix differs from the general
// shader's by more than a level: compilers round a result at a tie either way.
func compareToGeneral(t *testing.T, name string, pix, general []byte) {
	t.Helper()
	for i := range pix {
		if d := int(pix[i]) - int(general[i]); d < -1 || d > 1 {
			t.Fatalf("%s: byte %d is %d, and %d with the general shader", name, i, pix[i], general[i])
		}
	}
}

// TestClassesDrawAsTheGeneralShader draws scenes with the shaders of the
// classes of their batches and with the general shader, and with each class
// drawn by the shader holding them all, which tells kinds apart.
func TestClassesDrawAsTheGeneralShader(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, sc := range []struct {
		name string
		s    *scene.Scene
	}{{"scene", gputest.Scene()}, {"card", gputest.BorderedCardScene(false)}} {
		general := drawClasses(t, sc.s, func(r *Renderer) { r.b.NoClasses = true })
		compareToGeneral(t, sc.name, drawClasses(t, sc.s, func(r *Renderer) {}), general)
		compareToGeneral(t, sc.name+" joined", drawClasses(t, sc.s, func(r *Renderer) {
			all := r.classPipeFor(classShaders[len(classShaders)-1].classes)
			for c := gpu.ClassPlain; c < gpu.NumClasses; c++ {
				r.classes[1<<c] = all
			}
		}), general)
	}
}

// TestClassPipelines checks what batches draw with: only the shaders holding
// subpixel glyphs blend with a second color, and the shaders of a set of
// classes are those of the first set holding it.
func TestClassPipelines(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	r, err := New(hiddenWindow(t, 16, 16))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	vs, ps, blend := r.pipeline(&gpu.Batch{Class: gpu.ClassPlain})
	if vs == r.vs || ps == r.ps || blend != r.overBlend {
		t.Fatal("a plain fill does not draw with its class's shaders, blended by its alpha")
	}
	if _, _, blend := r.pipeline(&gpu.Batch{Class: gpu.ClassPlain, Hole: true}); blend != r.overHoleBlend {
		t.Fatal("a plain hole does not blend by its alpha")
	}
	if _, _, blend := r.pipeline(&gpu.Batch{Class: gpu.ClassSubpixel}); blend != r.blend {
		t.Fatal("a subpixel glyph does not blend with a second color")
	}
	if vs, ps, blend := r.pipeline(&gpu.Batch{}); vs != r.vs || ps != r.ps || blend != r.blend {
		t.Fatal("the general class does not draw with the general shader")
	}
	if _, ps, _ := r.pipeline(&gpu.Batch{Joined: 1<<gpu.ClassPlain | 1<<gpu.ClassShadow}); ps == r.ps {
		t.Fatal("joined batches draw with the general shader")
	}
	// A set no shaders are made for draws with those of the set holding all.
	_, ps1, blend1 := r.pipeline(&gpu.Batch{Joined: 1<<gpu.ClassFill | 1<<gpu.ClassGlyph})
	_, ps2, _ := r.pipeline(&gpu.Batch{Joined: 1<<gpu.ClassImage | 1<<gpu.ClassShadow})
	if ps1 == r.ps || ps1 != ps2 || blend1 != r.blend {
		t.Fatal("the set holding all classes does not draw what no other holds")
	}
}

// TestFrameCostFits checks that a frame that takes much of a refresh has the
// next ones build ahead, and that later ones of little cost end that.
func TestFrameCostFits(t *testing.T) {
	c := frameCost{period: 16 * time.Millisecond}
	if !c.fits() {
		t.Fatal("no frames yet, and frames do not fit")
	}
	c.add(2 * time.Millisecond)
	if !c.fits() {
		t.Fatal("a frame of 2 ms of 16 does not fit")
	}
	c.add(14 * time.Millisecond)
	if c.fits() {
		t.Fatal("a frame of 14 ms of 16 fits")
	}
	frames := 0
	for ; !c.fits(); frames++ {
		c.add(2 * time.Millisecond)
	}
	if frames < 2 || frames > 60 {
		t.Fatalf("%d frames after a heavy one until frames fit", frames)
	}
	unknown := frameCost{}
	if unknown.refreshPeriod() != minRefresh {
		t.Fatal("no default refresh period")
	}
}

// TestFrameCostWithoutReady checks that a frame no Ready started, as a resize's
// drawn at once, counts no main-thread time: counted from the last frame Ready
// started, a drag of seconds would have frames build ahead long after it.
func TestFrameCostWithoutReady(t *testing.T) {
	var c frameCost
	c.start = time.Now().Add(-3 * time.Second)
	if d := c.took(); d < 3*time.Second {
		t.Fatalf("a frame Ready started took %v", d)
	}
	if d := c.took(); d >= 0 {
		t.Fatalf("the next frame, which no Ready started, took %v", d)
	}
}

// TestFittingFrameWaitsForComposition checks a frame that fits: it presents at
// once, with nothing left pending, and the next waits for the window manager's
// composition, whose end wakes it, once the screen has room.
func TestFittingFrameWaitsForComposition(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer func(f func(_, _ time.Duration)) { awaitRefresh = f }(awaitRefresh)
	composed := make(chan struct{})
	bounds := make(chan [2]time.Duration, 1)
	awaitRefresh = func(least, most time.Duration) {
		bounds <- [2]time.Duration{least, most}
		<-composed
	}
	defer func(ms uintptr) { latencyTimeout = ms }(latencyTimeout)
	latencyTimeout = 50 // a window that never shows signals no room

	s := gputest.NewChanging(1).Scene()
	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	posted := make(chan func(), 4)
	r.SetPost(func(f func()) { posted <- f })
	r.cost.period = 16 * time.Millisecond
	// However the first frame got there, the second is what the test is about.
	for r.lat.ready(); !r.lat.ready(); {
		r.lat.waitThen(func() { posted <- func() {} })
		wait.For(t, "room", posted)()
	}
	if err := r.Render(s); err != nil {
		t.Fatal(err)
	}
	if r.pending {
		t.Fatal("a frame that fits waits to present")
	}
	if b := wait.For(t, "the wait for the composition", bounds); b != [2]time.Duration{8 * time.Millisecond, 32 * time.Millisecond} {
		t.Fatalf("waits within %v", b)
	}
	woke := make(chan struct{}, 1)
	if r.Ready(func() { woke <- struct{}{} }) {
		t.Fatal("the next frame is ready before the composition")
	}
	close(composed)
	for len(woke) == 0 {
		wait.For(t, "the composition's end, then room", posted)()
	}
	r.cost.estimate = time.Hour // the second frame builds ahead
	if !r.Ready(func() {}) {
		t.Fatal("a frame that does not fit waits before it builds")
	}
}

// TestFrameThatWaitedDoesNotKeepNextOnesWaiting checks that a frame that fits but
// waited to present, as the screen had no room as it drew, has the next wait for
// the composition: woken at once, that would draw before the screen has room and
// wait too, and so would every one after it, each a refresh late for good.
func TestFrameThatWaitedDoesNotKeepNextOnesWaiting(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer func(f func(_, _ time.Duration)) { awaitRefresh = f }(awaitRefresh)
	composed := make(chan struct{})
	awaitRefresh = func(_, _ time.Duration) { <-composed }
	defer func(ms uintptr) { latencyTimeout = ms }(latencyTimeout)
	latencyTimeout = 50 // a window that never shows signals no room

	s := gputest.NewChanging(1).Scene()
	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	posted := make(chan func(), 4)
	r.SetPost(func(f func()) { posted <- f })
	if err := r.draw(s); err != nil {
		t.Fatal(err)
	}
	woke := make(chan struct{}, 1)
	r.wake = func() { woke <- struct{}{} }
	r.pending = true
	r.presentPending()
	if !r.refreshing {
		t.Fatal("the next frame does not wait for the composition")
	}
	if len(woke) != 0 {
		t.Fatal("the next frame woke before the composition")
	}
	close(composed)
	for len(woke) == 0 {
		wait.For(t, "the composition's end, then room", posted)()
	}
}

// TestDwmFlushedShared checks that waits for a refresh share the call under
// way: at most one thread waits, whatever frames come meanwhile.
func TestDwmFlushedShared(t *testing.T) {
	a, b := dwmFlushed(), dwmFlushed()
	if a != b {
		// The first may have returned at once: then the next is a new wait.
		select {
		case <-a:
		default:
			t.Fatal("two calls under way")
		}
	}
	wait.For(t, "the window manager", a)
}
