package main

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// A suggestion asks the assistant, which thinks, the window glowing, then
// writes its reply word by word, and the glow fades once it is done.
func TestReply(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 860, 700)
	if err := tt.Click("Summarize my week"); err != nil {
		t.Fatal(err)
	}
	if len(a.messages) != 2 || !a.busy {
		t.Fatalf("%d messages, busy %v after a suggestion, want 2, busy", len(a.messages), a.busy)
	}
	tt.Frame()
	if !tt.HasText("Summarize my week") {
		t.Error("the question does not show")
	}
	reply := a.messages[1]
	a.step(thinkTime / 2)
	if reply.shown != 0 || a.glow <= 0 {
		t.Fatalf("%d words shown, glow %.2f while thinking, want none, glowing", reply.shown, a.glow)
	}
	a.step(thinkTime / 2)
	a.step(0.1)
	if reply.shown == 0 || reply.shown == len(reply.words) {
		t.Fatalf("%d of %d words shown a moment into writing, want some", reply.shown, len(reply.words))
	}
	for range 200 {
		a.step(0.05)
	}
	if a.busy || reply.shown != len(reply.words) || a.glow != 0 {
		t.Fatalf("busy %v, %d of %d words, glow %.2f long after, want done, all, no glow", a.busy, reply.shown, len(reply.words), a.glow)
	}
	if got := strings.Join(reply.words, ""); got != reply.text {
		t.Errorf("the words join into %q, want the reply", got)
	}
	tt.Frame()
	if !tt.HasText(reply.text) {
		t.Error("the reply does not show")
	}
}

// Return sends what was typed; Escape stops the assistant while it
// thinks.
func TestTypeAndStop(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 860, 700)
	tt.Type("Hello there")
	tt.Key(0, ui.KeyEnter)
	if len(a.messages) != 2 || a.messages[0].text != "Hello there" || a.input != "" {
		t.Fatalf("messages %d, input %q after Return, want the question sent", len(a.messages), a.input)
	}
	tt.Key(0, ui.KeyEscape)
	if a.busy || a.messages[1].shown != 1 || a.messages[1].words[0] != "Stopped." {
		t.Fatalf("busy %v, reply %v after Escape, want stopped", a.busy, a.messages[1].words)
	}
}

// splitWords keeps the spaces and line breaks, which joined give the text
// back.
func TestSplitWords(t *testing.T) {
	for _, s := range []string{"one", "one two", "a  b\n\nc ", "Day 1: x.\n\nDay 2"} {
		w := splitWords(s)
		if strings.Join(w, "") != s {
			t.Errorf("splitWords(%q) = %q", s, w)
		}
	}
	if n := len(splitWords("a b  c\n\nd")); n != 4 {
		t.Errorf("%d words in \"a b  c\\n\\nd\", want 4", n)
	}
}

// The conversation scrolls: once it is longer than the window, it shows
// its end, the whole of the last reply above the composer, not under it.
func TestScrollsToEnd(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 860, 500)
	for i := range 3 {
		if i == 0 {
			if err := tt.Click("Summarize my week"); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := tt.Click("Message"); err != nil {
				t.Fatal(err)
			}
			tt.Type("And then?")
			tt.Key(0, ui.KeyEnter)
		}
		// The reply comes in word by word, as it shows.
		for a.busy {
			a.step(0.1)
			tt.Frame()
		}
	}
	if len(a.messages) != 6 {
		t.Fatalf("%d messages, want 6", len(a.messages))
	}
	// The rows glide up to their place (FollowAnimation), on the clock.
	last := a.messages[len(a.messages)-1]
	tt.Advance(2 * time.Second)
	r, ok := tt.Find(last.text)
	field, _ := tt.Find("Message")
	// The list ends ringPadding and the field's own 6 above the field,
	// and its rows 16 above that once they settled: a reply still
	// gliding is cut off at the list's end, not that far above it.
	if !ok || !a.list.AtEnd() || r.H <= 0 || r.Y+r.H > field.Y-ringPadding-6-12 {
		t.Fatalf("the end of the conversation is not in view: the last reply at %+v, the field at %+v, at the end %v", r, field, a.list.AtEnd())
	}
	if r.H < 3*replySize {
		t.Errorf("the last reply shows %.0f DIPs of its lines, want all of them", r.H)
	}
}

// The edge glow's stops lie in order around the window, and move to new
// places every shuffleEvery seconds, easing there over shuffleTime.
func TestStops(t *testing.T) {
	n := 0
	s := newStops(func() float64 { n++; return float64(n%7) / 7 })
	sorted := func(st []stop) bool {
		for i := 1; i < len(st); i++ {
			if st[i].at < st[i-1].at {
				return false
			}
		}
		return true
	}
	if !sorted(s.show(0)) {
		t.Fatal("the stops are not in order")
	}
	s.step(0)
	from, to := s.from, s.to
	mid := s.show(shuffleTime / 2)
	for i := range mid {
		if want := (from[i].at + to[i].at) / 2; mid[i].at < want-1e-9 || mid[i].at > want+1e-9 {
			t.Errorf("stop %d halfway at %.3f, want %.3f", i, mid[i].at, want)
		}
	}
	if got := s.show(shuffleTime); !sorted(got) || got[0] != to[0] {
		t.Error("the stops did not arrive in order")
	}
	s.step(shuffleEvery / 2)
	if s.since != 0 {
		t.Error("the stops moved again before their time")
	}
	s.step(shuffleEvery)
	if s.since != shuffleEvery {
		t.Error("the stops did not move again on time")
	}
	g := s.gradient(1)
	if len(g) != len(glowColors)+2 || g[0].At != 0 || g[len(g)-1].At != 1 || g[0].Color != g[len(g)-1].Color {
		t.Errorf("gradient %v, want the colors between two stops of one color at 0 and 1", g)
	}
	// Its colors round-trip through Oklab.
	l, a, b := oklab(glowColors[3])
	if c := srgb(l, a, b); c != glowColors[3] {
		t.Errorf("%v through Oklab is %v", glowColors[3], c)
	}
}

// The assistant works in real time: a reply asked for before the window
// went out of sight, its frames paused, has come as far as the time
// tells once it shows again, done if long enough.
func TestCatchesUp(t *testing.T) {
	a := newApp()
	a.send("Summarize my week")
	a.step(thinkTime + 0.5)
	if m := a.messages[1]; m.shown == 0 || !a.busy {
		t.Fatalf("%d words, busy %v half a second into writing, want some, writing", m.shown, a.busy)
	}
	a = newApp()
	a.send("Summarize my week")
	a.step(60)
	if m := a.messages[1]; a.busy || m.shown != len(m.words) {
		t.Fatalf("busy %v, %d of %d words a minute later, want the whole reply", a.busy, m.shown, len(m.words))
	}
}

// A reply that comes all at once, as when the window shows again after it
// was hidden, shows its end at once rather than gliding up from below.
func TestJumpsOnCatchUp(t *testing.T) {
	a := newApp()
	tt := ui.NewTester(a.view, 860, 500)
	if err := tt.Click("Summarize my week"); err != nil {
		t.Fatal(err)
	}
	a.step(60)
	tt.Frame()
	last := a.messages[len(a.messages)-1]
	r, ok := tt.Find(last.text)
	field, _ := tt.Find("Message")
	if !ok || r.Y+r.H > field.Y-ringPadding-6-12 {
		t.Errorf("the reply that came at once ends at %.0f, want it in view above %.0f at once", r.Y+r.H, field.Y-ringPadding-6-12)
	}
}
