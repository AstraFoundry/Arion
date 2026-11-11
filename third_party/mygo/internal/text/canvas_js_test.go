package text

import (
	"slices"
	"syscall/js"
	"testing"
	"unicode/utf8"
)

// fakeCanvas has the canvas engine measure every character 10 DIPs wide,
// where no browser gives it a canvas, as Node.js.
func fakeCanvas(t *testing.T) {
	g := js.Global()
	if !g.Get("document").IsUndefined() || !g.Get("OffscreenCanvas").IsUndefined() {
		return
	}
	measure := js.FuncOf(func(this js.Value, args []js.Value) any {
		return map[string]any{
			"width":                   10 * utf8.RuneCountInString(args[0].String()),
			"fontBoundingBoxAscent":   8,
			"fontBoundingBoxDescent":  2,
			"actualBoundingBoxAscent": 5,
		}
	})
	getContext := js.FuncOf(func(this js.Value, args []js.Value) any {
		ctx := g.Get("Object").New()
		ctx.Set("measureText", measure)
		return ctx
	})
	canvas := js.FuncOf(func(this js.Value, args []js.Value) any {
		c := g.Get("Object").New()
		c.Set("getContext", getContext)
		return c
	})
	g.Set("OffscreenCanvas", canvas)
	t.Cleanup(func() {
		g.Delete("OffscreenCanvas")
		canvas.Release()
		getContext.Release()
		measure.Release()
	})
}

// The canvas engine breaks Korean between words (#1403).
func TestCanvasKeepAll(t *testing.T) {
	fakeCanvas(t)
	e := newCanvas()
	ends := func(text, lang string, width float32, wholeWords bool) []int {
		var out []int
		for _, l := range e.shape([]rune(text), Style{Size: 16, Lang: lang}, nil, width, false, wholeWords) {
			out = append(out, l.end)
		}
		return out
	}
	const text = "안녕하세요 반갑습니다 오늘은 날씨가 좋네요"
	for _, c := range []struct {
		text, lang string
		wholeWords bool
		want       []int
	}{
		{text, "ko", false, []int{6, 12, 16, 20, 23}},
		{text, "ko-KR", false, []int{6, 12, 16, 20, 23}},
		{text, "en", false, []int{6, 12, 17, 22, 23}},
		// A word longer than a line breaks within itself.
		{"가나 다라마바사아자 차", "ko", false, []int{3, 8, 12}},
		{"가나 다라마바사아자 차", "ko", true, []int{3, 11, 12}},
	} {
		if got := ends(c.text, c.lang, 50, c.wholeWords); !slices.Equal(got, c.want) {
			t.Errorf("%q in %q, whole words %v: lines end at %v, want %v", c.text, c.lang, c.wholeWords, got, c.want)
		}
	}
}
