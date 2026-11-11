package main

import (
	"testing"

	"github.com/egoist/mygo/plugins/loading"
	"github.com/egoist/mygo/ui"
)

// The gallery shows every indicator, named, and its options change them.
func TestGallery(t *testing.T) {
	g := &gallery{size: 1}
	tt := ui.NewTester(g.view, 780, 2000)
	for _, s := range loading.All() {
		r, ok := tt.Find(s.Name)
		if !ok || r.H != 24 {
			t.Errorf("%s: found %v, %v high, want 24", s.Name, ok, r.H)
		}
	}
	if err := tt.Click("48"); err != nil {
		t.Fatal(err)
	}
	if r, _ := tt.Find("Arc"); r.H != 48 {
		t.Errorf("Arc %v high after choosing 48", r.H)
	}
	if err := tt.Click("Paused"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !g.paused || tt.FramePending() {
		t.Errorf("paused %v, a frame pending %v, after Paused", g.paused, tt.FramePending())
	}
}
