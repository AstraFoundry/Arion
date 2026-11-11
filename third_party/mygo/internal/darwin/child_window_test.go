//go:build darwin

package darwin

import (
	"slices"
	"testing"
)

// TestChildOrder checks the order child windows go above their parent
// in: those that show as the app's windows are, front to back, then
// those that hid with the parent, by rank.
func TestChildOrder(t *testing.T) {
	ranks := map[string]int{"h0": 0, "h1": 1, "h2": 2, "a": 0}
	shows := func(w string) bool { return w[0] == 'a' || w[0] == 'b' } // a and b are children
	rank := func(w string) int { return ranks[w] }
	for _, c := range []struct {
		name          string
		front, hidden []string
		want          []string
	}{
		{"none", []string{"x", "parent"}, nil, nil},
		{"among other windows", []string{"b", "x", "a", "parent", "y"}, nil, []string{"b", "a"}},
		{"behind their parent too", []string{"parent", "a", "b"}, nil, []string{"a", "b"}},
		{"hidden ones by rank, behind", []string{"a", "parent"}, []string{"h2", "h0", "h1"}, []string{"a", "h0", "h1", "h2"}},
		{"hidden and shown again once", []string{"a", "parent"}, []string{"a"}, []string{"a"}},
	} {
		if got := childOrder(c.front, shows, c.hidden, rank); !slices.Equal(got, c.want) {
			t.Errorf("%s: childOrder(%q, %q) = %q, want %q", c.name, c.front, c.hidden, got, c.want)
		}
	}
}
