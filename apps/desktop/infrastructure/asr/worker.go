// Package asr adapts resident native Qwen3-ASR workers to recognition ports.
package asr

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"arion.local/core/domain"
)

type Kind string

const (
	QwenMLX  Kind = "qwen3-asr-mlx"
	QwenGGUF Kind = "qwen3-asr-gguf"
)

type Spec struct {
	Kind            Kind
	Directory       string
	Files           map[string]string
	SampleRate      int
	Threads         int
	EndpointSilence float32
}

// Resources is set once by the host before model activation. Tests and CLI
// drivers may instead supply ARION_ASR_WORKER explicitly.
var Resources string

func Locate(resources string) string {
	name := workerName()
	for _, dir := range []string{filepath.Join(resources, "asr"), filepath.Join(resources, runtime.GOOS+"-"+runtime.GOARCH, "asr")} {
		file := filepath.Join(dir, name)
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			return file
		}
	}
	return filepath.Join(resources, "asr", name)
}

type message struct {
	Type         string   `json:"type"`
	ID           string   `json:"id,omitempty"`
	Text         string   `json:"text,omitempty"`
	Stop         string   `json:"stop,omitempty"`
	Protocol     int      `json:"protocol,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Model        struct {
		Options []workerOption `json:"options"`
	} `json:"model,omitempty"`
}

type workerOption struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Steers bool   `json:"steers"`
}

func (m message) supportsPrompt(kind Kind) bool {
	if kind == QwenMLX {
		return slices.Contains(m.Capabilities, "prompt")
	}
	for _, option := range m.Model.Options {
		if option.Name == "prompt" && option.Type == "string" && option.Steers {
			return true
		}
	}
	return false
}

// engine serializes requests across process generations. Only a worker retired
// after a cancellation drain timeout may be loaded again for a later request.
type engine struct {
	spec        Spec
	worker      string
	serial      chan struct{}
	lifetime    context.Context
	stop        context.CancelFunc
	cancelGrace time.Duration
	mu          sync.Mutex
	current     *workerProcess
	needsReload bool
	failure     error
	closed      bool
	once        sync.Once
}

// workerProcess owns one subprocess and its request stream. Progress is
// nonterminal; cancellation drains its terminal acknowledgement before reuse.
type workerProcess struct {
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	write           sync.Mutex
	mu              sync.Mutex
	pending         map[string]chan message
	ready           chan message
	done            chan struct{}
	next            atomic.Uint64
	promptSupported bool
	once            sync.Once
}

func load(ctx context.Context, spec Spec) (*engine, error) {
	if spec.SampleRate != 16000 || (spec.Kind != QwenMLX && spec.Kind != QwenGGUF) {
		return nil, domain.Fail(domain.CodeModelUnsupported, "unsupported native ASR specification")
	}
	worker := os.Getenv("ARION_ASR_WORKER")
	if worker == "" {
		worker = Locate(Resources)
	}
	worker, err := exec.LookPath(worker)
	if err != nil {
		return nil, domain.Fail(domain.CodeModelUnsupported, "native ASR worker start failed")
	}
	worker, err = filepath.Abs(worker)
	if err != nil {
		return nil, domain.Fail(domain.CodeModelUnsupported, "native ASR worker path invalid")
	}
	spec.Directory, err = filepath.Abs(spec.Directory)
	if err != nil {
		return nil, domain.Fail(domain.CodeModelUnsupported, "native ASR model path invalid")
	}
	files := make(map[string]string, len(spec.Files))
	for role, name := range spec.Files {
		files[role] = name
	}
	spec.Files = files
	lifetime, stop := context.WithCancel(context.Background())
	e := &engine{spec: spec, worker: worker, lifetime: lifetime, stop: stop, serial: make(chan struct{}, 1), cancelGrace: 5 * time.Second}
	if err := e.start(ctx); err != nil {
		e.close()
		return nil, err
	}
	return e, nil
}

// start publishes a started process before waiting for model readiness, so
// Close also owns workers whose model is still loading. The same lock forbids
// starting or publishing any process after permanent closure.
func (e *engine) start(ctx context.Context) error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return domain.Fail(domain.CodeModelMissing, "native ASR engine closed")
	}
	e.needsReload = false
	p, err := startWorker(e.worker, e.spec)
	if err == nil {
		e.current = p
	}
	e.mu.Unlock()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	stop := context.AfterFunc(e.lifetime, cancel)
	defer stop()
	defer cancel()
	select {
	case m := <-p.ready:
		if m.Type == "ready" && m.Protocol == 3 {
			p.promptSupported = m.supportsPrompt(e.spec.Kind)
			if ctx.Err() != nil {
				p.close()
				return ctx.Err()
			}
			return nil
		}
	case <-ctx.Done():
		p.close()
		return ctx.Err()
	case <-p.done:
	}
	p.close()
	return domain.Fail(domain.CodeModelUnsupported, "native ASR load failed")
}

func startWorker(worker string, spec Spec) (*workerProcess, error) {
	model := spec.Directory
	if spec.Kind == QwenGGUF {
		model = filepath.Join(spec.Directory, spec.Files["model"])
	}
	args := []string{"worker", model}
	if spec.Kind == QwenGGUF {
		args = append(args, "--threads", strconv.Itoa(spec.Threads), "--no-warmup")
	}
	cmd := exec.Command(worker, args...)
	cmd.Env = append(os.Environ(), "SPEECH_NO_UPDATE_CHECK=1")
	prepareProcess(cmd)
	// Runtime diagnostics must not disclose audio, text, or user paths.
	cmd.Stderr = io.Discard
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		stdin.Close()
		return nil, domain.Fail(domain.CodeModelUnsupported, "native ASR worker start failed")
	}
	p := &workerProcess{cmd: cmd, stdin: stdin, pending: map[string]chan message{}, ready: make(chan message, 1), done: make(chan struct{})}
	go p.read(stdout)
	return p, nil
}

func (e *workerProcess) read(stdout io.Reader) {
	in := bufio.NewScanner(stdout)
	in.Buffer(make([]byte, 64<<10), 4<<20)
	for in.Scan() {
		var m message
		if json.Unmarshal(in.Bytes(), &m) != nil {
			break
		}
		if m.Type == "ready" || m.Type == "fatal" {
			select {
			case e.ready <- m:
			default:
			}
			continue
		}
		if m.Type == "progress" {
			continue
		}
		if m.ID == "" || (m.Type != "end" && m.Type != "error" && m.Type != "cancelled") {
			break
		}
		e.mu.Lock()
		ch := e.pending[m.ID]
		delete(e.pending, m.ID)
		e.mu.Unlock()
		if ch != nil {
			ch <- m
		}
	}
	stopProcessTree(e.cmd)
	_ = e.cmd.Wait()
	e.mu.Lock()
	for id, ch := range e.pending {
		close(ch)
		delete(e.pending, id)
	}
	e.mu.Unlock()
	close(e.done)
}
func (e *workerProcess) send(m any) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	e.write.Lock()
	defer e.write.Unlock()
	_, err = e.stdin.Write(append(data, '\n'))
	return err
}
func (e *engine) decode(ctx context.Context, samples []float32, hotwords ...string) (string, error) {
	select {
	case e.serial <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	case <-e.lifetime.Done():
		return "", domain.Fail(domain.CodeModelMissing, "native ASR engine closed")
	}
	defer func() { <-e.serial }()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(e.lifetime, cancel)
	defer stop()
	defer cancel()
	e.mu.Lock()
	closed, reload, failure, p := e.closed, e.needsReload, e.failure, e.current
	e.mu.Unlock()
	if closed {
		return "", domain.Fail(domain.CodeModelMissing, "native ASR engine closed")
	}
	if failure != nil {
		return "", failure
	}
	if reload {
		if err := e.start(ctx); err != nil {
			e.mu.Lock()
			if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
				// A caller cancelling its replacement load ends only that
				// request. start has reaped the incomplete process; another
				// explicit request may still replace the retired generation.
				e.current = nil
				e.needsReload = !e.closed
			} else {
				e.failure = err
			}
			e.mu.Unlock()
			return "", err
		}
		e.mu.Lock()
		p = e.current
		e.mu.Unlock()
	}
	prompt, err := domain.ASRContext(hotwords)
	if err != nil {
		return "", domain.Fail(domain.CodeContractUnsupported, "invalid ASR hotword context")
	}
	if prompt != "" && !p.promptSupported {
		return "", domain.Fail(domain.CodeModelUnsupported, "native ASR worker does not support hotword context")
	}
	text, err, retired := p.decode(ctx, samples, e.cancelGrace, prompt)
	if retired {
		e.mu.Lock()
		if e.current == p {
			e.current = nil
			e.needsReload = !e.closed
		}
		e.mu.Unlock()
	}
	return text, err
}

func (e *workerProcess) decode(ctx context.Context, samples []float32, cancelGrace time.Duration, prompt string) (string, error, bool) {
	select {
	case <-e.done:
		return "", domain.Fail(domain.CodeModelUnsupported, "native ASR worker exited"), false
	default:
	}
	if ctx.Err() != nil {
		return "", ctx.Err(), false
	}
	pcm := make([]byte, len(samples)*2)
	for i, x := range samples {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return "", domain.Fail(domain.CodeContractUnsupported, "invalid PCM"), false
		}
		x = max(-1, min(1, x))
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(int16(max(-32768, min(32767, math.Round(float64(x)*32768))))))
	}
	id := strconv.FormatUint(e.next.Add(1), 10)
	ch := make(chan message, 1)
	e.mu.Lock()
	e.pending[id] = ch
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.pending, id); e.mu.Unlock() }()
	if err := e.send(map[string]any{"type": "chunk", "id": id, "seq": 0, "pcm": base64.StdEncoding.EncodeToString(pcm)}); err != nil {
		return "", err, false
	}
	if err := e.send(map[string]any{"type": "transcribe", "id": id, "sample_rate": 16000, "prompt": prompt}); err != nil {
		return "", err, false
	}
	select {
	case m, ok := <-ch:
		if !ok || m.Type != "end" || m.Stop != "complete" {
			return "", domain.Fail(domain.CodeModelUnsupported, "native ASR inference incomplete"), false
		}
		if ctx.Err() != nil {
			return "", ctx.Err(), false
		}
		text, err := qwenTranscript(m.Text)
		return text, err, false
	case <-ctx.Done():
		_ = e.send(map[string]any{"type": "cancel", "id": id})
		select {
		case <-ch:
		case <-e.done:
		case <-time.After(cancelGrace):
			select {
			case <-e.done:
				return "", ctx.Err(), false
			default:
			}
			// This generation is no longer safe to reuse. Reap it now; a later
			// request may load the same runtime and specification, never retry
			// the cancelled request or switch to another backend.
			stopProcessTree(e.cmd)
			<-e.done
			return "", ctx.Err(), true
		}
		return "", ctx.Err(), false
	case <-e.done:
		return "", domain.Fail(domain.CodeModelUnsupported, "native ASR worker exited"), false
	}
}
func (e *engine) close() {
	e.once.Do(func() {
		e.mu.Lock()
		e.closed = true
		e.stop()
		p := e.current
		e.mu.Unlock()
		if p != nil {
			p.close()
		}
	})
}

func (e *workerProcess) close() {
	e.once.Do(func() {
		_ = e.stdin.Close()
		select {
		case <-e.done:
		case <-time.After(5 * time.Second):
			stopProcessTree(e.cmd)
			<-e.done
		}
	})
}
