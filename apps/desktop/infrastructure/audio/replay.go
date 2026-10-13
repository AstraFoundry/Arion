package audio

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// ReadWAV decodes a 16-bit PCM or 32-bit float mono WAV file at 16 kHz.
// Other formats are rejected explicitly; fixtures are converted beforehand.
func ReadWAV(path string) ([]float32, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var riff [12]byte
	if _, err := io.ReadFull(file, riff[:]); err != nil || string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a RIFF/WAVE file")
	}
	var format, channels, bits uint16
	var rate uint32
	for {
		var header [8]byte
		if _, err := io.ReadFull(file, header[:]); err != nil {
			return nil, fmt.Errorf("missing data chunk")
		}
		size := binary.LittleEndian.Uint32(header[4:])
		body := make([]byte, size+size%2)
		if _, err := io.ReadFull(file, body); err != nil && !(string(header[:4]) == "data" && err == io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("truncated chunk")
		}
		switch string(header[:4]) {
		case "fmt ":
			format = binary.LittleEndian.Uint16(body[0:])
			channels = binary.LittleEndian.Uint16(body[2:])
			rate = binary.LittleEndian.Uint32(body[4:])
			bits = binary.LittleEndian.Uint16(body[14:])
		case "data":
			if channels != 1 || rate != SampleRate {
				return nil, fmt.Errorf("fixture must be mono %d Hz", SampleRate)
			}
			body = body[:size]
			switch {
			case format == 1 && bits == 16:
				samples := make([]float32, len(body)/2)
				for i := range samples {
					samples[i] = float32(int16(binary.LittleEndian.Uint16(body[i*2:]))) / 32768
				}
				return samples, nil
			case format == 3 && bits == 32:
				samples := make([]float32, len(body)/4)
				for i := range samples {
					samples[i] = math.Float32frombits(binary.LittleEndian.Uint32(body[i*4:]))
				}
				return samples, nil
			default:
				return nil, fmt.Errorf("unsupported WAV encoding")
			}
		}
	}
}

// Replay feeds fixture samples in fixed chunks on a monotonic realtime
// schedule. It is an evaluation and E2E source, never a microphone substitute
// in the product.
type Replay struct {
	Samples []float32
	ChunkMs int

	mu      sync.Mutex
	started time.Time
	done    chan struct{}
}

// Started is the monotonic time the first chunk was delivered.
func (r *Replay) Started() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.started
}

// Done closes when replay ends: after the final chunk, or when stopped.
func (r *Replay) Done() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.done
}

type replayStream struct {
	cancel context.CancelFunc
	wait   chan struct{}
}

func (s *replayStream) Stop() error {
	s.cancel()
	<-s.wait
	return nil
}

func (r *Replay) Start(ctx context.Context, id contracts.Identity, sink ports.FrameSink) (ports.AudioStream, error) {
	if r.ChunkMs <= 0 || len(r.Samples) == 0 {
		return nil, domain.Fail(domain.CodeDeviceUnavailable, "empty replay fixture")
	}
	chunk := SampleRate * r.ChunkMs / 1000
	runCtx, cancel := context.WithCancel(ctx)
	stream := &replayStream{cancel: cancel, wait: make(chan struct{})}
	r.mu.Lock()
	r.done = make(chan struct{})
	r.started = time.Now()
	done, start := r.done, r.started
	r.mu.Unlock()
	go func() {
		defer close(stream.wait)
		finished := false
		defer func() {
			if !finished {
				close(done)
			}
		}()
		var sequence uint64
		for offset := 0; offset < len(r.Samples); offset += chunk {
			end := min(offset+chunk, len(r.Samples))
			// Deliver a chunk once its last sample has "arrived" in real time.
			due := start.Add(time.Duration(end) * time.Second / SampleRate)
			select {
			case <-runCtx.Done():
				return
			case <-time.After(time.Until(due)):
			}
			sink(contracts.Frame{Identity: id, Sequence: sequence, SampleOffset: uint64(offset), SampleRate: SampleRate, Channels: 1,
				Samples: append([]float32(nil), r.Samples[offset:end]...)})
			sequence++
		}
		close(done)
		finished = true
		<-runCtx.Done()
	}()
	return stream, nil
}
