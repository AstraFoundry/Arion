// Package asr adapts resident native Qwen3-ASR workers to recognition ports.
package asr

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	Type     string `json:"type"`
	ID       string `json:"id,omitempty"`
	Text     string `json:"text,omitempty"`
	Stop     string `json:"stop,omitempty"`
	Protocol int    `json:"protocol,omitempty"`
}

// engine owns one subprocess and its request stream. Progress is nonterminal;
// cancellation waits for the terminal acknowledgement before reusing the model.
type engine struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	write   sync.Mutex
	mu      sync.Mutex
	pending map[string]chan message
	ready   chan message
	done    chan struct{}
	serial  chan struct{}
	next    atomic.Uint64
	once    sync.Once
}

func load(ctx context.Context, spec Spec) (*engine, error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if spec.SampleRate != 16000 || (spec.Kind != QwenMLX && spec.Kind != QwenGGUF) {
		return nil, domain.Fail(domain.CodeModelUnsupported, "unsupported native ASR specification")
	}
	worker := os.Getenv("ARION_ASR_WORKER")
	if worker == "" {
		worker = Locate(Resources)
	}
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
	e := &engine{cmd: cmd, stdin: stdin, pending: map[string]chan message{}, ready: make(chan message, 1), done: make(chan struct{}), serial: make(chan struct{}, 1)}
	go e.read(stdout)
	select {
	case m := <-e.ready:
		if m.Type == "ready" && m.Protocol == 3 {
			return e, nil
		}
	case <-ctx.Done():
		e.close()
		return nil, ctx.Err()
	case <-e.done:
	}
	e.close()
	return nil, domain.Fail(domain.CodeModelUnsupported, "native ASR load failed")
}

func (e *engine) read(stdout io.Reader) {
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
func (e *engine) send(m any) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	e.write.Lock()
	defer e.write.Unlock()
	_, err = e.stdin.Write(append(data, '\n'))
	return err
}
func (e *engine) decode(ctx context.Context, samples []float32) (string, error) {
	select {
	case e.serial <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	case <-e.done:
		return "", domain.Fail(domain.CodeModelUnsupported, "native ASR worker exited")
	}
	defer func() { <-e.serial }()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	pcm := make([]byte, len(samples)*2)
	for i, x := range samples {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return "", domain.Fail(domain.CodeContractUnsupported, "invalid PCM")
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
		return "", err
	}
	if err := e.send(map[string]any{"type": "transcribe", "id": id, "sample_rate": 16000}); err != nil {
		return "", err
	}
	select {
	case m, ok := <-ch:
		if !ok || m.Type != "end" || m.Stop != "complete" {
			return "", domain.Fail(domain.CodeModelUnsupported, "native ASR inference incomplete")
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return qwenTranscript(m.Text)
	case <-ctx.Done():
		_ = e.send(map[string]any{"type": "cancel", "id": id})
		select {
		case <-ch:
		case <-e.done:
		case <-time.After(5 * time.Second):
			e.close()
		}
		return "", ctx.Err()
	case <-e.done:
		return "", domain.Fail(domain.CodeModelUnsupported, "native ASR worker exited")
	}
}
func (e *engine) close() {
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
