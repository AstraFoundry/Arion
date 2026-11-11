package ui

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/wait"
)

var (
	videoRed   = color.RGBA{255, 0, 0, 255}
	videoGreen = color.RGBA{0, 255, 0, 255}
	videoBlue  = color.RGBA{0, 0, 255, 255}
	videoWhite = color.RGBA{255, 255, 255, 255}
)

// solidGIF returns a GIF of 4×4 frames of solid colors, each showing for
// delay hundredths of a second.
func solidGIF(t *testing.T, delay int, colors ...color.RGBA) []byte {
	t.Helper()
	g := &gif.GIF{Config: image.Config{Width: 4, Height: 4}}
	for _, c := range colors {
		f := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{c})
		g.Image = append(g.Image, f)
		g.Delay = append(g.Delay, delay)
		g.Disposal = append(g.Disposal, gif.DisposalNone)
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// newVideo returns a player of a GIF of solid colors, a tenth of a second
// each, closed as the test ends.
func newVideo(t *testing.T, colors ...color.RGBA) *VideoPlayer {
	t.Helper()
	d, err := NewGIFDecoder(solidGIF(t, 10, colors...))
	if err != nil {
		t.Fatal(err)
	}
	p, err := NewVideoPlayer(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

// videoTester shows p, 4×4 pixels, as 16×16 DIPs: shows returns the color
// at its middle.
func videoTester(p *VideoPlayer, visible *bool) (tt *Tester, shows func() color.RGBA) {
	tt = coreNewTester(func(c *context) {
		if visible == nil || *visible {
			coreVideo(c, p).Size(16, 16).Fit(FillBox)
		}
	}, 40, 40)
	return tt, func() color.RGBA { return tt.Image().RGBAAt(8, 8) }
}

// decodedAhead waits for the goroutine decoding p to have n frames ready.
func decodedAhead(t *testing.T, p *VideoPlayer, n int) {
	t.Helper()
	wait.Until(t, "the frames decoded ahead", func() bool { return p.frames != nil && len(p.frames) >= n })
}

// TestGIFDecoderFrames checks that a GIF's frames draw over what the ones
// before left as their disposal says, with their transparent pixels
// showing what is under them, for as long as their delays say, and that
// the decoder starts over after Rewind.
func TestGIFDecoderFrames(t *testing.T) {
	transparent := color.RGBA{}
	full := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{videoRed})
	// Green over the top left, but for a transparent pixel at 1,1.
	corner := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{videoGreen, transparent})
	corner.SetColorIndex(1, 1, 1)
	low := image.NewPaletted(image.Rect(2, 2, 4, 4), color.Palette{videoBlue})
	dot := image.NewPaletted(image.Rect(3, 0, 4, 1), color.Palette{videoWhite})
	g := &gif.GIF{
		Config:   image.Config{Width: 4, Height: 4},
		Image:    []*image.Paletted{full, corner, low, dot},
		Delay:    []int{0, 5, 20, 1},
		Disposal: []byte{gif.DisposalNone, gif.DisposalBackground, gif.DisposalPrevious, gif.DisposalNone},
	}
	var b bytes.Buffer
	if err := gif.EncodeAll(&b, g); err != nil {
		t.Fatal(err)
	}
	d, err := NewGIFDecoder(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	type px struct {
		x, y int
		c    color.RGBA
	}
	frames := []struct {
		d    time.Duration
		want []px
	}{
		{100 * time.Millisecond, []px{{0, 0, videoRed}, {3, 3, videoRed}}},
		{50 * time.Millisecond, []px{{0, 0, videoGreen}, {1, 1, videoRed}, {3, 3, videoRed}}},
		// The green corner went, leaving nothing; blue covers the bottom
		// right.
		{200 * time.Millisecond, []px{{0, 0, transparent}, {1, 1, transparent}, {3, 3, videoBlue}, {3, 0, videoRed}}},
		// The blue went back to what was under it.
		{100 * time.Millisecond, []px{{3, 0, videoWhite}, {3, 3, videoRed}, {0, 0, transparent}}},
	}
	check := func(round int) {
		for i, want := range frames {
			f, err := d.Next()
			if err != nil {
				t.Fatalf("round %d, frame %d: %v", round, i, err)
			}
			if f.Duration != want.d {
				t.Errorf("round %d, frame %d shows for %v, want %v", round, i, f.Duration, want.d)
			}
			img := f.Image.(*image.RGBA)
			for _, p := range want.want {
				if got := img.RGBAAt(p.x, p.y); got != p.c {
					t.Errorf("round %d, frame %d at %d,%d: %v, want %v", round, i, p.x, p.y, got, p.c)
				}
			}
		}
		if _, err := d.Next(); err != io.EOF {
			t.Errorf("round %d: after the last frame, %v", round, err)
		}
	}
	check(1)
	if err := d.Rewind(); err != nil {
		t.Fatal(err)
	}
	check(2)

	if _, err := NewGIFDecoder([]byte("GIF89a, not quite")); err == nil {
		t.Error("data that is no GIF decoded")
	}
}

// TestVideoPlays checks that a video shows each frame as it comes due,
// waking the window with After rather than asking for every frame of the
// display, and that it ends at its last frame, or plays again when it
// loops.
func TestVideoPlays(t *testing.T) {
	p := newVideo(t, videoRed, videoGreen, videoBlue)
	tt, shows := videoTester(p, nil)
	start := tt.Now()
	if got := shows(); got != videoRed {
		t.Fatalf("first frame: %v", got)
	}
	if tt.FramePending() || !tt.h.afterAt.Equal(start.Add(100*time.Millisecond)) {
		t.Fatalf("asked for a frame %v, After due %v after the start: want only After, 100ms", tt.FramePending(), tt.h.afterAt.Sub(start))
	}
	decodedAhead(t, p, 2)
	builds := tt.rt.frame
	tt.Advance(99 * time.Millisecond)
	if got := shows(); got != videoRed || tt.rt.frame != builds {
		t.Fatalf("before the second frame is due: %v, %d frames built", got, tt.rt.frame-builds)
	}
	tt.Advance(time.Millisecond)
	if got := shows(); got != videoGreen || tt.rt.frame != builds+1 {
		t.Fatalf("as the second frame is due: %v, %d frames built", got, tt.rt.frame-builds)
	}
	decodedAhead(t, p, 2) // the third, and the end
	tt.Advance(100 * time.Millisecond)
	if got := shows(); got != videoBlue {
		t.Fatalf("third frame: %v", got)
	}
	tt.Advance(time.Second)
	if got := shows(); got != videoBlue || p.Playing() || p.Err() != nil || !tt.h.afterAt.IsZero() {
		t.Fatalf("after the end: %v, playing %v, error %v, After due %v", got, p.Playing(), p.Err(), tt.h.afterAt)
	}

	// Played again, it starts over.
	p.Play()
	tt.Frame()
	if got := shows(); got != videoRed || !p.Playing() {
		t.Fatalf("played again: %v, playing %v", got, p.Playing())
	}
	decodedAhead(t, p, 2)
	tt.Advance(100 * time.Millisecond)
	if got := shows(); got != videoGreen {
		t.Fatalf("played again, second frame: %v", got)
	}

	// Looping, the first frame follows the last.
	p = newVideo(t, videoRed, videoGreen).SetLoop(true)
	tt, shows = videoTester(p, nil)
	for i, want := range []color.RGBA{videoGreen, videoRed, videoGreen, videoRed} {
		decodedAhead(t, p, 2)
		tt.Advance(100 * time.Millisecond)
		if got := shows(); got != want {
			t.Fatalf("looping, frame %d: %v, want %v", i+1, got, want)
		}
	}
	if !p.Playing() {
		t.Error("a looping video stopped")
	}
}

// TestVideoPause checks that a paused video holds its frame and asks for
// no frames, and goes on from that frame, not from where the time spent
// paused would put it.
func TestVideoPause(t *testing.T) {
	p := newVideo(t, videoRed, videoGreen, videoBlue)
	tt, shows := videoTester(p, nil)
	decodedAhead(t, p, 2)
	tt.Advance(150 * time.Millisecond)
	p.Pause()
	tt.Frame()
	if got := shows(); got != videoGreen || p.Playing() || !tt.h.afterAt.IsZero() {
		t.Fatalf("paused: %v, playing %v, After due %v", got, p.Playing(), tt.h.afterAt)
	}
	tt.Advance(10 * time.Second)
	p.Play()
	tt.Frame()
	if got := shows(); got != videoGreen {
		t.Fatalf("played after a pause: %v", got)
	}
	decodedAhead(t, p, 2)
	tt.Advance(50 * time.Millisecond)
	if got := shows(); got != videoBlue {
		t.Fatalf("the rest of the second frame after a pause: %v", got)
	}
}

// slowDecoder decodes 1×1 frames of the colors given, each a tenth of a
// second, those after the first only as the test lets it.
type slowDecoder struct {
	colors []color.RGBA
	next   int
	gate   chan struct{}
	closed atomic.Bool
}

func (d *slowDecoder) Next() (VideoFrame, error) {
	if d.next >= len(d.colors) {
		return VideoFrame{}, io.EOF
	}
	if d.next > 0 {
		<-d.gate
	}
	c := d.colors[d.next]
	d.next++
	img := &image.RGBA{Pix: []uint8{c.R, c.G, c.B, c.A}, Stride: 4, Rect: image.Rect(0, 0, 1, 1)}
	return VideoFrame{Image: img, Duration: 100 * time.Millisecond}, nil
}

func (d *slowDecoder) Rewind() error { d.next = 0; return nil }
func (d *slowDecoder) Close() error  { d.closed.Store(true); return nil }

// TestVideoDropsLateFrames checks that a decoder slower than the display
// delays no frame: the window waits for the frame due without asking for
// frames meanwhile, the decoder wakes it as it has one, and of the frames
// that came due meanwhile only the last shows, at the time the clock puts
// the video at, rather than each in turn behind it.
func TestVideoDropsLateFrames(t *testing.T) {
	gate := make(chan struct{}, 8)
	d := &slowDecoder{colors: []color.RGBA{videoRed, videoGreen, videoBlue, videoWhite}, gate: gate}
	p, err := NewVideoPlayer(d)
	if err != nil {
		t.Fatal(err)
	}
	tt, shows := videoTester(p, nil)
	start := tt.Now()
	// The second frame is due, but not decoded: the window waits for it.
	tt.Advance(250 * time.Millisecond)
	if got := shows(); got != videoRed || tt.FramePending() || !tt.h.afterAt.IsZero() {
		t.Fatalf("waiting for the decoder: %v, asked for a frame %v, After due %v", got, tt.FramePending(), tt.h.afterAt)
	}
	version := p.bitmap.img.Version()
	gate <- struct{}{}
	gate <- struct{}{}
	wait.Until(t, "the decoder to wake the window", tt.FramePending)
	decodedAhead(t, p, 2)
	tt.Frame()
	// At 250 ms the third frame shows, the second passed over unshown, and
	// the fourth is due at 300 ms.
	if got := shows(); got != videoBlue {
		t.Fatalf("after the decoder caught up: %v", got)
	}
	if v := p.bitmap.img.Version(); v != version+1 {
		t.Errorf("the bitmap updated %d times for the frames that came due together", v-version)
	}
	if due := tt.h.afterAt.Sub(start); due != 300*time.Millisecond {
		t.Errorf("the next frame is due at %v", due)
	}
	// Closed while the decoder waits for the fourth frame, the player
	// closes the decoder once it decoded it.
	p.Close()
	close(gate)
	wait.Until(t, "the decoder to close", d.closed.Load)
}

// TestVideoReduceMotion checks that a video shows its first frame while
// the desktop asks for less motion, decoding and asking for nothing more,
// and that it goes on from where it stood once the desktop no longer asks.
func TestVideoReduceMotion(t *testing.T) {
	p := newVideo(t, videoRed, videoGreen, videoBlue)
	tt := coreNewTester(func(c *context) {}, 40, 40)
	tt.SetPreferences(Preferences{ReduceMotion: true})
	tt.rt.view = func(c *context) { coreVideo(c, p).Size(16, 16).Fit(FillBox) }
	tt.Frame()
	shows := func() color.RGBA { return tt.Image().RGBAAt(8, 8) }
	tt.Advance(time.Second)
	if got := shows(); got != videoRed || p.frames != nil || tt.FramePending() || !tt.h.afterAt.IsZero() {
		t.Fatalf("less motion: %v, decoding %v, asked for a frame %v, After due %v", got, p.frames != nil, tt.FramePending(), tt.h.afterAt)
	}

	// Less motion asked for as it plays: the first frame, then where it
	// stood.
	tt.SetPreferences(Preferences{})
	decodedAhead(t, p, 2)
	tt.Advance(100 * time.Millisecond)
	if got := shows(); got != videoGreen {
		t.Fatalf("playing: %v", got)
	}
	tt.SetPreferences(Preferences{ReduceMotion: true})
	if got := shows(); got != videoRed || !tt.h.afterAt.IsZero() {
		t.Fatalf("less motion while playing: %v, After due %v", got, tt.h.afterAt)
	}
	tt.Advance(time.Second)
	tt.SetPreferences(Preferences{})
	if got := shows(); got != videoGreen {
		t.Fatalf("motion again: %v", got)
	}
	tt.Advance(100 * time.Millisecond)
	if got := shows(); got != videoBlue {
		t.Fatalf("motion again, the next frame: %v", got)
	}
}

// TestVideoStopsWhileHidden checks that a video the view stops building,
// or in a window out of sight, neither moves on nor asks for frames, and
// goes on from where it stood once it shows again.
func TestVideoStopsWhileHidden(t *testing.T) {
	p := newVideo(t, videoRed, videoGreen, videoBlue)
	visible := true
	tt, shows := videoTester(p, &visible)
	decodedAhead(t, p, 2)
	tt.Advance(150 * time.Millisecond)
	if got := shows(); got != videoGreen {
		t.Fatalf("playing: %v", got)
	}
	visible = false
	tt.Frame()
	if tt.FramePending() || !tt.h.afterAt.IsZero() {
		t.Fatalf("out of the view: asked for a frame %v, After due %v", tt.FramePending(), tt.h.afterAt)
	}
	tt.Advance(10 * time.Second)
	visible = true
	tt.Frame()
	if got := shows(); got != videoGreen {
		t.Fatalf("back in the view: %v", got)
	}
	decodedAhead(t, p, 2)
	tt.Advance(100 * time.Millisecond)
	if got := shows(); got != videoBlue {
		t.Fatalf("back in the view, the next frame: %v", got)
	}

	// Out of sight, the frame the video asks for waits for the window to
	// show, and no other comes.
	p = newVideo(t, videoRed, videoGreen, videoBlue)
	tt, shows = videoTester(p, nil)
	decodedAhead(t, p, 2)
	tt.Advance(50 * time.Millisecond)
	tt.h.hidden = true
	version, builds := p.bitmap.img.Version(), tt.rt.frame
	tt.Advance(10 * time.Second)
	if !tt.rt.held || tt.rt.frame != builds || p.bitmap.img.Version() != version || !tt.h.afterAt.IsZero() {
		t.Fatalf("out of sight: held %v, %d frames built, bitmap updated %v, After due %v",
			tt.rt.held, tt.rt.frame-builds, p.bitmap.img.Version() != version, tt.h.afterAt)
	}
	tt.h.hidden = false
	tt.send(platform.SurfaceEvent{Kind: platform.SurfaceShown})
	if got := shows(); got != videoRed || tt.rt.frame == builds {
		t.Fatalf("shown again: %v, built %v", got, tt.rt.frame != builds)
	}
	// The 50 ms it showed before the window went out of sight, after its
	// last frame, do not count: the first frame shows for 100 ms more.
	tt.Advance(99 * time.Millisecond)
	if got := shows(); got != videoRed {
		t.Fatalf("shown again, before the second frame is due: %v", got)
	}
	tt.Advance(time.Millisecond)
	if got := shows(); got != videoGreen {
		t.Fatalf("shown again, the second frame: %v", got)
	}
}

// TestNewVideoPlayerErrors checks that a video of no frames, or whose
// first frame does not decode, makes no player.
func TestNewVideoPlayerErrors(t *testing.T) {
	if _, err := NewVideoPlayer(&slowDecoder{}); err == nil {
		t.Error("a video of no frames made a player")
	}
	broken := errors.New("broken")
	if _, err := NewVideoPlayer(failingDecoder{broken}); !errors.Is(err, broken) {
		t.Errorf("a first frame that does not decode: %v", err)
	}
}

type failingDecoder struct{ err error }

func (d failingDecoder) Next() (VideoFrame, error) { return VideoFrame{}, d.err }
func (d failingDecoder) Rewind() error             { return nil }
