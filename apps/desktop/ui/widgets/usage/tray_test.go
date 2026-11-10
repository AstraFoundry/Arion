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
func paint(dark bool, w, h float32, pose trayPose) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, int(w*2), int(h*2)))
	var k iso.Canvas
	view := ui.Rect{W: w, H: h}
	k.Begin(img, view, 2)
	drawTray(&k, view, trayInk(theme.Resolve(dark)), pose)
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

// The still pose is the sheet waiting, and the loop closes on it, so the
// figure never jumps between cycles.
func TestTrayLoopIsSeamless(t *testing.T) {
	still := trayAt(0)
	if still != (trayPose{lift: 1}) {
		t.Fatalf("still pose %+v", still)
	}
	end := trayAt(trayLoop - time.Millisecond)
	for name, d := range map[string]float32{"lift": end.lift - 1, "paper": end.paper, "typed": end.typed, "live": end.live, "give": end.give} {
		if math.Abs(float64(d)) > 0.01 {
			t.Errorf("%s ends %.3f off the still pose", name, d)
		}
	}
	// Every value moves without a step from one frame to the next.
	prev := trayAt(0)
	for t0 := time.Duration(0); t0 < trayLoop; t0 += 16 * time.Millisecond {
		p := trayAt(t0)
		for name, d := range map[string]float32{"lift": p.lift - prev.lift, "paper": p.paper - prev.paper, "typed": p.typed - prev.typed, "live": p.live - prev.live, "give": p.give - prev.give} {
			if math.Abs(float64(d)) > 0.15 {
				t.Fatalf("%s jumps %.3f at %v", name, d, t0)
			}
		}
		prev = p
	}
	// The sheet lands as paper with its text written before it lifts.
	landed := trayAt(3 * time.Second)
	if landed.lift != 0 || landed.paper != 1 || landed.typed != 1 {
		t.Fatalf("landed pose %+v", landed)
	}
}

// The figure is drawn in both appearances and changes as the loop runs.
func TestTrayDraws(t *testing.T) {
	const w, h = 300, 180
	for _, dark := range []bool{false, true} {
		appearance := map[bool]string{false: "light", true: "dark"}[dark]
		waiting := paint(dark, w, h, trayAt(0))
		ink := 0
		for i := 3; i < len(waiting.Pix); i += 4 {
			if waiting.Pix[i] > 0 {
				ink++
			}
		}
		if ink < len(waiting.Pix)/4/10 {
			t.Fatalf("%s: the figure covers only %d pixels", appearance, ink)
		}
		landed := paint(dark, w, h, trayAt(3*time.Second))
		if bytes.Equal(waiting.Pix, landed.Pix) {
			t.Fatalf("%s: the landed sheet draws as the waiting one", appearance)
		}
		for ms := 0; ms < int(trayLoop/time.Millisecond); ms += 300 {
			save(t, paint(dark, w, h, trayAt(time.Duration(ms)*time.Millisecond)), fmt.Sprintf("tray-%s-%04d", appearance, ms))
		}
	}
}
