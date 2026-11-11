package ui

import "testing"

// TestNewTesterWith starts a Tester under reduced motion, in the dark,
// with a coarse pointer that does not hover: its first build reads them,
// and a card that fades in as the view mounts it shows at once, where
// with NewTester it starts fading in before SetPreferences comes.
func TestNewTesterWith(t *testing.T) {
	var first *Preferences
	var firstDark, moved bool
	builds := 0
	var shown float32
	view := func(c *context) {
		builds++
		if p := c.Preferences(); first == nil {
			first, firstDark = &p, c.Theme().Dark
		}
		card := coreBox(c).Key("card").Size(20, 20)
		target := float32(0)
		if builds == 1 {
			c.Invalidate() // mounted: fade in from the next frame
		} else {
			target = 1
		}
		shown = card.AnimateTo("shown", target, Smooth)
		moved = moved || !card.Settled("shown")
	}
	prefs := Preferences{ReduceMotion: true, TextScale: 1, Pointer: PointerCoarse, AnyPointer: PointerCoarse, NoHover: true, NoAnyHover: true}
	coreNewTesterWith(view, 200, 100, TesterOptions{Preferences: prefs, Dark: true})
	if *first != prefs || !firstDark {
		t.Errorf("the first build reads %+v, dark %v", *first, firstDark)
	}
	if first.Hover() || first.AnyHover() {
		t.Error("a pointer that does not hover hovers")
	}
	if builds < 2 || moved || shown != 1 {
		t.Errorf("under reduced motion from the start, after %d builds the card moved %v and shows %v", builds, moved, shown)
	}

	// NewTester settles its first frames without them: the card starts
	// to fade in.
	first, moved, builds = nil, false, 0
	tt := coreNewTester(view, 200, 100)
	tt.SetPreferences(prefs)
	if first.ReduceMotion || first.Pointer != PointerFine || !first.Hover() || firstDark {
		t.Errorf("NewTester's first build reads %+v, dark %v", *first, firstDark)
	}
	if !moved {
		t.Error("with NewTester, the card does not start fading in")
	}
}
