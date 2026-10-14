// Command arion-rewrite-bench drives the production LLM worker over a text
// corpus. For each case it replays the live schedule — one rewrite per
// closed speech segment, then a final rewrite of the whole text — and
// records proposals, validation verdicts and engine timings as JSON lines.
// It is a development benchmark of the rewrite stage on fixed text.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/client"
)

type sample struct {
	ID             string   `json:"id"`
	Language       string   `json:"language"`
	Speech         string   `json:"speech"`
	SourceText     string   `json:"sourceText"`
	Scenario       string   `json:"scenario"`
	Mode           string   `json:"mode"`
	ApprovedMemory []string `json:"approvedMemory"`
	// ContextFixture holds a context case's window image, its true visible
	// text, cursor text and history.
	ContextFixture *struct {
		Image       string   `json:"image"`
		ScreenTruth string   `json:"screenTruth"`
		Before      string   `json:"before"`
		After       string   `json:"after"`
		History     []string `json:"history"`
	} `json:"contextFixture"`
	// Target is the focused app, field and site the case is dictated into.
	Target struct {
		App   string           `json:"app"`
		Field domain.FieldKind `json:"field"`
		Site  string           `json:"site"`
	} `json:"target"`
}

type live struct {
	Source  string  `json:"source"`
	Text    string  `json:"text"`
	Verdict string  `json:"verdict"`
	TotalMs float64 `json:"totalMs"`
}

type row struct {
	CaseID     string `json:"caseId"`
	VariantID  string `json:"variantId"`
	Scene      string `json:"scene"`
	SceneRoute string `json:"sceneRoute,omitempty"`
	// Context sources applied, their bounded size and the prompt used.
	ContextSources []string `json:"contextSources,omitempty"`
	ContextChars   int      `json:"contextChars,omitempty"`
	Template       string   `json:"template,omitempty"`
	ContextTerms   []string `json:"contextTerms,omitempty"`
	// Source is the text sent to the rewriter, after rule correction when
	// -correct is set.
	Source       string  `json:"source"`
	Live         []live  `json:"live"`
	Proposal     string  `json:"proposal"`
	Verdict      string  `json:"verdict"`
	Text         string  `json:"text"`
	Bypassed     bool    `json:"bypassed"`
	Truncated    bool    `json:"truncated"`
	FinalMs      float64 `json:"generationMs"`
	FirstTokenMs float64 `json:"firstTokenMs"`
	PrefillMs    float64 `json:"prefillMs"`
	DecodeMs     float64 `json:"decodeMs"`
	InputTokens  int     `json:"inputTokens"`
	ReusedTokens int     `json:"reusedTokens"`
	OutputTokens int     `json:"outputTokens"`
	PeakRSS      int64   `json:"peakRssBytes"`
}

func main() {
	corpusPath := flag.String("corpus", "", "text corpus JSON")
	worker := flag.String("worker", "", "arion-llm executable")
	runtimeDir := flag.String("runtime", "", "GenAI runtime directory")
	model := flag.String("model", "", "GenAI model directory")
	variant := flag.String("variant", "", "variant id recorded in rows")
	liveEdits := flag.Bool("live", true, "rewrite each closed segment before the final request")
	threads := flag.Int("threads", 4, "intra-op threads")
	correct := flag.Bool("correct", false, "apply the product's default rule correction (fillers) before rewriting, as dictation does")
	contextFlag := flag.String("context", "", "comma-separated context sources: cursor, history, screen")
	screenFlag := flag.String("screen", "oracle", "screen text: oracle (true visible text), accurate or fast (Vision OCR rows from -ocr)")
	ocrPath := flag.String("ocr", "", "arion-ocr JSONL for -screen accurate|fast")
	contextAs := flag.String("context-as", "prompt", "prompt: context sections and the context template; terms: spellings extracted from context into the plain prompt's terms; both")
	sceneMode := flag.String("scene", "corpus", "corpus: each case's scenario; auto: decided from its target like the product; or a fixed scene")
	out := flag.String("out", "", "output JSONL path (must not exist)")
	flag.Parse()
	if *sceneMode != "corpus" && *sceneMode != "auto" && !domain.Scene(*sceneMode).Known() {
		fmt.Fprintf(os.Stderr, "unknown scene mode %q\n", *sceneMode)
		os.Exit(2)
	}
	if *corpusPath == "" || *worker == "" || *runtimeDir == "" || *model == "" || *variant == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	data, err := os.ReadFile(*corpusPath)
	check(err)
	var corpus struct{ Samples []sample }
	check(json.Unmarshal(data, &corpus))
	sources := map[string]bool{}
	for _, name := range strings.Split(*contextFlag, ",") {
		switch name = strings.TrimSpace(name); name {
		case "":
		case "cursor", "history", "screen":
			sources[name] = true
		default:
			check(fmt.Errorf("unknown context source %q", name))
		}
	}
	if *contextAs != "prompt" && *contextAs != "terms" && *contextAs != "both" {
		check(fmt.Errorf("unknown -context-as %q", *contextAs))
	}
	ocr := map[string]string{}
	if sources["screen"] && *screenFlag != "oracle" {
		ocr = loadOCR(*ocrPath, *screenFlag)
	}
	file, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	check(err)
	defer file.Close()
	ctx := context.Background()
	c, info, err := client.Start(ctx, client.Config{Worker: *worker, Runtime: *runtimeDir, Model: *model, Threads: *threads, MaxLength: 4096, MaxNewTokens: 384}, os.Stderr)
	check(err)
	defer c.Close()
	fmt.Fprintf(os.Stderr, "loaded %s prompt=%s prefix=%d in %.0f ms\n", *variant, info.PromptID, info.PrefixTokens, info.LoadMs)
	encoder := json.NewEncoder(file)
	for i, s := range corpus.Samples {
		id := contracts.Identity{SessionID: s.ID, Generation: uint64(i + 1)}
		r := row{CaseID: s.ID, VariantID: *variant}
		switch *sceneMode {
		case "corpus":
			r.Scene = s.Scenario
		case "auto":
			d := domain.DecideScene(domain.SceneHints{App: s.Target.App, Field: s.Target.Field, Site: s.Target.Site}, nil)
			r.Scene, r.SceneRoute = string(d.Scene), string(d.Route)
		default:
			r.Scene = *sceneMode
		}
		source := s.SourceText
		if *correct {
			source = domain.Correct(source, domain.CorrectionPolicy{RemoveFillers: true}).Text
		}
		r.Source = source
		scene := domain.Scene(r.Scene)
		var rc domain.RewriteContext
		if f := s.ContextFixture; f != nil {
			if sources["screen"] {
				rc.Screen = f.ScreenTruth
				if *screenFlag != "oracle" {
					rc.Screen = ocr[s.ID]
				}
			}
			if sources["cursor"] {
				rc.Before, rc.After = f.Before, f.After
			}
			if sources["history"] {
				rc.History = f.History
			}
		}
		rc = rc.Bounded()
		for _, name := range []string{"cursor", "history", "screen"} {
			if sources[name] {
				r.ContextSources = append(r.ContextSources, name)
			}
		}
		if !rc.Empty() {
			r.ContextChars = len([]rune(strings.TrimSpace(rc.Evidence())))
		}
		// Validation always sees the whole context, as the product does;
		// -context-as only changes what reaches the model.
		evidence := rc
		terms := s.ApprovedMemory
		if *contextAs != "prompt" {
			r.ContextTerms = domain.ContextTerms(rc, source)
			terms = append(append([]string{}, terms...), r.ContextTerms...)
			if *contextAs == "terms" {
				rc = domain.RewriteContext{}
			}
		}
		if s.Mode == "verbatim" {
			// Verbatim dictation never reaches the model (009 step 5).
			r.Bypassed, r.Verdict, r.Text = true, "bypassed", source
			check(encoder.Encode(r))
			continue
		}
		request := func(text string, live bool) (ports.RewriteResult, float64) {
			start := time.Now()
			result, err := c.Rewrite(ctx, ports.RewriteRequest{Identity: id, Text: text, Scene: string(domain.ModelScene(scene, text)), Terms: terms, Context: rc, Live: live})
			check(err)
			return result, float64(time.Since(start).Microseconds()) / 1000
		}
		if *liveEdits {
			for _, prefix := range stablePrefixes(s, source) {
				result, ms := request(prefix, true)
				text := domain.AlignLiveEnding(prefix, result.Text)
				verdict := domain.ValidateRewrite(domain.RewriteCheck{Source: prefix, Candidate: text, Truncated: result.Truncated, Terms: terms, Context: evidence})
				r.Live = append(r.Live, live{Source: prefix, Text: text, Verdict: string(verdict), TotalMs: ms})
			}
		}
		result, ms := request(source, false)
		verdict := domain.ValidateRewrite(domain.RewriteCheck{Source: source, Candidate: result.Text, Truncated: result.Truncated, Terms: terms, Context: evidence})
		r.Template = result.Template
		r.Proposal, r.Verdict, r.Truncated = result.Text, string(verdict), result.Truncated
		r.Text = source
		if verdict == domain.RewriteAccepted {
			r.Text = result.Text
		}
		r.Text = domain.FinishScene(scene, r.Text)
		r.FinalMs, r.PrefillMs, r.DecodeMs = ms, result.PrefillMs, result.DecodeMs
		r.InputTokens, r.ReusedTokens, r.OutputTokens = result.InputTokens, result.ReusedTokens, result.OutputTokens
		c.End(id)
		check(encoder.Encode(r))
		fmt.Fprintf(os.Stderr, "%-18s %-7s %-20s %6.0f ms reuse=%d/%d\n", s.ID, r.Scene, verdict, ms, result.ReusedTokens, result.InputTokens)
	}
}

// stablePrefixes are the cumulative texts of closed segments, excluding the
// complete utterance: what a live edit sees while the speaker continues.
// Segment boundaries come from the case's pause-marked speech script, or
// its commas when it has none.
func stablePrefixes(s sample, sourceText string) []string {
	speech := s.Speech
	if speech == "" {
		speech = strings.NewReplacer("，", "|", ", ", "|").Replace(strings.TrimRight(sourceText, "。.!?！？"))
	}
	parts := strings.Split(speech, "|")
	if len(parts) < 2 {
		return nil
	}
	var out []string
	cursor := 0
	source := []rune(sourceText)
	for _, part := range parts[:len(parts)-1] {
		// Advance through the source past this spoken part and its trailing
		// punctuation, so the prefix uses the source's own characters.
		cursor = advance(source, cursor, []rune(part))
		if cursor <= 0 {
			return out
		}
		out = append(out, string(source[:cursor]))
	}
	return out
}

func advance(source []rune, from int, part []rune) int {
	matched := 0
	i := from
	for i < len(source) && matched < len(part) {
		if source[i] == part[matched] {
			matched++
		} else if matched > 0 && !isSkippable(source[i]) {
			return -1
		}
		i++
	}
	for i < len(source) && isSkippable(source[i]) {
		i++
	}
	if matched < len(part) {
		return -1
	}
	return i
}

func isSkippable(r rune) bool { return strings.ContainsRune("，,。. 、；;：:！!？?", r) }

// loadOCR reads one recognition level's text per case from arion-ocr rows.
func loadOCR(path, level string) map[string]string {
	if path == "" {
		check(fmt.Errorf("-ocr is required for -screen %s", level))
	}
	data, err := os.ReadFile(path)
	check(err)
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var r struct{ CaseID, Level, Text string }
		check(json.Unmarshal([]byte(line), &r))
		if r.Level == level {
			out[r.CaseID] = r.Text
		}
	}
	return out
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
