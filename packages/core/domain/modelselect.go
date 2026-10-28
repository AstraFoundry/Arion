package domain

import (
	"math"
	"slices"
	"sort"
)

// ModelEvaluation is the aggregate, content-free result of a completed local
// benchmark run for one recognition bundle on one hardware class.
type ModelEvaluation struct {
	RunID string
	// Hardware is the platform tag the run was measured on, e.g. "darwin-arm64".
	Hardware string
	// ErrorRates holds corpus micro-averaged rates per language stratum:
	// "zh" (CER), "en" (WER) and "mixed" (MER).
	ErrorRates map[string]float64
	// StopToReviewP95Ms is speech end to review-ready, warm, p95.
	StopToReviewP95Ms float64
	// FirstPartialP95Ms is speech onset to first nonempty preview, warm, p95.
	FirstPartialP95Ms float64
	// RealTimeFactor is duration-weighted inference time / audio time.
	RealTimeFactor float64
	Observations   int
}

// ModelOption is one approved, provisionable recognition bundle.
type ModelOption struct {
	ID string
	// Adopted marks an explicit product migration, independently of benchmark gates.
	Adopted    bool
	Platforms  []string
	Languages  []string
	SizeBytes  int64
	Evaluation []ModelEvaluation
}

// SelectionPolicy states release gates from 006 Models. Budgets are the
// provisional engineering targets; a bundle failing them is not eligible.
type SelectionPolicy struct {
	Platform          string
	RequiredLanguages []string
	MaxStopToReviewMs float64
	MaxRealTimeFactor float64
	MinObservations   int
}

// DefaultSelectionPolicy mirrors the 006 provisional budgets.
func DefaultSelectionPolicy(platform string) SelectionPolicy {
	return SelectionPolicy{
		Platform:          platform,
		RequiredLanguages: []string{"zh", "en", "mixed"},
		MaxStopToReviewMs: 1000,
		MaxRealTimeFactor: 1,
		MinObservations:   20,
	}
}

// Selection explains which bundle loads by default and why.
type Selection struct {
	Option     ModelOption
	Evaluation ModelEvaluation
	Score      float64
	// LatencyMeasured is false when no run on this hardware class exists and
	// the choice rests on accuracy measured elsewhere. The UI states this.
	LatencyMeasured bool
}

// Score is the unweighted mean of the required language error rates: each
// language counts equally whatever its sample count.
func (p SelectionPolicy) Score(e ModelEvaluation) (float64, bool) {
	if len(p.RequiredLanguages) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, language := range p.RequiredLanguages {
		rate, ok := e.ErrorRates[language]
		if !ok || math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 {
			return 0, false
		}
		sum += rate
	}
	return sum / float64(len(p.RequiredLanguages)), true
}

func (p SelectionPolicy) eligible(e ModelEvaluation) bool {
	return e.Hardware == p.Platform && e.Observations >= p.MinObservations &&
		e.StopToReviewP95Ms <= p.MaxStopToReviewMs && e.RealTimeFactor < p.MaxRealTimeFactor
}

// SelectDefaultModel first honors an explicitly adopted platform bundle,
// reporting measurements only when they exist. Otherwise it returns the best evaluated bundle: the
// lowest mean error among bundles whose latency/RTF measured on this hardware
// class pass the gates. When no bundle was measured on this hardware class,
// recognition accuracy is hardware-independent for identical weights and
// runtime, so the most accurate bundle measured elsewhere is chosen with
// LatencyMeasured false. Ties prefer lower latency, then smaller downloads,
// then the ID. Unmeasured bundles are never chosen.
func SelectDefaultModel(options []ModelOption, policy SelectionPolicy) (Selection, error) {
	var adopted *ModelOption
	for i := range options {
		option := &options[i]
		if !option.Adopted || !slices.Contains(option.Platforms, policy.Platform) || !containsAll(option.Languages, policy.RequiredLanguages) {
			continue
		}
		if adopted != nil {
			return Selection{}, Fail(CodeModelUnsupported, "multiple adopted models for platform")
		}
		adopted = option
	}
	if adopted != nil {
		selection := Selection{Option: *adopted}
		for _, evaluation := range adopted.Evaluation {
			if score, ok := policy.Score(evaluation); ok && evaluation.Hardware == policy.Platform && evaluation.Observations >= policy.MinObservations {
				selection.Evaluation, selection.Score, selection.LatencyMeasured = evaluation, score, true
				break
			}
		}
		return selection, nil
	}

	var local, elsewhere []Selection
	for _, option := range options {
		if !slices.Contains(option.Platforms, policy.Platform) || !containsAll(option.Languages, policy.RequiredLanguages) {
			continue
		}
		for _, evaluation := range option.Evaluation {
			score, ok := policy.Score(evaluation)
			if !ok || evaluation.Observations < policy.MinObservations {
				continue
			}
			candidate := Selection{Option: option, Evaluation: evaluation, Score: score}
			if evaluation.Hardware == policy.Platform {
				if policy.eligible(evaluation) {
					candidate.LatencyMeasured = true
					local = append(local, candidate)
				}
			} else {
				elsewhere = append(elsewhere, candidate)
			}
		}
	}
	candidates := local
	if len(candidates) == 0 && !policy.measuredLocally(options) {
		candidates = elsewhere
	}
	if len(candidates) == 0 {
		return Selection{}, Fail(CodeEvaluationUnavailable, "no evaluated bundle passes the gates for "+policy.Platform)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.Score != b.Score {
			return a.Score < b.Score
		}
		if a.Evaluation.StopToReviewP95Ms != b.Evaluation.StopToReviewP95Ms {
			return a.Evaluation.StopToReviewP95Ms < b.Evaluation.StopToReviewP95Ms
		}
		if a.Option.SizeBytes != b.Option.SizeBytes {
			return a.Option.SizeBytes < b.Option.SizeBytes
		}
		return a.Option.ID < b.Option.ID
	})
	return candidates[0], nil
}

// measuredLocally reports whether any run exists on this hardware class; if
// one does, its gates decide and other hardware cannot override them.
func (p SelectionPolicy) measuredLocally(options []ModelOption) bool {
	for _, option := range options {
		for _, evaluation := range option.Evaluation {
			if evaluation.Hardware == p.Platform {
				return true
			}
		}
	}
	return false
}

func containsAll(values, wanted []string) bool {
	for _, want := range wanted {
		if !slices.Contains(values, want) {
			return false
		}
	}
	return true
}
