package capsule

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arion.local/desktop/ui/entities/dictation"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// mixed is dictation that alternates Chinese and English, where shaping adds
// spacing that single-character widths do not have.
func mixed(runes int) string {
	const unit = "今天我们讨论 machine learning 的 deployment 方案，"
	var b strings.Builder
	for len([]rune(b.String())) < runes {
		b.WriteString(unit)
	}
	return string([]rune(b.String())[:runes])
}

// repeated builds runes characters from unit.
func repeated(unit string, runes int) string {
	var b strings.Builder
	for len([]rune(b.String())) < runes {
		b.WriteString(unit)
	}
	return string([]rune(b.String())[:runes])
}

// inkEnd is the right-most column holding text ink, or -1.
func inkEnd(img *image.RGBA) int {
	bg := img.RGBAAt(0, 0)
	end := -1
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Max.X - 1; x > end; x-- {
			p := img.RGBAAt(x, y)
			if d := int(p.R) - int(bg.R); d > 40 || d < -40 {
				end = x
				break
			}
		}
	}
	return end
}

// The end of a long draft stays at the right edge of the line however much
// has been said; it must not drift left as the text grows.
func TestLongDraftEndStaysAtTheRightEdge(t *testing.T) {
	for name, build := range map[string]func(int) string{
		"mixed":   mixed,
		"chinese": func(n int) string { return repeated("今天天气很好我们去公园散步。", n) },
		"english": func(n int) string { return repeated("the quick brown fox jumps over the lazy dog ", n) },
	} {
		t.Run(name, func(t *testing.T) { checkEnd(t, build) })
	}
}

func checkEnd(t *testing.T, build func(int) string) {
	tokens := theme.Resolve(false)
	var ends []int
	for _, runes := range []int{80, 160, 320, 599} {
		state := &State{}
		text := build(runes)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Padding(20).Children(func() {
				draft(c, tokens, state, "s1", dictation.Frame{Text: text}, ui.RGB(0, 0, 0))
			})
		}, 400, 80)
		for deadline := time.Now().Add(1500 * time.Millisecond); time.Now().Before(deadline); {
			tt.Frame()
			time.Sleep(16 * time.Millisecond)
		}
		ends = append(ends, inkEnd(tt.Image()))
		if dir := os.Getenv("DRAFT_DUMP"); dir != "" {
			var b bytes.Buffer
			_ = png.Encode(&b, tt.Image())
			_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s-%d.png", strings.ReplaceAll(t.Name(), "/", "-"), runes)), b.Bytes(), 0o600)
		}
	}
	t.Logf("ink ends by length: %v", ends)
	for _, e := range ends[1:] {
		if d := e - ends[0]; d > 6 || d < -6 {
			t.Fatalf("end of text drifts with length: %v", ends)
		}
	}
}
