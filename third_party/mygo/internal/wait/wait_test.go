package wait

import (
	"sync/atomic"
	"testing"
)

func TestUntil(t *testing.T) {
	var n atomic.Int32
	go func() {
		for range 5 {
			n.Add(1)
		}
	}()
	Until(t, "five", func() bool { return n.Load() == 5 })
}

func TestFor(t *testing.T) {
	ch := make(chan int, 1)
	go func() { ch <- 7 }()
	if v := For(t, "a value", ch); v != 7 {
		t.Fatalf("For = %d", v)
	}
}
