package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The intro is one timeline: the tweens the view arranges add up to a
// duration it can read, and the controls hold and play all of them
// together.
func TestTimelineIntro(t *testing.T) {
	s := &app{}
	tt := ui.NewTester(s.view, windowW, windowH)
	if !slices.ContainsFunc(tt.Texts(), func(s string) bool { return strings.HasSuffix(s, " / 870 ms") }) {
		t.Fatalf("the intro does not show that it lasts 870ms: %q", tt.Texts())
	}

	if err := tt.Click("Pause"); err != nil {
		t.Fatal(err)
	}
	if !s.paused {
		t.Fatal("Pause did not hold the intro")
	}

	if err := tt.Click("Play"); err != nil {
		t.Fatal(err)
	}
	if s.paused {
		t.Fatal("Play did not go on")
	}

	if err := tt.Click("Reverse"); err != nil {
		t.Fatal(err)
	}
	if !s.reverse {
		t.Fatal("Reverse did not turn the intro around")
	}

	if err := tt.Click("Restart"); err != nil {
		t.Fatal(err)
	}
	if s.run != 1 {
		t.Fatalf("Restart played the intro %d times, want 1", s.run)
	}
}
