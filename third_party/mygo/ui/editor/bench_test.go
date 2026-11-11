package editor

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// bigGo is a Go file of about n lines.
func bigGo(n int) string {
	var b strings.Builder
	b.WriteString("package big\n\nimport \"fmt\"\n\n")
	for i := 0; b.Len() < n*40; i++ {
		b.WriteString("// handle" + strconv.Itoa(i) + " handles the request, with \"quotes\" and 0x1f.\n")
		b.WriteString("func handle" + strconv.Itoa(i) + "(w io.Writer, r *Request) error {\n")
		b.WriteString("\tif err := r.Check(\"x\\n\", 42); err != nil {\n\t\treturn fmt.Errorf(`bad: %w`, err)\n\t}\n")
		b.WriteString("\tfor i, v := range r.Values { // each\n\t\tfmt.Fprintln(w, i, v)\n\t}\n\treturn nil\n}\n\n")
	}
	return b.String()
}

func newBenchEditor(b *testing.B, text string, wrap bool) (*State, *ui.Tester) {
	st := &State{Buffer: NewBuffer(text), SoftWrap: wrap}
	st.Buffer.SetLanguage(LanguageNamed("go"))
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, st).Fill().AutoFocus()
	}, 1200, 800)
	tt.SetScale(2)
	// Highlight the whole text first, as an idle editor has.
	st.ed.view.advanceSyntax(st.ed.view.text.LineCount(), time.Hour)
	tt.Frame()
	tt.Move(600, 400)
	return st, tt
}

// BenchmarkCodeEditorScroll renders the frames of scrolling a Go file of
// 100,000 lines a few lines at a time: building, laying out the rows that
// come into view, painting and drawing on the CPU.
func BenchmarkCodeEditorScroll(b *testing.B) {
	_, tt := newBenchEditor(b, bigGo(100_000), false)
	b.ReportAllocs()
	for b.Loop() {
		tt.Scroll(600, 400, 0, 61)
	}
}

// BenchmarkCodeEditorScrollWrap scrolls with soft wraps.
func BenchmarkCodeEditorScrollWrap(b *testing.B) {
	_, tt := newBenchEditor(b, bigGo(100_000), true)
	b.ReportAllocs()
	for b.Loop() {
		tt.Scroll(600, 400, 0, 61)
	}
}

// BenchmarkCodeEditorType types a character in the middle of a Go file
// of 100,000 lines and renders the frame.
func BenchmarkCodeEditorType(b *testing.B) {
	st, tt := newBenchEditor(b, bigGo(100_000), false)
	mid := st.Buffer.Offset(Pos{Line: 50_000})
	st.Select(mid, mid)
	tt.Frame()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		if i%2 == 0 {
			tt.Type("x")
		} else {
			tt.Key(0, ui.KeyBackspace)
		}
	}
}

// BenchmarkCodeEditorOpen makes an editor of a Go file of 100,000 lines
// and renders its first frame.
func BenchmarkCodeEditorOpen(b *testing.B) {
	text := bigGo(100_000)
	b.ReportAllocs()
	for b.Loop() {
		st := &State{Buffer: NewBuffer(text)}
		st.Buffer.SetLanguage(LanguageNamed("go"))
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, st).Fill()
		}, 1200, 800)
		tt.SetScale(2)
		tt.Frame()
	}
}

// BenchmarkDiffViewerScroll scrolls a split diff of two versions of a Go
// file of 20,000 lines, a change every 50 lines, all lines shown.
func BenchmarkDiffViewerScroll(b *testing.B) {
	old := bigGo(20_000)
	lines := strings.Split(old, "\n")
	for i := 10; i < len(lines); i += 50 {
		lines[i] = strings.Replace(lines[i], "err", "e", 1) + " // changed"
	}
	d := &DiffState{Old: NewBuffer(old), New: NewBuffer(strings.Join(lines, "\n")), ExpandUnchanged: true}
	d.Old.SetLanguage(LanguageNamed("go"))
	d.New.SetLanguage(LanguageNamed("go"))
	tt := ui.NewTester(func(c *ui.Context) {
		DiffView(c, d).Fill()
	}, 1400, 800)
	tt.SetScale(2)
	tt.Frame()
	tt.Move(700, 400)
	b.ReportAllocs()
	for b.Loop() {
		tt.Scroll(700, 400, 0, 61)
	}
}

// BenchmarkDiffViewerType types in a diff of 20,000 lines, which compares
// again around the edit.
func BenchmarkDiffViewerType(b *testing.B) {
	old := bigGo(20_000)
	lines := strings.Split(old, "\n")
	for i := 10; i < len(lines); i += 50 {
		lines[i] += " // changed"
	}
	d := &DiffState{Old: NewBuffer(old), New: NewBuffer(strings.Join(lines, "\n")), Editable: true}
	tt := ui.NewTester(func(c *ui.Context) {
		DiffView(c, d).Fill()
	}, 1400, 800)
	tt.SetScale(2)
	tt.Frame()
	ed := &d.eds[1]
	off := d.New.Offset(Pos{Line: 10_010})
	ed.sels = []sel{{anchor: off, head: off, goal: -1}}
	ed.reveal = revealCenter
	tt.ClickAt(1000, 400)
	ed.sels = []sel{{anchor: off, head: off, goal: -1}}
	tt.Frame()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		if i%2 == 0 {
			tt.Type("x")
		} else {
			tt.Key(0, ui.KeyBackspace)
		}
	}
}

// newFitBench shows a diff of two versions of a Go file of about n lines,
// a change every 50 lines, as high as its content in a scrolling list
// under a header, as an app listing a change's files shows it.
func newFitBench(n int, layout DiffLayout, wrap bool) *ui.Tester {
	old := bigGo(n)
	lines := strings.Split(old, "\n")
	for i := 10; i < len(lines); i += 50 {
		lines[i] = strings.Replace(lines[i], "err", "e", 1) + " // changed, and a few more words to wrap on narrow windows"
	}
	d := &DiffState{Old: NewBuffer(old), New: NewBuffer(strings.Join(lines, "\n")), Layout: layout, Wrap: wrap, FitContent: true}
	d.Old.SetLanguage(LanguageNamed("go"))
	d.New.SetLanguage(LanguageNamed("go"))
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			ui.Box(c).Height(200)
			DiffView(c, d).FillWidth()
		})
	}, 1400, 800)
	tt.SetScale(2)
	for range 4 {
		tt.Frame()
	}
	tt.Move(700, 400)
	return tt
}

func benchFitScroll(b *testing.B, layout DiffLayout) {
	tt := newFitBench(20_000, layout, false)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		dy := float32(61)
		if i/200%2 == 1 {
			dy = -61
		}
		tt.Scroll(700, 400, 0, dy)
	}
}

// BenchmarkDiffFitScroll scrolls a split diff of 20,000 lines as high as
// its content, down and back up, in the list it is in.
func BenchmarkDiffFitScroll(b *testing.B) { benchFitScroll(b, DiffSplit) }

// BenchmarkDiffFitScrollUnified scrolls a unified one.
func BenchmarkDiffFitScrollUnified(b *testing.B) { benchFitScroll(b, DiffUnified) }

func benchFitResize(b *testing.B, layout DiffLayout, wrap bool) {
	tt := newFitBench(20_000, layout, wrap)
	widths := []int{1400, 1380, 1360, 1340, 1320, 1300, 1320, 1340, 1360, 1380}
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		tt.SetSize(widths[i%len(widths)], 800)
		tt.Frame()
	}
}

// BenchmarkDiffFitResize renders the frames of a window resized through
// widths over a split diff of 20,000 lines as high as its content.
func BenchmarkDiffFitResize(b *testing.B) { benchFitResize(b, DiffSplit, false) }

// BenchmarkDiffFitResizeWrap resizes with soft wraps, which each width
// measures again.
func BenchmarkDiffFitResizeWrap(b *testing.B) { benchFitResize(b, DiffSplit, true) }

// BenchmarkDiffFitResizeUnifiedWrap resizes a unified one with soft wraps.
func BenchmarkDiffFitResizeUnifiedWrap(b *testing.B) { benchFitResize(b, DiffUnified, true) }

// BenchmarkDiffFitFrame renders a frame of a split diff of 20,000 lines
// as high as its content in which nothing changed, as the pointer moving
// over it asks for.
func BenchmarkDiffFitFrame(b *testing.B) {
	tt := newFitBench(20_000, DiffSplit, false)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		tt.Move(700, float32(400+i%2))
	}
}
