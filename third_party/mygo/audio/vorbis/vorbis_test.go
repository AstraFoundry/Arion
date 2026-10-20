package vorbis

import (
	"bytes"
	"math"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/audio"
)

// tone.ogg is 0.5 s of a 440 Hz sine at 22.05 kHz, in stereo.
func TestDecode(t *testing.T) {
	data, err := os.ReadFile("testdata/tone.ogg")
	if err != nil {
		t.Fatal(err)
	}
	d, name, err := audio.NewDecoder(bytes.NewReader(data))
	if err != nil || name != "vorbis" {
		t.Fatal(name, err)
	}
	if d.SampleRate() != 22050 || d.Channels() != 2 || d.Len() < 11025 || d.Len() > 11025+128 {
		t.Fatalf("%d Hz, %d channels, %d frames", d.SampleRate(), d.Channels(), d.Len())
	}
	buf := make([]float32, 2*d.Len())
	n := 0
	for n < len(buf) {
		m, err := d.Read(buf[n:])
		n += m
		if err != nil {
			break
		}
	}
	if n != len(buf) {
		t.Fatalf("decoded %d samples", n)
	}
	// Zero crossings of the left channel, 2 a period.
	zc := 0
	for i := 2; i < n; i += 2 {
		if (buf[i-2] < 0) != (buf[i] < 0) {
			zc++
		}
	}
	if f := float64(zc) / 2 / (float64(n/2) / 22050); math.Abs(f-440) > 5 {
		t.Errorf("decoded %v Hz", f)
	}
	s, err := audio.DecodeSound(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Duration(); got < 500*time.Millisecond || got > 510*time.Millisecond {
		t.Errorf("Duration = %v", got)
	}
	if err := d.SetPosition(5000); err != nil {
		t.Error(err)
	}
}
