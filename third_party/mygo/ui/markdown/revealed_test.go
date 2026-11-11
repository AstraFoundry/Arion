package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// revealChat is a chat's reply as the issue shows it (#1380): streamed,
// then shown with View once done, its time stamp built below it once it
// is done and Revealed.
type revealChat struct {
	text     string
	done     bool
	o        *Options
	revealed bool       // what Revealed said in the last frame
	st       *viewState // the view's state, to check it against
}

func (m *revealChat) view(c *ui.Context) {
	ui.Column(c).Children(func() {
		var md ui.Element
		if m.done {
			md = View(c, m.text, m.o)
		} else {
			md = Stream(c, m.text, m.o)
		}
		m.revealed = Revealed(md)
		m.st = *ui.Local(md, stateKey{}, newState)
		if m.done && m.revealed {
			ui.Text(c, "12:04")
		}
	})
}

// shows returns the text of the reply the view shows, without the time.
func shows(tt *ui.Tester) string {
	return strings.TrimSuffix(strings.Join(tt.Texts(), ""), "12:04")
}

// TestRevealedChat streams a reply in bursts, as a model's come, and
// checks that its time shows only once all of the reply shows and has
// faded in, after the stream is done, on frames the view asks for.
func TestRevealedChat(t *testing.T) {
	const fade = 300 * time.Millisecond
	src := strings.Repeat("Some words of a reply, and **more** of them. ", 12) + "\n\n- an item\n- another"
	m := &revealChat{o: &Options{}}
	tt := ui.NewTester(m.view, 600, 800)
	for i := 0; i < len(src); {
		i = min(i+40, len(src))
		m.text = src[:i]
		tt.Frame()
		if tt.HasText("12:04") {
			t.Fatal("the time shows while the reply streams")
		}
		tt.Advance(16 * time.Millisecond)
	}
	// The last bytes came: the stream is done, with most of it still to
	// show.
	m.done = true
	tt.Frame()
	if m.revealed || tt.HasText("12:04") {
		t.Fatalf("revealed as the stream completes, showing %d of %d bytes", len(shows(tt)), len(src))
	}
	whole := ui.NewTester(func(c *ui.Context) { View(c, src, nil) }, 600, 800)
	want := strings.Join(whole.Texts(), "")
	var allAt time.Time
	for range 300 {
		// Only the clock moves: the view asks for the frames.
		tt.Advance(10 * time.Millisecond)
		all := shows(tt) == want
		if all && allAt.IsZero() {
			allAt = tt.Now()
		}
		if m.revealed != (all && !m.st.reveal.fading()) {
			t.Fatalf("Revealed is %v, with all shown %v and fading %v", m.revealed, all, m.st.reveal.fading())
		}
		if m.revealed {
			break
		}
	}
	if !tt.HasText("12:04") {
		t.Fatalf("the time never showed: %d of %d bytes show", len(shows(tt)), len(want))
	}
	// The last unit faded in before the time showed.
	if d := tt.Now().Sub(allAt); d < fade-20*time.Millisecond || d > fade+50*time.Millisecond {
		t.Errorf("the time showed %v after the whole reply did, want about the fade's %v", d, fade)
	}
	if tt.FramePending() {
		t.Error("the view still asks for frames once revealed")
	}
}

// TestRevealedGrows checks that a Stream revealed turns unrevealed as
// more text comes, and revealed again once that shows.
func TestRevealedGrows(t *testing.T) {
	m := &revealChat{o: &Options{}, text: "First words. "}
	tt := ui.NewTester(m.view, 600, 400)
	settle := func(what string) {
		t.Helper()
		for range 100 {
			if m.revealed {
				return
			}
			tt.Advance(16 * time.Millisecond)
		}
		t.Fatalf("%s: never revealed", what)
	}
	settle("the first words")
	m.text += strings.Repeat("and more words ", 20)
	tt.Frame()
	if m.revealed {
		t.Fatal("still revealed as more text came")
	}
	settle("more text")
	if got, want := shows(tt), strings.TrimSpace(m.text); got != want {
		t.Errorf("revealed, showing %q, want %q", got, want)
	}
	m.done = true
	tt.Frame()
	if !m.revealed || !tt.HasText("12:04") {
		t.Error("done with all shown, not revealed")
	}
}

// TestRevealedAtOnce checks text that shows at once: revealed in the
// frame it comes, as is no text.
func TestRevealedAtOnce(t *testing.T) {
	for _, c := range []struct {
		name   string
		text   string
		stream bool
		o      *Options
	}{
		{"empty view", "", false, nil},
		{"empty stream", "", true, nil},
		{"view", sample, false, nil},
		{"view, revealed by word", sample, false, &Options{Reveal: &Reveal{}}},
		{"stream at once", sample, true, &Options{Reveal: &Reveal{Unit: AtOnce}}},
	} {
		m := &revealChat{text: c.text, done: !c.stream, o: c.o}
		ui.NewTester(m.view, 600, 800)
		if !m.revealed {
			t.Errorf("%s: not revealed", c.name)
		}
	}
	// Nor is an edit, which shows at once.
	m := &revealChat{o: &Options{}, text: "Some words that come in."}
	tt := ui.NewTester(m.view, 600, 400)
	for range 100 {
		tt.Advance(16 * time.Millisecond)
	}
	m.text = "Other words."
	tt.Frame()
	if !m.revealed {
		t.Error("an edit: not revealed")
	}
}

// TestRevealedWithoutFade checks that without a fade, by Reveal.Fade or
// the desktop's Reduce Motion, text is revealed once its last word shows.
func TestRevealedWithoutFade(t *testing.T) {
	text := strings.Repeat("word ", 100)
	for _, c := range []struct {
		name   string
		o      *Options
		reduce bool
	}{
		{"negative fade", &Options{Reveal: &Reveal{Fade: -1}}, false},
		{"reduce motion", &Options{}, true},
	} {
		m := &revealChat{o: c.o}
		tt := ui.NewTester(m.view, 600, 800)
		if c.reduce {
			tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
		}
		m.text = text
		tt.Frame()
		for range 200 {
			all := shows(tt) == strings.TrimSpace(text)
			if m.revealed != all {
				t.Fatalf("%s: Revealed is %v with all shown %v", c.name, m.revealed, all)
			}
			if all {
				break
			}
			tt.Advance(16 * time.Millisecond)
		}
		if !m.revealed {
			t.Errorf("%s: never revealed", c.name)
		}
		if tt.FramePending() {
			t.Errorf("%s: the view still asks for frames once revealed", c.name)
		}
	}
}

// TestRevealedBySentence checks that a sentence Stream holds back keeps
// the view unrevealed, without frames, until View shows it.
func TestRevealedBySentence(t *testing.T) {
	m := &revealChat{o: &Options{Reveal: &Reveal{Unit: BySentence, Fade: -1}}, text: "A whole sentence. And half of"}
	tt := ui.NewTester(m.view, 600, 400)
	tt.Advance(time.Second)
	if m.revealed {
		t.Error("revealed with a sentence held back")
	}
	if tt.FramePending() {
		t.Error("the view asks for frames while a sentence waits for its end")
	}
	m.done = true
	tt.Frame()
	tt.Advance(time.Second)
	if !m.revealed || !tt.HasText("12:04") {
		t.Errorf("done: not revealed, showing %q", shows(tt))
	}
}

// TestRevealedOtherElement checks Revealed of an element no view
// returned.
func TestRevealedOtherElement(t *testing.T) {
	var box, zero bool
	ui.NewTester(func(c *ui.Context) {
		box = Revealed(ui.Box(c))
		zero = Revealed(ui.Element{})
	}, 100, 100)
	if box || zero {
		t.Errorf("Revealed of a box %v, of no element %v", box, zero)
	}
}
