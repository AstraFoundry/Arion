package whatsnew

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	catalog "arion.local/desktop/ui/entities/whatsnew"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// TestEverySlideHasAScene is the catalog's other half: a slide listed for
// a release must have its recording and copy here.
func TestEverySlideHasAScene(t *testing.T) {
	listed := map[string]bool{}
	for _, release := range catalog.Releases() {
		for _, id := range release.Slides {
			listed[id] = true
			s, ok := scenes[id]
			if !ok {
				t.Errorf("%s lists slide %q, which has no scene", release.Version, id)
				continue
			}
			if s.final >= s.cycle || s.draw == nil {
				t.Errorf("%s: final frame %v outside its %v loop, or nothing to draw", id, s.final, s.cycle)
			}
			for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
				messages := copy.For(locale)
				for _, key := range []copy.Key{s.title, s.body, s.label} {
					if messages.Text(key) == "" {
						t.Errorf("%s: empty copy %q in %s", id, key, locale)
					}
				}
			}
		}
	}
	for id := range scenes {
		if !listed[id] {
			t.Errorf("scene %q is in no release", id)
		}
	}
}

// The recordings place the pointer by the mock phrases' widths, so the
// phrases are the same text in both languages.
func TestDemoPhrasesMatchAcrossLocales(t *testing.T) {
	for _, key := range []copy.Key{copy.WhatsNewDemoPhrase, copy.WhatsNewDemoTerm, copy.WhatsNewDemoPrompt} {
		if en, zh := copy.For(copy.English).Text(key), copy.For(copy.SimplifiedChinese).Text(key); en != zh {
			t.Errorf("%s differs: %q and %q", key, en, zh)
		}
	}
}

func TestAskStoryOrder(t *testing.T) {
	steps := []struct {
		ms   int
		want func(askFrame) bool
		what string
	}{
		{0, func(f askFrame) bool { return f.Selected == 0 && !f.Bar && f.Cursor.At == rest && f.Lens.Zoom == 1 }, "opens on the document"},
		{1700, func(f askFrame) bool { return f.Selected > 0 && f.Selected < 1 && !f.Bar && f.Lens.Zoom > 1.3 }, "the drag selects, zoomed in"},
		{2300, func(f askFrame) bool { return f.Selected == 1 && f.Bar && !f.Menu && f.Hot == "" }, "the bar follows the selection"},
		{3200, func(f askFrame) bool { return f.Menu && f.Hot == "claude" && f.Row == -1 }, "resting on Claude opens its menu"},
		{4000, func(f askFrame) bool { return f.Menu && f.Row == 1 && !f.Answered }, "the pointer is on Explain"},
		{4200, func(f askFrame) bool { return !f.Bar && !f.Menu && f.Cursor.Press > 0 }, "the click closes the bar"},
		{6500, func(f askFrame) bool { return f.Answered && f.Lens.Focus == chatFocus && f.Lens.Zoom > 1.1 }, "Claude opens, its input in focus"},
		{9100, func(f askFrame) bool { return f.Answered && f.Lens.Zoom == 1 }, "zooms out before looping"},
	}
	for _, step := range steps {
		if f := askAt(ms(step.ms)); !step.want(f) {
			t.Errorf("%d ms: %s: %+v", step.ms, step.what, f)
		}
	}
	if final := askAt(askFinal); !final.Answered || final.Bar {
		t.Errorf("final frame %+v", final)
	}
	if loop := askAt(askCycle + time.Millisecond); loop.Answered || loop.Selected != 0 {
		t.Errorf("the loop starts over: %+v", loop)
	}
}

func TestTermStoryOrder(t *testing.T) {
	steps := []struct {
		ms   int
		want func(termFrame) bool
		what string
	}{
		{0, func(f termFrame) bool { return f.Selected == 0 && !f.Bar && !f.Vocabulary }, "opens on the document"},
		{2000, func(f termFrame) bool { return f.Selected == 1 && f.Bar && !f.Hot }, "the bar follows the selection"},
		{3000, func(f termFrame) bool { return f.Bar && f.Hot && f.Cursor.At == vocabAt }, "the pointer is on Add to vocabulary"},
		{3150, func(f termFrame) bool { return !f.Bar && f.Cursor.Press > 0 && !f.Vocabulary }, "the click closes the bar"},
		{3600, func(f termFrame) bool { return f.Vocabulary && !f.Added }, "the vocabulary opens"},
		{6000, func(f termFrame) bool { return f.Added && f.Lens.Focus == termRow }, "the term is in the list, in focus"},
		{8300, func(f termFrame) bool { return f.Added && f.Lens.Zoom == 1 }, "zooms out before looping"},
	}
	for _, step := range steps {
		if f := termAt(ms(step.ms)); !step.want(f) {
			t.Errorf("%d ms: %s: %+v", step.ms, step.what, f)
		}
	}
	if final := termAt(termFinal); !final.Added {
		t.Errorf("final frame %+v", final)
	}
}

func TestPointersAndLensesMoveContinuously(t *testing.T) {
	check := func(name string, cycle time.Duration, at func(time.Duration) (point, lens)) {
		p, l := at(0)
		for t0 := 10 * time.Millisecond; t0 < cycle; t0 += 10 * time.Millisecond {
			q, m := at(t0)
			if d := abs(q.X-p.X) + abs(q.Y-p.Y); d > 8 {
				t.Fatalf("%s: pointer jumps %.1f DIP at %v", name, d, t0)
			}
			if d := abs(m.Zoom - l.Zoom); d > 0.03 {
				t.Fatalf("%s: zoom jumps %.3f at %v", name, d, t0)
			}
			p, l = q, m
		}
		if q, m := at(cycle); q != p || m != l {
			t.Fatalf("%s: the loop does not end where it starts: %+v %+v, %+v %+v", name, p, l, q, m)
		}
	}
	check("ask", askCycle, func(d time.Duration) (point, lens) { f := askAt(d); return f.Cursor.At, f.Lens })
	check("term", termCycle, func(d time.Duration) (point, lens) { f := termAt(d); return f.Cursor.At, f.Lens })
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

type actions struct {
	runtime.Actions
	state     *runtime.State
	dismissed int
}

func (a *actions) DismissWhatsNew() { a.dismissed++; a.state.WhatsNew.Slides = nil }

func dialog(t *testing.T, locale copy.Locale, slides []runtime.WhatsNewSlide, width, height int, options ...func(*ui.Tester)) (*ui.Tester, *actions, *runtime.State) {
	t.Helper()
	state := &runtime.State{WhatsNew: runtime.WhatsNew{Slides: slides, Seq: 1}}
	state.Preferences.UILanguage = runtime.UILanguage(locale)
	recorder := &actions{state: state}
	store := runtime.Store{State: state, Actions: recorder}
	d := &Dialog{}
	tt := ui.NewTester(func(c *ui.Context) {
		tokens := theme.Apply(c)
		ui.Column(c).Fill().Background(tokens.Canvas)
		d.Render(c, store, tokens)
	}, width, height)
	for _, option := range options {
		option(tt)
	}
	settle(tt)
	return tt, recorder, state
}

func settle(tt *ui.Tester) {
	for range 40 {
		tt.Frame()
	}
}

var both = []runtime.WhatsNewSlide{{ID: "selection-actions", Version: "0.1.6"}, {ID: "selection-vocabulary", Version: "0.1.6"}}

func has(tt *ui.Tester, text string) bool { return slices.Contains(tt.Texts(), text) }

func TestDialogStepsThroughItsSlides(t *testing.T) {
	for _, locale := range []copy.Locale{copy.English, copy.SimplifiedChinese} {
		messages := copy.For(locale)
		tt, recorder, _ := dialog(t, locale, both, 760, 560)
		text := messages.Text
		if !has(tt, text(copy.WhatsNewSelectionTitle)) || !has(tt, messages.WhatsNewVersion("0.1.6")) {
			t.Fatalf("%s: first slide missing from %q", locale, tt.Texts())
		}
		if has(tt, text(copy.WhatsNewPrevious)) || !has(tt, text(copy.Skip)) || !has(tt, text(copy.WhatsNewNext)) {
			t.Fatalf("%s: first slide offers Skip and Next only: %q", locale, tt.Texts())
		}
		tt.Click(text(copy.WhatsNewNext))
		settle(tt)
		if !has(tt, text(copy.WhatsNewVocabularyTitle)) || has(tt, text(copy.WhatsNewSelectionTitle)) {
			t.Fatalf("%s: Next shows the second slide: %q", locale, tt.Texts())
		}
		if has(tt, text(copy.Skip)) || !has(tt, text(copy.Done)) || !has(tt, text(copy.WhatsNewPrevious)) {
			t.Fatalf("%s: last slide offers Previous and Done: %q", locale, tt.Texts())
		}
		tt.Click(text(copy.WhatsNewPrevious))
		settle(tt)
		if !has(tt, text(copy.WhatsNewSelectionTitle)) {
			t.Fatalf("%s: Previous returns to the first slide: %q", locale, tt.Texts())
		}
		tt.Key(0, ui.KeyRight)
		settle(tt)
		if !has(tt, text(copy.WhatsNewVocabularyTitle)) {
			t.Fatalf("%s: the right arrow advances: %q", locale, tt.Texts())
		}
		if recorder.dismissed != 0 {
			t.Fatalf("%s: dismissed while stepping", locale)
		}
		tt.Click(text(copy.Done))
		settle(tt)
		if recorder.dismissed != 1 || has(tt, text(copy.WhatsNewVocabularyTitle)) {
			t.Fatalf("%s: Done dismisses once and closes: %d %q", locale, recorder.dismissed, tt.Texts())
		}
	}
}

func TestSkipAndEscapeDismiss(t *testing.T) {
	messages := copy.For(copy.English)
	tt, recorder, _ := dialog(t, copy.English, both, 760, 560)
	tt.Click(messages.Text(copy.Skip))
	settle(tt)
	if recorder.dismissed != 1 {
		t.Fatalf("Skip dismissed %d times", recorder.dismissed)
	}
	tt, recorder, _ = dialog(t, copy.English, both, 760, 560)
	tt.Key(0, ui.KeyEscape)
	settle(tt)
	if recorder.dismissed != 1 {
		t.Fatalf("Escape dismissed %d times", recorder.dismissed)
	}
}

func TestSingleSlideOffersOnlyDone(t *testing.T) {
	messages := copy.For(copy.English)
	tt, _, _ := dialog(t, copy.English, both[:1], 760, 560)
	if has(tt, messages.Text(copy.Skip)) || has(tt, messages.Text(copy.WhatsNewNext)) || !has(tt, messages.Text(copy.Done)) {
		t.Fatalf("one slide: %q", tt.Texts())
	}
}

func TestNewPresentationStartsOver(t *testing.T) {
	messages := copy.For(copy.English)
	tt, _, state := dialog(t, copy.English, both, 760, 560)
	tt.Click(messages.Text(copy.WhatsNewNext))
	settle(tt)
	state.WhatsNew.Seq++
	settle(tt)
	if !has(tt, messages.Text(copy.WhatsNewSelectionTitle)) {
		t.Fatalf("a new presentation starts at its first slide: %q", tt.Texts())
	}
}

// TestFramesRender draws moments of each recording in both appearances
// and, with ARION_UI_SNAPSHOTS, saves them.
func TestFramesRender(t *testing.T) {
	messages := copy.For(copy.English)
	for _, step := range []struct {
		id   string
		ms   int
		want string
	}{
		{"selection-actions", 1700, messages.Text(copy.WhatsNewDemoPhrase)},
		{"selection-actions", 2900, messages.Text(copy.SelectionVocabulary)},
		{"selection-actions", 4000, messages.Text(copy.SelectionExplain)},
		{"selection-actions", 7000, messages.Text(copy.WhatsNewDemoPrompt)},
		{"selection-vocabulary", 3000, messages.Text(copy.SelectionVocabulary)},
		{"selection-vocabulary", 6400, messages.Text(copy.NavVocabulary)},
	} {
		for _, dark := range []bool{false, true} {
			s := scenes[step.id]
			tt := ui.NewTester(func(c *ui.Context) {
				tokens := theme.Apply(c)
				ui.Column(c).Fill().Clip().Background(tokens.Scrim).Children(func() {
					s.draw(c, tokens, messages, ms(step.ms), false)
				})
			}, Width, StageHeight)
			tt.SetDark(dark)
			settle(tt)
			if !has(tt, step.want) {
				t.Errorf("%s %d ms: %q missing from %q", step.id, step.ms, step.want, tt.Texts())
			}
			save(t, tt, fmt.Sprintf("whatsnew-%s-%05d-dark-%v.png", step.id, step.ms, dark))
		}
	}
}

func TestDialogRendersInBothAppearancesAndSmallWindows(t *testing.T) {
	for _, shot := range []struct {
		name          string
		width, height int
		dark          bool
		locale        copy.Locale
	}{
		{"light", 760, 560, false, copy.English}, {"dark", 760, 560, true, copy.SimplifiedChinese}, {"small", 640, 440, false, copy.English},
	} {
		tt, _, _ := dialog(t, shot.locale, both, shot.width, shot.height, func(tt *ui.Tester) { tt.SetDark(shot.dark) })
		if !has(tt, copy.For(shot.locale).Text(copy.WhatsNewSelectionTitle)) {
			t.Errorf("%s: %q", shot.name, tt.Texts())
		}
		save(t, tt, "whatsnew-dialog-"+shot.name+".png")
	}
}

func TestReducedMotionHoldsTheFinalFrame(t *testing.T) {
	tt, _, _ := dialog(t, copy.English, both, 760, 560, func(tt *ui.Tester) { tt.SetPreferences(ui.Preferences{ReduceMotion: true}) })
	if !has(tt, copy.For(copy.English).Text(copy.WhatsNewDemoPrompt)) {
		t.Errorf("the final frame shows the prompt in Claude: %q", tt.Texts())
	}
}

func save(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("ARION_UI_SNAPSHOTS")
	if dir == "" {
		return
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, tt.Image()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}
