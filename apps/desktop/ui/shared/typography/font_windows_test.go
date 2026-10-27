//go:build windows

package typography

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Compare native shaping with metrics from the actual bundled font, so an
// installed font or generic system substitution cannot pass the speech check.
func TestBundledSpeechFontGlyphMetrics(t *testing.T) {
	if err := Register(); err != nil {
		t.Fatal(err)
	}
	original, err := sfnt.Parse(regular)
	if err != nil {
		t.Fatal(err)
	}
	var buffer sfnt.Buffer
	for _, character := range []rune{'m', '1', '\u4f60'} {
		index, err := original.GlyphIndex(&buffer, character)
		if err != nil || index == 0 {
			t.Fatalf("bundled glyph %U missing: %v", character, err)
		}
		advance, err := original.GlyphAdvance(&buffer, index, fixed.I(16), font.HintingNone)
		if err != nil {
			t.Fatal(err)
		}
		glyphs := ui.Shape(string(character), ui.Font{Family: Family, Size: 16})
		if len(glyphs) != 1 {
			t.Fatalf("expected one glyph for %U, got %d", character, len(glyphs))
		}
		expected := float64(advance) / 64
		if math.Abs(float64(glyphs[0].Advance)-expected) > 0.05 {
			t.Errorf("native %U advance %g differs from bundled Source Han %g", character, glyphs[0].Advance, expected)
		}
	}
}
