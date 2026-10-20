package text

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"unsafe"
	"weak"
)

// Scrolling must not retain hundreds of frames of glyph arrays that
// cannot expire once the window stops drawing.
func TestLayoutCacheBoundsScrollingMemory(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	const rows = 40
	for frame := range 300 {
		for row := range rows {
			s.Layout(Params{Text: fmt.Sprintf("line %d: %s", frame*rows+row, strings.Repeat("code ", 20))})
		}
		s.EndFrame()
		if s.layoutBytes > maxLayoutBytes || len(s.layouts) > maxLayouts {
			t.Fatalf("frame %d retained %d bytes in %d layouts", frame, s.layoutBytes, len(s.layouts))
		}
	}
}

func TestLayoutCacheReusesDisplayedText(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	p := Params{Text: "unchanged visible text"}
	l := s.Layout(p)
	for range 10 {
		s.EndFrame()
		if got := s.Layout(p); got != l {
			t.Fatal("a new frame reshaped unchanged visible text")
		}
		if got := s.Layout(p); got != l {
			t.Fatal("repeated measurement in one frame reshaped the text")
		}
	}
	if n := s.LayoutsMade(); n != 1 {
		t.Fatalf("made %d layouts for unchanged text", n)
	}
}

func TestLayoutCacheReleasesEvictedText(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	old := weak.Make(s.Layout(Params{Text: "the first viewport"}))
	for i := range maxLayouts + 1 {
		s.Layout(Params{Text: fmt.Sprint(i)})
	}
	runtime.GC()
	if old.Value() != nil {
		t.Fatal("evicted text is retained while the window is idle")
	}
	runtime.KeepAlive(s)
}

func TestLayoutCacheKeepsFrequentlyUsedText(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	p := Params{Text: "a frequently used label"}
	l := s.Layout(p)
	for i := range maxLayouts + 1 {
		s.Layout(Params{Text: fmt.Sprint(i)})
		if got := s.Layout(p); got != l {
			t.Fatal("cache pressure evicted a frequently used label")
		}
	}
}

func TestLayoutCacheSkipsOversizedParagraph(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	p := Params{Text: "a small label"}
	l := s.Layout(p)
	large := s.Layout(Params{Text: strings.Repeat("x", maxLayoutBytes/32)})
	if len(large.Runes) != maxLayoutBytes/32 || len(large.Lines) == 0 {
		t.Fatal("oversized text did not lay out")
	}
	if len(s.layouts) != 1 || s.Layout(p) != l {
		t.Fatal("an oversized paragraph displaced the ordinary labels")
	}
}

func TestRetainedLayoutDoesNotRetainClearedCache(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	kept := s.Layout(Params{Text: "kept by an element"})
	other := weak.Make(s.Layout(Params{Text: "no longer used"}))
	s.clearLayouts()
	runtime.GC()
	if other.Value() != nil || s.layoutBytes != 0 {
		t.Fatal("a retained layout keeps the cleared cache alive")
	}
	runtime.KeepAlive(kept)
	runtime.KeepAlive(s)
}

func cacheDocumentExcerpt(s *System) weak.Pointer[byte] {
	document := strings.Repeat("x", 1<<20)
	old := weak.Make(unsafe.StringData(document))
	s.Layout(Params{Text: document[100:110]})
	return old
}

func TestLayoutCacheDoesNotRetainExcerptSource(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	old := cacheDocumentExcerpt(s)
	runtime.GC()
	if old.Value() != nil {
		t.Fatal("a cached ten-character excerpt retains its entire source document")
	}
	if s.Layout(Params{Text: strings.Repeat("x", 10)}).Params.Text != strings.Repeat("x", 10) {
		t.Fatal("the cache changed the excerpt's text")
	}
	runtime.KeepAlive(s)
}

// documentRow is a paragraph of a long document, and document the
// paragraphs whose layouts take a little more than the cache's budget,
// however large the layouts of this engine are.
func documentRow(i int) Params {
	return Params{Text: fmt.Sprintf("paragraph %04d: %s", i, strings.Repeat("word ", 60))}
}

func document(s *System) []Params {
	s.Layout(documentRow(0))
	ps := make([]Params, maxLayoutBytes*9/8/s.layoutBytes+1)
	for i := range ps {
		ps[i] = documentRow(i)
	}
	return ps
}

// A window drawing a document a little over the budget must not evict
// each paragraph just before its next frame lays it out again.
func TestLayoutCacheKeepsDisplayedDocument(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	ps := document(s)
	for _, p := range ps {
		s.Layout(p)
	}
	if s.layoutBytes <= maxLayoutBytes {
		t.Fatalf("the document takes %d bytes, within the budget", s.layoutBytes)
	}
	s.EndFrame()
	made := s.LayoutsMade()
	for range 20 {
		for _, p := range ps {
			s.Layout(p)
		}
		s.EndFrame()
	}
	if n := s.LayoutsMade() - made; n != 0 {
		t.Fatalf("made %d layouts for the unchanged document", n)
	}
}

// Once no frame draws the document, the cache goes back to its budget,
// whichever window's frames go by, and later forgets the rest.
func TestLayoutCacheTrimsUnusedDocument(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	for _, p := range document(s) {
		s.Layout(p)
	}
	for range liveLayoutFrames - 1 {
		s.EndFrame()
		if s.layoutBytes <= maxLayoutBytes {
			t.Fatal("the document was trimmed while still recent")
		}
	}
	s.Layout(Params{Text: "another window"})
	s.EndFrame()
	if s.layoutBytes > maxLayoutBytes {
		t.Fatalf("kept %d bytes of unused layouts", s.layoutBytes)
	}
	for range 241 {
		s.EndFrame()
	}
	if len(s.layouts) != 0 || s.layoutBytes != 0 {
		t.Fatal("old layouts survived expiry")
	}
}

// Windows take turns: another window's frames between two of the
// document's keep its layouts.
func TestLayoutCacheKeepsDocumentAcrossWindows(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	ps := document(s)
	first := s.Layout(ps[0])
	for _, p := range ps[1:] {
		s.Layout(p)
	}
	s.EndFrame()
	for range liveLayoutFrames - 2 {
		s.Layout(Params{Text: "another window"})
		s.EndFrame()
	}
	if s.Layout(ps[0]) != first {
		t.Fatal("another window's frames evicted the document")
	}
}

// Recent layouts still stay within a hard limit.
func TestLayoutCacheCapsRecentBytes(t *testing.T) {
	s := newSystem()
	s.eng = &stubEngine{}
	first := s.Layout(Params{Text: "first"})
	for i := range 100 {
		s.Layout(Params{Text: fmt.Sprintf("%d: %s", i, strings.Repeat("word ", 2000))})
		if s.layoutBytes > maxLiveLayoutBytes {
			t.Fatalf("kept %d bytes", s.layoutBytes)
		}
	}
	if s.Layout(Params{Text: "first"}) == first {
		t.Fatal("the hard limit did not evict the oldest layout")
	}
}
