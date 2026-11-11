package scene

import (
	"sync"
	"testing"
)

func TestFoldTranslations(t *testing.T) {
	m, between := Translation(10, 20), Translation(0.5, 1)
	s := &Scene{Width: 100, Height: 100, Glyphs: []Glyph{{X: 3.4, Y: 5.6}, {X: 1, Y: 1}}, Ops: []Op{
		{Kind: OpFill, Rect: Rect{X: 1, Y: 2, W: 3, H: 4}, Paint: PaintLinear, Gradient: [4]float32{0, 0, 3, 4}, Transform: m},
		{Kind: OpFill, Rect: Rect{W: 3, H: 4}, Paint: PaintStripes, Gradient: [4]float32{1, 0, 2, 4}, Transform: between},
		{Kind: OpShadow, Rect: Rect{W: 5, H: 5}, Cast: Rect{X: 1, W: 4, H: 4}, Transform: m},
		{Kind: OpGlyphs, Start: 0, End: 1, Paint: PaintConic, Gradient: [4]float32{1, 1, 2, 0}, Transform: m},
		{Kind: OpGlyphs, Start: 1, End: 2, Transform: between},
		{Kind: OpImage, Rect: Rect{W: 2, H: 2}, Transform: m},
		{Kind: OpImage, Rect: Rect{W: 2, H: 2}, Transform: between},
		{Kind: OpPushClip, Rect: Rect{W: 50, H: 50}, Transform: m},
		{Kind: OpFill, Rect: Rect{W: 3, H: 4}, Transform: m.Mul(Rotation(10))},
		{Kind: OpPopClip},
		{Kind: OpPushClip, Rect: Rect{W: 50, H: 50}, Transform: m.Mul(Rotation(10))},
		{Kind: OpPopClip},
		{Kind: OpFill, Rect: Rect{W: 4, H: 4}, Paint: PaintRadial, Gradient: [4]float32{2, 1, 3, 0}, Transform: m},
	}}
	var f TranslationFolder
	if f.Fold(s) != s {
		t.Fatal("folded before FoldTranslations")
	}
	FoldTranslations()
	got := f.Fold(s)
	want := []Op{
		{Kind: OpFill, Rect: Rect{X: 11, Y: 22, W: 3, H: 4}, Paint: PaintLinear, Gradient: [4]float32{10, 20, 13, 24}},
		{Kind: OpFill, Rect: Rect{X: 0.5, Y: 1, W: 3, H: 4}, Paint: PaintStripes, Gradient: [4]float32{1, 0, 2, 4}},
		{Kind: OpShadow, Rect: Rect{X: 10, Y: 20, W: 5, H: 5}, Cast: Rect{X: 11, Y: 20, W: 4, H: 4}},
		{Kind: OpGlyphs, Rect: Rect{X: 10, Y: 20}, Start: 0, End: 1, Paint: PaintConic, Gradient: [4]float32{11, 21, 2, 0}},
		s.Ops[4], // glyphs between pixels
		{Kind: OpImage, Rect: Rect{X: 10, Y: 20, W: 2, H: 2}},
		s.Ops[6], // an image between pixels
		{Kind: OpPushClip, Rect: Rect{X: 10, Y: 20, W: 50, H: 50}}, // a clip's edges are computed where they fall
		s.Ops[8], s.Ops[9], s.Ops[10], s.Ops[11],
		{Kind: OpFill, Rect: Rect{X: 10, Y: 20, W: 4, H: 4}, Paint: PaintRadial, Gradient: [4]float32{12, 21, 3, 0}}, // its radius kept
	}
	for i := range want {
		if got.Ops[i] != want[i] {
			t.Errorf("op %d: %+v, want %+v", i, got.Ops[i], want[i])
		}
	}
	if g := got.Glyphs; g[0].X != 13 || g[0].Y != 26 || g[1] != s.Glyphs[1] {
		t.Errorf("glyphs %+v", g)
	}
	if s.Ops[0].Transform != m || s.Glyphs[0].X != 3.4 {
		t.Error("folding changed the scene")
	}
	plain := &Scene{Ops: []Op{s.Ops[8], s.Ops[10], s.Ops[11]}}
	if f.Fold(plain) != plain {
		t.Error("a scene without translations was copied")
	}
}

// Package ui calls FoldTranslations wherever it makes a translation, on
// any goroutine, while renderers fold on theirs: with -race, this checks
// that they share what FoldTranslations sets safely.
func TestFoldTranslationsConcurrently(t *testing.T) {
	s := &Scene{Width: 10, Height: 10, Ops: []Op{{Kind: OpFill, Rect: Rect{W: 1, H: 1}, Transform: Translation(1, 0)}}}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 100 {
				FoldTranslations()
			}
		})
	}
	var f TranslationFolder
	for range 100 {
		f.Fold(s)
	}
	wg.Wait()
	if f.Fold(s) == s {
		t.Error("a translated fill not folded after FoldTranslations")
	}
}
