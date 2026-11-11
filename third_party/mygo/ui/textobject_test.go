package ui

import "testing"

// TestTextObjects lays out an inline object between words: it takes its
// width in the line, stands its height on the baseline, and lines break
// around it.
func TestTextObjects(t *testing.T) {
	font := Font{Size: 14}
	plain := ShapeRichText([]Span{{Text: "ab"}}, font, 0)
	pw, ph := plain.Size()
	l := ShapeRichText([]Span{{Text: "a"}, {Text: "￼", ObjectWidth: 40, ObjectHeight: 50}, {Text: "b"}}, font, 0)
	w, h := l.Size()
	if w < pw+39 || w > pw+41 {
		t.Errorf("width %v, want %v and the object's 40", w, pw)
	}
	if h < 50 || h <= ph {
		t.Errorf("height %v: the line does not hold the object", h)
	}
	objects := l.Objects()
	if len(objects) != 1 || objects[0].Index != 1 {
		t.Fatalf("objects %+v", objects)
	}
	r := objects[0].Rect
	if r.W < 39 || r.W > 41 || r.H != 50 || r.X <= 0 || r.Y < 0 {
		t.Errorf("object box %+v", r)
	}
	// The caret after it is past its box.
	if c := l.Caret(2); c.X < r.X+r.W-1 {
		t.Errorf("caret after the object at %v, box %+v", c.X, r)
	}
	// Narrow, the line breaks around it rather than overflowing.
	narrow := ShapeRichText([]Span{{Text: "words words "}, {Text: "￼", ObjectWidth: 60, ObjectHeight: 20}, {Text: " more"}}, font, 90)
	if _, nh := narrow.Size(); nh < 2*20 {
		t.Errorf("narrow height %v: no line broke at the object", nh)
	}
}

// TestTextObjectDescent lays out inline objects reaching below the
// baseline: the line makes room for their descent, their boxes hold it,
// and without one the layout is as before.
func TestTextObjectDescent(t *testing.T) {
	font := Font{Size: 14}
	obj := func(d float32) []Span {
		return []Span{{Text: "a"}, {Text: "￼", ObjectWidth: 30, ObjectHeight: 8, ObjectDescent: d}, {Text: "b"}}
	}
	zero := ShapeRichText(obj(0), font, 0)
	before := ShapeRichText([]Span{{Text: "a"}, {Text: "￼", ObjectWidth: 30, ObjectHeight: 8}, {Text: "b"}}, font, 0)
	zw, zh := zero.Size()
	if bw, bh := before.Size(); zw != bw || zh != bh {
		t.Errorf("zero descent %vx%v, without one %vx%v", zw, zh, bw, bh)
	}
	deep := ShapeRichText(obj(40), font, 0)
	w, h := deep.Size()
	if w != zw || h < zh+30 {
		t.Errorf("with a descent of 40: %vx%v, without %vx%v", w, h, zw, zh)
	}
	zr, r := zero.Objects()[0].Rect, deep.Objects()[0].Rect
	if zr.H != 8 || r.H != 48 || r.W != zr.W {
		t.Errorf("boxes %+v and %+v", zr, r)
	}
	// The object's box stands on the baseline its text stands on, and
	// fits in the line.
	if caret := deep.Caret(0); r.Y+8 < caret.Y || r.Y+8 > caret.Y+caret.H {
		t.Errorf("box %+v, caret %+v: not on the text's baseline", r, caret)
	}
	if r.Y < 0 || r.Y+r.H > h+0.01 {
		t.Errorf("box %+v out of the text, %v high", r, h)
	}
	// Wrapped, the second line holds the deep object below the first.
	narrow := ShapeRichText([]Span{{Text: "words words words "}, {Text: "￼", ObjectWidth: 30, ObjectHeight: 8, ObjectDescent: 40}, {Text: " more"}}, font, 100)
	nr := narrow.Objects()[0].Rect
	if first := narrow.Caret(0); nr.Y < first.Y+first.H {
		t.Errorf("box %+v on the first line, caret %+v", nr, first)
	}
	if _, nh := narrow.Size(); nr.Y+nr.H > nh+0.01 || nh < 48 {
		t.Errorf("box %+v, text %v high", nr, nh)
	}
}

// TestDrawObjects draws a text's inline objects in their boxes, after
// its glyphs, those of the text elements inside it included, and not
// those its clip hides.
func TestDrawObjects(t *testing.T) {
	type drawn struct {
		at   int
		r    Rect
		clip Rect
	}
	var got []drawn
	var box Rect
	obj := Span{Text: "￼", ObjectWidth: 20, ObjectHeight: 10, ObjectDescent: 6}
	tt := NewTester(func(c *Context) {
		Column(c).Size(300, 60).Clip().Children(func() {
			RichText(c, Span{Text: "ab"}, obj, Span{Text: "\n\n\n\n\n\n\n\n\n\nfar "}, obj).
				Draw(func(p *Painter, r Rect) { box, got = r, got[:0] }). // painted again, drawn again
				DrawObjects(func(p *Painter, at int, r Rect) { got = append(got, drawn{at, r, p.Visible()}) }).
				Children(func() { RichText(c, Span{Text: " x"}, obj) })
		})
	}, 400, 400)
	defer tt.rt.close()
	tt.Frame()
	// "ab" and the object, ten lines and "far" with an object (cut off by
	// the clip), then " x" and its object.
	if len(got) != 1 {
		t.Fatalf("drew %+v, want the first object alone", got)
	}
	d := got[0]
	if d.at != 2 || d.r.W != 20 || d.r.H != 16 {
		t.Errorf("drew %+v", d)
	}
	if d.r.X <= box.X || d.r.Y < box.Y || d.r.Y+d.r.H > box.Y+box.H {
		t.Errorf("object %+v outside the text's box %+v", d.r, box)
	}
	if d.clip.H > 60.01 {
		t.Errorf("drawn without the clip: %+v", d.clip)
	}

	// Unclipped, every object draws, the inline elements' at their runes
	// of the whole paragraph, moved as their element is.
	got = got[:0]
	tt2 := NewTester(func(c *Context) {
		RichText(c, Span{Text: "ab"}, obj).
			Draw(func(*Painter, Rect) { got = got[:0] }).
			DrawObjects(func(p *Painter, at int, r Rect) {
				got = append(got, drawn{at: at, r: r, clip: transformRect(p.transform, r)})
			}).
			Children(func() {
				RichText(c, Span{Text: " x"}, obj)
				RichText(c, Span{Text: " y"}, obj).Translate(0, 30)
			})
	}, 400, 400)
	defer tt2.rt.close()
	tt2.Frame()
	if len(got) != 3 || got[0].at != 2 || got[1].at != 5 || got[2].at != 8 || got[1].r.X <= got[0].r.X {
		t.Fatalf("drew %+v", got)
	}
	if got[1].clip != got[1].r || got[2].clip.Y-got[2].r.Y != 30 {
		t.Errorf("drew in %+v and %+v: not where their elements are", got[1].clip, got[2].clip)
	}
}
