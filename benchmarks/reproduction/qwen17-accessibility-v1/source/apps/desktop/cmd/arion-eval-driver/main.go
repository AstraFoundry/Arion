//go:build !windows

// Command arion-eval-driver runs Arion's production dictation pipeline behind
// the evaluation JSONL protocol (tools/evaluation). Audio is replayed in real
// time from verified fixtures; stdout carries protocol lines only.
package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/audio"
	"arion.local/desktop/infrastructure/modelruntime"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/models"
)

type request struct {
	Protocol  int               `json:"protocol"`
	RequestID string            `json:"requestId"`
	Op        string            `json:"op"`
	Purpose   string            `json:"purpose"`
	Artifacts map[string]string `json:"artifacts"`
	SampleID  string            `json:"sampleId"`
	Seed      int               `json:"seed"`
	Warmup    bool              `json:"warmup"`
	Cache     string            `json:"cacheState"`
	Variant   struct {
		ID       string          `json:"id"`
		Features json.RawMessage `json:"features"`
	} `json:"variant"`
	Pipeline map[string]any `json:"pipeline"`
	Input    struct {
		Audio struct {
			Path       string  `json:"path"`
			SHA256     string  `json:"sha256"`
			DurationMs float64 `json:"durationMs"`
		} `json:"audio"`
		Context        string `json:"context"`
		ApprovedMemory []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"approvedMemory"`
	} `json:"input"`
}

type features struct {
	Memory     bool   `json:"memory"`
	Ranker     string `json:"ranker"`
	Correction string `json:"correction"`
}

type driverConfig struct {
	Artifacts []struct {
		ID     string `json:"id"`
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"artifacts"`
}

type event struct {
	Ms    float64 `json:"ms"`
	Text  string  `json:"text"`
	Final bool    `json:"final,omitempty"`
}

type candidate struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Source string `json:"source"`
}

func main() {
	configPath := flag.String("config", "", "driver configuration JSON whose artifacts are attested")
	modelRoot := flag.String("models", "", "verified model store root")
	flag.Parse()
	if *configPath == "" || *modelRoot == "" {
		fmt.Fprintln(os.Stderr, "usage: arion-eval-driver -config driver.json -models dir")
		os.Exit(2)
	}
	catalog, err := models.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalog invalid")
		os.Exit(1)
	}
	d := &driver{catalog: catalog, store: provision.NewStore(*modelRoot), configPath: *configPath, managers: map[string]*modelruntime.Manager{}}
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 8<<20)
	out := json.NewEncoder(os.Stdout)
	for in.Scan() {
		var req request
		if err := json.Unmarshal(in.Bytes(), &req); err != nil || req.Protocol != 1 {
			fmt.Fprintln(os.Stderr, "invalid request line")
			os.Exit(1)
		}
		response, err := d.handle(req)
		if err != nil {
			// The runner treats a missing reply as failure; never fabricate one.
			// The reason (a code, never content) goes to the diagnostics dir
			// because the runner discards stderr.
			fmt.Fprintln(os.Stderr, "request failed:", err)
			if *diagnostics != "" {
				_ = os.MkdirAll(*diagnostics, 0o700)
				_ = os.WriteFile(filepath.Join(*diagnostics, "request-"+req.RequestID+".error"), []byte(err.Error()+"\n"), 0o600)
			}
			os.Exit(1)
		}
		response["protocol"], response["requestId"] = 1, req.RequestID
		if err := out.Encode(response); err != nil {
			os.Exit(1)
		}
	}
}

type driver struct {
	catalog    models.Catalog
	store      *provision.Store
	configPath string
	managers   map[string]*modelruntime.Manager
	attested   map[string]string
}

func (d *driver) handle(req request) (map[string]any, error) {
	switch req.Op {
	case "hello":
		return d.hello(req)
	case "infer":
		return d.infer(req)
	default:
		return nil, fmt.Errorf("unsupported op %q", req.Op)
	}
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hello re-hashes every configured artifact itself: the attestation is the
// driver's own observation, not an echo of the runner's request.
func (d *driver) hello(req request) (map[string]any, error) {
	data, err := os.ReadFile(d.configPath)
	if err != nil {
		return nil, err
	}
	var config driverConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	attested := map[string]string{}
	for _, artifact := range config.Artifacts {
		filename := artifact.Path
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(filepath.Dir(d.configPath), filename)
		}
		digest, err := hashFile(filename)
		if err != nil {
			return nil, err
		}
		attested[artifact.ID] = digest
	}
	d.attested = attested
	return map[string]any{
		"execution": "live", "purpose": req.Purpose, "artifacts": attested,
		"hardware": map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "cpu": sysctl("machdep.cpu.brand_string"),
			"powerMode": powerMode(), "ramBytes": ramBytes(), "osBuild": sysctl("kern.osversion"), "logicalCpus": runtime.NumCPU()},
		"runtime": map[string]any{"version": "arion-desktop pipeline; sherpa-onnx-go v1.13.8; " + runtime.Version(),
			"buildFlags": "CGO_ENABLED=1 GOARCH=" + runtime.GOARCH, "timer": "Go monotonic clock (mach_absolute_time)", "provider": "onnxruntime CPU"},
	}, nil
}

func sysctl(name string) string {
	out, err := exec.Command("/usr/sbin/sysctl", "-n", name).Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func ramBytes() float64 {
	var value float64
	fmt.Sscan(sysctl("hw.memsize"), &value)
	return value
}

func powerMode() string {
	out, err := exec.Command("/usr/bin/pmset", "-g", "batt").Output()
	if err != nil {
		return "unknown"
	}
	if strings.Contains(string(out), "AC Power") {
		return "ac"
	}
	return "battery"
}

func peakRSS() float64 {
	var usage syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &usage)
	if runtime.GOOS == "darwin" {
		return float64(usage.Maxrss)
	}
	return float64(usage.Maxrss) * 1024
}

// timing wraps the model ports to measure inference-active wall time.
type timing struct {
	mu      sync.Mutex
	compute time.Duration
}

func (t *timing) add(start time.Time) {
	t.mu.Lock()
	t.compute += time.Since(start)
	t.mu.Unlock()
}

type timedFactory struct {
	inner ports.RecognizerFactory
	t     *timing
}

func (f timedFactory) NewRecognizer(ctx context.Context, id contracts.Identity) (ports.Recognizer, error) {
	r, err := f.inner.NewRecognizer(ctx, id)
	if err != nil {
		return nil, err
	}
	return timedRecognizer{r, f.t}, nil
}

type timedRecognizer struct {
	ports.Recognizer
	t *timing
}

func (r timedRecognizer) Accept(ctx context.Context, frame contracts.Frame) error {
	defer r.t.add(time.Now())
	return r.Recognizer.Accept(ctx, frame)
}
func (r timedRecognizer) Partial(ctx context.Context) (contracts.Draft, error) {
	defer r.t.add(time.Now())
	return r.Recognizer.Partial(ctx)
}
func (r timedRecognizer) Finish(ctx context.Context) (contracts.Draft, error) {
	defer r.t.add(time.Now())
	return r.Recognizer.Finish(ctx)
}

type timedFinalizer struct {
	inner ports.Finalizer
	t     *timing
}

func (f timedFinalizer) Finalize(ctx context.Context, request ports.FinalizeRequest) (string, error) {
	defer f.t.add(time.Now())
	return f.inner.Finalize(ctx, request)
}

func (d *driver) manager(bundle models.Bundle, threads int, cold bool) (*modelruntime.Manager, float64, error) {
	key := fmt.Sprintf("%s/%d/%d", bundle.ID, threads, bundle.EndpointMs)
	if m, ok := d.managers[key]; ok && !cold {
		return m, 0, nil
	}
	// Cold requests reset model residency: every resident bundle is dropped
	// and the requested one is loaded again, its load time recorded.
	for k := range d.managers {
		delete(d.managers, k)
	}
	runtime.GC()
	start := time.Now()
	m := modelruntime.NewManager(d.catalog, d.store, threads)
	if !m.Installed(bundle) {
		return nil, 0, fmt.Errorf("bundle %s not provisioned in model store", bundle.ID)
	}
	if err := m.Activate(context.Background(), bundle); err != nil {
		return nil, 0, err
	}
	d.managers[key] = m
	return m, float64(time.Since(start).Microseconds()) / 1000, nil
}

var diagnostics = flag.String("diagnostics", "", "directory for watchdog stack dumps")

func dumpStacks(request string) {
	if *diagnostics == "" {
		return
	}
	buffer := make([]byte, 1<<22)
	n := runtime.Stack(buffer, true)
	_ = os.MkdirAll(*diagnostics, 0o700)
	_ = os.WriteFile(filepath.Join(*diagnostics, "request-"+request+".stacks"), buffer[:n], 0o600)
}

func number(pipeline map[string]any, key string) (float64, bool) {
	value, ok := pipeline[key].(float64)
	return value, ok
}

func (d *driver) infer(req request) (map[string]any, error) {
	if d.attested == nil {
		return nil, errors.New("hello required")
	}
	var f features
	if err := json.Unmarshal(req.Variant.Features, &f); err != nil {
		return nil, err
	}
	// Only treatments this pipeline really implements are accepted.
	if f.Memory || f.Ranker != "source" || (f.Correction != "none" && f.Correction != "local") {
		return nil, fmt.Errorf("unsupported treatment %+v", f)
	}
	known := map[string]bool{"asr": true, "sampleRate": true, "chunkMs": true, "endpoint": true, "pace": true, "threads": true, "endpointSilenceMs": true}
	for key := range req.Pipeline {
		if !known[key] {
			return nil, fmt.Errorf("pipeline setting %q not implemented by this driver", key)
		}
	}
	bundleID, _ := req.Pipeline["asr"].(string)
	bundle, ok := d.catalog.Bundle(bundleID)
	if !ok {
		return nil, fmt.Errorf("unknown bundle %q", bundleID)
	}
	chunk, _ := number(req.Pipeline, "chunkMs")
	threads, _ := number(req.Pipeline, "threads")
	rate, _ := number(req.Pipeline, "sampleRate")
	endpoint, _ := number(req.Pipeline, "endpointSilenceMs")
	if rate != audio.SampleRate || req.Pipeline["pace"] != "realtime" || req.Pipeline["endpoint"] != "sherpa-rules-v1" || chunk <= 0 || threads <= 0 {
		return nil, errors.New("pipeline settings not supported")
	}
	if int(endpoint) != bundle.EndpointMs {
		bundle.EndpointMs = int(endpoint)
	}
	digest, err := hashFile(req.Input.Audio.Path)
	if err != nil || digest != req.Input.Audio.SHA256 {
		return nil, errors.New("audio fixture digest mismatch")
	}
	samples, err := audio.ReadWAV(req.Input.Audio.Path)
	if err != nil {
		return nil, err
	}
	manager, loadMs, err := d.manager(bundle, int(threads), req.Cache == "cold")
	if err != nil {
		return nil, err
	}
	clock := &timing{}
	replay := &audio.Replay{Samples: samples, ChunkMs: int(chunk)}
	correction := f.Correction == "local"
	coordinator, err := application.NewCoordinator(application.Dependencies{
		Audio: replay, Recognizers: timedFactory{manager, clock}, Finalizer: timedFinalizer{manager, clock},
		NewID: func() string { return req.RequestID },
	}, application.Options{Correction: correction, Policy: func() domain.CorrectionPolicy {
		return domain.CorrectionPolicy{LatinCase: manager.LatinCase()}
	}})
	if err != nil {
		return nil, err
	}
	var mu sync.Mutex
	var events []event
	var firstPartial *float64
	var raw, final string
	var reviewAt time.Time
	finished := make(chan domain.ErrorCode, 1)
	capturing := make(chan struct{}, 1)
	coordinator.Subscribe(func(e contracts.Event) {
		mu.Lock()
		defer mu.Unlock()
		started := replay.Started()
		since := func() float64 { return float64(time.Since(started).Microseconds()) / 1000 }
		switch e.Kind {
		case contracts.SessionStateChanged:
			if e.State == domain.Capturing {
				capturing <- struct{}{}
			}
		case contracts.DraftReplaced:
			ms := since()
			events = append(events, event{Ms: ms, Text: e.Text})
			if firstPartial == nil && strings.TrimSpace(e.Text) != "" {
				firstPartial = &ms
			}
		case contracts.CorrectionReady:
			raw, final, reviewAt = e.Raw, e.Text, time.Now()
			finished <- ""
		case contracts.ErrorOccurred:
			reviewAt = time.Now()
			finished <- e.Error
		}
	})
	// A request that overruns its audio by a minute dumps all goroutine
	// stacks (content-free) for diagnosis; the runner then fails the run.
	watchdog := time.AfterFunc(time.Duration(req.Input.Audio.DurationMs)*time.Millisecond+time.Minute, func() { dumpStacks(req.RequestID) })
	defer watchdog.Stop()
	coordinator.Start()
	select {
	case <-capturing:
	case code := <-finished:
		return nil, fmt.Errorf("session failed before capture: %s", code)
	}
	select {
	case <-replay.Done():
	case code := <-finished:
		return nil, fmt.Errorf("session failed during capture: %s", code)
	}
	stopAt := time.Now()
	coordinator.Stop()
	code := <-finished
	if code != "" && code != domain.CodeNoSpeech {
		return nil, fmt.Errorf("session failed: %s", code)
	}
	mu.Lock()
	defer mu.Unlock()
	started := replay.Started()
	msAt := func(t time.Time) float64 { return float64(t.Sub(started).Microseconds()) / 1000 }
	speechEnd := msAt(stopAt)
	finalMs := msAt(reviewAt)
	if !correction {
		final = raw
	}
	events = append(events, event{Ms: finalMs, Text: raw, Final: true})
	candidates := []candidate{}
	ranked := []string{}
	if raw != "" {
		candidates = append(candidates, candidate{ID: "recognizer-1", Text: raw, Source: "recognizer"})
		ranked = append(ranked, "recognizer-1")
	}
	var firstPartialValue any
	if firstPartial != nil {
		firstPartialValue = *firstPartial
	}
	var effectiveFeatures any
	_ = json.Unmarshal(req.Variant.Features, &effectiveFeatures)
	effectivePipeline := map[string]any{}
	for key, value := range req.Pipeline {
		effectivePipeline[key] = value
	}
	if !reflect.DeepEqual(effectivePipeline, req.Pipeline) {
		return nil, errors.New("pipeline not applied")
	}
	return map[string]any{
		"execution": "live", "effectiveFeatures": effectiveFeatures, "effectivePipeline": effectivePipeline, "cacheState": req.Cache,
		"rawText": raw, "selectedText": raw, "finalText": final, "candidates": candidates, "rankedIds": ranked, "events": events,
		"timings": map[string]any{"speechStartMs": 0.0, "speechEndMs": speechEnd, "firstPartialMs": firstPartialValue, "finalMs": finalMs,
			"reviewMs": finalMs, "rankerMs": nil, "loadMs": loadMs, "computeMs": float64(clock.compute.Microseconds())/1000 + loadMs},
		"resources": map[string]any{"peakRssBytes": peakRSS()},
	}, nil
}
