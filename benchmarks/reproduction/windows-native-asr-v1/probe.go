// Command probe records actual native Windows ASR adapter observations.
// References stay in the evaluator and never enter recognition requests.
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
	"runtime"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/asr"
	"arion.local/desktop/infrastructure/audio"
	"arion.local/models"
)

type sample struct {
	ID       string `json:"id"`
	Split    string `json:"split"`
	Language string `json:"language"`
	Audio    struct {
		Path       string  `json:"path"`
		SHA256     string  `json:"sha256"`
		DurationMs float64 `json:"durationMs"`
	} `json:"audio"`
	Reference string `json:"reference"`
}

func digest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func main() {
	root := flag.String("root", "", "repository root")
	model := flag.String("model", "", "pinned GGUF file")
	worker := flag.String("worker", "", "native worker file")
	out := flag.String("out", "", "new evidence directory")
	flag.Parse()
	if err := run(*root, *model, *worker, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, model, worker, out string) error {
	if root == "" || model == "" || worker == "" || out == "" {
		return errors.New("all arguments are required")
	}
	if err := os.Mkdir(out, 0700); err != nil {
		return err
	}
	log, err := os.Create(filepath.Join(out, "observations.jsonl"))
	if err != nil {
		return err
	}
	defer log.Close()
	encoder := json.NewEncoder(io.MultiWriter(log, os.Stdout))
	emit := func(v any) {
		if err := encoder.Encode(v); err != nil {
			panic(err)
		}
	}
	started := time.Now()
	defer func() {
		emit(map[string]any{"event": "exit", "elapsedMs": float64(time.Since(started).Microseconds()) / 1000})
	}()
	catalog, err := models.Load()
	if err != nil {
		return err
	}
	artifact, ok := catalog.Artifact("qwen3-asr-1.7b-gguf-q8")
	if !ok {
		return errors.New("missing catalog artifact")
	}
	hash, err := digest(model)
	if err != nil {
		return err
	}
	if hash != artifact.Files["model"].SHA256 {
		return errors.New("model digest mismatch")
	}
	workerHash, err := digest(worker)
	if err != nil {
		return err
	}
	emit(map[string]any{"event": "provenance", "os": runtime.GOOS, "arch": runtime.GOARCH, "go": runtime.Version(), "logicalCpus": runtime.NumCPU(), "processor": os.Getenv("PROCESSOR_IDENTIFIER"), "modelSha256": hash, "worker": worker, "workerSha256": workerHash, "runtime": artifact.Runtime, "threads": 4, "audioRate": 16000, "scope": "development smoke diagnostics; no aggregate quality or release performance claim"})
	if err = os.Setenv("ARION_ASR_WORKER", worker); err != nil {
		return err
	}
	spec := asr.Spec{Kind: asr.QwenGGUF, Directory: filepath.Dir(model), Files: map[string]string{"model": filepath.Base(model)}, SampleRate: 16000, Threads: 4, EndpointSilence: 0.8}
	corpusDir := filepath.Join(root, "benchmarks", "corpora", "asr-v1")
	raw, err := os.ReadFile(filepath.Join(corpusDir, "corpus.json"))
	if err != nil {
		return err
	}
	var corpus struct {
		Samples []sample `json:"samples"`
	}
	if err = json.Unmarshal(raw, &corpus); err != nil {
		return err
	}
	ids := []string{"aishell-BAC009S0766W0137", "aishell-BAC009S0766W0325", "libri-3081-166546-0006", "libri-251-137823-0017", "tts-dev-00-eddy", "tts-dev-01-flo"}
	selected := make([]sample, 0, len(ids))
	pcm := map[string][]float32{}
	for _, id := range ids {
		found := false
		for _, s := range corpus.Samples {
			if s.ID != id {
				continue
			}
			found = true
			if s.Split != "dev" {
				return errors.New("non-development sample")
			}
			path := filepath.Join(corpusDir, s.Audio.Path)
			hash, err := digest(path)
			if err != nil {
				return err
			}
			if hash != s.Audio.SHA256 {
				return fmt.Errorf("audio hash mismatch %s", id)
			}
			samples, err := audio.ReadWAV(path)
			if err != nil {
				return err
			}
			selected = append(selected, s)
			pcm[id] = samples
		}
		if !found {
			return fmt.Errorf("sample missing %s", id)
		}
	}
	loadStart := time.Now()
	offline, err := asr.LoadOffline(spec)
	if err != nil {
		return fmt.Errorf("offline load: %w", err)
	}
	emit(map[string]any{"event": "loaded", "mode": "offline", "elapsedMs": float64(time.Since(loadStart).Microseconds()) / 1000})
	for _, s := range selected {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		start := time.Now()
		text, decodeErr := offline.Finalize(ctx, ports.FinalizeRequest{SampleRate: 16000, Samples: pcm[s.ID]})
		cancel()
		emit(map[string]any{"event": "offline", "id": s.ID, "split": s.Split, "language": s.Language, "audioSha256": s.Audio.SHA256, "reference": s.Reference, "text": text, "elapsedMs": float64(time.Since(start).Microseconds()) / 1000, "error": errorString(decodeErr)})
		if decodeErr != nil {
			offline.Close()
			return decodeErr
		}
		if text == "" {
			offline.Close()
			return fmt.Errorf("empty speech output %s", s.ID)
		}
	}
	long := []float32{}
	for range 5 {
		long = append(long, pcm[selected[0].ID]...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	start := time.Now()
	text, cancelErr := offline.Finalize(ctx, ports.FinalizeRequest{SampleRate: 16000, Samples: long})
	cancel()
	emit(map[string]any{"event": "cancel", "elapsedMs": float64(time.Since(start).Microseconds()) / 1000, "text": text, "error": errorString(cancelErr)})
	if !errors.Is(cancelErr, context.DeadlineExceeded) {
		offline.Close()
		return fmt.Errorf("cancellation returned %v", cancelErr)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	start = time.Now()
	text, err = offline.Finalize(ctx, ports.FinalizeRequest{SampleRate: 16000, Samples: pcm[selected[2].ID]})
	cancel()
	emit(map[string]any{"event": "reuse-after-cancel", "id": selected[2].ID, "text": text, "elapsedMs": float64(time.Since(start).Microseconds()) / 1000, "error": errorString(err)})
	offline.Close()
	if err != nil {
		return err
	}
	if text == "" {
		return errors.New("empty reuse output")
	}
	loadStart = time.Now()
	stream, err := asr.LoadStreaming(spec)
	if err != nil {
		return err
	}
	defer stream.Close()
	emit(map[string]any{"event": "loaded", "mode": "streaming", "elapsedMs": float64(time.Since(loadStart).Microseconds()) / 1000})
	for _, index := range []int{0, 2, 4} {
		s := selected[index]
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		id := contracts.Identity{SessionID: s.ID, Generation: uint64(index + 1)}
		recognizer, err := stream.NewRecognizer(ctx, id)
		if err != nil {
			cancel()
			return err
		}
		start := time.Now()
		firstPartialMs := -1.0
		var previews []map[string]any
		samples := pcm[s.ID]
		for offset, sequence := 0, uint64(0); offset < len(samples); offset, sequence = offset+320, sequence+1 {
			end := min(offset+320, len(samples))
			time.Sleep(time.Until(start.Add(time.Duration(end) * time.Second / 16000)))
			err = recognizer.Accept(ctx, contracts.Frame{Identity: id, Sequence: sequence, SampleOffset: uint64(offset), SampleRate: 16000, Channels: 1, Samples: samples[offset:end]})
			if err != nil {
				recognizer.Close()
				cancel()
				return err
			}
			draft, partialErr := recognizer.Partial(ctx)
			if partialErr != nil {
				recognizer.Close()
				cancel()
				return partialErr
			}
			if draft.Text != "" && (len(previews) == 0 || draft.Revision != uint64(previews[len(previews)-1]["revision"].(uint64))) {
				ms := float64(time.Since(start).Microseconds()) / 1000
				if firstPartialMs < 0 {
					firstPartialMs = ms
				}
				previews = append(previews, map[string]any{"revision": draft.Revision, "text": draft.Text, "elapsedMs": ms})
			}
		}
		stop := time.Now()
		draft, finishErr := recognizer.Finish(ctx)
		compute := time.Duration(0)
		if timed, ok := recognizer.(interface{ ComputeTime() time.Duration }); ok {
			compute = timed.ComputeTime()
		}
		recognizer.Close()
		cancel()
		emit(map[string]any{"event": "streaming", "id": s.ID, "reference": s.Reference, "text": draft.Text, "identity": draft.Identity, "final": draft.Final, "revision": draft.Revision, "previews": previews, "firstPartialMs": firstPartialMs, "stopToResultMs": float64(time.Since(stop).Microseconds()) / 1000, "computeMs": float64(compute.Microseconds()) / 1000, "audioSamples": len(samples), "error": errorString(finishErr)})
		if finishErr != nil {
			return finishErr
		}
		if draft.Identity != id || !draft.Final || draft.Text == "" {
			return errors.New("invalid streaming final")
		}
	}
	emit(map[string]any{"event": "completed", "offlineSpeechCases": len(selected), "realtimeStreamingCases": 3, "cancellation": true, "reuseAfterCancellation": true})
	return nil
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
