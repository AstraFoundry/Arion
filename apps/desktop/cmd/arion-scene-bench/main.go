// Command arion-scene-bench scores scene selection (domain.DecideScene)
// against labelled target contexts and prints a content-free JSON summary:
// accuracy, per-scene precision and recall, the confusion matrix, which
// signal decided, and the decision time. Labels come from the corpus author,
// so it measures rule coverage of the listed apps and sites, not what users
// would choose.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"arion.local/core/domain"
)

type context struct {
	ID       string           `json:"id"`
	App      string           `json:"app"`
	Field    domain.FieldKind `json:"field"`
	Site     string           `json:"site"`
	Expected domain.Scene     `json:"expected"`
}

type miss struct {
	ID       string `json:"id"`
	Expected string `json:"expected"`
	Got      string `json:"got"`
	Route    string `json:"route"`
}

type perScene struct {
	Expected  int     `json:"expected"`
	Predicted int     `json:"predicted"`
	Correct   int     `json:"correct"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
}

func main() {
	path := flag.String("contexts", "", "labelled contexts JSON")
	showMisses := flag.Bool("misses", false, "include misclassified context IDs")
	flag.Parse()
	if *path == "" {
		flag.Usage()
		os.Exit(2)
	}
	data, err := os.ReadFile(*path)
	check(err)
	var corpus struct{ Contexts []context }
	check(json.Unmarshal(data, &corpus))
	if len(corpus.Contexts) == 0 {
		check(fmt.Errorf("no contexts in %s", *path))
	}
	confusion := map[string]map[string]int{}
	routes := map[string]int{}
	scenes := map[string]*perScene{}
	at := func(s string) *perScene {
		if scenes[s] == nil {
			scenes[s] = &perScene{}
		}
		return scenes[s]
	}
	var misses []miss
	correct := 0
	for _, c := range corpus.Contexts {
		if !c.Expected.Known() {
			check(fmt.Errorf("context %s expects unknown scene %q", c.ID, c.Expected))
		}
		d := domain.DecideScene(domain.SceneHints{App: c.App, Field: c.Field, Site: c.Site}, nil)
		want, got := string(c.Expected), string(d.Scene)
		if confusion[want] == nil {
			confusion[want] = map[string]int{}
		}
		confusion[want][got]++
		routes[string(d.Route)]++
		at(want).Expected++
		at(got).Predicted++
		if want == got {
			correct++
			at(want).Correct++
		} else {
			misses = append(misses, miss{c.ID, want, got, string(d.Route)})
		}
	}
	for _, s := range scenes {
		s.Precision, s.Recall = ratio(s.Correct, s.Predicted), ratio(s.Correct, s.Expected)
	}
	// Decision time over the whole set, repeated for a stable mean.
	const rounds = 2000
	start := time.Now()
	for range rounds {
		for _, c := range corpus.Contexts {
			domain.DecideScene(domain.SceneHints{App: c.App, Field: c.Field, Site: c.Site}, nil)
		}
	}
	perDecision := float64(time.Since(start).Nanoseconds()) / float64(rounds*len(corpus.Contexts))
	sort.Slice(misses, func(i, j int) bool { return misses[i].ID < misses[j].ID })
	summary := map[string]any{
		"contexts": len(corpus.Contexts), "correct": correct, "accuracy": ratio(correct, len(corpus.Contexts)),
		"scenes": scenes, "confusion": confusion, "routes": routes, "decisionNs": perDecision,
	}
	if *showMisses {
		summary["misses"] = misses
	}
	out, err := json.MarshalIndent(summary, "", "  ")
	check(err)
	fmt.Println(string(out))
}

func ratio(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
