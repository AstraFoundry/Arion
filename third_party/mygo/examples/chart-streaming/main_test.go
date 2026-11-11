package main

import (
	"testing"
	"time"
)

func TestWaveformRing(t *testing.T) {
	s := newWaveform("raw")
	start := time.Unix(1, 0)
	s.update(start)
	for _, elapsed := range []time.Duration{time.Millisecond, 17 * time.Millisecond, 16 * time.Second, 40*time.Second + time.Millisecond/2} {
		s.update(start.Add(elapsed))
		want := int64(samples) + elapsed.Milliseconds()
		if s.next != want {
			t.Fatal(s.next, want)
		}
		for k := range 3 {
			for i := range samples {
				if s.ys[k][i] != value(want-samples+int64(i), k) {
					t.Fatalf("bad ring at %v, %d/%d", elapsed, k, i)
				}
			}
		}
		if s.xs[0] > 0 || s.xs[samples-1] > 1 || s.xs[samples-1] < .999 {
			t.Fatal("bad rolling range")
		}
	}
}
