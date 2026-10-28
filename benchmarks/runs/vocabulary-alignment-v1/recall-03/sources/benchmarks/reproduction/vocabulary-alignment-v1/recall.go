// Run from the repository root. This driver observes production vocabulary
// correction and recall; evaluator labels are held in a separate file.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"arion.local/core/domain"
)

type sample struct {
	ID       string               `json:"id"`
	Source   string               `json:"source"`
	Previous string               `json:"previous"`
	Entries  []domain.Replacement `json:"entries"`
}

func main() {
	if len(os.Args) != 2 {
		panic("input JSON path required")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var cases []sample
	if err := json.Unmarshal(raw, &cases); err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	for _, c := range cases {
		v := domain.NewVocabulary(c.Entries)
		corrected := domain.Correct(c.Source, domain.CorrectionPolicy{Vocabulary: v}).Text
		terms := v.TermsForSources(domain.MaxRecallTerms, corrected, c.Previous)
		latencies := make([]int64, 100)
		for i := range latencies {
			started := time.Now()
			domain.Correct(c.Source, domain.CorrectionPolicy{Vocabulary: v})
			v.TermsForSources(domain.MaxRecallTerms, corrected, c.Previous)
			latencies[i] = time.Since(started).Nanoseconds()
		}
		if err := encoder.Encode(map[string]any{"id": c.ID, "source": c.Source, "corrected": corrected, "terms": terms, "latencyNs": latencies}); err != nil {
			panic(err)
		}
	}
	fmt.Fprintln(os.Stderr, "Production correction/recall observations complete; no ASR or model inference in this driver.")
}
