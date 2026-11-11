package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/wait"
)

// setup mixes by hand, with nothing playing.
func setup(t *testing.T) {
	t.Helper()
	mu.Lock()
	testMode = true
	for _, v := range voices {
		v.active = false
	}
	voices = nil
	volume = 1
	held, unheard = 0, 0
	mu.Unlock()
}

// hold makes the output hold n frames ahead of what is heard, as a device
// that just opened.
func hold(n int64) {
	mu.Lock()
	defer mu.Unlock()
	held, unheard = n, 0
}

// render mixes n frames.
func render(n int) []float32 {
	out := make([]float32, n*channels)
	for i := 0; i < n; i += 256 {
		mix(out[i*channels : min(n, i+256)*channels])
	}
	return out
}

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-4 }

func ramp(n int) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = float32(i%100) / 200
	}
	return s
}

func TestSoundPlaysAsDecoded(t *testing.T) {
	setup(t)
	samples := ramp(1000)
	p := NewSound(sampleRate, 1, samples).Play()
	out := render(1200)
	for i, s := range samples {
		if !near(out[2*i], s) || !near(out[2*i+1], s) {
			t.Fatalf("frame %d = %v %v, want %v", i, out[2*i], out[2*i+1], s)
		}
	}
	for i := 1000; i < 1200; i++ {
		if out[2*i] != 0 {
			t.Fatalf("frame %d after the end = %v", i, out[2*i])
		}
	}
	if p.Playing() {
		t.Error("still playing after its end")
	}
	if got := p.Position(); got != p.Duration() {
		t.Errorf("Position = %v at the end, want %v", got, p.Duration())
	}
	if len(voices) != 0 {
		t.Errorf("%d voices left", len(voices))
	}
}

func TestStereoKeepsChannels(t *testing.T) {
	setup(t)
	NewSound(sampleRate, 2, []float32{0.1, 0.2, 0.3, 0.4}).Play()
	out := render(4)
	if !near(out[0], 0.1) || !near(out[1], 0.2) || !near(out[2], 0.3) || !near(out[3], 0.4) {
		t.Errorf("out = %v", out)
	}
}

func TestPlayAgainAfterEnd(t *testing.T) {
	setup(t)
	p := NewSound(sampleRate, 1, ramp(100)).Play()
	render(200)
	p.Play()
	if !p.Playing() {
		t.Fatal("Play at the end did not start again")
	}
	out := render(100)
	if !near(out[2*50], 0.25) {
		t.Errorf("frame 50 = %v, want 0.25", out[2*50])
	}
}

func TestOnEnd(t *testing.T) {
	setup(t)
	p := NewSound(sampleRate, 1, ramp(100)).NewPlayer()
	ended := make(chan struct{}, 4)
	off := p.OnEnd(func() { ended <- struct{}{} })
	p.Play()
	render(200)
	wait.For(t, "the end", ended)
	render(200) // ended once
	p.Play()
	render(50)
	p.Pause()
	render(200) // paused, not at the end
	p.Seek(0)
	off()
	p.Play()
	render(200)
	if n := len(ended); n != 0 {
		t.Errorf("%d more ends, want 0", n)
	}

	loop := NewSound(sampleRate, 1, ramp(100)).NewPlayer().SetLoop(true)
	loop.OnEnd(func() { ended <- struct{}{} })
	loop.Play()
	render(350)
	loop.Stop()
	render(200)
	if n := len(ended); n != 0 {
		t.Errorf("a loop or a stopped player ended %d times", n)
	}
}

func TestSoundsOverlap(t *testing.T) {
	setup(t)
	s := NewSound(sampleRate, 1, []float32{0.25, 0.25, 0.25, 0.25})
	s.Play()
	s.Play()
	out := render(2)
	if !near(out[0], 0.5) {
		t.Errorf("two players mix to %v, want 0.5", out[0])
	}
}

func TestClipping(t *testing.T) {
	setup(t)
	s := NewSound(sampleRate, 1, []float32{0.8, 0.8})
	s.Play()
	s.Play()
	if out := render(1); out[0] != 1 {
		t.Errorf("out = %v, want clipped to 1", out[0])
	}
}

func TestLoop(t *testing.T) {
	setup(t)
	samples := ramp(100)
	p := NewSound(sampleRate, 1, samples).NewPlayer().SetLoop(true)
	p.Play()
	out := render(350)
	for i := range 350 {
		if want := samples[i%100]; !near(out[2*i], want) {
			t.Fatalf("frame %d = %v, want %v", i, out[2*i], want)
		}
	}
	if !p.Playing() {
		t.Error("a loop stopped")
	}
	if got, want := p.Position(), frames(50, sampleRate); got != want {
		t.Errorf("Position = %v, want %v", got, want)
	}
}

func TestPauseFadesOutAndResumes(t *testing.T) {
	setup(t)
	samples := make([]float32, 10000)
	for i := range samples {
		samples[i] = 0.5
	}
	p := NewSound(sampleRate, 1, samples).Play()
	render(1000)
	p.Pause()
	if p.Playing() {
		t.Error("Playing after Pause")
	}
	out := render(rampFrames + 10)
	if !(out[0] > 0.45 && out[2*(rampFrames/2)] < 0.3 && out[2*(rampFrames/2)] > 0.2) {
		t.Errorf("fade out = %v, %v", out[0], out[2*(rampFrames/2)])
	}
	if out[2*(rampFrames+5)] != 0 || len(voices) != 0 {
		t.Errorf("after the fade: %v, %d voices", out[2*(rampFrames+5)], len(voices))
	}
	at := p.Position()
	render(500)
	if p.Position() != at {
		t.Error("a paused player moved")
	}
	p.Play()
	out = render(rampFrames + 10)
	if !(out[0] < 0.05 && near(out[2*(rampFrames+5)], 0.5)) {
		t.Errorf("fade in = %v … %v", out[0], out[2*(rampFrames+5)])
	}
}

func TestVolumeRamps(t *testing.T) {
	setup(t)
	samples := make([]float32, 10000)
	for i := range samples {
		samples[i] = 0.5
	}
	p := NewSound(sampleRate, 1, samples).Play()
	render(10)
	p.SetVolume(0.5)
	out := render(rampFrames * 2)
	if !(out[0] > 0.45 && near(out[2*(rampFrames+10)], 0.25)) {
		t.Errorf("volume ramp = %v … %v", out[0], out[2*(rampFrames+10)])
	}
	SetVolume(0)
	out = render(rampFrames * 2)
	if out[2*(rampFrames+10)] != 0 {
		t.Errorf("master volume 0 = %v", out[2*(rampFrames+10)])
	}
}

func TestPan(t *testing.T) {
	setup(t)
	mono := NewSound(sampleRate, 1, []float32{0.5, 0.5, 0.5})
	mono.NewPlayer().SetPan(-1).Play()
	out := render(2)
	if !near(out[0], 0.5*math.Sqrt2) || !near(out[1], 0) {
		t.Errorf("mono panned left = %v %v", out[0], out[1])
	}
	setup(t)
	stereo := NewSound(sampleRate, 2, []float32{0.2, 0.4, 0.2, 0.4})
	stereo.NewPlayer().SetPan(1).Play()
	out = render(1)
	if !near(out[0], 0) || !near(out[1], 0.6) {
		t.Errorf("stereo panned right = %v %v", out[0], out[1])
	}
}

func TestSpeed(t *testing.T) {
	setup(t)
	p := NewSound(sampleRate, 1, ramp(1000)).NewPlayer().SetSpeed(2)
	p.Play()
	out := render(100)
	if !near(out[2*10], ramp(1000)[20]) {
		t.Errorf("frame 10 at speed 2 = %v, want %v", out[2*10], ramp(1000)[20])
	}
	if got, want := p.Position(), frames(200, sampleRate); got != want {
		t.Errorf("Position = %v, want %v", got, want)
	}
}

func TestResamples(t *testing.T) {
	setup(t)
	// A 1 kHz sine at 24 kHz plays at 1 kHz.
	const rate = 24000
	samples := make([]float32, rate/10)
	for i := range samples {
		samples[i] = float32(math.Sin(2 * math.Pi * 1000 * float64(i) / rate))
	}
	p := NewSound(rate, 1, samples).Play()
	out := render(sampleRate / 20)
	for i := 100; i < sampleRate/20; i++ {
		want := math.Sin(2 * math.Pi * 1000 * float64(i) / sampleRate)
		if math.Abs(float64(out[2*i])-want) > 0.01 {
			t.Fatalf("frame %d = %v, want %v", i, out[2*i], want)
		}
	}
	if got, want := p.Position(), 50*time.Millisecond; got != want {
		t.Errorf("Position = %v, want %v", got, want)
	}
}

func TestSeek(t *testing.T) {
	setup(t)
	samples := ramp(48000)
	p := NewSound(sampleRate, 1, samples).NewPlayer()
	p.Seek(500 * time.Millisecond)
	if got := p.Position(); got != 500*time.Millisecond {
		t.Errorf("Position = %v after Seek", got)
	}
	p.Play()
	out := render(rampFrames + 20)
	if i := rampFrames + 10; !near(out[2*i], samples[24000+i]) {
		t.Errorf("frame %d after seeking = %v, want %v", i, out[2*i], samples[24000+i])
	}
	p.Stop()
	if p.Playing() || p.Position() != 0 {
		t.Errorf("after Stop: playing %v at %v", p.Playing(), p.Position())
	}
}

func TestClose(t *testing.T) {
	setup(t)
	p := NewSound(sampleRate, 1, ramp(1000)).Play()
	p.Close()
	p.Play()
	if p.Playing() {
		t.Error("a closed player plays")
	}
	render(10)
	if len(voices) != 0 {
		t.Errorf("%d voices after Close", len(voices))
	}
}

// genDecoder generates frames i/1000 of a channel each, with a length.
type genDecoder struct {
	ch     int
	pos    int64
	length int64
	closed atomic.Bool
}

func (d *genDecoder) SampleRate() int { return sampleRate }
func (d *genDecoder) Channels() int   { return d.ch }
func (d *genDecoder) Len() int64      { return d.length }
func (d *genDecoder) Close() error    { d.closed.Store(true); return nil }

func (d *genDecoder) Read(buf []float32) (int, error) {
	n := 0
	for ; n+d.ch <= len(buf) && d.pos < d.length; n += d.ch {
		for c := range d.ch {
			buf[n+c] = float32(d.pos%1000)/1000 + float32(c)
		}
		d.pos++
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (d *genDecoder) SetPosition(f int64) error {
	if f > d.length {
		return errors.New("out of range")
	}
	d.pos = f
	return nil
}

func waitDecoded(t *testing.T, p *Player) {
	t.Helper()
	s := p.v.src.(*stream)
	for range 1000 {
		s.mu.Lock()
		n := s.n
		s.mu.Unlock()
		if n > 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("the stream decoded nothing")
}

func TestStream(t *testing.T) {
	setup(t)
	d := &genDecoder{ch: 3, length: 5000}
	p := NewPlayer(d)
	if p.Duration() != frames(5000, sampleRate) {
		t.Errorf("Duration = %v", p.Duration())
	}
	waitDecoded(t, p)
	p.Play()
	out := render(1000)
	for i := range 1000 {
		if l, r := out[2*i], out[2*i+1]; !near(l, float32(i)/1000) || !near(r, float32(i)/1000+1) && r != 1 {
			t.Fatalf("frame %d = %v %v", i, l, r)
		}
	}
	p.Close()
	for range 1000 {
		if d.closed.Load() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Error("Close did not close the decoder")
}

func TestStreamLoopsWithoutGap(t *testing.T) {
	setup(t)
	d := &genDecoder{ch: 1, length: 300}
	p := NewPlayer(d).SetLoop(true)
	waitDecoded(t, p)
	p.Play()
	out := make([]float32, 0, 2000*channels)
	for len(out) < 2000*channels {
		chunk := make([]float32, 64*channels)
		// The decoder keeps a second ahead: wait for it, as the device
		// would not.
		for {
			s := p.v.src.(*stream)
			s.mu.Lock()
			n := s.n
			s.mu.Unlock()
			if n >= 64 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		mix(chunk)
		out = append(out, chunk...)
	}
	for i := range 2000 {
		if want := float32(i%300) / 1000; !near(out[2*i], want) {
			t.Fatalf("frame %d = %v, want %v", i, out[2*i], want)
		}
	}
	p.Close()
}

func TestStreamSeek(t *testing.T) {
	setup(t)
	d := &genDecoder{ch: 1, length: 48000}
	p := NewPlayer(d)
	p.Seek(frames(700, sampleRate))
	waitDecoded(t, p)
	p.Play()
	out := render(rampFrames + 20)
	if i := rampFrames + 10; !near(out[2*i], float32((700+i)%1000)/1000) {
		t.Errorf("frame %d = %v, want %v", i, out[2*i], float32((700+i)%1000)/1000)
	}
	p.Close()
}

func TestOrphanedStreamPlaysToItsEnd(t *testing.T) {
	setup(t)
	d := &genDecoder{ch: 1, length: 500}
	v := NewPlayer(d).v
	for range 1000 {
		s := v.src.(*stream)
		s.mu.Lock()
		n := s.n
		s.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	mu.Lock()
	v.playing, v.active, v.snap = true, true, true
	voices = append(voices, v)
	v.orphan = true
	mu.Unlock()
	render(600)
	if !v.closed {
		t.Error("an orphaned stream was not closed at its end")
	}
}

func wav(tag, bits, ch int, data []byte) []byte {
	var b bytes.Buffer
	w := func(v any) { binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	w(uint32(36 + len(data) + 10))
	b.WriteString("WAVE")
	b.WriteString("LIST")
	w(uint32(1)) // odd, padded
	b.Write([]byte{0, 0})
	b.WriteString("fmt ")
	w(uint32(16))
	w(uint16(tag))
	w(uint16(ch))
	w(uint32(22050))
	w(uint32(22050 * ch * bits / 8))
	w(uint16(ch * bits / 8))
	w(uint16(bits))
	b.WriteString("data")
	w(uint32(len(data)))
	b.Write(data)
	return b.Bytes()
}

func TestWAV(t *testing.T) {
	le := binary.LittleEndian
	for _, tc := range []struct {
		name      string
		tag, bits int
		data      []byte
	}{
		{"u8", 1, 8, []byte{128, 192, 64, 0}},
		{"s16", 1, 16, le.AppendUint16(le.AppendUint16(le.AppendUint16(le.AppendUint16(nil, 0), 0x4000), 0xc000), 0x8000)},
		{"s24", 1, 24, []byte{0, 0, 0, 0, 0, 0x40, 0, 0, 0xc0, 0, 0, 0x80}},
		{"s32", 1, 32, le.AppendUint32(le.AppendUint32(le.AppendUint32(le.AppendUint32(nil, 0), 0x40000000), 0xc0000000), 0x80000000)},
		{"f32", 3, 32, le.AppendUint32(le.AppendUint32(le.AppendUint32(le.AppendUint32(nil, 0), math.Float32bits(0.5)), math.Float32bits(-0.5)), math.Float32bits(-1))},
		{"f64", 3, 64, le.AppendUint64(le.AppendUint64(le.AppendUint64(le.AppendUint64(nil, 0), math.Float64bits(0.5)), math.Float64bits(-0.5)), math.Float64bits(-1))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := DecodeSound(wav(tc.tag, tc.bits, 2, tc.data))
			if err != nil {
				t.Fatal(err)
			}
			want := []float32{0, 0.5, -0.5, -1}
			if s.rate != 22050 || s.channels != 2 || len(s.samples) != 4 {
				t.Fatalf("decoded %d Hz, %d channels, %d samples", s.rate, s.channels, len(s.samples))
			}
			for i := range want {
				if s.samples[i] != want[i] {
					t.Errorf("sample %d = %v, want %v", i, s.samples[i], want[i])
				}
			}
			if s.Duration() != frames(2, 22050) {
				t.Errorf("Duration = %v", s.Duration())
			}
		})
	}
}

func TestWAVSeek(t *testing.T) {
	data := make([]byte, 0, 200)
	for i := range 100 {
		data = binary.LittleEndian.AppendUint16(data, uint16(i*100))
	}
	d, name, err := NewDecoder(bytes.NewReader(wav(1, 16, 1, data)))
	if err != nil || name != "wav" {
		t.Fatal(name, err)
	}
	if d.Len() != 100 {
		t.Errorf("Len = %d", d.Len())
	}
	if err := d.SetPosition(40); err != nil {
		t.Fatal(err)
	}
	buf := make([]float32, 4)
	if n, err := d.Read(buf); n != 4 || err != nil || buf[0] != float32(4000)/(1<<15) {
		t.Errorf("after SetPosition(40): %d %v %v", n, err, buf)
	}
}

func TestUnknownFormat(t *testing.T) {
	if _, err := DecodeSound([]byte("not audio at all")); err != ErrFormat {
		t.Errorf("err = %v, want ErrFormat", err)
	}
}

// The output plays on time where no device opens, and says why.
func TestNullOutput(t *testing.T) {
	setup(t)
	mu.Lock()
	testMode = false
	mu.Unlock()
	defer func() {
		mu.Lock()
		testMode = true
		mu.Unlock()
	}()
	failure := errors.New("no device here")
	openOutput = func(time.Duration) (device, int, error) { return nil, 0, failure }
	defer func() { openOutput = openDevice }()
	p := NewSound(sampleRate, 1, make([]float32, sampleRate/10)).Play()
	// How long it takes is TestNullDeviceKeepsTime's: a minute only
	// guards against a hang.
	for start := time.Now(); p.Playing(); time.Sleep(5 * time.Millisecond) {
		if time.Since(start) > time.Minute {
			t.Fatal("100 ms still playing after a minute")
		}
	}
	if Err() != failure {
		t.Errorf("Err = %v", Err())
	}
}

// Setting the latency opens the output again, to hold as far ahead as it
// says.
func TestSetLatencyReopens(t *testing.T) {
	setup(t)
	opened := make(chan time.Duration, 8)
	openOutput = func(ahead time.Duration) (device, int, error) {
		opened <- ahead
		return pacedDevice{}, nullFrames, nil
	}
	defer func() { openOutput = openDevice }()
	mu.Lock()
	testMode = false
	mu.Unlock()
	defer func() {
		mu.Lock()
		testMode = true
		latency = LatencyInteractive
		mu.Unlock()
	}()
	p := NewSound(sampleRate, 1, make([]float32, sampleRate)).NewPlayer().SetLoop(true)
	p.Play()
	defer p.Close()
	if got := wait.For(t, "the output to open", opened); got != 40*time.Millisecond {
		t.Errorf("opened %v ahead, want 40ms", got)
	}
	SetLatency(LatencyPlayback)
	if got := wait.For(t, "the output to open again", opened); got != 250*time.Millisecond {
		t.Errorf("opened again %v ahead, want 250ms", got)
	}
}

// A pacedDevice takes what it is given a little at a time.
type pacedDevice struct{}

func (pacedDevice) write([]float32) error { time.Sleep(time.Millisecond); return nil }
func (pacedDevice) close()                {}

// A null device takes as long as what it plays would, on a clock of its
// own here, does not hurry to catch up once behind, and fails after a
// while for the device to be tried again.
func TestNullDeviceKeepsTime(t *testing.T) {
	start := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	clock := start
	d := newNullDevice(true)
	d.next, d.retry = start, start.Add(2*time.Second)
	d.now = func() time.Time { return clock }
	d.sleep = func(dt time.Duration) { clock = clock.Add(max(dt, 0)) }
	chunk := make([]float32, nullFrames*channels) // 10 ms
	for range 10 {
		if err := d.write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if got := clock.Sub(start); got != 100*time.Millisecond {
		t.Errorf("100 ms played in %v", got)
	}
	// Late by a second, as after the computer slept: the next chunk takes
	// its 10 ms from now.
	clock = clock.Add(time.Second)
	late := clock
	if err := d.write(chunk); err != nil {
		t.Fatal(err)
	}
	if got := clock.Sub(late); got != 10*time.Millisecond {
		t.Errorf("10 ms played late in %v", got)
	}
	clock = start.Add(2*time.Second + time.Millisecond)
	if err := d.write(chunk); err != errRetry {
		t.Errorf("after 2 s: %v, want errRetry", err)
	}
}
