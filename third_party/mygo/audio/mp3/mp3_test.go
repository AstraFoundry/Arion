package mp3

import (
	"bytes"
	"math"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/audio"
)

// tone.mp3 is 0.5 s of a 440 Hz sine at 44.1 kHz, in mono, which decodes
// to stereo with the encoder's padding.
func TestDecode(t *testing.T) {
	data, err := os.ReadFile("testdata/tone.mp3")
	if err != nil {
		t.Fatal(err)
	}
	d, name, err := audio.NewDecoder(bytes.NewReader(data))
	if err != nil || name != "mp3" {
		t.Fatal(name, err)
	}
	if d.SampleRate() != 44100 || d.Channels() != 2 {
		t.Fatalf("%d Hz, %d channels", d.SampleRate(), d.Channels())
	}
	// go-mp3 keeps the encoder's delay and padding, a few frames of 1152.
	if n := d.Len(); n < 22050 || n > 22050+4*1152 {
		t.Errorf("Len = %d", n)
	}
	s, err := audio.DecodeSound(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Duration(); got < 500*time.Millisecond || got > 610*time.Millisecond {
		t.Errorf("Duration = %v", got)
	}
	if err := d.SetPosition(10000); err != nil {
		t.Fatal(err)
	}
	buf := make([]float32, 2*8820)
	n := 0
	for n < len(buf) {
		m, err := d.Read(buf[n:])
		n += m
		if err != nil {
			break
		}
	}
	zc := 0
	for i := 2; i < n; i += 2 {
		if (buf[i-2] < 0) != (buf[i] < 0) {
			zc++
		}
	}
	if f := float64(zc) / 2 / (float64(n/2) / 44100); math.Abs(f-440) > 5 {
		t.Errorf("decoded %v Hz from frame 10000", f)
	}
}
