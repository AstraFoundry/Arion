package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/vocabulary"
	"arion.local/desktop/infrastructure/wordseg"
)

// stats are latency quantiles in the unit named by the field.
type stats struct {
	Count int     `json:"count"`
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Max   float64 `json:"max"`
}

func summarize(values []float64) stats {
	if len(values) == 0 {
		return stats{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	q := func(p float64) float64 {
		pos := float64(len(sorted)-1) * p
		lo := int(pos)
		hi := min(lo+1, len(sorted)-1)
		return sorted[lo] + (sorted[hi]-sorted[lo])*(pos-float64(lo))
	}
	return stats{Count: len(sorted), P50: q(0.5), P95: q(0.95), P99: q(0.99), Max: sorted[len(sorted)-1]}
}

func micros(d time.Duration) float64 { return float64(d.Nanoseconds()) / 1e3 }
func millis(d time.Duration) float64 { return float64(d.Nanoseconds()) / 1e6 }

// syntheticEntries mixes learned real pairs with generated Latin and Han
// terms, so lookups see realistic first-rune collisions.
func syntheticEntries(n int, seed []domain.Replacement, rng *rand.Rand) []domain.Replacement {
	han := []rune("阿里昂杰森飞书送火纵晓王小内存部署服务模型芯片间距流程信息数组函数变量分支合并提交日志调试接口")
	out := append([]domain.Replacement(nil), seed[:min(len(seed), n)]...)
	for i := len(out); i < n; i++ {
		if i%2 == 0 {
			out = append(out, domain.Replacement{From: fmt.Sprintf("term%05d", i), To: fmt.Sprintf("Term%05d", i)})
			continue
		}
		var b strings.Builder
		for k := 0; k < 2+rng.Intn(3); k++ {
			b.WriteRune(han[rng.Intn(len(han))])
		}
		out = append(out, domain.Replacement{From: b.String(), To: fmt.Sprintf("T%d", i)})
	}
	return out
}

// legacyApply is the replacement loop Arion used before the index: one
// case-insensitive scan of the text per entry. It is the baseline the
// index is measured against.
func legacyApply(s string, reps []domain.Replacement) string {
	for _, rep := range reps {
		if rep.From == "" || rep.From == rep.To {
			continue
		}
		lowerS, lowerFrom := strings.ToLower(s), strings.ToLower(rep.From)
		if len(lowerS) != len(s) || len(lowerFrom) != len(rep.From) {
			s = strings.ReplaceAll(s, rep.From, rep.To)
			continue
		}
		var b strings.Builder
		for i := 0; i < len(s); {
			k := strings.Index(lowerS[i:], lowerFrom)
			if k < 0 {
				b.WriteString(s[i:])
				break
			}
			b.WriteString(s[i : i+k])
			b.WriteString(rep.To)
			i += k + len(rep.From)
		}
		s = b.String()
	}
	return s
}

// recallScale measures building, applying and recalling memory against the
// number of entries, on real dictation texts.
type recallScale struct {
	Entries      int     `json:"entries"`
	BuildMs      float64 `json:"buildMs"`
	ApplyUs      stats   `json:"applyUs"`
	TermsUs      stats   `json:"termsUs"`
	CorrectUs    stats   `json:"correctUs"`
	LegacyUs     stats   `json:"legacyApplyUs"`
	MaxTermsSent int     `json:"maxPromptTerms"`
}

func measureRecall(sizes []int, seed []domain.Replacement, texts []string, reps int) []recallScale {
	rng := rand.New(rand.NewSource(1729))
	var out []recallScale
	for _, n := range sizes {
		entries := syntheticEntries(n, seed, rng)
		start := time.Now()
		v := domain.NewVocabulary(entries)
		row := recallScale{Entries: n, BuildMs: millis(time.Since(start))}
		var apply, terms, correct, legacy []float64
		policy := domain.CorrectionPolicy{Vocabulary: v}
		for r := 0; r < reps; r++ {
			for _, text := range texts {
				t := time.Now()
				v.Apply(text)
				apply = append(apply, micros(time.Since(t)))
				t = time.Now()
				got := v.Terms(text, domain.MaxRecallTerms)
				terms = append(terms, micros(time.Since(t)))
				row.MaxTermsSent = max(row.MaxTermsSent, len(got))
				t = time.Now()
				domain.Correct(text, policy)
				correct = append(correct, micros(time.Since(t)))
				if r == 0 {
					t = time.Now()
					legacyApply(text, entries)
					legacy = append(legacy, micros(time.Since(t)))
				}
			}
		}
		row.ApplyUs, row.TermsUs, row.CorrectUs, row.LegacyUs = summarize(apply), summarize(terms), summarize(correct), summarize(legacy)
		out = append(out, row)
	}
	return out
}

// storeScale measures the encrypted, fsynced store against its size.
type storeScale struct {
	Entries   int   `json:"entries"`
	FileBytes int64 `json:"fileBytes"`
	LearnMs   stats `json:"learnMs"`
	RecallMs  stats `json:"recallMs"`
	ListMs    stats `json:"listMs"`
}

func measureStore(dir string, sizes []int, reps int) ([]storeScale, error) {
	rng := rand.New(rand.NewSource(7))
	key := bytes.Repeat([]byte{9}, 32)
	var out []storeScale
	for _, n := range sizes {
		path := filepath.Join(dir, fmt.Sprintf("store-%d.bin", n))
		store, err := vocabulary.Open(path, key)
		if err != nil {
			return nil, err
		}
		now := time.Now()
		if n > 0 {
			if _, err := store.Learn(syntheticEntries(n, nil, rng), now); err != nil {
				return nil, err
			}
		}
		row := storeScale{Entries: n}
		var learn, recall, list []float64
		for r := 0; r < reps; r++ {
			pair := domain.Replacement{From: fmt.Sprintf("probe%d", r), To: "Probe"}
			t := time.Now()
			if _, err := store.Learn([]domain.Replacement{pair}, now); err != nil {
				return nil, err
			}
			learn = append(learn, millis(time.Since(t)))
			t = time.Now()
			if err := store.Recall([]domain.Replacement{pair}, now); err != nil {
				return nil, err
			}
			recall = append(recall, millis(time.Since(t)))
			t = time.Now()
			if _, err := store.List(); err != nil {
				return nil, err
			}
			list = append(list, millis(time.Since(t)))
			// Keep the size constant: undo the probe.
			if _, err := store.Learn([]domain.Replacement{{From: "Probe", To: pair.From}}, now); err != nil {
				return nil, err
			}
		}
		info, _ := os.Stat(path)
		row.FileBytes = info.Size()
		row.LearnMs, row.RecallMs, row.ListMs = summarize(learn), summarize(recall), summarize(list)
		out = append(out, row)
	}
	return out, nil
}

// extractScale measures ExtractCorrections against the inserted length.
type extractScale struct {
	Runes int   `json:"insertedRunes"`
	Us    stats `json:"latencyUs"`
}

func measureExtract(reps int) []extractScale {
	var out []extractScale
	for _, n := range []int{50, 200, 1000, domain.MaxEditRunes} {
		words := make([]string, 0, n/5)
		for len(strings.Join(words, " ")) < n {
			words = append(words, fmt.Sprintf("w%03d", len(words)%997))
		}
		inserted := string([]rune(strings.Join(words, " "))[:n])
		settled := strings.Replace(inserted, "w002", "W-two", 1)
		var lat []float64
		for r := 0; r < reps; r++ {
			t := time.Now()
			domain.ExtractCorrections(domain.FieldEdit{Inserted: inserted, Baseline: inserted, Settled: settled}, wordseg.Segmenter{})
			lat = append(lat, micros(time.Since(t)))
		}
		out = append(out, extractScale{Runes: len([]rune(inserted)), Us: summarize(lat)})
	}
	return out
}

// trackerReport covers the tracker's effect on the dictation path and its
// behaviour with a host that stops answering, at production timings.
type trackerReport struct {
	TrackUs          stats   `json:"trackCallUs"`
	SettleUs         stats   `json:"settleCallUs"`
	SettleToReport   stats   `json:"settleToReportMs"`
	HungAbandonMs    float64 `json:"hungHostAbandonMs"`
	HungReads        int     `json:"hungHostReads"`
	ReadsPerIdleEdit float64 `json:"readsPerIdleEdit"`
}

type benchField struct {
	mu      sync.Mutex
	text    string
	hang    bool
	focused bool
}

func (f *benchField) Follow(contracts.TargetToken) bool { return true }
func (f *benchField) Release(contracts.TargetToken)     {}
func (f *benchField) ReadField(ctx context.Context, _ contracts.TargetToken) (ports.FieldRead, error) {
	f.mu.Lock()
	hang, read := f.hang, ports.FieldRead{Text: f.text, Focused: f.focused}
	f.mu.Unlock()
	if hang {
		<-ctx.Done()
		return ports.FieldRead{}, errors.New("host not answering")
	}
	return read, nil
}

func measureTracker(reps int) trackerReport {
	var r trackerReport
	var track, settle, settleReport []float64
	for i := 0; i < reps; i++ {
		reports := make(chan application.EditReport, 1)
		tracker := &application.EditTracker{Segmenter: wordseg.Segmenter{}, Report: func(e application.EditReport) { reports <- e }}
		field := &benchField{text: "ask jason about it", focused: true}
		t := time.Now()
		tracker.Track(field, "t", contracts.Identity{SessionID: "bench"}, "ask jason about it")
		track = append(track, micros(time.Since(t)))
		time.Sleep(application.EditPoll + 50*time.Millisecond)
		field.mu.Lock()
		field.text = "ask Jayson about it"
		field.mu.Unlock()
		time.Sleep(application.EditPoll + 50*time.Millisecond)
		t = time.Now()
		tracker.Settle()
		settle = append(settle, micros(time.Since(t)))
		<-reports
		settleReport = append(settleReport, millis(time.Since(t)))
	}
	r.TrackUs, r.SettleUs, r.SettleToReport = summarize(track), summarize(settle), summarize(settleReport)

	reports := make(chan application.EditReport, 1)
	tracker := &application.EditTracker{Segmenter: wordseg.Segmenter{}, Report: func(e application.EditReport) { reports <- e }}
	start := time.Now()
	tracker.Track(&benchField{hang: true}, "t", contracts.Identity{}, "x")
	report := <-reports
	r.HungAbandonMs, r.HungReads = millis(time.Since(start)), report.Reads
	r.ReadsPerIdleEdit = float64(application.EditIdle/application.EditPoll) + 1
	return r
}

// concurrencyReport measures recall while memory is being learned and the
// index rebuilt, as the app does after each learned correction.
type concurrencyReport struct {
	Readers         int   `json:"readers"`
	DurationMs      int   `json:"durationMs"`
	IdleApplyUs     stats `json:"idleApplyUs"`
	LoadedApplyUs   stats `json:"underWritesApplyUs"`
	Applies         int64 `json:"appliesUnderWrites"`
	Writes          int64 `json:"writes"`
	Rebuilds        int64 `json:"indexRebuilds"`
	WriteErrors     int64 `json:"writeErrors"`
	FinalEntries    int   `json:"finalEntries"`
	ExpectedEntries int   `json:"expectedEntries"`
}

func measureConcurrency(dir string, texts []string, readers int, duration time.Duration) (concurrencyReport, error) {
	r := concurrencyReport{Readers: readers, DurationMs: int(duration.Milliseconds())}
	store, err := vocabulary.Open(filepath.Join(dir, "concurrent.bin"), bytes.Repeat([]byte{3}, 32))
	if err != nil {
		return r, err
	}
	rng := rand.New(rand.NewSource(11))
	seed := syntheticEntries(1000, nil, rng)
	if _, err := store.Learn(seed, time.Now()); err != nil {
		return r, err
	}
	stored, err := store.List()
	if err != nil {
		return r, err
	}
	var current atomic.Pointer[domain.Vocabulary]
	current.Store(domain.NewVocabulary(domain.Replacements(vocabulary.Memory(stored))))
	sample := func(n int) []float64 {
		var lat []float64
		for i := 0; i < n; i++ {
			t := time.Now()
			current.Load().Apply(texts[i%len(texts)])
			lat = append(lat, micros(time.Since(t)))
		}
		return lat
	}
	r.IdleApplyUs = summarize(sample(20 * len(texts)))

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	var loaded []float64
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var local []float64
			for k := i; ; k++ {
				select {
				case <-stop:
					mu.Lock()
					loaded = append(loaded, local...)
					mu.Unlock()
					return
				default:
				}
				t := time.Now()
				current.Load().Apply(texts[k%len(texts)])
				local = append(local, micros(time.Since(t)))
				atomic.AddInt64(&r.Applies, 1)
			}
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for k := 0; ; k++ {
			select {
			case <-stop:
				return
			default:
			}
			if _, err := store.Learn([]domain.Replacement{{From: fmt.Sprintf("live%05d", k), To: "Live"}}, time.Now()); err != nil {
				atomic.AddInt64(&r.WriteErrors, 1)
				continue
			}
			atomic.AddInt64(&r.Writes, 1)
			entries, err := store.List()
			if err != nil {
				atomic.AddInt64(&r.WriteErrors, 1)
				continue
			}
			current.Store(domain.NewVocabulary(domain.Replacements(vocabulary.Memory(entries))))
			atomic.AddInt64(&r.Rebuilds, 1)
		}
	}()
	time.Sleep(duration)
	close(stop)
	wg.Wait()
	r.LoadedApplyUs = summarize(loaded)
	entries, err := store.List()
	if err != nil {
		return r, err
	}
	r.FinalEntries = len(entries)
	r.ExpectedEntries = min(len(stored)+int(r.Writes), domain.MaxMemoryEntries)
	return r, nil
}

// faultReport injects storage faults; every case must leave dictation
// usable and memory either intact or explicitly unavailable.
type faultReport struct {
	Name           string `json:"name"`
	LearnFailed    bool   `json:"learnFailedExplicitly"`
	FileUnchanged  bool   `json:"fileUnchanged"`
	ListUsable     bool   `json:"listUsable"`
	DictationWorks bool   `json:"dictationCorrectionWorks"`
	Pass           bool   `json:"pass"`
}

func injectFaults(dir string) ([]faultReport, error) {
	key := bytes.Repeat([]byte{5}, 32)
	fresh := func(name string) (*vocabulary.Store, string, error) {
		path := filepath.Join(dir, "fault-"+name, "vocabulary.bin")
		store, err := vocabulary.Open(path, key)
		if err != nil {
			return nil, "", err
		}
		if _, err := store.Add("阿里昂", "Arion"); err != nil {
			return nil, "", err
		}
		return store, path, nil
	}
	// Dictation keeps working with whatever memory loaded, even none.
	dictation := func(store *vocabulary.Store) bool {
		entries, _ := store.List()
		v := domain.NewVocabulary(domain.Replacements(vocabulary.Memory(entries)))
		return domain.Correct("打开阿里昂", domain.CorrectionPolicy{Vocabulary: v}).Text != ""
	}
	var out []faultReport
	for _, name := range []string{"bit-flip", "truncated", "wrong-key", "crash-leftover", "read-only-dir"} {
		store, path, err := fresh(name)
		if err != nil {
			return nil, err
		}
		before, _ := os.ReadFile(path)
		r := faultReport{Name: name}
		switch name {
		case "bit-flip":
			corrupt := append([]byte(nil), before...)
			corrupt[len(corrupt)/2] ^= 0xff
			_ = os.WriteFile(path, corrupt, 0o600)
			before = corrupt
		case "truncated":
			before = before[:5]
			_ = os.WriteFile(path, before, 0o600)
		case "wrong-key":
			store, _ = vocabulary.Open(path, bytes.Repeat([]byte{6}, 32))
		case "crash-leftover":
			_ = os.WriteFile(path+".tmp", []byte("half-written"), 0o600)
		case "read-only-dir":
			_ = os.Chmod(filepath.Dir(path), 0o500)
		}
		_, learnErr := store.Learn([]domain.Replacement{{From: "fault", To: "Fault"}}, time.Now())
		r.LearnFailed = learnErr != nil
		after, _ := os.ReadFile(path)
		r.FileUnchanged = bytes.Equal(before, after)
		_, listErr := store.List()
		r.ListUsable = listErr == nil
		r.DictationWorks = dictation(store)
		switch name {
		case "crash-leftover":
			// The leftover is ignored and replaced; learning succeeds.
			r.Pass = !r.LearnFailed && r.ListUsable && r.DictationWorks
		case "read-only-dir":
			_ = os.Chmod(filepath.Dir(path), 0o700)
			r.Pass = r.LearnFailed && r.FileUnchanged && r.ListUsable && r.DictationWorks
		default:
			r.Pass = r.LearnFailed && r.FileUnchanged && !r.ListUsable && r.DictationWorks
		}
		out = append(out, r)
	}
	return out, nil
}
