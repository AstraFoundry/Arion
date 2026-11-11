package markdown

import (
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
)

func llm(b *testing.B) string {
	data, err := os.ReadFile("parser/testdata/llm.md")
	if err != nil {
		b.Fatal(err)
	}
	return string(data)
}

// BenchmarkViewSteady draws frames of the 50 KB answer shown whole, in a
// window of 800 by 1000, as nothing changes.
func BenchmarkViewSteady(b *testing.B) {
	src := llm(b)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() { View(c, src, nil) })
	}, 800, 1000)
	b.ReportAllocs()
	for b.Loop() {
		tt.Frame()
	}
}

// BenchmarkViewStream draws the frames of the 50 KB answer streaming in,
// 16 bytes a frame, as a chat follows a reply's end; it reports the time
// of a frame with its token.
func BenchmarkViewStream(b *testing.B) {
	src := llm(b)
	o := &Options{Reveal: &Reveal{Unit: AtOnce}}
	n := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() { Stream(c, src[:n], o) })
	}, 800, 1000)
	b.ReportAllocs()
	for b.Loop() {
		n += 16
		if n > len(src) {
			n = 16
		}
		tt.Frame()
	}
}
