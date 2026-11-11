package markdown

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestViewAnchorInitialAndRepeated(t *testing.T) {
	var scroll ui.ScrollState
	o := Options{Anchor: "#target"}
	source := "# Intro\n\n" + strings.Repeat("A paragraph before the destination.\n\n", 30) + "# Target\n\nDestination.\n"
	tt := ui.NewTester(func(c *ui.Context) {
		c.DisableAnimations()
		ui.Scroll(c).Fill().TrackScroll(&scroll).Children(func() { View(c, source, &o) })
	}, 400, 160)
	if scroll.Y <= 0 {
		t.Fatal("initial anchor did not reveal the heading")
	}
	scroll.Y = 0
	tt.Frame()
	if scroll.Y != 0 {
		t.Fatalf("unchanged anchor kept forcing scroll: %g", scroll.Y)
	}
	o.AnchorVersion++
	tt.Frame()
	if scroll.Y <= 0 {
		t.Fatal("a new version did not repeat the same anchor jump")
	}
	scroll.Y = 0
	source = "# Another document\n\n" + strings.Repeat("Different text before the target.\n\n", 20) + "# Target\n\nNew destination.\n"
	tt.Frame()
	if scroll.Y <= 0 {
		t.Fatal("a replacement document did not apply the same anchor")
	}
}

// TestViewOnLinkSetsAnchor checks that OnLink keeps getting #a-heading
// links, and scrolls the view to the heading by setting Anchor.
func TestViewOnLinkSetsAnchor(t *testing.T) {
	var scroll ui.ScrollState
	var clicked []string
	var o Options
	o.OnLink = func(dest string) {
		clicked = append(clicked, dest)
		if strings.HasPrefix(dest, "#") {
			o.Anchor = dest
		}
	}
	source := "[Jump](#target) [File](next.md)\n\n" + strings.Repeat("A paragraph before the destination.\n\n", 30) + "# Target\n\nDestination.\n"
	tt := ui.NewTester(func(c *ui.Context) {
		c.DisableAnimations()
		ui.Scroll(c).Fill().TrackScroll(&scroll).Children(func() { View(c, source, &o) })
	}, 400, 160)
	if err := tt.Click("File"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Jump"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(clicked) != 2 || clicked[0] != "next.md" || clicked[1] != "#target" {
		t.Fatalf("OnLink got %q, want next.md and #target", clicked)
	}
	if scroll.Y <= 0 {
		t.Fatal("the anchor OnLink set did not reveal its heading")
	}
	if len(tt.OpenedURLs()) != 0 {
		t.Fatalf("links escaped to the system opener: %q", tt.OpenedURLs())
	}
}
