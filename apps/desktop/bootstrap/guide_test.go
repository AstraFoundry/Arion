package bootstrap

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestGuideBounds(t *testing.T) {
	area := mygo.Rectangle{X: 0, Y: 25, Width: 1512, Height: 920}
	cases := []struct {
		name     string
		settings mygo.Rectangle
		want     mygo.Rectangle
	}{
		{"under the pane", mygo.Rectangle{X: 300, Y: 120, Width: 715, Height: 600},
			mygo.Rectangle{X: 530, Y: 728, Width: 485, Height: 112}},
		{"narrow window keeps a readable width", mygo.Rectangle{X: 300, Y: 120, Width: 500, Height: 600},
			mygo.Rectangle{X: 530, Y: 728, Width: guideMinWidth, Height: 112}},
		{"tall window overlaps its foot", mygo.Rectangle{X: 300, Y: 40, Width: 715, Height: 880},
			mygo.Rectangle{X: 530, Y: 25 + 920 - guideInset - 112, Width: 485, Height: 112}},
		{"window past the right edge", mygo.Rectangle{X: 1300, Y: 120, Width: 715, Height: 600},
			mygo.Rectangle{X: 1512 - guideInset - 485, Y: 728, Width: 485, Height: 112}},
		{"window past the left edge", mygo.Rectangle{X: -400, Y: 120, Width: 715, Height: 600},
			mygo.Rectangle{X: guideInset, Y: 728, Width: 485, Height: 112}},
	}
	for _, tc := range cases {
		if got := guideBounds(tc.settings, area, 112); got != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestGuideBoundsSecondaryDisplay(t *testing.T) {
	area := mygo.Rectangle{X: 1512, Y: -200, Width: 1920, Height: 1055}
	got := guideBounds(mygo.Rectangle{X: 1700, Y: -100, Width: 715, Height: 600}, area, 112)
	want := mygo.Rectangle{X: 1930, Y: 508, Width: 485, Height: 112}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
