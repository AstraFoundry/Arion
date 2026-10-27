package typewriter

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

const width, height = 320, 560

// render shows the figure as it stands elapsed after the step was entered.
func render(dark bool, elapsed *time.Duration) *ui.Tester {
	tt := ui.NewTester(func(c *ui.Context) {
		Render(c, theme.Resolve(dark), copy.For(copy.English), copy.TypewriterFigureLabel, c.Now().Add(-*elapsed))
	}, width, height)
	tt.SetDark(dark)
	return tt
}

func differs(a, b *image.RGBA) int {
	n := 0
	for i := 0; i+3 < len(a.Pix); i += 4 {
		for j := i; j < i+3; j++ {
			if d := int(a.Pix[j]) - int(b.Pix[j]); d > 2 || d < -2 {
				n++
				break
			}
		}
	}
	return n
}

func clone(img *image.RGBA) *image.RGBA {
	out := *img
	out.Pix = append([]byte(nil), img.Pix...)
	return &out
}

func snapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	directory := os.Getenv("ARION_UI_SNAPSHOTS")
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, tt.Image()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name+".png"), buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// The figure turns round, closes in on the paper and types: each stage
// of the arrival is another picture, and so is each stroke after it.
func TestFigureArrivesAndTypes(t *testing.T) {
	for _, dark := range []bool{false, true} {
		appearance := map[bool]string{false: "light", true: "dark"}[dark]
		var elapsed time.Duration
		tt := render(dark, &elapsed)
		var last *image.RGBA
		moments := []struct {
			name string
			at   time.Duration
		}{
			{"0-back", 0}, {"1-turning", 800 * time.Millisecond}, {"2-turned", 1450 * time.Millisecond},
			{"3-closing", 2100 * time.Millisecond}, {"4-front", typingFrom},
			{"5-typing", typingFrom + text.lines[1].start + 30*time.Millisecond},
			{"6-returning", typingFrom + text.lines[3].back + returning/2},
			{"7-page", typingFrom + text.lines[14].start + 40*time.Millisecond},
			{"8-looped", typingFrom + text.pass + text.lines[2].start + 40*time.Millisecond},
		}
		for _, m := range moments {
			elapsed = m.at
			tt.Frame()
			if _, ok := tt.Find(copy.For(copy.English).Text(copy.TypewriterFigureLabel)); !ok {
				t.Fatalf("%s: the figure is not labeled", appearance)
			}
			img := clone(tt.Image())
			if last != nil && differs(last, img) == 0 {
				t.Errorf("%s %s: the figure did not change", appearance, m.name)
			}
			last = img
			snapshot(t, tt, "typewriter-"+appearance+"-"+m.name)
		}
	}
}

// Every character of the story has a glyph in the serif, and every
// letter a key it is typed with.
func TestStoryTypes(t *testing.T) {
	for _, r := range assets.Arion {
		if r == '\n' {
			continue
		}
		if !serif.has(r) {
			t.Errorf("no glyph for %q", r)
		}
		if r != ' ' && keyOf(r) < 0 && !isPunctuation(r) {
			t.Errorf("no key types %q", r)
		}
	}
	for _, k := range keys {
		if !serif.has(k.legend) {
			t.Errorf("no glyph for the key %q", k.legend)
		}
	}
	if len(text.lines) < 20 {
		t.Fatalf("the story set in %d lines", len(text.lines))
	}
	for i, l := range text.lines {
		if len(l.strokes) > 0 && margin+l.end*em > paperW-margin+0.01 {
			t.Errorf("line %d runs %g ems, past the margin", i, l.end)
		}
	}
}

func isPunctuation(r rune) bool { return r == '·' || r == '·' || r == ',' || r == '.' || r == ';' }

// The typing goes on: the carriage steps along a line, returns, and the
// story starts again once typed out.
func TestTypingRuns(t *testing.T) {
	first := text.at(text.lines[0].strokes[3].at + step)
	later := text.at(text.lines[0].strokes[9].at + step)
	if later.pen <= first.pen || later.count <= first.count {
		t.Errorf("the pen went from %v to %v", first, later)
	}
	back := text.at(text.lines[0].back + returning)
	if back.feed != 1 || back.pen != text.lines[1].strokes[0].x {
		t.Errorf("after the return the paper stands at %v", back)
	}
	again := text.at(text.pass + text.lines[0].strokes[3].at + step)
	if again.line != len(text.lines) || again.count != first.count {
		t.Errorf("the second pass stands at %v", again)
	}
}

// With Reduce Motion the figure is the typed page, still.
func TestStillWithReducedMotion(t *testing.T) {
	at := poseAt(0, true)
	if at.turn != 1 || at.zoom != 1 || at.front != 1 || at.typed.line != stillLines {
		t.Errorf("still pose %v", at)
	}
}

// BenchmarkFrame draws a typing frame in front of the paper at Retina
// scale: it has to fit in a frame at 60 Hz.
func BenchmarkFrame(b *testing.B) {
	f := frame{colorsOf(theme.Resolve(false)), poseAt(typingFrom+text.lines[14].start+40*time.Millisecond, false)}
	var v picture
	for i := 0; b.Loop(); i++ {
		f.at.typed.since = time.Duration(i%4) * 20 * time.Millisecond
		v.show(ui.Rect{W: width, H: height}, 2, f)
	}
}
