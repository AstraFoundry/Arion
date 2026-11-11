//go:build linux && (amd64 || arm64)

package linux

import (
	"testing"
	"unsafe"
)

// The frame clock spaces frames by the interval of the monitor's rate in
// millihertz, as GDK computes it from a Wayland output's: 165.001 Hz is
// 6060 µs, not the 16667 of 60 Hz, which only an unknown rate keeps.
func TestRefreshInterval(t *testing.T) {
	for _, c := range []struct {
		mhz  int32
		want int64
	}{{165001, 6060}, {144000, 6944}, {60000, 16666}, {59940, 16683}, {0, 0}, {-1, 0}} {
		if got := refreshInterval(c.mhz); got != c.want {
			t.Errorf("refreshInterval(%d) = %d µs, want %d", c.mhz, got, c.want)
		}
	}
}

// setRefreshInterval finds refresh_interval in GdkFrameTimings of either
// layout of GTK 3.24 through its getter, and leaves the other fields as
// they were.
func TestSetRefreshInterval(t *testing.T) {
	defer func(at int) { refreshIntervalAt = at }(refreshIntervalAt)
	// ref_count, frame_counter, cookie, frame_time, smoothed_frame_time
	// (3.24.24 on), drawn_time, presentation_time, refresh_interval,
	// predicted_presentation_time.
	for _, c := range []struct {
		name     string
		interval int // index of refresh_interval in int64s
		fields   map[int]int64
	}{
		{"GTK 3.24.24", 7, map[int]int64{0: 1, 1: 42, 3: 1000, 4: 1000, 8: 26000}},
		{"GTK 3.24.24 frozen", 7, map[int]int64{0: 1, 1: 42, 3: 1000, 4: 1000}},
		{"GTK 3.22", 6, map[int]int64{0: 1, 1: 42, 3: 1000, 7: 26000}},
		{"GTK 3.22 frozen", 6, map[int]int64{0: 1, 1: 42, 3: 1000}},
	} {
		t.Run(c.name, func(t *testing.T) {
			refreshIntervalAt = 0
			timings := new([12]int64)
			for i, v := range c.fields {
				timings[i] = v
			}
			p := ptr(unsafe.Pointer(timings))
			get := func(q ptr) int64 {
				if q != p {
					t.Fatal("got the interval of other timings")
				}
				return timings[c.interval]
			}
			for frame := range 2 {
				timings[c.interval] = 0
				if !setRefreshInterval(p, 6060, get) {
					t.Fatalf("frame %d: refresh interval not set", frame)
				}
				for i, v := range timings {
					want := c.fields[i]
					if i == c.interval {
						want = 6060
					}
					if v != want {
						t.Fatalf("frame %d: timings[%d] = %d, want %d", frame, i, v, want)
					}
				}
			}
			if refreshIntervalAt != c.interval*8 {
				t.Errorf("found the interval at %d, want %d", refreshIntervalAt, c.interval*8)
			}
		})
	}

	// Timings where neither offset holds it are left alone, and not
	// probed again.
	refreshIntervalAt = 0
	timings := new([12]int64)
	if setRefreshInterval(ptr(unsafe.Pointer(timings)), 6060, func(ptr) int64 { return 0 }) {
		t.Fatal("set an interval its getter does not read")
	}
	if *timings != ([12]int64{}) || refreshIntervalAt != -1 {
		t.Fatalf("timings %v, offset %d", *timings, refreshIntervalAt)
	}
}
