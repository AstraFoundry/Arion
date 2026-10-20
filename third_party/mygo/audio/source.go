package audio

import (
	"io"
	"sync"
)

// A memSource reads a sound's samples.
type memSource struct {
	samples  []float32
	channels int
	pos      int // sample
	loop     bool
}

func (s *memSource) read(dst []float32) (int, bool) {
	n := 0
	for n < len(dst) {
		if s.pos >= len(s.samples) {
			if !s.loop || len(s.samples) == 0 {
				return n / s.channels, true
			}
			s.pos = 0
		}
		c := copy(dst[n:], s.samples[s.pos:])
		c -= c % s.channels
		if c == 0 {
			break
		}
		n += c
		s.pos += c
	}
	return n / s.channels, false
}

func (s *memSource) seek(frame int64) {
	s.pos = int(min(frame*int64(s.channels), int64(len(s.samples))))
}

func (s *memSource) setLoop(loop bool) { s.loop = loop }
func (s *memSource) close()            {}

// A stream decodes ahead of a voice on a goroutine of its own, into a ring
// the voice reads without waiting.
type stream struct {
	d        Decoder
	channels int // of d
	keep     int // channels kept: 1 or 2

	mu     sync.Mutex
	cond   sync.Cond
	ring   []float32 // frames of keep channels
	r, n   int       // where the next sample is read, and how many there are
	loop   bool
	atEOF  bool  // d reached its end
	seekTo int64 // where to seek d next, or -1
	gen    int   // counts seeks, to drop what was decoded before one
	closed bool
}

func newStream(d Decoder) *stream {
	ch := max(d.Channels(), 1)
	s := &stream{d: d, channels: ch, keep: min(ch, 2), seekTo: -1}
	s.cond.L = &s.mu
	// About a second ahead.
	frames := max(d.SampleRate(), 4096)
	s.ring = make([]float32, frames*s.keep)
	go s.decode()
	return s
}

func (s *stream) decode() {
	buf := make([]float32, 2048*s.channels)
	var kept []float32
	if s.keep != s.channels {
		kept = make([]float32, 2048*s.keep)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		for !s.closed && s.seekTo < 0 && (s.atEOF && !s.loop || len(s.ring)-s.n < 2048*s.keep) {
			s.cond.Wait()
		}
		if s.closed {
			if c, ok := s.d.(io.Closer); ok {
				c.Close()
			}
			return
		}
		if s.seekTo >= 0 || s.atEOF {
			// A seek, or a loop back to the start.
			to := max(s.seekTo, 0)
			s.seekTo = -1
			gen := s.gen
			s.mu.Unlock()
			err := s.d.SetPosition(to)
			s.mu.Lock()
			if gen == s.gen {
				// Where it cannot go, the stream ends.
				s.atEOF = err != nil
				s.loop = s.loop && err == nil
			}
			continue
		}
		gen := s.gen
		s.mu.Unlock()
		n, err := s.d.Read(buf)
		n -= n % s.channels
		samples := buf[:n]
		if kept != nil {
			k := 0
			for i := 0; i < n; i += s.channels {
				k += copy(kept[k:], buf[i:i+s.keep])
			}
			samples = kept[:k]
		}
		s.mu.Lock()
		if gen != s.gen {
			// Seeked meanwhile.
			continue
		}
		for len(samples) > 0 {
			w := (s.r + s.n) % len(s.ring)
			c := copy(s.ring[w:min(len(s.ring), w+len(s.ring)-s.n)], samples)
			s.n += c
			samples = samples[c:]
		}
		if err != nil || n == 0 {
			// The end, or an error, which ends the stream as well.
			s.atEOF = true
			if err != io.EOF && err != nil || n == 0 && err == nil {
				// A decoder that fails or makes no progress loops no more.
				s.loop = false
			}
		}
	}
}

func (s *stream) read(dst []float32) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	want := min(len(dst), s.n)
	want -= want % s.keep
	got := 0
	for got < want {
		c := copy(dst[got:want], s.ring[s.r:])
		got += c
		s.r = (s.r + c) % len(s.ring)
		s.n -= c
	}
	if got > 0 {
		s.cond.Signal()
	}
	ended := got < len(dst)/s.keep*s.keep && s.n == 0 && s.atEOF && !s.loop && s.seekTo < 0
	return got / s.keep, ended
}

func (s *stream) seek(frame int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seekTo = frame
	s.atEOF = false
	s.r, s.n = 0, 0
	s.gen++
	s.cond.Signal()
}

func (s *stream) setLoop(loop bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loop = loop
	s.cond.Signal()
}

func (s *stream) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.cond.Signal()
}
