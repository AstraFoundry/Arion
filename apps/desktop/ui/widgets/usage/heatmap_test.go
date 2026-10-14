package usage

import "testing"

func TestLevelByRank(t *testing.T) {
	active := []int{10, 20, 30, 40, 50, 60, 70, 5000}
	for words, want := range map[int]int{0: 0, 10: 1, 20: 1, 30: 2, 40: 2, 60: 3, 70: 4, 5000: 4} {
		if got := level(words, active); got != want {
			t.Errorf("level(%d) = %d, want %d", words, got, want)
		}
	}
	if level(12, []int{12}) != levels {
		t.Error("a lone active day is not the busiest")
	}
}

func TestGridFitsWidth(t *testing.T) {
	for _, width := range []float32{300, 470, 770, 2000} {
		g := gridFor(width)
		if right := g.x + float32(g.weeks)*g.pitch - cellGap; right > width+0.01 || g.weeks > maxWeeks || g.cell < minCell-0.01 && g.weeks > 1 {
			t.Errorf("width %v: %+v ends at %v", width, g, right)
		}
		if week, day, ok := g.at(g.x+g.pitch*float32(g.weeks-1)+1, monthLabels+g.pitch*6+1); !ok || week != g.weeks-1 || day != 6 {
			t.Errorf("width %v: last cell hit %d %d %v", width, week, day, ok)
		}
	}
}
