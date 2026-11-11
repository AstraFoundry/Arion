package ui

import "testing"

// TestInViewAmount scrolls a card 100 DIPs tall into a container 200 tall:
// it counts at a fifth, at half and whole as that much of it shows, with
// a margin as the root grown, and invisible not at all.
func TestInViewAmount(t *testing.T) {
	var some, fifth, half, all, over, grown, invisible, scaled, scaledHalf bool
	var allocs float64
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Height(200).Children(func() {
			coreBox(c).Height(195)
			card := coreBox(c).Key("card").Height(100)
			coreBox(c).Height(400)
			some, fifth, half, all = card.InViewAmount(0), card.InViewAmount(0.2), card.InViewAmount(0.5), card.InViewAmount(1)
			over = card.InViewAmount(2)
			grown = card.InViewAmount(0.5, 0, 0, 50, 0)
			invisible = coreBox(c).Key("hidden").Height(10).Invisible().InViewAmount(0.1)
			allocs = testing.AllocsPerRun(10, func() { card.InViewAmount(0.5, 20) })
		})
		// A box scaled twice as large around its center, at the window's
		// top left corner: 35 of its 100 DIPs across show, and 75 of 100
		// down, a quarter of it, though more than its unscaled area.
		box := coreBox(c).Key("scaled").Absolute().Left(-40).Top(0).Size(50, 50).Scale(2, 2)
		scaled, scaledHalf = box.InViewAmount(0.25), box.InViewAmount(0.5)
	}, 200, 400)
	tt.Frame()
	if !some || fifth || half || all || !grown || invisible {
		t.Fatalf("5 of 100 DIPs showing: some %v, a fifth %v, half %v, all %v, half within 50 below %v, invisible %v",
			some, fifth, half, all, grown, invisible)
	}
	if allocs != 0 {
		t.Errorf("InViewAmount allocated %v times", allocs)
	}
	if !scaled || scaledHalf {
		t.Errorf("a box scaled about its center, a quarter of it showing: a quarter %v, half %v", scaled, scaledHalf)
	}
	tt.Scroll(50, 50, 0, 25) // 30 show
	tt.Frame()
	if !fifth || half || all {
		t.Fatalf("30 of 100 DIPs showing: a fifth %v, half %v, all %v", fifth, half, all)
	}
	tt.Scroll(50, 50, 0, 25) // 55 show
	tt.Frame()
	if !half || all {
		t.Fatalf("55 of 100 DIPs showing: half %v, all %v", half, all)
	}
	tt.Scroll(50, 50, 0, 50) // all of it
	tt.Frame()
	if !all || !over {
		t.Fatalf("all of it showing: all %v, more than all %v", all, over)
	}
}
