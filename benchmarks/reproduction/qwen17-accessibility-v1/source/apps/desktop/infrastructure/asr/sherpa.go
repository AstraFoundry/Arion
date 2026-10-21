// Package asr adapts sherpa-onnx recognizers to the core recognition ports.
package asr

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// Kind is the sherpa model family of an artifact.
type Kind string

const (
	OnlineTransducer  Kind = "online-transducer"
	OnlineParaformer  Kind = "online-paraformer"
	OfflineTransducer Kind = "offline-transducer"
	OfflineSenseVoice Kind = "offline-sense-voice"
	OfflineParaformer Kind = "offline-paraformer"
	OfflineQwen3ASR   Kind = "offline-qwen3-asr"
)

// Spec locates one verified artifact's files inside its cache directory.
type Spec struct {
	Kind         Kind
	Directory    string
	Files        map[string]string
	ModelingUnit string
	SampleRate   int
	Threads      int
	// EndpointSilence is the trailing silence that closes an internal
	// segment; it never ends the user's session (SPH-04).
	EndpointSilence float32
}

func (s Spec) file(role string) string {
	name := s.Files[role]
	if name == "" {
		return ""
	}
	return filepath.Join(s.Directory, name)
}

// Streaming is a resident online recognizer shared by sessions.
type Streaming struct {
	mu         sync.Mutex
	recognizer *sherpa.OnlineRecognizer
	sampleRate int
}

// LoadStreaming loads an online model. A nil recognizer from the runtime is
// an explicit unsupported-model error.
func LoadStreaming(spec Spec) (*Streaming, error) {
	config := sherpa.OnlineRecognizerConfig{
		FeatConfig:     sherpa.FeatureConfig{SampleRate: spec.SampleRate, FeatureDim: 80},
		DecodingMethod: "greedy_search",
		EnableEndpoint: 1,
		// Rule 1 closes long silence without speech; rule 2 closes a segment
		// after speech; rule 3 bounds segment length for revision stability.
		Rule1MinTrailingSilence: 2.4,
		Rule2MinTrailingSilence: spec.EndpointSilence,
		Rule3MinUtteranceLength: 20,
	}
	config.ModelConfig.Tokens = spec.file("tokens")
	config.ModelConfig.NumThreads = spec.Threads
	config.ModelConfig.Provider = "cpu"
	config.ModelConfig.ModelingUnit = spec.ModelingUnit
	switch spec.Kind {
	case OnlineTransducer:
		config.ModelConfig.Transducer = sherpa.OnlineTransducerModelConfig{Encoder: spec.file("encoder"), Decoder: spec.file("decoder"), Joiner: spec.file("joiner")}
	case OnlineParaformer:
		config.ModelConfig.Paraformer = sherpa.OnlineParaformerModelConfig{Encoder: spec.file("encoder"), Decoder: spec.file("decoder")}
	default:
		return nil, domain.Fail(domain.CodeModelUnsupported, "not a streaming kind")
	}
	recognizer := sherpa.NewOnlineRecognizer(&config)
	if recognizer == nil {
		return nil, domain.Fail(domain.CodeModelUnsupported, "runtime rejected streaming model")
	}
	return &Streaming{recognizer: recognizer, sampleRate: spec.SampleRate}, nil
}

// Close frees the resident model. Callers ensure no session still uses it.
func (s *Streaming) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recognizer != nil {
		sherpa.DeleteOnlineRecognizer(s.recognizer)
		s.recognizer = nil
	}
}

// NewRecognizer creates a session stream. It satisfies ports.RecognizerFactory.
func (s *Streaming) NewRecognizer(_ context.Context, id contracts.Identity) (ports.Recognizer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recognizer == nil {
		return nil, domain.Fail(domain.CodeModelMissing, "streaming model unloaded")
	}
	return &session{engine: s, id: id, stream: sherpa.NewOnlineStream(s.recognizer)}, nil
}

type session struct {
	engine   *Streaming
	id       contracts.Identity
	stream   *sherpa.OnlineStream
	segments []string
	revision uint64
	closed   bool
}

func (s *session) Accept(_ context.Context, frame contracts.Frame) error {
	if int(frame.SampleRate) != s.engine.sampleRate || frame.Channels != 1 {
		return domain.Fail(domain.CodeModelUnsupported, "frame format differs from model metadata")
	}
	s.engine.mu.Lock()
	defer s.engine.mu.Unlock()
	if s.closed || s.engine.recognizer == nil {
		return domain.Fail(domain.CodeModelMissing, "stream closed")
	}
	s.stream.AcceptWaveform(s.engine.sampleRate, frame.Samples)
	for s.engine.recognizer.IsReady(s.stream) {
		s.engine.recognizer.Decode(s.stream)
	}
	return nil
}

// Partial returns the whole session hypothesis: closed segments plus the
// current one. An endpoint closes the current segment and resets decoding.
func (s *session) Partial(context.Context) (contracts.Draft, error) {
	s.engine.mu.Lock()
	defer s.engine.mu.Unlock()
	if s.closed || s.engine.recognizer == nil {
		return contracts.Draft{}, domain.Fail(domain.CodeModelMissing, "stream closed")
	}
	current := strings.TrimSpace(s.engine.recognizer.GetResult(s.stream).Text)
	if s.engine.recognizer.IsEndpoint(s.stream) {
		if current != "" {
			s.segments = append(s.segments, current)
		}
		current = ""
		s.engine.recognizer.Reset(s.stream)
	}
	return s.draft(current, false), nil
}

// draft reports closed segments separately from the open one, so a live
// editor can treat closed text as stable.
func (s *session) draft(current string, final bool) contracts.Draft {
	s.revision++
	segments := append([]string(nil), s.segments...)
	parts := append(append([]string(nil), segments...), current)
	return contracts.Draft{Identity: s.id, Revision: s.revision, Text: JoinSegments(parts), Final: final, Segments: segments, Tail: current}
}

// Finish flushes the model's right context with trailing silence so the
// last syllable is decoded once, then returns the final hypothesis (SPH-A05).
func (s *session) Finish(context.Context) (contracts.Draft, error) {
	s.engine.mu.Lock()
	defer s.engine.mu.Unlock()
	if s.closed || s.engine.recognizer == nil {
		return contracts.Draft{}, domain.Fail(domain.CodeModelMissing, "stream closed")
	}
	s.stream.AcceptWaveform(s.engine.sampleRate, make([]float32, s.engine.sampleRate*4/10))
	s.stream.InputFinished()
	for s.engine.recognizer.IsReady(s.stream) {
		s.engine.recognizer.Decode(s.stream)
	}
	current := strings.TrimSpace(s.engine.recognizer.GetResult(s.stream).Text)
	return s.draft(current, true), nil
}

func (s *session) Close() error {
	s.engine.mu.Lock()
	defer s.engine.mu.Unlock()
	if !s.closed {
		s.closed = true
		sherpa.DeleteOnlineStream(s.stream)
	}
	return nil
}

// JoinSegments concatenates recognizer segments (see domain.JoinText).
func JoinSegments(parts []string) string { return domain.JoinText(parts...) }

// Offline is a resident non-streaming recognizer used for the acoustic
// second pass over retained session audio.
type Offline struct {
	mu         sync.Mutex
	recognizer *sherpa.OfflineRecognizer
	sampleRate int
}

func LoadOffline(spec Spec) (*Offline, error) {
	config := sherpa.OfflineRecognizerConfig{
		FeatConfig:     sherpa.FeatureConfig{SampleRate: spec.SampleRate, FeatureDim: 80},
		DecodingMethod: "greedy_search",
	}
	config.ModelConfig.Tokens = spec.file("tokens")
	config.ModelConfig.NumThreads = spec.Threads
	config.ModelConfig.Provider = "cpu"
	config.ModelConfig.ModelingUnit = spec.ModelingUnit
	switch spec.Kind {
	case OfflineSenseVoice:
		config.ModelConfig.SenseVoice = sherpa.OfflineSenseVoiceModelConfig{Model: spec.file("model"), Language: "auto", UseInverseTextNormalization: 1}
	case OfflineTransducer:
		config.ModelConfig.Transducer = sherpa.OfflineTransducerModelConfig{Encoder: spec.file("encoder"), Decoder: spec.file("decoder"), Joiner: spec.file("joiner")}
	case OfflineParaformer:
		config.ModelConfig.Paraformer = sherpa.OfflineParaformerModelConfig{Model: spec.file("model")}
	case OfflineQwen3ASR:
		// The tokenizer files are installed flat beside the graphs, so the
		// artifact directory is the tokenizer directory. Greedy decoding;
		// 128 new tokens bound one segment.
		config.ModelConfig.Qwen3ASR = sherpa.OfflineQwen3ASRModelConfig{ConvFrontend: spec.file("conv_frontend"), Encoder: spec.file("encoder"),
			Decoder: spec.file("decoder"), Tokenizer: spec.Directory, MaxTotalLen: 512, MaxNewTokens: 128, Temperature: 1e-6, TopP: 1, Seed: 11}
	default:
		return nil, domain.Fail(domain.CodeModelUnsupported, "not an offline kind")
	}
	recognizer := sherpa.NewOfflineRecognizer(&config)
	if recognizer == nil {
		return nil, domain.Fail(domain.CodeModelUnsupported, "runtime rejected offline model")
	}
	return &Offline{recognizer: recognizer, sampleRate: spec.SampleRate}, nil
}

func (o *Offline) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.recognizer != nil {
		sherpa.DeleteOfflineRecognizer(o.recognizer)
		o.recognizer = nil
	}
}

// maxSegmentSeconds bounds one offline decode; longer audio is split at the
// quietest point near the bound.
const maxSegmentSeconds = 25

// Finalize satisfies ports.Finalizer.
func (o *Offline) Finalize(ctx context.Context, request ports.FinalizeRequest) (string, error) {
	if request.SampleRate != o.sampleRate {
		return "", domain.Fail(domain.CodeModelUnsupported, "sample rate differs from model metadata")
	}
	var parts []string
	for _, segment := range SplitAtQuiet(request.Samples, o.sampleRate, maxSegmentSeconds) {
		if ctx.Err() != nil {
			return "", domain.Fail(domain.CodeCancelled, "finalize cancelled")
		}
		text, err := o.decode(segment)
		if err != nil {
			return "", err
		}
		parts = append(parts, text)
	}
	return JoinSegments(parts), nil
}

func (o *Offline) decode(samples []float32) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.recognizer == nil {
		return "", domain.Fail(domain.CodeModelMissing, "offline model unloaded")
	}
	stream := sherpa.NewOfflineStream(o.recognizer)
	defer sherpa.DeleteOfflineStream(stream)
	stream.AcceptWaveform(o.sampleRate, samples)
	o.recognizer.Decode(stream)
	return strings.TrimSpace(stream.GetResult().Text), nil
}

// SplitAtQuiet cuts audio into segments no longer than limit seconds,
// choosing the lowest-energy 20 ms window in the last third of each span.
func SplitAtQuiet(samples []float32, rate, limit int) [][]float32 {
	window := rate / 50
	max := rate * limit
	var out [][]float32
	for len(samples) > max {
		best, bestEnergy := max, float32(-1)
		for start := max * 2 / 3; start+window <= max; start += window {
			var energy float32
			for _, s := range samples[start : start+window] {
				energy += s * s
			}
			if bestEnergy < 0 || energy < bestEnergy {
				best, bestEnergy = start+window/2, energy
			}
		}
		out = append(out, samples[:best])
		samples = samples[best:]
	}
	if len(samples) > 0 {
		out = append(out, samples)
	}
	return out
}
