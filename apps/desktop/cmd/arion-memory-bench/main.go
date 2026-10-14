// Command arion-memory-bench evaluates edit learning and vocabulary recall
// (ADR 014): extraction quality on labeled edits, the held-out effect of
// memory learned from dev-split corrections replayed on recorded test-split
// recognition, and the latency, scaling, concurrency and fault behaviour of
// the recall index, the encrypted store and the edit tracker.
//
// The output directory must not exist. summary.json holds aggregates only;
// private.jsonl keeps per-sample texts and stays in .workspace.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"arion.local/core/domain"
)

type summary struct {
	Version     int               `json:"version"`
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	Environment map[string]string `json:"environment"`
	Inputs      map[string]string `json:"inputs"`
	Extraction  extractionReport  `json:"extraction"`
	Replay      *replayReport     `json:"replay,omitempty"`
	Recall      []recallScale     `json:"recallScaling"`
	Store       []storeScale      `json:"storeScaling"`
	Extract     []extractScale    `json:"extractScaling"`
	Tracker     trackerReport     `json:"tracker"`
	Concurrency concurrencyReport `json:"concurrency"`
	Faults      []faultReport     `json:"faults"`
}

func main() {
	cases := flag.String("cases", "benchmarks/configs/edit-learning-cases.json", "labeled edit scenarios")
	corpus := flag.String("corpus", "", "private ASR corpus manifest (optional)")
	dev := flag.String("dev", "", "results.jsonl with dev-split recognition")
	devVariant := flag.String("dev-variant", "xasr-160-xasr-final", "variant whose dev recognition is learned from")
	test := flag.String("test", "", "results.jsonl with test-split recognition")
	testVariant := flag.String("test-variant", "xasr-160-xasr-final", "variant whose test recognition is corrected")
	rulesVariant := flag.String("rules-variant", "xasr-160-xasr-final-rules", "test variant with the driver's local correction, to validate the replay")
	out := flag.String("out", "", "new output directory")
	reps := flag.Int("reps", 20, "repetitions per latency measurement")
	flag.Parse()
	if err := run(*cases, *corpus, *dev, *devVariant, *test, *testVariant, *rulesVariant, *out, *reps); err != nil {
		fmt.Fprintln(os.Stderr, "arion-memory-bench:", err)
		os.Exit(1)
	}
}

func run(cases, corpus, dev, devVariant, test, testVariant, rulesVariant, out string, reps int) error {
	if out == "" {
		return fmt.Errorf("-out is required")
	}
	if err := os.Mkdir(out, 0o700); err != nil {
		return fmt.Errorf("output directory must be new: %w", err)
	}
	scratch, err := os.MkdirTemp("", "arion-memory-bench")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)

	s := summary{Version: 1, ID: "edit-learning-v1", Status: "failed", Environment: environment(),
		Inputs: map[string]string{"cases": filepath.Base(cases), "devVariant": devVariant, "testVariant": testVariant}}
	defer func() { _ = writeJSON(filepath.Join(out, "summary.json"), s) }()

	step("extraction")
	if s.Extraction, err = scoreCases(cases); err != nil {
		return err
	}
	texts := []string{"打开阿里昂的设置，然后把这个 PR merge 到 main 分支之前先跑一下 CI。", "ask jason about the release before friday",
		"我们用 Kubernetes 部署这个服务，内存占用涨到了八个 G。"}
	var seed []domain.Replacement
	if corpus != "" {
		step("replay")
		r, err := replay(corpus, dev, devVariant, test, testVariant, rulesVariant)
		if err != nil {
			return err
		}
		s.Replay = &r
		seed = r.LearnedExamples
		texts = texts[:0]
		for _, row := range r.PerSample {
			texts = append(texts, row["off"].(string))
		}
		if err := writePrivate(filepath.Join(out, "private.jsonl"), r); err != nil {
			return err
		}
	}
	step("recall scaling")
	s.Recall = measureRecall([]int{0, 10, 100, 1000, domain.MaxMemoryEntries, 10000}, seed, texts, reps)
	step("store scaling")
	if s.Store, err = measureStore(scratch, []int{0, 100, 1000, domain.MaxMemoryEntries - 1}, reps); err != nil {
		return err
	}
	step("extract scaling")
	s.Extract = measureExtract(reps * 10)
	step("tracker")
	s.Tracker = measureTracker(5)
	step("concurrency")
	if s.Concurrency, err = measureConcurrency(scratch, texts, 8, 3*time.Second); err != nil {
		return err
	}
	step("faults")
	if s.Faults, err = injectFaults(scratch); err != nil {
		return err
	}
	s.Status = "completed"
	return nil
}

func step(name string) { fmt.Fprintf(os.Stderr, "== %s\n", name) }

func environment() map[string]string {
	env := map[string]string{"goos": runtime.GOOS, "goarch": runtime.GOARCH, "go": runtime.Version(),
		"cpus": fmt.Sprint(runtime.NumCPU()), "at": time.Now().UTC().Format(time.RFC3339)}
	if b, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		env["cpu"] = strings.TrimSpace(string(b))
	}
	if b, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		env["gitRevision"] = strings.TrimSpace(string(b))
	}
	if b, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		env["gitDirty"] = fmt.Sprint(len(strings.TrimSpace(string(b))) > 0)
	}
	return env
}

func writePrivate(path string, r replayReport) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	for _, pair := range r.LearnedExamples {
		if err := enc.Encode(map[string]any{"learned": pair}); err != nil {
			return err
		}
	}
	for _, row := range r.PerSample {
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}
