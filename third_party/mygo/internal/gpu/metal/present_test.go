//go:build darwin

package metal

import (
	"testing"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/wait"
)

// TestPresenter checks the presenter's fetches: a frame is not ready while
// one is in flight, the presenter wakes the window once it has the
// drawable, and taking or draining one waits for a fetch in flight.
func TestPresenter(t *testing.T) {
	if err := load(); err != nil {
		t.Skip("no Objective-C runtime:", err)
	}
	gate := make(chan struct{})
	fetched := 0
	nextDrawable = func(id) id {
		<-gate
		fetched++
		return send(send(class("NSObject"), "new"), "autorelease")
	}
	defer func() { nextDrawable = func(layer id) id { return send(layer, "nextDrawable") } }()
	var p presenter
	defer p.stop()

	if !p.ready(nil) {
		t.Fatal("not ready before any fetch")
	}
	p.start(0)
	woke := make(chan struct{})
	if p.ready(func() { close(woke) }) {
		t.Fatal("ready while a fetch is in flight")
	}
	gate <- struct{}{}
	wait.For(t, "the presenter's wake", woke)
	if !p.ready(nil) {
		t.Fatal("not ready once fetched")
	}
	// A fetched drawable is kept until taken: starting again fetches none.
	p.start(0)
	d := p.take()
	if d == 0 || fetched != 1 {
		t.Fatalf("took %#x after %d fetches", d, fetched)
	}
	pool(func() { send(d, "release") })

	// Taking waits for the fetch in flight, as draining does.
	p.start(0)
	go func() { gate <- struct{}{} }()
	if d = p.take(); d == 0 || fetched != 2 {
		t.Fatalf("took %#x after %d fetches", d, fetched)
	}
	pool(func() { send(d, "release") })
	p.start(0)
	go func() { gate <- struct{}{} }()
	p.drain()
	if fetched != 3 || p.drawable != 0 || p.fetching {
		t.Fatalf("drained after %d fetches: drawable %#x, fetching %v", fetched, p.drawable, p.fetching)
	}
}

// TestPrefetchedFrames renders frames following each other into a layer:
// the first, changing the layer, takes its drawable on the main thread;
// each after presents as soon as drawn and has the presenter fetch the
// next frame's, which that frame takes; a frame resizing the layer drains
// it.
func TestPrefetchedFrames(t *testing.T) {
	if err := load(); err != nil {
		t.Skip("no Metal:", err)
	}
	var layer id
	pool(func() { layer = send(class("CALayer"), "new") })
	if layer == 0 {
		t.Skip("no Core Animation")
	}
	defer pool(func() { release(&layer) })
	r, err := New(uintptr(layer))
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	var fetched int
	next := nextDrawable
	nextDrawable = func(l id) id { fetched++; return next(l) }
	defer func() { nextDrawable = next }()
	frame := func(w, i int) {
		t.Helper()
		s := &scene.Scene{}
		s.Reset(w, 32, scene.Color{A: 255})
		s.Ops = append(s.Ops, scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(i), W: 8, H: 8}, Color: scene.Color{R: 255, A: 255}})
		r.pre.wait()
		parked := drawableID(r.pre.drawable)
		if err := r.Render(s); err != nil {
			t.Fatal(err)
		}
		r.pre.wait()
		if parked != 0 && drawableID(r.pre.drawable) == parked {
			t.Fatalf("frame %d did not take the drawable fetched for it", i)
		}
		if !r.pre.ready(nil) {
			t.Fatalf("frame %d: not ready after the fetch", i)
		}
	}
	frame(64, 0)
	if r.synced == false || r.pre.started {
		t.Fatalf("the first frame: synced %v, presenter started %v", r.synced, r.pre.started)
	}
	for i := 1; i <= 5; i++ {
		frame(64, i)
		if r.pre.drawable == 0 {
			t.Fatalf("frame %d: no drawable fetched for the next", i)
		}
	}
	// The presenter fetched after each frame but the first, which
	// presented with the transaction.
	if fetched != 5 {
		t.Errorf("%d fetches after 6 frames, want 5", fetched)
	}
	frame(96, 6)
	if r.pre.drawable != 0 || !r.synced {
		t.Errorf("a resize: drawable fetched %#x, synced %v", r.pre.drawable, r.synced)
	}
}

// TestPendingFrames renders a frame while its drawable is being fetched:
// it is built at once and waits, frames paced by the display are not
// ready meanwhile, and the presenter has it drawn on the main thread
// (post) once it has the drawable; a frame not paced, as a resize's,
// draws the pending one first.
func TestPendingFrames(t *testing.T) {
	if err := load(); err != nil {
		t.Skip("no Metal:", err)
	}
	var layer id
	pool(func() { layer = send(class("CALayer"), "new") })
	if layer == 0 {
		t.Skip("no Core Animation")
	}
	defer pool(func() { release(&layer) })
	r, err := New(uintptr(layer))
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	posted := make(chan func(), 4)
	r.SetPost(func(fn func()) { posted <- fn })
	gate := make(chan struct{})
	next := nextDrawable
	nextDrawable = func(l id) id { <-gate; return next(l) }
	defer func() { nextDrawable = next }()
	frame := func(w, i int) {
		t.Helper()
		s := &scene.Scene{}
		s.Reset(w, 32, scene.Color{A: 255})
		s.Ops = append(s.Ops, scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(i), W: 8, H: 8}, Color: scene.Color{R: 255, A: 255}})
		if err := r.Render(s); err != nil {
			t.Fatal(err)
		}
	}
	frame(64, 0) // with the transaction: no fetch after
	frame(64, 1) // takes its drawable, then the presenter fetches the next
	if !r.pre.fetching {
		t.Fatal("no drawable being fetched after a frame presented as soon as drawn")
	}
	frame(64, 2) // built, waiting for the fetch
	if r.pending.cb == 0 {
		t.Fatal("the frame did not wait for the drawable being fetched")
	}
	woke := false
	if r.Ready(func() { woke = true }) {
		t.Fatal("ready while a frame waits")
	}
	gate <- struct{}{} // the fetch ends
	finish := wait.For(t, "the post", posted)
	finish()
	if r.pending.cb != 0 || !woke {
		t.Fatalf("after the drawable came: pending %v, woke %v", r.pending.cb != 0, woke)
	}
	if !r.Ready(nil) {
		t.Fatal("not ready once drawn")
	}
	// A frame waiting is drawn first by a frame not paced, as a resize's.
	frame(64, 3)
	if r.pending.cb == 0 {
		t.Fatal("the frame did not wait")
	}
	go func() { gate <- struct{}{} }()
	frame(96, 4)
	if r.pending.cb != 0 || !r.synced || r.pre.fetching {
		t.Fatalf("a resize: pending %v, synced %v, fetching %v", r.pending.cb != 0, r.synced, r.pre.fetching)
	}
	close(gate)
	// The post of the fetch the resize waited for finds nothing pending.
	select {
	case fn := <-posted:
		fn()
	default:
	}
}

// drawableID returns the identity of drawable d, or 0 for none: its
// address may come back for another.
func drawableID(d id) uintptr {
	if d == 0 {
		return 0
	}
	return send(d, "drawableID")
}

// TestDroppedPendingFrame checks that a frame waiting for its drawable,
// dropped as the drawables are trimmed or the renderer let go, wakes what
// waited for it (Ready): the frame paced by the display waiting draws
// then, rather than never.
func TestDroppedPendingFrame(t *testing.T) {
	if err := load(); err != nil {
		t.Skip("no Metal:", err)
	}
	var layer id
	pool(func() { layer = send(class("CALayer"), "new") })
	if layer == 0 {
		t.Skip("no Core Animation")
	}
	defer pool(func() { release(&layer) })
	r, err := New(uintptr(layer))
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	r.SetPost(func(func()) {})
	gate := make(chan struct{})
	next := nextDrawable
	nextDrawable = func(l id) id { <-gate; return next(l) }
	defer func() { nextDrawable = next }()
	frame := func(i int) {
		t.Helper()
		s := &scene.Scene{}
		s.Reset(64, 32, scene.Color{A: 255})
		s.Ops = append(s.Ops, scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(i), W: 8, H: 8}, Color: scene.Color{R: 255, A: 255}})
		if err := r.Render(s); err != nil {
			t.Fatal(err)
		}
	}
	frame(0) // with the transaction: no fetch after
	frame(1) // takes its drawable, then the presenter fetches the next
	frame(2) // built, waiting for the fetch
	if r.pending.cb == 0 {
		t.Fatal("the frame did not wait for the drawable being fetched")
	}
	woke := false
	if r.Ready(func() { woke = true }) {
		t.Fatal("ready while a frame waits")
	}
	pool(r.dropPending)
	if r.pending.cb != 0 || !woke {
		t.Fatalf("dropped: pending %v, woke %v", r.pending.cb != 0, woke)
	}
	close(gate)
	r.pre.drain()
}

// TestPendingFramesSweep checks that frames drawn once their drawable
// comes (finishPending) free the textures of the transient canvases of
// frames before, as frames drawn at once do. Only these swept, so an
// animation, whose frames mostly wait for their drawable, kept a texture
// for every frame's transient canvases: hundreds of megabytes on the GPU
// within seconds of a player's moving lyrics.
func TestPendingFramesSweep(t *testing.T) {
	if err := load(); err != nil {
		t.Skip("no Metal:", err)
	}
	var layer id
	pool(func() { layer = send(class("CALayer"), "new") })
	if layer == 0 {
		t.Skip("no Core Animation")
	}
	defer pool(func() { release(&layer) })
	r, err := New(uintptr(layer))
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	posted := make(chan func(), 4)
	r.SetPost(func(fn func()) { posted <- fn })
	gate := make(chan struct{})
	next := nextDrawable
	nextDrawable = func(l id) id { <-gate; return next(l) }
	defer func() { nextDrawable = next }()
	frame := func(i int) {
		t.Helper()
		s := &scene.Scene{}
		s.Reset(64, 32, scene.Color{A: 255})
		s.Ops = append(s.Ops, scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(i % 56), W: 8, H: 8}, Color: scene.Color{R: 255, A: 255}})
		c := scene.NewCanvas(16, 16)
		c.Transient = true
		s.Canvases = []*scene.Image{c}
		if err := r.Render(s); err != nil {
			t.Fatal(err)
		}
	}
	frame(0) // with the transaction: no fetch after
	frame(1) // takes its drawable, then the presenter fetches the next
	for i := 2; i < 2+3*sweepEvery; i++ {
		frame(i) // built, waiting for the fetch
		if r.pending.cb == 0 {
			t.Fatalf("frame %d did not wait for its drawable", i)
		}
		gate <- struct{}{} // the fetch ends
		finish := wait.For(t, "the post", posted)
		finish()
	}
	if n := len(r.canvases); n > sweepEvery+1 {
		t.Errorf("%d canvas textures after %d frames waiting for their drawables, want at most %d", n, 3*sweepEvery, sweepEvery+1)
	}
	close(gate)
	r.pre.drain()
}

// TestStalePost checks that the post of a fetch whose frame pending a
// frame not paced drew meanwhile neither blocks the main thread for the
// next frame's drawable, being fetched, nor draws that frame before it
// comes: the next fetch posts it.
func TestStalePost(t *testing.T) {
	if err := load(); err != nil {
		t.Skip("no Metal:", err)
	}
	var layer id
	pool(func() { layer = send(class("CALayer"), "new") })
	if layer == 0 {
		t.Skip("no Core Animation")
	}
	defer pool(func() { release(&layer) })
	r, err := New(uintptr(layer))
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	posted := make(chan func(), 4)
	r.SetPost(func(fn func()) { posted <- fn })
	gate := make(chan struct{})
	next := nextDrawable
	nextDrawable = func(l id) id { <-gate; return next(l) }
	defer func() { nextDrawable = next }()
	defer close(gate)
	frame := func(i int) {
		t.Helper()
		s := &scene.Scene{}
		s.Reset(64, 32, scene.Color{A: 255})
		s.Ops = append(s.Ops, scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(i), W: 8, H: 8}, Color: scene.Color{R: 255, A: 255}})
		if err := r.Render(s); err != nil {
			t.Fatal(err)
		}
	}
	frame(0) // with the transaction: no fetch after
	frame(1) // takes its drawable, then the presenter fetches the next
	frame(2) // built, waiting for the fetch
	gate <- struct{}{}
	stale := wait.For(t, "the post", posted)
	// A frame not paced draws the one pending, the presenter fetches the
	// next drawable, and this frame waits for it.
	frame(3)
	if r.pending.cb == 0 || !r.pre.fetching {
		t.Fatalf("pending %v, fetching %v", r.pending.cb != 0, r.pre.fetching)
	}
	stale() // would wait for the fetch, the gate closed
	if r.pending.cb == 0 {
		t.Fatal("the stale post drew the frame pending")
	}
	gate <- struct{}{}
	finish := wait.For(t, "the post of the next fetch", posted)
	finish()
	if r.pending.cb != 0 {
		t.Fatal("the frame pending was not drawn once its drawable came")
	}
}
