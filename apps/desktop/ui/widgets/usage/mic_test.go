package usage

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"arion.local/desktop/ui/shared/iso"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// paint draws the figure at pose in a box of w by h DIPs at 2×.
func paint(dark bool, w, h float32, pose micPose) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(w*2), int(h*2)))
	var k iso.Canvas
	view := ui.Rect{W: w, H: h}
	k.Begin(img, view, 2)
	drawMic(&k, view, micInk(theme.Resolve(dark)), pose)
	k.End()
	return img
}

func save(t *testing.T, img image.Image, name string) {
	t.Helper()
	directory := os.Getenv("ARION_UI_SNAPSHOTS")
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name+".png"), buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// The loop starts and ends turned away, turns round and settles aimed
// with the band lit, and nothing jumps between frames.
func TestMicLoopIsSeamless(t *testing.T) {
	if p := micAt(0); p.yaw != awayYaw || math.Abs(float64(p.tilt-awayTilt)) > 0.01 || p.on != 0 {
		t.Fatalf("the loop starts at %+v", p)
	}
	end := micAt(time.Duration(micLoop) - time.Millisecond)
	if math.Abs(float64(end.yaw-awayYaw)) > 0.05 || math.Abs(float64(end.tilt-awayTilt)) > 0.05 || end.on > 0.001 {
		t.Fatalf("the loop ends at %+v", end)
	}
	if p := micAt(2500 * time.Millisecond); p.yaw != aimedYaw || math.Abs(float64(p.tilt-aimedTilt)) > 0.01 || p.on != 1 {
		t.Fatalf("aimed pose %+v", p)
	}
	prev, past := micAt(0), float32(aimedTilt)
	for t0 := time.Duration(0); t0 < time.Duration(micLoop); t0 += 16 * time.Millisecond {
		p := micAt(t0)
		if math.Abs(float64(p.yaw-prev.yaw)) > 4 || math.Abs(float64(p.tilt-prev.tilt)) > 4 || math.Abs(float64(p.on-prev.on)) > 0.15 {
			t.Fatalf("the pose jumps at %v: %+v after %+v", t0, p, prev)
		}
		past = max(past, p.tilt)
		prev = p
	}
	// The tip settles past its mark and back, by a few degrees.
	if over := past - aimedTilt; over < 1 || over > 8 {
		t.Fatalf("the tip overshoots %.1f°", over)
	}
}

// The figure draws in both appearances, and it moves.
func TestMicDraws(t *testing.T) {
	const w, h = 160, 172
	for _, dark := range []bool{false, true} {
		appearance := map[bool]string{false: "light", true: "dark"}[dark]
		ready := paint(dark, w, h, aimed)
		ink := 0
		for i := 3; i < len(ready.Pix); i += 4 {
			if ready.Pix[i] > 0 {
				ink++
			}
		}
		if ink < len(ready.Pix)/4/10 {
			t.Fatalf("%s: the figure covers only %d pixels", appearance, ink)
		}
		if bytes.Equal(ready.Pix, paint(dark, w, h, micAt(0)).Pix) {
			t.Fatalf("%s: turning away does not show", appearance)
		}
		save(t, ready, "mic-"+appearance+"-aimed")
		for ms := 0; ms < int(micLoop/float64(time.Millisecond)); ms += 200 {
			save(t, paint(dark, w, h, micAt(time.Duration(ms)*time.Millisecond)), fmt.Sprintf("mic-%s-%04d", appearance, ms))
		}
	}
}
