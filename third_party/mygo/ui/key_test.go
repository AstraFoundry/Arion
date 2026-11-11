package ui

import (
	"math"
	"testing"
)

type keyItem struct{ title string }

// keyedStates builds a box keyed by each key under one parent, and returns
// the local state of each.
func keyedStates(c *Context, keys []any) []*int {
	var states []*int
	Column(c).Children(func() {
		for _, k := range keys {
			e := Box(c.Key(k)).Size(10, 10)
			states = append(states, Local(e, "n", func() int { return 0 }))
		}
	})
	return states
}

// A pointer keys the object it points to: its element keeps its state as
// the object changes, and two objects alike keep a state each. The Tester
// panics on two siblings with one key.
func TestPointerKeysFollowTheirObjects(t *testing.T) {
	a, b := &keyItem{"a"}, &keyItem{"b"}
	var states []*int
	tt := NewTester(func(c *Context) { states = keyedStates(c, []any{a, b}) }, 50, 50)
	*states[0], *states[1] = 1, 2
	a.title = "renamed"
	tt.Frame()
	if *states[0] != 1 {
		t.Fatalf("renaming the object reset its element's state to %d", *states[0])
	}
	b.title = "renamed" // alike, and still another object
	tt.Frame()
	if states[0] == states[1] || *states[1] != 2 {
		t.Fatalf("two objects alike share a state: %d and %d", *states[0], *states[1])
	}
}

// Keys equal in value but of different types are different keys, as in a
// map.
func TestKeysOfDifferentTypesDiffer(t *testing.T) {
	type id int32
	var states []*int
	NewTester(func(c *Context) {
		states = keyedStates(c, []any{int(1), int64(1), uint64(1), int32(1), uint32(1), id(1), [1]int32{1}, keyItem{"a"}})
	}, 50, 50)
	seen := map[*int]bool{}
	for _, s := range states {
		if seen[s] {
			t.Fatal("two keys of different types share a state")
		}
		seen[s] = true
	}
}

// A key Go cannot compare, as a slice, still keys its element by its value.
func TestUncomparableKeys(t *testing.T) {
	var states []*int
	tt := NewTester(func(c *Context) { states = keyedStates(c, []any{[]int{1}, []int{2}}) }, 50, 50)
	*states[0] = 1
	tt.Frame()
	if *states[0] != 1 || states[0] == states[1] {
		t.Fatalf("slice keys: states %d and %d", *states[0], *states[1])
	}
}

// A key not equal to itself, as a NaN, keeps its element's state from
// frame to frame, though a map would hash it anew each time.
func TestNaNKeys(t *testing.T) {
	nan := math.NaN()
	var states []*int
	tt := NewTester(func(c *Context) { states = keyedStates(c, []any{nan, [1]float64{nan}}) }, 50, 50)
	*states[0], *states[1] = 1, 2
	tt.Frame()
	if *states[0] != 1 || *states[1] != 2 {
		t.Fatalf("NaN keys lost their states: %d and %d", *states[0], *states[1])
	}
}

// Keying by a pointer or a struct allocates nothing, frame after frame.
func TestKeysAllocateNothing(t *testing.T) {
	p, v := &keyItem{"a"}, keyItem{"b"}
	for _, k := range []any{p, v, int32(1), 1, "a"} {
		if n := testing.AllocsPerRun(100, func() { keyedID(1, k) }); n != 0 {
			t.Errorf("keying by %T allocates %v times", k, n)
		}
	}
}
