package ui

import (
	"strings"
	"testing"
)

// Pretty wraps a paragraph as it would, but for its last line, which takes
// a word from the line before rather than hold one alone.
func TestPrettyText(t *testing.T) {
	last := func(n *node) string {
		l := n.tl.Lines[len(n.tl.Lines)-1]
		return strings.TrimSpace(string(n.tl.Runes[l.Start:l.End]))
	}
	// The first of these texts whose last line, wrapped as it is, is a
	// word alone: where it wraps depends on the system's fonts.
	words := strings.Fields("Body text wraps its words over as many lines as it takes and then ends on a word that is alone x")
	var plain, pretty *node
	for n := 8; n <= len(words); n++ {
		s := strings.Join(words[:n], " ")
		coreNewTester(func(c *context) {
			coreColumn(c).Width(240).AlignItems(Start).Children(func() {
				plain = coreText(c, s).FontSize(20)
				pretty = coreText(c, s).FontSize(20).Pretty()
			})
		}, 400, 400)
		if len(plain.tl.Lines) >= 2 && !strings.Contains(last(plain), " ") {
			break
		}
	}
	pl, pr := plain.tl.Lines, pretty.tl.Lines
	if len(pl) < 2 || strings.Contains(last(plain), " ") {
		t.Fatalf("no text ends on a word alone: %d lines, the last %q", len(pl), last(plain))
	}
	if len(pr) != len(pl) || !strings.Contains(last(pretty), " ") {
		t.Errorf("%d lines pretty, the last %q", len(pr), last(pretty))
	}
	for i := range len(pl) - 2 {
		if pl[i].Start != pr[i].Start || pl[i].End != pr[i].End {
			t.Errorf("line %d moved: %d-%d, pretty %d-%d", i, pl[i].Start, pl[i].End, pr[i].Start, pr[i].End)
		}
	}
	if !pretty.tl.Params.Pretty {
		t.Error("the layout is not pretty")
	}
}
