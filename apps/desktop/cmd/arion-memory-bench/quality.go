package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"arion.local/core/domain"
)

// caseFile is benchmarks/configs/edit-learning-cases.json.
type caseFile struct {
	ID    string `json:"id"`
	Cases []struct {
		ID       string `json:"id"`
		Kind     string `json:"kind"`
		Language string `json:"language"`
		Inserted string `json:"inserted"`
		Baseline string `json:"baseline"`
		Settled  string `json:"settled"`
		Expect   []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"expect"`
	} `json:"cases"`
}

// extractionReport scores ExtractCorrections against labeled edits.
type extractionReport struct {
	Cases           int            `json:"cases"`
	Corrections     int            `json:"corrections"`
	ExpectedPairs   int            `json:"expectedPairs"`
	LearnedPairs    int            `json:"learnedPairs"`
	CorrectPairs    int            `json:"correctPairs"`
	Precision       float64        `json:"pairPrecision"`
	Recall          float64        `json:"pairRecall"`
	Negatives       int            `json:"negatives"`
	FalseLearned    int            `json:"negativesLearned"`
	FalseLearnRate  float64        `json:"falseLearningRate"`
	Risks           int            `json:"riskCases"`
	RisksLearned    int            `json:"riskCasesLearned"`
	RiskIDs         []string       `json:"riskCasesLearnedIds"`
	Missed          []string       `json:"missedCorrectionIds"`
	Verdicts        map[string]int `json:"verdicts"`
	ExtractLatencyU stats          `json:"extractLatencyUs"`
}

func scoreCases(path string) (extractionReport, error) {
	var file caseFile
	if err := readJSON(path, &file); err != nil {
		return extractionReport{}, err
	}
	r := extractionReport{Verdicts: map[string]int{}}
	var latencies []float64
	for _, c := range file.Cases {
		r.Cases++
		edit := domain.FieldEdit{Inserted: c.Inserted, Baseline: c.Baseline, Settled: c.Settled}
		start := time.Now()
		out := domain.ExtractCorrections(edit)
		latencies = append(latencies, float64(time.Since(start).Nanoseconds())/1e3)
		r.Verdicts[string(out.Verdict)]++
		switch c.Kind {
		case "correction":
			r.Corrections++
			r.ExpectedPairs += len(c.Expect)
			r.LearnedPairs += len(out.Pairs)
			hit := 0
			for _, e := range c.Expect {
				for _, p := range out.Pairs {
					if p.From == e.From && p.To == e.To {
						hit++
						break
					}
				}
			}
			r.CorrectPairs += hit
			if hit < len(c.Expect) {
				r.Missed = append(r.Missed, c.ID)
			}
		case "negative":
			r.Negatives++
			if len(out.Pairs) > 0 {
				r.FalseLearned++
			}
		case "risk":
			r.Risks++
			if len(out.Pairs) > 0 {
				r.RisksLearned++
				r.RiskIDs = append(r.RiskIDs, c.ID)
			}
		}
	}
	r.Precision = ratio(r.CorrectPairs, r.LearnedPairs)
	r.Recall = ratio(r.CorrectPairs, r.ExpectedPairs)
	r.FalseLearnRate = ratio(r.FalseLearned, r.Negatives)
	r.ExtractLatencyU = summarize(latencies)
	return r, nil
}

// corpusSample is the part of the private ASR corpus the replay needs.
type corpusSample struct {
	ID        string `json:"id"`
	Split     string `json:"split"`
	SpeakerID string `json:"speakerId"`
	Language  string `json:"language"`
	Unit      string `json:"unit"`
	Reference string `json:"reference"`
}

// hypothesis is one recorded recognition from a benchmark results file.
type hypothesis struct {
	CaseID    string `json:"caseId"`
	VariantID string `json:"variantId"`
	Response  struct {
		Candidates []struct {
			Text string `json:"text"`
		} `json:"candidates"`
	} `json:"response"`
}

func loadHypotheses(path, variant string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<26)
	for scanner.Scan() {
		var h hypothesis
		if err := json.Unmarshal(scanner.Bytes(), &h); err != nil {
			return nil, err
		}
		if h.VariantID != variant || len(h.Response.Candidates) == 0 {
			continue
		}
		out[h.CaseID] = h.Response.Candidates[0].Text
	}
	return out, scanner.Err()
}

// replayReport is the held-out effect of memory learned from dev
// corrections, replayed on recorded recognition (recognition is not re-run).
type replayReport struct {
	DevSamples      int                  `json:"devSamples"`
	DevVerdicts     map[string]int       `json:"devVerdicts"`
	LearnedEntries  int                  `json:"learnedEntries"`
	TestSamples     int                  `json:"testSamples"`
	HeldOut         map[string]effect    `json:"heldOut"`
	InSample        map[string]effect    `json:"inSampleCeiling"`
	RulesAgreement  *agreement           `json:"rulesAgreement,omitempty"`
	PromptTerms     stats                `json:"promptTermsPerTestSample"`
	PromptTermsAll  int                  `json:"promptTermsIfAllEntries"`
	LearnedExamples []domain.Replacement `json:"-"`
	PerSample       []map[string]any     `json:"-"`
}

// effect aggregates error units with memory off and on.
type effect struct {
	Samples       int     `json:"samples"`
	ReferenceUnit int     `json:"referenceUnits"`
	ErrorsOff     int     `json:"errorsMemoryOff"`
	ErrorsOn      int     `json:"errorsMemoryOn"`
	RateOff       float64 `json:"rateMemoryOff"`
	RateOn        float64 `json:"rateMemoryOn"`
	Changed       int     `json:"samplesChanged"`
	Improved      int     `json:"samplesImproved"`
	Worsened      int     `json:"samplesWorsened"`
}

// agreement checks that the offline rule correction equals the driver's
// local correction on the same recordings, which validates the replay.
type agreement struct {
	Compared int `json:"compared"`
	Equal    int `json:"equal"`
}

func replay(corpusPath, devPath, devVariant, testPath, testVariant, rulesVariant string) (replayReport, error) {
	var corpus struct {
		Samples []corpusSample `json:"samples"`
	}
	if err := readJSON(corpusPath, &corpus); err != nil {
		return replayReport{}, err
	}
	dev, err := loadHypotheses(devPath, devVariant)
	if err != nil {
		return replayReport{}, err
	}
	test, err := loadHypotheses(testPath, testVariant)
	if err != nil {
		return replayReport{}, err
	}
	base := domain.CorrectionPolicy{LatinCase: domain.LatinCaseNatural}
	r := replayReport{DevVerdicts: map[string]int{}, HeldOut: map[string]effect{}, InSample: map[string]effect{}}

	// Learn: the user corrects each inserted dev dictation to its reference,
	// in corpus order, as one person would over time. Test references are
	// never read for learning (EVAL-A04).
	var memory []domain.MemoryEntry
	now := time.Unix(1_800_000_000, 0)
	for _, s := range corpus.Samples {
		hyp, ok := dev[s.ID]
		if s.Split != "dev" || !ok || strings.TrimSpace(s.Reference) == "" {
			continue
		}
		r.DevSamples++
		inserted := domain.Correct(hyp, base).Text
		out := domain.ExtractCorrections(domain.FieldEdit{Inserted: inserted, Baseline: inserted, Settled: s.Reference})
		r.DevVerdicts[string(out.Verdict)]++
		for i, pair := range out.Pairs {
			memory, _ = domain.Learn(memory, pair, fmt.Sprintf("%s-%d", s.ID, i), now)
		}
		now = now.Add(time.Minute)
	}
	memory = domain.PruneMemory(memory, now)
	r.LearnedEntries = len(memory)
	r.LearnedExamples = domain.Replacements(memory)
	vocabulary := domain.NewVocabulary(r.LearnedExamples)
	r.PromptTermsAll = vocabulary.Len()
	withMemory := base
	withMemory.Vocabulary = vocabulary

	apply := func(split string, hyps map[string]string, into map[string]effect, record bool) []float64 {
		var terms []float64
		for _, s := range corpus.Samples {
			hyp, ok := hyps[s.ID]
			if s.Split != split || !ok || strings.TrimSpace(s.Reference) == "" {
				continue
			}
			off := domain.Correct(hyp, base).Text
			on := domain.Correct(hyp, withMemory)
			refUnits, eOff := errorCount(s.Reference, off, s.Unit)
			_, eOn := errorCount(s.Reference, on.Text, s.Unit)
			for _, key := range []string{s.Language, "overall"} {
				e := into[key]
				e.Samples++
				e.ReferenceUnit += refUnits
				e.ErrorsOff += eOff
				e.ErrorsOn += eOn
				if on.Text != off {
					e.Changed++
				}
				if eOn < eOff {
					e.Improved++
				}
				if eOn > eOff {
					e.Worsened++
				}
				into[key] = e
			}
			terms = append(terms, float64(len(vocabulary.Terms(on.Text, domain.MaxRecallTerms))))
			if record {
				r.PerSample = append(r.PerSample, map[string]any{"id": s.ID, "reference": s.Reference, "off": off, "on": on.Text,
					"errorsOff": eOff, "errorsOn": eOn, "recalled": on.Recalled})
			}
		}
		for key, e := range into {
			e.RateOff, e.RateOn = ratio(e.ErrorsOff, e.ReferenceUnit), ratio(e.ErrorsOn, e.ReferenceUnit)
			into[key] = e
		}
		return terms
	}
	r.PromptTerms = summarize(apply("test", test, r.HeldOut, true))
	for _, e := range r.HeldOut {
		r.TestSamples = max(r.TestSamples, e.Samples)
	}
	apply("dev", dev, r.InSample, false)

	if rulesVariant != "" {
		rules, err := loadHypotheses(testPath, rulesVariant)
		if err != nil {
			return r, err
		}
		a := agreement{}
		for id, hyp := range test {
			if driver, ok := rules[id]; ok {
				a.Compared++
				if domain.Correct(hyp, base).Text == driver {
					a.Equal++
				}
			}
		}
		r.RulesAgreement = &a
	}
	sort.Slice(r.LearnedExamples, func(i, j int) bool { return r.LearnedExamples[i].From < r.LearnedExamples[j].From })
	return r, nil
}

// The evaluator's normalization (tools/evaluation/src/metrics.mjs,
// arion-nfkc-lower-punctuation-space-v1). NFKC is approximated by folding
// fullwidth ASCII, the only compatibility forms in this corpus; the
// baseline error counts are checked against the harness report.
var (
	punctuation = regexp.MustCompile(`[\p{P}\p{S}]`)
	spaces      = regexp.MustCompile(`\s+`)
	merUnit     = regexp.MustCompile(`\p{Han}|[^\p{Han}\s]+`)
)

func normalize(text string) string {
	text = strings.Map(func(r rune) rune {
		if r >= 0xFF01 && r <= 0xFF5E {
			return r - 0xFEE0
		}
		if r == 0x3000 {
			return ' '
		}
		return r
	}, text)
	text = strings.ToLower(text)
	text = punctuation.ReplaceAllString(text, " ")
	return strings.TrimSpace(spaces.ReplaceAllString(text, " "))
}

func units(text, unit string) []string {
	n := normalize(text)
	switch unit {
	case "cer":
		var out []string
		for _, r := range n {
			if !unicode.IsSpace(r) {
				out = append(out, string(r))
			}
		}
		return out
	case "wer":
		if n == "" {
			return nil
		}
		return strings.Split(n, " ")
	default:
		return merUnit.FindAllString(n, -1)
	}
}

// errorCount returns reference units and the Levenshtein distance.
func errorCount(reference, hypothesis, unit string) (int, int) {
	ref, hyp := units(reference, unit), units(hypothesis, unit)
	prev := make([]int, len(hyp)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ref); i++ {
		cur := make([]int, len(hyp)+1)
		cur[0] = i
		for j := 1; j <= len(hyp); j++ {
			if ref[i-1] == hyp[j-1] {
				cur[j] = prev[j-1]
			} else {
				cur[j] = 1 + min(prev[j-1], prev[j], cur[j-1])
			}
		}
		prev = cur
	}
	return len(ref), prev[len(hyp)]
}
