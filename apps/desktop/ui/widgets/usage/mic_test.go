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

// still is the pose the loop starts and ends on: off, quiet.
var still = micPose{rings: [rings]float32{-1, -1, -1}}

// The loop closes on the still pose, and nothing jumps between frames: a
// ring fades in from nothing and out to nothing.
func TestMicLoopIsSeamless(t *testing.T) {
	if p := micAt(0); p != still {
		t.Fatalf("still pose %+v", p)
	}
	if p := micAt(time.Duration(micLoop) - time.Millisecond); p.on > 0.01 || p.rings != still.rings {
		t.Fatalf("the loop ends on %+v", p)
	}
	visible := func(u float32) float32 {
		if u < 0 {
			return 0
		}
		return min(1, u*6) * (1 - u)
	}
	prev := micAt(0)
	lit := false
	for t0 := time.Duration(0); t0 < time.Duration(micLoop); t0 += 16 * time.Millisecond {
		p := micAt(t0)
		if math.Abs(float64(p.on-prev.on)) > 0.15 {
			t.Fatalf("the switch jumps at %v", t0)
		}
		for i := range p.rings {
			if math.Abs(float64(visible(p.rings[i])-visible(prev.rings[i]))) > 0.15 {
				t.Fatalf("ring %d pops at %v", i, t0)
			}
		}
		lit = lit || p.on == 1
		prev = p
	}
	if !lit {
		t.Fatal("the band never lights fully")
	}
}

// The figure draws in both appearances, and its rings show.
func TestMicDraws(t *testing.T) {
	const w, h = 220, 172
	for _, dark := range []bool{false, true} {
		appearance := map[bool]string{false: "light", true: "dark"}[dark]
		quiet := paint(dark, w, h, still)
		ink := 0
		for i := 3; i < len(quiet.Pix); i += 4 {
			if quiet.Pix[i] > 0 {
				ink++
			}
		}
		if ink < len(quiet.Pix)/4/10 {
			t.Fatalf("%s: the figure covers only %d pixels", appearance, ink)
		}
		if bytes.Equal(quiet.Pix, paint(dark, w, h, micAt(1500*time.Millisecond)).Pix) {
			t.Fatalf("%s: the rings do not show", appearance)
		}
		for ms := 0; ms < int(micLoop/float64(time.Millisecond)); ms += 300 {
			save(t, paint(dark, w, h, micAt(time.Duration(ms)*time.Millisecond)), fmt.Sprintf("mic-%s-%04d", appearance, ms))
		}
	}
}
