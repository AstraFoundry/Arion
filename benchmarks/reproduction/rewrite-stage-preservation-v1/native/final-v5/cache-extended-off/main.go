//go:build !windows

// Command arion-live-bench replays pause-segmented speech fixtures in real
// time through the production dictation coordinator: streaming recognition,
// optional per-segment finalization, live LLM previews in the arion-llm
// worker and the final validated rewrite. It records every displayed draft
// and the final review on one monotonic clock as JSON lines. It is a
// development benchmark; it does not measure native insertion.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
 "arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/audio"
	"arion.local/desktop/infrastructure/llm/client"
	"arion.local/desktop/infrastructure/modelruntime"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/models"
)

type manifest struct {
	TextCorpusSha256 string `json:"textCorpusSha256"`
	Samples          []struct {
		ID       string `json:"id"`
		Segments int    `json:"segments"`
		Audio    struct {
			Path       string  `json:"path"`
			SHA256     string  `json:"sha256"`
			DurationMs float64 `json:"durationMs"`
		} `json:"audio"`
	} `json:"samples"`
}

type textSample struct {
	ID             string   `json:"id"`
	Scenario       string   `json:"scenario"`
	ApprovedMemory []string `json:"approvedMemory"`
 Replacements []domain.Replacement `json:"replacements"`
 Screen string `json:"screen"`
}

type draftEvent struct {
	Ms      float64 `json:"ms"`
	Text    string  `json:"text"`
	Preview bool    `json:"preview,omitempty"`
}

type row struct {
	RemoveFillers  bool         `json:"removeFillers"`
	CaseID         string       `json:"caseId"`
	VariantID      string       `json:"variantId"`
	Events         []draftEvent `json:"events"`
	RawText        string       `json:"rawText"`
	SourceText     string       `json:"sourceText"`
	FinalText      string       `json:"finalText"`
	Rewrite        string       `json:"rewrite"`
	Error          string       `json:"error,omitempty"`
	SpeechEndMs    float64      `json:"speechEndMs"`
	FinalMs        float64      `json:"finalMs"`
	StopToFinalMs  float64      `json:"stopToFinalMs"`
	FirstPartialMs *float64     `json:"firstPartialMs"`
	FirstPreviewMs *float64     `json:"firstPreviewMs"`
	PreviewEvents  int          `json:"previewEvents"`
	Segments       int          `json:"segments"`
}

func main() {
	manifestPath := flag.String("manifest", "", "live audio manifest")
	corpusPath := flag.String("corpus", "", "text corpus the manifest was built from")
	storeRoot := flag.String("models", "", "verified model store root")
	bundleID := flag.String("bundle", "", "catalog bundle id")
	removeFillers := flag.Bool("remove-fillers", true, "remove hesitation tokens before rewriting, matching the product default")
	segment := flag.Bool("segment-finalize", false, "finalize closed segments while speaking")
	worker := flag.String("llm-worker", "", "arion-llm executable (enables rewriting)")
	runtimeDir := flag.String("llm-runtime", "", "GenAI runtime directory")
	llmModel := flag.String("llm-model", "", "GenAI model directory")
	threads := flag.Int("threads", 4, "ASR and LLM intra-op threads")
	chunk := flag.Int("chunk", 20, "replay chunk in ms")
	variant := flag.String("variant", "", "variant id recorded in rows")
	out := flag.String("out", "", "output JSONL path (must not exist)")
	flag.Parse()
	if *manifestPath == "" || *corpusPath == "" || *storeRoot == "" || *bundleID == "" || *variant == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	var m manifest
	readJSON(*manifestPath, &m)
	corpusBytes, err := os.ReadFile(*corpusPath)
	check(err)
	if digest(corpusBytes) != m.TextCorpusSha256 {
		check(errors.New("text corpus differs from the manifest's source corpus"))
	}
	var corpus struct{ Samples []textSample }
	check(json.Unmarshal(corpusBytes, &corpus))
	texts := map[string]textSample{}
	for _, s := range corpus.Samples {
		texts[s.ID] = s
	}
	catalog, err := models.Load()
	check(err)
	bundle, ok := catalog.Bundle(*bundleID)
	if !ok {
		check(fmt.Errorf("unknown bundle %s", *bundleID))
	}
	manager := modelruntime.NewManager(catalog, provision.NewStore(*storeRoot), *threads)
	start := time.Now()
	check(manager.Activate(context.Background(), bundle))
	fmt.Fprintf(os.Stderr, "activated %s in %.0f ms\n", bundle.ID, ms(time.Since(start)))
	var rewriter *client.Client
	if *worker != "" {
		var info client.LoadInfo
		rewriter, info, err = client.Start(context.Background(), client.Config{Worker: *worker, Runtime: *runtimeDir, Model: *llmModel,
			Threads: *threads, MaxLength: 4096, MaxNewTokens: 256}, os.Stderr)
		check(err)
		defer rewriter.Close()
		fmt.Fprintf(os.Stderr, "llm loaded prompt=%s in %.0f ms\n", info.PromptID, info.LoadMs)
	}
	file, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	check(err)
	defer file.Close()
	encoder := json.NewEncoder(file)
	dir := filepath.Dir(*manifestPath)
	for i, s := range m.Samples {
		path := filepath.Join(dir, s.Audio.Path)
		data, err := os.ReadFile(path)
		check(err)
		if digest(data) != s.Audio.SHA256 {
			check(fmt.Errorf("%s audio digest mismatch", s.ID))
		}
		samples, err := audio.ReadWAV(path)
		check(err)
		text := texts[s.ID]
		r := runCase(manager, rewriter, bundle, *segment, *removeFillers, *chunk, samples, text, fmt.Sprintf("%s-%d", s.ID, i))
		r.CaseID, r.VariantID, r.Segments = s.ID, *variant, s.Segments
		check(encoder.Encode(r))
		fmt.Fprintf(os.Stderr, "%-18s stop→final %6.0f ms previews=%d %s\n", s.ID, r.StopToFinalMs, r.PreviewEvents, r.Rewrite)
	}
}

func runCase(manager *modelruntime.Manager, rewriter *client.Client, bundle models.Bundle, segment, removeFillers bool, chunk int, samples []float32, text textSample, id string) row {
	replay := &audio.Replay{Samples: samples, ChunkMs: chunk}
	deps := application.Dependencies{Audio: replay, Recognizers: manager, NewID: func() string { return id }}
	if bundle.Finalizer != "" {
		deps.Finalizer = manager
	}
	policy := application.RewritePolicy{}
	if rewriter != nil {
		traceFile, err := os.OpenFile(filepath.Join(filepath.Dir(os.Args[0]), id+"-rewrite.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
 check(err)
 defer traceFile.Close()
 deps.Rewriter = &screenTraceRewriter{client: rewriter, screen: text.Screen, out: json.NewEncoder(traceFile)}
		policy = application.RewritePolicy{Enabled: true, Scene: text.Scenario, Terms: text.ApprovedMemory, Vocabulary: domain.NewVocabulary(text.Replacements), Timeout: 10 * time.Second}
	}
	coordinator, err := application.NewCoordinator(deps, application.Options{Correction: true, SegmentFinalize: segment,
		Policy: func() domain.CorrectionPolicy {
			return domain.CorrectionPolicy{LatinCase: manager.LatinCase(), RemoveFillers: removeFillers, Vocabulary: domain.NewVocabulary(text.Replacements)}
		},
		Rewrite: func() application.RewritePolicy { return policy }})
	check(err)
	var mu sync.Mutex
	r := row{RemoveFillers: removeFillers}
	var finalAt time.Time
	finished := make(chan struct{}, 1)
	capturing := make(chan struct{}, 1)
	coordinator.Subscribe(func(e contracts.Event) {
		mu.Lock()
		defer mu.Unlock()
		since := ms(time.Since(replay.Started()))
		switch e.Kind {
		case contracts.SessionStateChanged:
			if e.State == domain.Capturing {
				capturing <- struct{}{}
			}
		case contracts.DraftReplaced:
			preview := e.Rewrite == application.RewritePreview
			r.Events = append(r.Events, draftEvent{Ms: since, Text: e.Text, Preview: preview})
			if r.FirstPartialMs == nil && strings.TrimSpace(e.Text) != "" {
				r.FirstPartialMs = &since
			}
			if preview {
				r.PreviewEvents++
				if r.FirstPreviewMs == nil {
					r.FirstPreviewMs = &since
				}
			}
		case contracts.CorrectionReady:
			r.RawText, r.SourceText, r.FinalText, r.Rewrite = e.Raw, e.Source, e.Text, e.Rewrite
			finalAt = time.Now()
			finished <- struct{}{}
		case contracts.ErrorOccurred:
			r.Error = string(e.Error)
			finalAt = time.Now()
			finished <- struct{}{}
		}
	})
	coordinator.Start()
	<-capturing
	select {
	case <-replay.Done():
	case <-finished:
		return r
	}
	stopAt := time.Now()
	coordinator.Stop()
	<-finished
	// Let the pump release recognizer and worker session state.
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	r.SpeechEndMs = ms(stopAt.Sub(replay.Started()))
	r.FinalMs = ms(finalAt.Sub(replay.Started()))
	r.StopToFinalMs = ms(finalAt.Sub(stopAt))
	return r
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func readJSON(path string, v any) {
	f, err := os.Open(path)
	check(err)
	defer f.Close()
	data, err := io.ReadAll(f)
	check(err)
	check(json.Unmarshal(data, v))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// screenTraceRewriter supplies an explicit fixture context to actual inference.
// It records native requests/results without replacing recognition or model output.
type screenTraceRewriter struct { client *client.Client; screen string; out *json.Encoder; mu sync.Mutex }
func (r *screenTraceRewriter) Rewrite(ctx context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
 req.Context.Screen = r.screen
 start := time.Now()
 result, err := r.client.Rewrite(ctx, req)
 errorText := ""; if err != nil { errorText = err.Error() }
 r.mu.Lock(); defer r.mu.Unlock()
 check(r.out.Encode(struct { Request ports.RewriteRequest; Result ports.RewriteResult; Error string; WallMs float64 }{req,result,errorText,ms(time.Since(start))}))
 return result, err
}
func (r *screenTraceRewriter) End(id contracts.Identity) { r.client.End(id) }
