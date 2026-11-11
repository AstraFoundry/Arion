package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"strings"
	"testing"
)

// Test-only application state; the framework owns no document buffer.
type primitiveClient struct {
	text             string
	selection        TextInputSelection
	mark             *TextInputRange
	queries          []TextInputRange
	changes, unmarks int
}

func (c *primitiveClient) TextForRange(r TextInputRange) (string, TextInputRange) {
	c.queries = append(c.queries, r)
	a, z := UTF16ByteOffset(c.text, r.Start), UTF16ByteOffset(c.text, r.End)
	return c.text[a:z], TextInputRange{Start: UTF16Len(c.text[:a]), End: UTF16Len(c.text[:z])}
}
func (c *primitiveClient) Selection() TextInputSelection { return c.selection }
func (c *primitiveClient) MarkedRange() (TextInputRange, bool) {
	if c.mark != nil {
		return *c.mark, true
	}
	return TextInputRange{}, false
}
func (c *primitiveClient) rangeFor(r *TextInputRange) TextInputRange {
	if r != nil {
		return *r
	}
	if c.mark != nil {
		return *c.mark
	}
	return c.selection.Range
}
func (c *primitiveClient) replace(r TextInputRange, text string) {
	a, z := UTF16ByteOffset(c.text, r.Start), UTF16ByteOffset(c.text, r.End)
	c.text = c.text[:a] + text + c.text[z:]
	end := r.Start + UTF16Len(text)
	c.selection = TextInputSelection{Range: TextInputRange{Start: end, End: end}}
	c.changes++
}
func (c *primitiveClient) ReplaceText(r *TextInputRange, text string) {
	c.replace(c.rangeFor(r), text)
	c.mark = nil
}
func (c *primitiveClient) SetMarkedText(r *TextInputRange, text string, selected TextInputRange) {
	wanted := c.rangeFor(r)
	c.replace(wanted, text)
	mark := TextInputRange{Start: wanted.Start, End: wanted.Start + UTF16Len(text)}
	c.mark = &mark
	c.selection = TextInputSelection{Range: TextInputRange{Start: mark.Start + selected.Start, End: mark.Start + selected.End}}
}
func (c *primitiveClient) UnmarkText() { c.mark = nil; c.unmarks++ }
func (c *primitiveClient) BoundsForRange(r TextInputRange) (Rect, TextInputRange, bool) {
	return Rect{float32(r.Start) * 4, 5, max(1, float32(r.End-r.Start)*4), 20}, r, true
}
func (c *primitiveClient) IndexForPoint(p Point) (int, bool) {
	return max(0, min(int(p.X/4), UTF16Len(c.text))), true
}
func primitiveTester(c *primitiveClient) *Tester {
	return coreNewTester(func(ctx *context) {
		coreBox(ctx).Size(250, 80).HandleTextInput(c).AutoFocus().HandleInput(func(ev InputEvent) bool { return ev.Kind == InputCommand })
	}, 300, 100)
}

func TestTextInputClientCompositionAndReplacement(t *testing.T) {
	c := &primitiveClient{text: "A😀cafe", selection: TextInputSelection{Range: TextInputRange{Start: 7, End: 7}}}
	tt := primitiveTester(c)
	defer tt.rt.close()
	if tt.h.ime.Client == nil || tt.h.ime.Text != "" {
		t.Fatal("client copied a document snapshot")
	}
	tt.Type("!")
	if c.text != "A😀cafe!" {
		t.Fatal(c.text)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.TextInput, Text: "é", Replace: true, From: 6, To: 7})
	if c.text != "A😀café!" {
		t.Fatal("absolute UTF-16 replacement", c.text)
	}
	c.selection = TextInputSelection{Range: TextInputRange{Start: 6, End: 7}}
	tt.Frame()
	tt.Compose("é", 2)
	if c.text != "A😀café!" || c.mark == nil || *c.mark != (TextInputRange{Start: 6, End: 8}) {
		t.Fatal("preedit", c.text, c.mark)
	}
	tt.Compose("😀", 1)
	if c.selection.Range != (TextInputRange{Start: 8, End: 8}) {
		t.Fatal("caret confused runes with UTF-16", c.selection)
	}
	tt.Type("é")
	if c.text != "A😀café!" || c.mark != nil {
		t.Fatal("commit", c.text)
	}
}

// TestCompositionTaken: the window tells the system whether the focus took
// a composition, which input methods give up otherwise on macOS. An element
// taking text itself (TextCaret) that leaves one, as a terminal leaves the
// dead key of a key it sent as Alt, does not take it; a text input client
// takes it, whatever its HandleInput answers.
func TestCompositionTaken(t *testing.T) {
	take := true
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(100, 40).Focusable().AutoFocus().TextCaret(Rect{0, 0, 1, 16}).HandleInput(func(ev InputEvent) bool {
			return ev.Kind != InputCompose || take
		})
	}, 200, 100)
	defer tt.rt.close()
	compose := platform.SurfaceEvent{Kind: platform.TextComposition, Text: "´", Caret: 1}
	if !tt.send2(compose) {
		t.Error("a composition the element took was not taken")
	}
	take = false
	if tt.send2(compose) {
		t.Error("a composition the element left was taken")
	}
	client := primitiveTester(&primitiveClient{})
	defer client.rt.close()
	if !client.send2(compose) {
		t.Error("a text input client's composition was not taken")
	}
}

func TestTextInputClientNativeEndBeforeCommit(t *testing.T) {
	c := &primitiveClient{text: "abc", selection: TextInputSelection{Range: TextInputRange{Start: 3, End: 3}}}
	tt := primitiveTester(c)
	defer tt.rt.close()
	client := tt.h.ime.Client
	client.SetMarkedText(nil, "😀", TextInputRange{Start: 2, End: 2})
	var composition platform.InputComposition
	composition.End(client)
	if c.mark != nil || c.text != "abc😀" {
		t.Fatal("unmark removed document content")
	}
	composition.Replace(client, nil, "é")
	if c.text != "abcé" {
		t.Fatal("end-before-commit duplicated preedit", c.text)
	}
	client.SetMarkedText(nil, "候補", TextInputRange{Start: 2, End: 2})
	composition.End(client)
	composition.Reset() // the native client was switched or reset
	composition.Replace(client, nil, "!")
	if c.text != "abcé候補!" {
		t.Fatal("stale commit range survived reset", c.text)
	}
}

// An input method that gives its composition up with no empty preedit
// first, as IMM32's Microsoft Japanese IME does on Esc, leaves the text
// as it was before the composition.
func TestTextInputClientNativeCancel(t *testing.T) {
	c := &primitiveClient{text: "abc", selection: TextInputSelection{Range: TextInputRange{Start: 1, End: 3}}}
	tt := primitiveTester(c)
	defer tt.rt.close()
	client := tt.h.ime.Client
	var composition platform.InputComposition
	composition.Cancel(client) // nothing marked: the selection stays
	if c.text != "abc" || c.selection.Range != (TextInputRange{Start: 1, End: 3}) {
		t.Fatal("cancel without a composition changed the document", c.text, c.selection)
	}
	client.SetMarkedText(nil, "にほｎ", TextInputRange{Start: 3, End: 3})
	composition.Cancel(client)
	composition.End(client)
	if c.mark != nil || c.text != "a" {
		t.Fatal("cancelled composition was kept", c.text, c.mark)
	}
	client.SetMarkedText(nil, "にほん", TextInputRange{Start: 3, End: 3})
	composition.Replace(client, nil, "日本")
	composition.End(client)
	if c.mark != nil || c.text != "a日本" {
		t.Fatal("commit after a cancel", c.text, c.mark)
	}
}

func TestTextInputClientQueriesGeometryAndLifetime(t *testing.T) {
	c := &primitiveClient{text: strings.Repeat("a", 10000), selection: TextInputSelection{Range: TextInputRange{Start: 8000, End: 8000}}}
	show := true
	tt := coreNewTester(func(ctx *context) {
		coreColumn(ctx).Padding(10).Children(func() {
			if show {
				coreBox(ctx).Size(200, 40).HandleTextInput(c).AutoFocus()
			}
			coreButton(ctx, "Other")
		})
	}, 300, 100)
	defer tt.rt.close()
	client := tt.h.ime.Client
	text, actual := client.TextForRange(TextInputRange{Start: 6000, End: 6004})
	if text != "aaaa" || actual != (TextInputRange{Start: 6000, End: 6004}) {
		t.Fatal("query clipped to IME context")
	}
	if b, used, ok := client.BoundsForRange(TextInputRange{Start: 3, End: 5}); !ok || b.X != 22 || b.Y != 15 || used != (TextInputRange{Start: 3, End: 5}) {
		t.Fatal("element-to-surface geometry", b, used, ok)
	}
	if index, ok := client.IndexForPoint(26, 15); !ok || index != 4 {
		t.Fatal("surface-to-element hit test", index, ok)
	}
	c.mark = &TextInputRange{Start: 3, End: 4}
	tt.Click("Other")
	if c.mark != nil || c.unmarks != 1 {
		t.Fatal("focus loss composition cleanup", c.unmarks)
	}
	before := c.changes
	client.ReplaceText(nil, "stale")
	if c.changes != before {
		t.Fatal("unfocused client mutated")
	}
	show = false
	tt.Frame()
	client.ReplaceText(nil, "removed")
	if c.changes != before {
		t.Fatal("disposed client mutated")
	}
}

func TestTextInputClientSwappingAndTypedNil(t *testing.T) {
	first, second := &primitiveClient{text: "one"}, &primitiveClient{text: "two"}
	current := first
	tt := coreNewTester(func(ctx *context) { coreBox(ctx).Size(200, 50).HandleTextInput(current).AutoFocus() }, 250, 60)
	defer tt.rt.close()
	old := tt.h.ime.Client
	current = second
	tt.Frame()
	old.ReplaceText(nil, "wrong")
	if first.changes != 0 || second.changes != 0 {
		t.Fatal("old callback targeted replacement")
	}
	tt.Type("new")
	if second.text != "newtwo" {
		t.Fatal(second.text)
	}
	current = nil
	tt.Frame()
	if tt.h.ime.Active {
		t.Fatal("typed nil did not disconnect")
	}
}

func TestTextInputSurroundingUnicodeAndReversedSelection(t *testing.T) {
	c := &primitiveClient{text: "A😀bc", selection: TextInputSelection{Range: TextInputRange{Start: 3, End: 5}, Reversed: true}}
	tt := primitiveTester(c)
	defer tt.rt.close()
	ctx := platform.ClientTextContext(tt.h.ime.Client)
	if ctx.Caret != 3 || UTF16ByteOffset(ctx.Text, ctx.Caret) != 5 {
		t.Fatal("reversed caret", ctx)
	}
	if got := platform.ClientDeleteRange(tt.h.ime.Client, -1, 1); got != (TextInputRange{Start: 1, End: 3}) {
		t.Fatal("GTK delete split surrogate", got)
	}
	c.text = strings.Repeat("a", 6000) + "😀"
	c.selection = TextInputSelection{Range: TextInputRange{Start: 6002, End: 6002}}
	if got := platform.ClientDeleteRange(tt.h.ime.Client, -5000, 5000); got != (TextInputRange{Start: 1001, End: 6002}) {
		t.Fatal("large delete clipped", got)
	}
}

func TestTextLayoutUTF16GraphemesAndBidiRanges(t *testing.T) {
	l := ShapeText("é😀", Font{Size: 18}, 0)
	if l.Caret(1) != l.Caret(0) || l.Caret(3) != l.Caret(2) {
		t.Fatal("caret split grapheme or surrogate")
	}
	for _, index := range []int{0, 2, 4} {
		r := l.Caret(index)
		if got := l.IndexAt(Point{r.X, r.Y + r.H/2}); got != index {
			t.Fatal("caret hit test", index, got)
		}
	}
	bidi := ShapeText("abc אבג def", Font{Size: 18}, 0)
	rects := bidi.SelectionRects(TextInputRange{Start: 4, End: 6}, TextInputRange{Start: 7, End: 8})
	if len(rects) < 2 {
		t.Fatal("bidi range set collapsed", rects)
	}
	for _, r := range rects {
		if r.W <= 0 || r.H <= 0 {
			t.Fatal("invalid selection rectangle", r)
		}
	}
}

func TestTextLayoutVisualCaretUTF16(t *testing.T) {
	l := ShapeText("😀 abc אבג def", Font{Size: 18}, 0)
	up := l.CaretAt(TextCaretPosition{Index: 7, Affinity: TextUpstream})
	down := l.CaretAt(TextCaretPosition{Index: 7, Affinity: TextDownstream})
	if up.X == down.X {
		t.Fatal("lost secondary bidi caret")
	}
	p := TextCaretPosition{Index: 9}
	for _, index := range []int{8, 7, 11} {
		p = l.MoveCaret(p, 1)
		if p.Index != index {
			t.Fatalf("visual UTF-16 navigation %+v, want %d", p, index)
		}
	}
	ranges := l.SelectionRanges(TextCaretPosition{Index: 9}, p)
	if len(ranges) != 2 || ranges[0] != (TextInputRange{Start: 7, End: 9}) || ranges[1] != (TextInputRange{Start: 10, End: 11}) {
		t.Fatal("visual UTF-16 ranges", ranges)
	}
	position := l.PositionAt(Point{X: up.X, Y: up.Y + up.H/2})
	if got := l.CaretAt(position); got.X != up.X || got.Y != up.Y {
		t.Fatalf("hit-test lost visual edge: %+v %+v", up, got)
	}
}

// tallClient is the text of a code editor as high as its lines: its caret
// is far down its box.
type tallClient struct {
	primitiveClient
	caretY float32
}

func (c *tallClient) BoundsForRange(r TextInputRange) (Rect, TextInputRange, bool) {
	return Rect{X: 40, Y: c.caretY, W: 1, H: 20}, r, true
}

// Focusing an app's text field taller than the scroll view around it, by
// Tab or by assistive technology and the input methods that act as it,
// scrolls its caret into view, not the top of its box.
func TestFocusRevealsClientCaret(t *testing.T) {
	var s ScrollState
	c := &tallClient{caretY: 1500}
	tt := coreNewTester(func(ctx *context) {
		coreScroll(ctx).Fill().TrackScroll(&s).Children(func() {
			coreBox(ctx).Height(500).Shrink(0)
			coreBox(ctx).Size(250, 2000).Shrink(0).HandleTextInput(c).Role(RoleTextField).Label("code")
			coreBox(ctx).Height(500).Shrink(0)
		})
	}, 300, 200)
	defer tt.rt.close()
	// The caret is 500+1500 down the content, 20 high, in a view 200 high.
	shows := func() bool { return s.Y <= 2000 && s.Y+200 >= 2020 }
	tt.Key(0, KeyTab)
	tt.Frame()
	if !shows() {
		t.Fatalf("Tab into the field scrolled to %v, which doesn't show its caret", s.Y)
	}
	s.Y = 0
	tt.Frame()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	field := accessNode(t, tt.h.access, platform.RoleTextField, "code")
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessFocus})
	tt.Frame()
	if !shows() {
		t.Fatalf("focusing the field scrolled to %v, which doesn't show its caret", s.Y)
	}
	// Up the field, the caret comes into view from above.
	c.caretY = 100
	s.Y = 2200
	tt.Frame()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessFocus})
	tt.Frame()
	if s.Y > 600 || s.Y+200 < 620 {
		t.Fatalf("focusing the field scrolled to %v, which doesn't show its caret at 600", s.Y)
	}
}

// Space types a space into an app's text field in a scroll view, and
// Shift+Space a space too, without paging the view, as in a web page's
// field (#1410); a key that types nothing, as PageDown, still scrolls it.
func TestSpaceInTextClientDoesNotScroll(t *testing.T) {
	var s ScrollState
	c := &primitiveClient{}
	tt := coreNewTester(func(ctx *context) {
		coreScroll(ctx).Fill().TrackScroll(&s).Children(func() {
			coreBox(ctx).Size(250, 40).Shrink(0).HandleTextInput(c).AutoFocus()
			coreBox(ctx).Height(2000).Shrink(0)
		})
	}, 300, 200)
	defer tt.rt.close()
	tt.TypeKey(0, KeySpace, " ")
	tt.Frame()
	if c.text != " " || s.Y != 0 {
		t.Fatalf("Space typed %q and scrolled to %v", c.text, s.Y)
	}
	tt.Key(0, KeyPageDown)
	tt.Frame()
	y := s.Y
	if y == 0 {
		t.Fatal("PageDown did not scroll the view around the field")
	}
	tt.TypeKey(Shift, KeySpace, " ")
	tt.Frame()
	if c.text != "  " || s.Y != y {
		t.Fatalf("Shift+Space typed %q and scrolled from %v to %v", c.text, y, s.Y)
	}
}

// An element taking text itself (TextCaret) whose HandleInput leaves the
// key of Space, for the text that comes after it, doesn't page the view
// around it either; a read-only text input, which takes no text, does.
func TestSpaceScrollsOnlyWithoutTextFocus(t *testing.T) {
	var s ScrollState
	typed, readOnly, value := "", false, "fixed"
	tt := coreNewTester(func(ctx *context) {
		coreScroll(ctx).Fill().TrackScroll(&s).Children(func() {
			if readOnly {
				coreTextInput(ctx, &value).ReadOnly(true).AutoFocus().Width(250)
			} else {
				coreBox(ctx).Size(250, 40).Shrink(0).Focusable().AutoFocus().TextCaret(Rect{W: 1, H: 16}).HandleInput(func(ev InputEvent) bool {
					if ev.Kind == InputText {
						typed += ev.Text
						return true
					}
					return false
				})
			}
			coreBox(ctx).Height(2000).Shrink(0)
		})
	}, 300, 200)
	defer tt.rt.close()
	tt.TypeKey(0, KeySpace, " ")
	tt.Frame()
	if typed != " " || s.Y != 0 {
		t.Fatalf("Space typed %q and scrolled to %v", typed, s.Y)
	}
	readOnly = true
	tt.Frame()
	tt.Frame()
	tt.TypeKey(0, KeySpace, " ")
	tt.Frame()
	if value != "fixed" || s.Y == 0 {
		t.Fatalf("Space in a read-only input made it %q and scrolled to %v", value, s.Y)
	}
}
