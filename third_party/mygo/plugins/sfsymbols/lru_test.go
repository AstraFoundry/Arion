package sfsymbols

import "testing"

// TestLRU checks that the cache holds no more than its limit, dropping the
// least recently used, and keeps what was used last.
func TestLRU(t *testing.T) {
	c := newLRU[int, int](3)
	for i := range 3 {
		c.put(i, i*10)
	}
	c.get(0) // 1 is now the least recently used
	c.put(3, 30)
	if c.len() != 3 {
		t.Fatalf("%d entries, want 3", c.len())
	}
	if _, ok := c.get(1); ok {
		t.Error("1 kept, though the least recently used")
	}
	for _, k := range []int{0, 2, 3} {
		if v, ok := c.get(k); !ok || v != k*10 {
			t.Errorf("%d: %d, %v, want %d", k, v, ok, k*10)
		}
	}
	c.put(2, 21)
	if v, _ := c.get(2); v != 21 || c.len() != 3 {
		t.Errorf("2 is %d with %d entries after an update, want 21 with 3", v, c.len())
	}
}
