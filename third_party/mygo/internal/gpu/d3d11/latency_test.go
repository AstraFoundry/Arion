//go:build windows

package d3d11

import (
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/wait"
)

var (
	procCreateSemaphore  = kernel32.NewProc("CreateSemaphoreW")
	procReleaseSemaphore = kernel32.NewProc("ReleaseSemaphore")
)

// TestFrameLatencyTokens checks the frame latency object's tokens, with a
// semaphore standing in for DXGI's: a present is ready when the object is
// signaled, holding its token until a present takes it; otherwise a
// goroutine waits for it and calls on; a present drawn at once waits for
// that goroutine; and each present takes one, so the count stays the
// frames the screen can take.
func TestFrameLatencyTokens(t *testing.T) {
	h, _, err := procCreateSemaphore.Call(0, 0, 8, 0)
	if h == 0 {
		t.Fatal(err)
	}
	l := latency{handle: h}
	defer l.release()
	signal := func() { procReleaseSemaphore.Call(h, 1, 0) }

	if l.ready() {
		t.Fatal("ready before the screen took a frame")
	}
	woke := make(chan struct{}, 2)
	l.waitThen(func() { woke <- struct{}{} })
	l.waitThen(func() { woke <- struct{}{} }) // the same wait
	if l.ready() {
		t.Fatal("ready while waiting")
	}
	signal()
	wait.For(t, "the wait's end", woke)
	if len(woke) != 0 {
		t.Fatal("one wait called on twice")
	}
	if !l.ready() || !l.ready() {
		t.Fatal("not ready with the token")
	}
	// A present takes it: the next is not ready.
	l.take()
	if l.ready() {
		t.Fatal("ready after a present took the token")
	}
	// A present drawn at once waits for the goroutine waiting.
	l.waitThen(nil)
	go signal()
	l.take()
	if l.token || l.waiting {
		t.Fatalf("after a present drawn at once: token %v, waiting %v", l.token, l.waiting)
	}
	// Signaled when asked: ready at once, with no goroutine.
	signal()
	if !l.ready() || l.waiting {
		t.Fatal("not ready at once when signaled")
	}
	l.take()
	// Every success taken: the count is back to none.
	if l.waitObject(0) {
		t.Fatal("the object's count drifted")
	}
}

// TestLapsedWaitPresentsAtOnce checks that a present whose goroutine's
// wait on the object timed out, as it does while the window shows
// nowhere, takes no token and does not wait for one again on the main
// thread, which would stall it a whole timeout more for each such frame.
func TestLapsedWaitPresentsAtOnce(t *testing.T) {
	defer func(ms uintptr) { latencyTimeout = ms }(latencyTimeout)
	latencyTimeout = 400
	h, _, err := procCreateSemaphore.Call(0, 0, 8, 0)
	if h == 0 {
		t.Fatal(err)
	}
	l := latency{handle: h}
	defer l.release()

	woke := make(chan struct{}, 1)
	l.waitThen(func() { woke <- struct{}{} })
	wait.For(t, "the wait's timeout", woke)
	start := time.Now()
	l.take()
	if d := time.Since(start); d >= 200*time.Millisecond {
		t.Fatalf("the present waited %v again after a lapsed wait", d)
	}
	if l.token || l.lapsed {
		t.Fatalf("after the present: token %v, lapsed %v", l.token, l.lapsed)
	}
	// The next present that the screen has room for takes its token as
	// before.
	procReleaseSemaphore.Call(h, 1, 0)
	if !l.ready() {
		t.Fatal("not ready when signaled")
	}
	l.take()
	if l.waitObject(0) {
		t.Fatal("the object's count drifted")
	}
}

// TestUnchangedFrameWaitsForRefresh checks that a frame that changes
// nothing, and so presents nothing, still has the next wait for the
// display (Ready), until the refresh it waits for comes: an animation
// whose frames move less than a pixel keeps the display's pace rather
// than draw as fast as the main thread can (#608).
func TestUnchangedFrameWaitsForRefresh(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	refresh := make(chan struct{})
	defer func(f func(_, _ time.Duration)) { awaitRefresh = f }(awaitRefresh)
	awaitRefresh = func(_, _ time.Duration) { <-refresh }

	s := gputest.NewChanging(1).Scene()
	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()
	if r.ctx1 == 0 {
		t.Skip("no Direct3D 11.1")
	}
	posted := make(chan func(), 4)
	r.SetPost(func(f func()) { posted <- f })
	r.cost.estimate = time.Hour // frames that go through the frame latency object
	if err := r.Render(s); err != nil {
		t.Fatal(err)
	}
	for r.pending {
		wait.For(t, "the first frame's present", posted)() // once the screen has room
	}
	if !r.Ready(func() {}) {
		t.Fatal("not ready after a frame presented")
	}
	if err := r.Render(s); err != nil {
		t.Fatal(err)
	}
	if r.redrew != redrewNothing {
		t.Fatal("the same scene drew again")
	}
	woke := make(chan struct{}, 2)
	if r.Ready(func() { woke <- struct{}{} }) {
		t.Fatal("ready at once after a frame that changed nothing")
	}
	// The next frame that changes nothing waits for the same refresh.
	if err := r.Render(s); err != nil {
		t.Fatal(err)
	}
	refresh <- struct{}{}
	wait.For(t, "the refresh's end", posted)()
	wait.For(t, "the wake", woke)
	if !r.Ready(func() { woke <- struct{}{} }) {
		t.Fatal("not ready after the refresh")
	}
	if len(woke) != 0 || len(posted) != 0 {
		t.Fatal("woken twice")
	}
}
