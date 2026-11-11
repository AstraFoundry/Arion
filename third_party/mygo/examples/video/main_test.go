package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The background plays behind the content, and the button pauses it and
// plays it again.
func TestVideoBackground(t *testing.T) {
	s, err := newApp()
	if err != nil {
		t.Fatal(err)
	}
	defer s.waves.Close()
	if w, h := s.waves.Size(); w != 192 || h != 108 {
		t.Errorf("the clip is %d×%d", w, h)
	}
	tt := ui.NewTester(s.view, 640, 360)
	if !tt.HasText("Wavelength") || !s.waves.Playing() || !s.waves.Loop() {
		t.Fatalf("texts %q, playing %v, looping %v", tt.Texts(), s.waves.Playing(), s.waves.Loop())
	}
	if err := tt.Click("Pause"); err != nil {
		t.Fatal(err)
	}
	if s.waves.Playing() {
		t.Fatal("Pause did not hold the clip")
	}
	if err := tt.Click("Play"); err != nil {
		t.Fatal(err)
	}
	if !s.waves.Playing() {
		t.Fatal("Play did not go on")
	}
}
